package wayland

import (
	"math"
	"sync"
	"time"

	"github.com/AvengeMedia/danklinux/internal/errdefs"
	"github.com/godbus/dbus/v5"
	wlclient "github.com/yaslama/go-wayland/wayland/client"
)

type Config struct {
	Outputs        []string
	LowTemp        int
	HighTemp       int
	Latitude       *float64
	Longitude      *float64
	UseIPLocation  bool
	ManualSunrise  *time.Time
	ManualSunset   *time.Time
	ManualDuration *time.Duration
	Gamma          float64
	Enabled        bool
}

type State struct {
	Config         Config    `json:"config"`
	CurrentTemp    int       `json:"currentTemp"`
	NextTransition time.Time `json:"nextTransition"`
	SunriseTime    time.Time `json:"sunriseTime"`
	SunsetTime     time.Time `json:"sunsetTime"`
	IsDay          bool      `json:"isDay"`
}

type cmd struct {
	fn func()
}

type Manager struct {
	config      Config
	configMutex sync.RWMutex
	state       *State
	stateMutex  sync.RWMutex

	display             *wlclient.Display
	registry            *wlclient.Registry
	gammaControl        interface{}
	availableOutputs    []*wlclient.Output
	outputRegNames      map[uint32]uint32
	outputs             map[uint32]*outputState
	outputsMutex        sync.RWMutex
	controlsInitialized bool

	cmdq  chan cmd
	alive bool

	stopChan      chan struct{}
	updateTicker  *time.Ticker
	updateTrigger chan struct{}
	wg            sync.WaitGroup

	currentTemp      int
	targetTemp       int
	transitionSerial int64
	transitionMutex  sync.RWMutex

	applyTimer *time.Timer

	cachedIPLat   *float64
	cachedIPLon   *float64
	locationMutex sync.RWMutex

	subscribers  map[string]chan State
	subMutex     sync.RWMutex
	dirty        chan struct{}
	notifierWg   sync.WaitGroup
	lastNotified *State

	dbusConn   *dbus.Conn
	dbusSignal chan *dbus.Signal
}

type outputState struct {
	id           uint32
	name         string
	registryName uint32
	output       *wlclient.Output
	gammaControl interface{}
	rampSize     uint32
	failed       bool
	isVirtual    bool
}

type SunTimes struct {
	Sunrise time.Time
	Sunset  time.Time
}

func DefaultConfig() Config {
	return Config{
		Outputs:  []string{},
		LowTemp:  4000,
		HighTemp: 6500,
		Gamma:    1.0,
		Enabled:  false,
	}
}

func (c *Config) Validate() error {
	if c.LowTemp < 1000 || c.LowTemp > 10000 {
		return errdefs.ErrInvalidTemperature
	}
	if c.HighTemp < 1000 || c.HighTemp > 10000 {
		return errdefs.ErrInvalidTemperature
	}
	if c.LowTemp > c.HighTemp {
		return errdefs.ErrInvalidTemperature
	}
	if c.Gamma <= 0 || c.Gamma > 10 {
		return errdefs.ErrInvalidGamma
	}
	if c.Latitude != nil && (math.Abs(*c.Latitude) > 90) {
		return errdefs.ErrInvalidLocation
	}
	if c.Longitude != nil && (math.Abs(*c.Longitude) > 180) {
		return errdefs.ErrInvalidLocation
	}
	if (c.Latitude != nil) != (c.Longitude != nil) {
		return errdefs.ErrInvalidLocation
	}
	if (c.ManualSunrise != nil) != (c.ManualSunset != nil) {
		return errdefs.ErrInvalidManualTimes
	}
	return nil
}

func (m *Manager) GetState() State {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()
	if m.state == nil {
		return State{}
	}
	stateCopy := *m.state
	return stateCopy
}

func (m *Manager) Subscribe(id string) chan State {
	ch := make(chan State, 64)
	m.subMutex.Lock()
	m.subscribers[id] = ch
	m.subMutex.Unlock()
	return ch
}

func (m *Manager) Unsubscribe(id string) {
	m.subMutex.Lock()
	if ch, ok := m.subscribers[id]; ok {
		close(ch)
		delete(m.subscribers, id)
	}
	m.subMutex.Unlock()
}

func (m *Manager) notifySubscribers() {
	select {
	case m.dirty <- struct{}{}:
	default:
	}
}

func stateChanged(old, new *State) bool {
	if old == nil || new == nil {
		return true
	}
	if old.CurrentTemp != new.CurrentTemp {
		return true
	}
	if old.IsDay != new.IsDay {
		return true
	}
	if !old.NextTransition.Equal(new.NextTransition) {
		return true
	}
	if !old.SunriseTime.Equal(new.SunriseTime) {
		return true
	}
	if !old.SunsetTime.Equal(new.SunsetTime) {
		return true
	}
	if old.Config.Enabled != new.Config.Enabled {
		return true
	}
	return false
}

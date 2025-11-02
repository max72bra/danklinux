package brightness

import (
	"sync"
	"time"
)

type DeviceClass string

const (
	ClassBacklight DeviceClass = "backlight"
	ClassLED       DeviceClass = "leds"
	ClassDDC       DeviceClass = "ddc"
)

type Device struct {
	Class          DeviceClass `json:"class"`
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Current        int         `json:"current"`
	Max            int         `json:"max"`
	CurrentPercent int         `json:"currentPercent"`
	Backend        string      `json:"backend"`
}

type State struct {
	Devices []Device `json:"devices"`
}

type Request struct {
	ID     interface{}            `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type Manager struct {
	sysfsBackend *SysfsBackend
	ddcBackend   *DDCBackend

	sysfsReady bool
	ddcReady   bool

	stateMutex sync.RWMutex
	state      State

	subscribers map[string]chan State
	subMutex    sync.RWMutex

	debounceTimers  map[string]*time.Timer
	debouncePending map[string]int
	debounceMutex   sync.Mutex

	stopChan chan struct{}
	wg       sync.WaitGroup
}

type debounceRequest struct {
	deviceID string
	percent  int
}

type backend interface {
	GetDevices() ([]Device, error)
	SetBrightness(id string, percent int) error
	Close()
}

type SysfsBackend struct {
	basePath string
	classes  []string

	deviceCache      map[string]*sysfsDevice
	deviceCacheMutex sync.RWMutex
}

type sysfsDevice struct {
	class         DeviceClass
	id            string
	name          string
	maxBrightness int
	minValue      int
}

type DDCBackend struct {
	devices      map[string]*ddcDevice
	devicesMutex sync.RWMutex

	scanMutex    sync.Mutex
	lastScan     time.Time
	scanInterval time.Duration
}

type ddcDevice struct {
	bus  int
	addr int
	id   string
	name string
}

type ddcCapability struct {
	vcp     byte
	max     int
	current int
}

type SetBrightnessParams struct {
	Device  string `json:"device"`
	Percent int    `json:"percent"`
}

func (m *Manager) Subscribe(id string) chan State {
	ch := make(chan State, 16)
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

func (m *Manager) NotifySubscribers() {
	m.stateMutex.RLock()
	state := m.state
	m.stateMutex.RUnlock()

	m.subMutex.RLock()
	defer m.subMutex.RUnlock()

	for _, ch := range m.subscribers {
		select {
		case ch <- state:
		default:
		}
	}
}

func (m *Manager) GetState() State {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()
	return m.state
}

func (m *Manager) Close() {
	close(m.stopChan)
	m.wg.Wait()

	m.debounceMutex.Lock()
	for _, timer := range m.debounceTimers {
		timer.Stop()
	}
	m.debounceTimers = make(map[string]*time.Timer)
	m.debouncePending = make(map[string]int)
	m.debounceMutex.Unlock()

	m.subMutex.Lock()
	for _, ch := range m.subscribers {
		close(ch)
	}
	m.subscribers = make(map[string]chan State)
	m.subMutex.Unlock()

	if m.ddcBackend != nil {
		m.ddcBackend.Close()
	}
}

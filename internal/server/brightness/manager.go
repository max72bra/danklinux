package brightness

import (
	"fmt"
	"sort"
	"time"

	"github.com/AvengeMedia/danklinux/internal/log"
)

func NewManager() (*Manager, error) {
	m := &Manager{
		subscribers:     make(map[string]chan State),
		debounceTimers:  make(map[string]*time.Timer),
		debouncePending: make(map[string]int),
		stopChan:        make(chan struct{}),
	}

	go m.initSysfs()
	go m.initDDC()

	m.wg.Add(1)
	go m.pollLoop()

	return m, nil
}

func (m *Manager) initSysfs() {
	log.Debug("Initializing sysfs backend...")
	sysfs, err := NewSysfsBackend()
	if err != nil {
		log.Warnf("Failed to initialize sysfs backend: %v", err)
		return
	}

	devices, err := sysfs.GetDevices()
	if err != nil {
		log.Warnf("Failed to get initial sysfs devices: %v", err)
	} else {
		log.Infof("Sysfs backend initialized with %d devices", len(devices))
		for _, d := range devices {
			log.Debugf("  - %s: %s (%d%%)", d.ID, d.Name, d.CurrentPercent)
		}
	}

	m.sysfsBackend = sysfs
	m.sysfsReady = true

	m.updateState()
}

func (m *Manager) initDDC() {
	ddc, err := NewDDCBackend()
	if err != nil {
		log.Debugf("Failed to initialize DDC backend: %v", err)
		return
	}

	m.ddcBackend = ddc
	m.ddcReady = true
	log.Info("DDC backend initialized")

	m.updateState()
}

func (m *Manager) pollLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.updateState()
		}
	}
}

func sortDevices(devices []Device) {
	sort.Slice(devices, func(i, j int) bool {
		classOrder := map[DeviceClass]int{
			ClassBacklight: 0,
			ClassDDC:       1,
			ClassLED:       2,
		}

		orderI := classOrder[devices[i].Class]
		orderJ := classOrder[devices[j].Class]

		if orderI != orderJ {
			return orderI < orderJ
		}

		return devices[i].Name < devices[j].Name
	})
}

func stateChanged(old, new State) bool {
	if len(old.Devices) != len(new.Devices) {
		return true
	}

	oldMap := make(map[string]Device)
	for _, d := range old.Devices {
		oldMap[d.ID] = d
	}

	for _, newDev := range new.Devices {
		oldDev, exists := oldMap[newDev.ID]
		if !exists {
			return true
		}
		if oldDev.Current != newDev.Current || oldDev.Max != newDev.Max {
			return true
		}
	}

	return false
}

func (m *Manager) updateState() {
	allDevices := make([]Device, 0)

	if m.sysfsReady && m.sysfsBackend != nil {
		devices, err := m.sysfsBackend.GetDevices()
		if err != nil {
			log.Debugf("Failed to get sysfs devices: %v", err)
		} else {
			allDevices = append(allDevices, devices...)
		}
	}

	if m.ddcReady && m.ddcBackend != nil {
		devices, err := m.ddcBackend.GetDevices()
		if err != nil {
			log.Debugf("Failed to get DDC devices: %v", err)
		} else {
			allDevices = append(allDevices, devices...)
		}
	}

	sortDevices(allDevices)

	m.stateMutex.Lock()
	oldState := m.state
	newState := State{Devices: allDevices}

	if stateChanged(oldState, newState) {
		m.state = newState
		m.stateMutex.Unlock()
		m.NotifySubscribers()
	} else {
		m.stateMutex.Unlock()
	}
}

func (m *Manager) SetBrightness(deviceID string, percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("percent out of range: %d", percent)
	}

	isDDC := false
	isSysfs := false

	if m.sysfsBackend != nil {
		devices, _ := m.sysfsBackend.GetDevices()
		for _, dev := range devices {
			if dev.ID == deviceID {
				isSysfs = true
				break
			}
		}
	}

	if !isSysfs && m.ddcBackend != nil {
		devices, _ := m.ddcBackend.GetDevices()
		for _, dev := range devices {
			if dev.ID == deviceID {
				isDDC = true
				break
			}
		}
	}

	if !isSysfs && !isDDC {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	debounceDelay := 50 * time.Millisecond
	if isDDC {
		debounceDelay = 200 * time.Millisecond
	}

	m.debounceMutex.Lock()
	defer m.debounceMutex.Unlock()

	m.debouncePending[deviceID] = percent

	if timer, exists := m.debounceTimers[deviceID]; exists {
		timer.Stop()
	}

	m.debounceTimers[deviceID] = time.AfterFunc(debounceDelay, func() {
		m.debounceMutex.Lock()
		pendingPercent, exists := m.debouncePending[deviceID]
		if exists {
			delete(m.debouncePending, deviceID)
		}
		m.debounceMutex.Unlock()

		if !exists {
			return
		}

		var err error
		if isSysfs {
			err = m.sysfsBackend.SetBrightness(deviceID, pendingPercent)
		} else if isDDC {
			err = m.ddcBackend.SetBrightness(deviceID, pendingPercent)
		}

		if err != nil {
			log.Debugf("Failed to set brightness for %s: %v", deviceID, err)
		}

		m.updateState()
	})

	return nil
}

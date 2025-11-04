package brightness

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AvengeMedia/danklinux/internal/log"
)

func NewManager() (*Manager, error) {
	m := &Manager{
		subscribers:       make(map[string]chan State),
		updateSubscribers: make(map[string]chan DeviceUpdate),
		stopChan:          make(chan struct{}),
	}

	go m.initLogind()
	go m.initSysfs()
	go m.initDDC()

	return m, nil
}

func (m *Manager) initLogind() {
	log.Debug("Initializing logind backend...")
	logind, err := NewLogindBackend()
	if err != nil {
		log.Infof("Logind backend not available: %v", err)
		log.Info("Will use direct sysfs access for brightness control")
		return
	}

	m.logindBackend = logind
	m.logindReady = true
	log.Info("Logind backend initialized - will use for brightness control")
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
		m.sysfsBackend = sysfs
		m.sysfsReady = true
		m.updateState()
		return
	}

	log.Infof("Sysfs backend initialized with %d devices", len(devices))
	for _, d := range devices {
		log.Debugf("  - %s: %s (%d%%)", d.ID, d.Name, d.CurrentPercent)
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

func (m *Manager) Rescan() {
	log.Debug("Rescanning brightness devices...")
	m.updateState()
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
		}
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
	}

	if m.ddcReady && m.ddcBackend != nil {
		devices, err := m.ddcBackend.GetDevices()
		if err != nil {
			log.Debugf("Failed to get DDC devices: %v", err)
		}
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
	}

	sortDevices(allDevices)

	m.stateMutex.Lock()
	oldState := m.state
	newState := State{Devices: allDevices}

	if !stateChanged(oldState, newState) {
		m.stateMutex.Unlock()
		return
	}

	m.state = newState
	m.stateMutex.Unlock()
	log.Debugf("State changed, notifying subscribers")
	m.NotifySubscribers()
}

func (m *Manager) SetBrightness(deviceID string, percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("percent out of range: %d", percent)
	}

	log.Debugf("SetBrightness: %s to %d%%", deviceID, percent)

	m.stateMutex.Lock()
	currentState := m.state
	var found bool
	var deviceClass DeviceClass
	var deviceIndex int

	log.Debugf("Current state has %d devices", len(currentState.Devices))

	for i, dev := range currentState.Devices {
		if dev.ID == deviceID {
			found = true
			deviceClass = dev.Class
			deviceIndex = i
			break
		}
	}

	if !found {
		m.stateMutex.Unlock()
		log.Debugf("Device not found in state: %s", deviceID)
		return fmt.Errorf("device not found: %s", deviceID)
	}

	log.Debugf("Updating cached state for %s from %d%% to %d%%", deviceID, m.state.Devices[deviceIndex].CurrentPercent, percent)
	m.state.Devices[deviceIndex].CurrentPercent = percent
	m.stateMutex.Unlock()

	var err error
	if deviceClass == ClassDDC {
		log.Debugf("Calling DDC backend for %s", deviceID)
		err = m.ddcBackend.SetBrightness(deviceID, percent)
	} else if m.logindReady && m.logindBackend != nil {
		log.Debugf("Calling logind backend for %s", deviceID)
		err = m.setViaSysfsWithLogind(deviceID, percent)
	} else {
		log.Debugf("Calling sysfs backend for %s", deviceID)
		err = m.sysfsBackend.SetBrightness(deviceID, percent)
	}

	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	log.Debugf("Queueing broadcast for %s", deviceID)
	m.debouncedBroadcast(deviceID, deviceClass, percent)
	return nil
}

func (m *Manager) IncrementBrightness(deviceID string, step int) error {
	m.stateMutex.RLock()
	currentState := m.state
	m.stateMutex.RUnlock()

	var currentPercent int
	var found bool

	for _, dev := range currentState.Devices {
		if dev.ID == deviceID {
			currentPercent = dev.CurrentPercent
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	newPercent := currentPercent + step
	if newPercent > 100 {
		newPercent = 100
	}
	if newPercent < 0 {
		newPercent = 0
	}

	return m.SetBrightness(deviceID, newPercent)
}

func (m *Manager) DecrementBrightness(deviceID string, step int) error {
	return m.IncrementBrightness(deviceID, -step)
}

func (m *Manager) setViaSysfsWithLogind(deviceID string, percent int) error {
	parts := strings.SplitN(deviceID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid device id: %s", deviceID)
	}

	subsystem := parts[0]
	name := parts[1]

	dev, err := m.sysfsBackend.getDevice(deviceID)
	if err != nil {
		return err
	}

	value := m.sysfsBackend.percentToValue(percent, dev)

	if m.logindBackend == nil {
		return m.sysfsBackend.SetBrightness(deviceID, percent)
	}

	err = m.logindBackend.SetBrightness(subsystem, name, uint32(value))
	if err != nil {
		log.Debugf("logind SetBrightness failed, falling back to direct sysfs: %v", err)
		return m.sysfsBackend.SetBrightness(deviceID, percent)
	}

	log.Debugf("set %s to %d%% (%d/%d) via logind", deviceID, percent, value, dev.maxBrightness)
	return nil
}

func (m *Manager) debouncedBroadcast(deviceID string, deviceClass DeviceClass, percent int) {
	m.broadcastMutex.Lock()
	defer m.broadcastMutex.Unlock()

	m.broadcastPending = true
	m.pendingDeviceID = deviceID

	if m.broadcastTimer != nil {
		m.broadcastTimer.Stop()
	}

	m.broadcastTimer = time.AfterFunc(150*time.Millisecond, func() {
		m.broadcastMutex.Lock()
		pending := m.broadcastPending
		deviceID := m.pendingDeviceID
		m.broadcastPending = false
		m.pendingDeviceID = ""
		m.broadcastMutex.Unlock()

		if !pending || deviceID == "" {
			return
		}

		m.broadcastDeviceUpdate(deviceID)
	})
}

func (m *Manager) broadcastDeviceUpdate(deviceID string) {
	m.stateMutex.RLock()
	var targetDevice *Device
	for _, dev := range m.state.Devices {
		if dev.ID == deviceID {
			devCopy := dev
			targetDevice = &devCopy
			break
		}
	}
	m.stateMutex.RUnlock()

	if targetDevice == nil {
		log.Debugf("Device not found for broadcast: %s", deviceID)
		return
	}

	update := DeviceUpdate{Device: *targetDevice}

	m.subMutex.RLock()
	defer m.subMutex.RUnlock()

	if len(m.updateSubscribers) == 0 {
		log.Debugf("No update subscribers for device: %s", deviceID)
		return
	}

	log.Debugf("Broadcasting device update: %s at %d%%", deviceID, targetDevice.CurrentPercent)

	for _, ch := range m.updateSubscribers {
		select {
		case ch <- update:
		default:
		}
	}
}

package brightness

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AvengeMedia/danklinux/internal/log"
)

func NewSysfsBackend() (*SysfsBackend, error) {
	b := &SysfsBackend{
		basePath:    "/sys/class",
		classes:     []string{"backlight", "leds"},
		deviceCache: make(map[string]*sysfsDevice),
	}

	logind, err := NewLogindBackend()
	if err != nil {
		log.Debugf("logind backend not available: %v", err)
	} else {
		b.logind = logind
		log.Debug("logind backend available for brightness control")
	}

	if err := b.scanDevices(); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *SysfsBackend) scanDevices() error {
	b.deviceCacheMutex.Lock()
	defer b.deviceCacheMutex.Unlock()

	for _, class := range b.classes {
		classPath := filepath.Join(b.basePath, class)
		entries, err := os.ReadDir(classPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", classPath, err)
		}

		for _, entry := range entries {
			devicePath := filepath.Join(classPath, entry.Name())

			stat, err := os.Stat(devicePath)
			if err != nil || !stat.IsDir() {
				continue
			}
			maxPath := filepath.Join(devicePath, "max_brightness")

			maxData, err := os.ReadFile(maxPath)
			if err != nil {
				log.Debugf("skip %s/%s: no max_brightness", class, entry.Name())
				continue
			}

			maxBrightness, err := strconv.Atoi(strings.TrimSpace(string(maxData)))
			if err != nil || maxBrightness <= 0 {
				log.Debugf("skip %s/%s: invalid max_brightness", class, entry.Name())
				continue
			}

			deviceClass := ClassBacklight
			minValue := 1
			if class == "leds" {
				deviceClass = ClassLED
				minValue = 0
			}

			deviceID := fmt.Sprintf("%s:%s", class, entry.Name())
			b.deviceCache[deviceID] = &sysfsDevice{
				class:         deviceClass,
				id:            deviceID,
				name:          entry.Name(),
				maxBrightness: maxBrightness,
				minValue:      minValue,
			}

			log.Debugf("found %s device: %s (max=%d)", class, entry.Name(), maxBrightness)
		}
	}

	return nil
}

func (b *SysfsBackend) GetDevices() ([]Device, error) {
	b.deviceCacheMutex.RLock()
	defer b.deviceCacheMutex.RUnlock()

	devices := make([]Device, 0, len(b.deviceCache))

	for _, dev := range b.deviceCache {
		parts := strings.SplitN(dev.id, ":", 2)
		if len(parts) != 2 {
			continue
		}

		class := parts[0]
		name := parts[1]

		devicePath := filepath.Join(b.basePath, class, name)
		brightnessPath := filepath.Join(devicePath, "brightness")

		brightnessData, err := os.ReadFile(brightnessPath)
		if err != nil {
			log.Debugf("failed to read brightness for %s: %v", dev.id, err)
			continue
		}

		current, err := strconv.Atoi(strings.TrimSpace(string(brightnessData)))
		if err != nil {
			log.Debugf("failed to parse brightness for %s: %v", dev.id, err)
			continue
		}

		percent := b.valueToPercent(current, dev)

		devices = append(devices, Device{
			Class:          dev.class,
			ID:             dev.id,
			Name:           dev.name,
			Current:        current,
			Max:            dev.maxBrightness,
			CurrentPercent: percent,
			Backend:        "sysfs",
		})
	}

	return devices, nil
}

func (b *SysfsBackend) SetBrightness(id string, percent int) error {
	b.deviceCacheMutex.RLock()
	dev, ok := b.deviceCache[id]
	b.deviceCacheMutex.RUnlock()

	if !ok {
		return fmt.Errorf("device not found: %s", id)
	}

	if percent < 0 || percent > 100 {
		return fmt.Errorf("percent out of range: %d", percent)
	}

	value := b.percentToValue(percent, dev)

	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid device id: %s", id)
	}

	class := parts[0]
	name := parts[1]

	if b.logind != nil {
		if err := b.logind.SetBrightness(class, name, uint32(value)); err != nil {
			log.Debugf("logind SetBrightness failed, falling back to direct write: %v", err)
		} else {
			log.Debugf("set %s to %d%% (%d/%d) via logind", id, percent, value, dev.maxBrightness)
			return nil
		}
	}

	devicePath := filepath.Join(b.basePath, class, name)
	brightnessPath := filepath.Join(devicePath, "brightness")

	data := []byte(fmt.Sprintf("%d", value))
	if err := os.WriteFile(brightnessPath, data, 0644); err != nil {
		return fmt.Errorf("write brightness: %w", err)
	}

	log.Debugf("set %s to %d%% (%d/%d) via direct sysfs", id, percent, value, dev.maxBrightness)

	return nil
}

func (b *SysfsBackend) percentToValue(percent int, dev *sysfsDevice) int {
	// LEDs can go to 0, backlight devices must stay at minimum 1
	if percent == 0 {
		return dev.minValue
	}

	// Map 1-100% to minValue-maxBrightness range
	usableRange := dev.maxBrightness - dev.minValue
	value := dev.minValue + ((percent - 1) * usableRange / 99)

	if value < dev.minValue {
		value = dev.minValue
	}
	if value > dev.maxBrightness {
		value = dev.maxBrightness
	}

	return value
}

func (b *SysfsBackend) valueToPercent(value int, dev *sysfsDevice) int {
	// Handle minimum values
	if value <= dev.minValue {
		if dev.minValue == 0 && value == 0 {
			return 0
		}
		return 1
	}

	// Map minValue-maxBrightness range to 1-100%
	usableRange := dev.maxBrightness - dev.minValue
	if usableRange == 0 {
		return 100
	}

	percent := 1 + ((value - dev.minValue) * 99 / usableRange)

	if percent > 100 {
		percent = 100
	}
	if percent < 1 {
		percent = 1
	}

	return percent
}

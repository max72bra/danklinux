package brightness

import (
	"fmt"
	"math"
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

		percent := b.ValueToPercent(current, dev, false)

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

func (b *SysfsBackend) GetDevice(id string) (*sysfsDevice, error) {
	b.deviceCacheMutex.RLock()
	defer b.deviceCacheMutex.RUnlock()

	dev, ok := b.deviceCache[id]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", id)
	}

	return dev, nil
}

func (b *SysfsBackend) SetBrightness(id string, percent int, logarithmic bool) error {
	dev, err := b.GetDevice(id)
	if err != nil {
		return err
	}

	if percent < 0 || percent > 100 {
		return fmt.Errorf("percent out of range: %d", percent)
	}

	value := b.PercentToValue(percent, dev, logarithmic)

	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid device id: %s", id)
	}

	class := parts[0]
	name := parts[1]

	devicePath := filepath.Join(b.basePath, class, name)
	brightnessPath := filepath.Join(devicePath, "brightness")

	data := []byte(fmt.Sprintf("%d", value))
	if err := os.WriteFile(brightnessPath, data, 0644); err != nil {
		return fmt.Errorf("write brightness: %w", err)
	}

	log.Debugf("set %s to %d%% (%d/%d) via direct sysfs", id, percent, value, dev.maxBrightness)

	return nil
}

func (b *SysfsBackend) PercentToValue(percent int, dev *sysfsDevice, logarithmic bool) int {
	if percent == 0 {
		return dev.minValue
	}

	usableRange := dev.maxBrightness - dev.minValue
	var value int

	if logarithmic {
		const exponent = 2.0
		normalizedPercent := float64(percent) / 100.0
		hardwarePercent := math.Pow(normalizedPercent, 1.0/exponent)
		value = dev.minValue + int(math.Round(hardwarePercent*float64(usableRange)))
	} else {
		value = dev.minValue + ((percent - 1) * usableRange / 99)
	}

	if value < dev.minValue {
		value = dev.minValue
	}
	if value > dev.maxBrightness {
		value = dev.maxBrightness
	}

	return value
}

func (b *SysfsBackend) ValueToPercent(value int, dev *sysfsDevice, logarithmic bool) int {
	if value <= dev.minValue {
		if dev.minValue == 0 && value == 0 {
			return 0
		}
		return 1
	}

	usableRange := dev.maxBrightness - dev.minValue
	if usableRange == 0 {
		return 100
	}

	var percent int

	if logarithmic {
		const exponent = 2.0
		linearPercent := 1 + ((value - dev.minValue) * 99 / usableRange)
		normalizedLinear := float64(linearPercent) / 100.0
		logPercent := math.Pow(normalizedLinear, exponent)
		percent = int(math.Round(logPercent * 100.0))
	} else {
		percent = 1 + ((value - dev.minValue) * 99 / usableRange)
	}

	if percent > 100 {
		percent = 100
	}
	if percent < 1 {
		percent = 1
	}

	return percent
}

package brightness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSysfsBackend_PercentConversions(t *testing.T) {
	tests := []struct {
		name      string
		device    *sysfsDevice
		percent   int
		wantValue int
	}{
		{
			name:      "backlight 0% should be 1",
			device:    &sysfsDevice{maxBrightness: 100, minValue: 1, class: ClassBacklight},
			percent:   0,
			wantValue: 1,
		},
		{
			name:      "backlight 50%",
			device:    &sysfsDevice{maxBrightness: 100, minValue: 1, class: ClassBacklight},
			percent:   50,
			wantValue: 50,
		},
		{
			name:      "backlight 100%",
			device:    &sysfsDevice{maxBrightness: 100, minValue: 1, class: ClassBacklight},
			percent:   100,
			wantValue: 100,
		},
		{
			name:      "led 0% should be 0",
			device:    &sysfsDevice{maxBrightness: 255, minValue: 0, class: ClassLED},
			percent:   0,
			wantValue: 0,
		},
		{
			name:      "led 50%",
			device:    &sysfsDevice{maxBrightness: 255, minValue: 0, class: ClassLED},
			percent:   50,
			wantValue: 127,
		},
	}

	b := &SysfsBackend{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.percentToValue(tt.percent, tt.device)
			if got != tt.wantValue {
				t.Errorf("percentToValue() = %v, want %v", got, tt.wantValue)
			}

			gotPercent := b.valueToPercent(got, tt.device)
			if tt.percent > 0 && gotPercent == 0 {
				t.Errorf("valueToPercent() returned 0 for non-zero input")
			}
		})
	}
}

func TestSysfsBackend_ValueToPercent(t *testing.T) {
	tests := []struct {
		name        string
		device      *sysfsDevice
		value       int
		wantPercent int
	}{
		{
			name:        "backlight min value",
			device:      &sysfsDevice{maxBrightness: 100, minValue: 1, class: ClassBacklight},
			value:       1,
			wantPercent: 1,
		},
		{
			name:        "backlight max value",
			device:      &sysfsDevice{maxBrightness: 100, minValue: 1, class: ClassBacklight},
			value:       100,
			wantPercent: 100,
		},
		{
			name:        "led zero",
			device:      &sysfsDevice{maxBrightness: 255, minValue: 0, class: ClassLED},
			value:       0,
			wantPercent: 0,
		},
	}

	b := &SysfsBackend{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.valueToPercent(tt.value, tt.device)
			if got != tt.wantPercent {
				t.Errorf("valueToPercent() = %v, want %v", got, tt.wantPercent)
			}
		})
	}
}

func TestSysfsBackend_ScanDevices(t *testing.T) {
	tmpDir := t.TempDir()

	backlightDir := filepath.Join(tmpDir, "backlight", "test_backlight")
	if err := os.MkdirAll(backlightDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(backlightDir, "max_brightness"), []byte("100\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlightDir, "brightness"), []byte("50\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ledsDir := filepath.Join(tmpDir, "leds", "test_led")
	if err := os.MkdirAll(ledsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(ledsDir, "max_brightness"), []byte("255\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ledsDir, "brightness"), []byte("128\n"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &SysfsBackend{
		basePath:    tmpDir,
		classes:     []string{"backlight", "leds"},
		deviceCache: make(map[string]*sysfsDevice),
	}

	if err := b.scanDevices(); err != nil {
		t.Fatalf("scanDevices() error = %v", err)
	}

	if len(b.deviceCache) != 2 {
		t.Errorf("expected 2 devices, got %d", len(b.deviceCache))
	}

	backlightID := "backlight:test_backlight"
	if _, ok := b.deviceCache[backlightID]; !ok {
		t.Errorf("backlight device not found")
	}

	ledID := "leds:test_led"
	if _, ok := b.deviceCache[ledID]; !ok {
		t.Errorf("LED device not found")
	}
}

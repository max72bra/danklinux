package brightness

import (
	"os"
	"path/filepath"
	"testing"

	mocks_brightness "github.com/AvengeMedia/danklinux/internal/mocks/brightness"
	mock_dbus "github.com/AvengeMedia/danklinux/internal/mocks/github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/mock"
)

func TestSysfsBackend_SetBrightness_LogindSuccess(t *testing.T) {
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

	mockConn := mocks_brightness.NewMockDBusConn(t)
	mockObj := mock_dbus.NewMockBusObject(t)

	mockLogind := NewLogindBackendWithConn(mockConn)

	b := &SysfsBackend{
		basePath:    tmpDir,
		classes:     []string{"backlight"},
		deviceCache: make(map[string]*sysfsDevice),
		logind:      mockLogind,
	}

	if err := b.scanDevices(); err != nil {
		t.Fatal(err)
	}

	mockConn.EXPECT().
		Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1/session/auto")).
		Return(mockObj).
		Once()

	mockObj.EXPECT().
		Call("org.freedesktop.login1.Session.SetBrightness", mock.Anything, "backlight", "test_backlight", uint32(75)).
		Return(&dbus.Call{Err: nil}).
		Once()

	err := b.SetBrightness("backlight:test_backlight", 75)
	if err != nil {
		t.Errorf("SetBrightness() with logind error = %v, want nil", err)
	}

	data, _ := os.ReadFile(filepath.Join(backlightDir, "brightness"))
	if string(data) == "75\n" {
		t.Error("Direct sysfs write occurred when logind should have been used")
	}
}

func TestSysfsBackend_SetBrightness_LogindFailsFallbackToSysfs(t *testing.T) {
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

	mockConn := mocks_brightness.NewMockDBusConn(t)
	mockObj := mock_dbus.NewMockBusObject(t)

	mockLogind := NewLogindBackendWithConn(mockConn)

	b := &SysfsBackend{
		basePath:    tmpDir,
		classes:     []string{"backlight"},
		deviceCache: make(map[string]*sysfsDevice),
		logind:      mockLogind,
	}

	if err := b.scanDevices(); err != nil {
		t.Fatal(err)
	}

	mockConn.EXPECT().
		Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1/session/auto")).
		Return(mockObj).
		Once()

	mockObj.EXPECT().
		Call("org.freedesktop.login1.Session.SetBrightness", mock.Anything, "backlight", "test_backlight", mock.Anything).
		Return(&dbus.Call{Err: dbus.ErrMsgNoObject}).
		Once()

	err := b.SetBrightness("backlight:test_backlight", 75)
	if err != nil {
		t.Errorf("SetBrightness() with fallback error = %v, want nil", err)
	}

	data, _ := os.ReadFile(filepath.Join(backlightDir, "brightness"))
	brightness := string(data)
	if brightness != "75" {
		t.Errorf("Fallback sysfs write did not occur, got brightness = %q, want %q", brightness, "75")
	}
}

func TestSysfsBackend_SetBrightness_NoLogind(t *testing.T) {
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

	b := &SysfsBackend{
		basePath:    tmpDir,
		classes:     []string{"backlight"},
		deviceCache: make(map[string]*sysfsDevice),
		logind:      nil,
	}

	if err := b.scanDevices(); err != nil {
		t.Fatal(err)
	}

	err := b.SetBrightness("backlight:test_backlight", 75)
	if err != nil {
		t.Errorf("SetBrightness() without logind error = %v, want nil", err)
	}

	data, _ := os.ReadFile(filepath.Join(backlightDir, "brightness"))
	brightness := string(data)
	if brightness != "75" {
		t.Errorf("Direct sysfs write = %q, want %q", brightness, "75")
	}
}

func TestSysfsBackend_SetBrightness_LEDWithLogind(t *testing.T) {
	tmpDir := t.TempDir()

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

	mockConn := mocks_brightness.NewMockDBusConn(t)
	mockObj := mock_dbus.NewMockBusObject(t)

	mockLogind := NewLogindBackendWithConn(mockConn)

	b := &SysfsBackend{
		basePath:    tmpDir,
		classes:     []string{"leds"},
		deviceCache: make(map[string]*sysfsDevice),
		logind:      mockLogind,
	}

	if err := b.scanDevices(); err != nil {
		t.Fatal(err)
	}

	mockConn.EXPECT().
		Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1/session/auto")).
		Return(mockObj).
		Once()

	mockObj.EXPECT().
		Call("org.freedesktop.login1.Session.SetBrightness", mock.Anything, "leds", "test_led", uint32(0)).
		Return(&dbus.Call{Err: nil}).
		Once()

	err := b.SetBrightness("leds:test_led", 0)
	if err != nil {
		t.Errorf("SetBrightness() LED with logind error = %v, want nil", err)
	}
}

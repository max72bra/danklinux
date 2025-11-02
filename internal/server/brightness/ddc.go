package brightness

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/AvengeMedia/danklinux/internal/log"
)

const (
	I2C_SLAVE      = 0x0703
	DDCCI_ADDR     = 0x37
	DDCCI_VCP_GET  = 0x01
	DDCCI_VCP_SET  = 0x03
	VCP_BRIGHTNESS = 0x10
)

func NewDDCBackend() (*DDCBackend, error) {
	b := &DDCBackend{
		devices:      make(map[string]*ddcDevice),
		scanInterval: 30 * time.Second,
	}

	if err := b.scanI2CDevices(); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *DDCBackend) scanI2CDevices() error {
	b.scanMutex.Lock()
	defer b.scanMutex.Unlock()

	if time.Since(b.lastScan) < b.scanInterval {
		return nil
	}

	b.devicesMutex.Lock()
	defer b.devicesMutex.Unlock()

	b.devices = make(map[string]*ddcDevice)

	for i := 0; i < 32; i++ {
		busPath := fmt.Sprintf("/dev/i2c-%d", i)
		if _, err := os.Stat(busPath); os.IsNotExist(err) {
			continue
		}

		if dev, err := b.probeDDCDevice(i); err == nil && dev != nil {
			id := fmt.Sprintf("ddc:i2c-%d", i)
			dev.id = id
			b.devices[id] = dev
			log.Debugf("found DDC device on i2c-%d", i)
		}
	}

	b.lastScan = time.Now()

	return nil
}

func (b *DDCBackend) probeDDCDevice(bus int) (*ddcDevice, error) {
	busPath := fmt.Sprintf("/dev/i2c-%d", bus)

	fd, err := syscall.Open(busPath, syscall.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), I2C_SLAVE, uintptr(DDCCI_ADDR)); errno != 0 {
		return nil, errno
	}

	cap, err := b.getVCPFeature(fd, VCP_BRIGHTNESS)
	if err != nil {
		return nil, err
	}

	if cap.max == 0 {
		return nil, fmt.Errorf("invalid max brightness")
	}

	name := b.getDDCName(bus)

	return &ddcDevice{
		bus:  bus,
		addr: DDCCI_ADDR,
		name: name,
	}, nil
}

func (b *DDCBackend) getDDCName(bus int) string {
	sysfsPath := fmt.Sprintf("/sys/class/i2c-adapter/i2c-%d/name", bus)
	data, err := os.ReadFile(sysfsPath)
	if err != nil {
		return fmt.Sprintf("I2C-%d", bus)
	}

	name := strings.TrimSpace(string(data))
	if name == "" {
		name = fmt.Sprintf("I2C-%d", bus)
	}

	return name
}

func (b *DDCBackend) GetDevices() ([]Device, error) {
	if err := b.scanI2CDevices(); err != nil {
		log.Debugf("DDC scan error: %v", err)
	}

	b.devicesMutex.RLock()
	defer b.devicesMutex.RUnlock()

	devices := make([]Device, 0, len(b.devices))

	for id, dev := range b.devices {
		busPath := fmt.Sprintf("/dev/i2c-%d", dev.bus)

		fd, err := syscall.Open(busPath, syscall.O_RDWR, 0)
		if err != nil {
			log.Debugf("failed to open %s: %v", busPath, err)
			continue
		}

		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), I2C_SLAVE, uintptr(dev.addr)); errno != 0 {
			syscall.Close(fd)
			log.Debugf("failed to set i2c slave addr for %s: %v", id, errno)
			continue
		}

		cap, err := b.getVCPFeature(fd, VCP_BRIGHTNESS)
		syscall.Close(fd)

		if err != nil {
			log.Debugf("failed to get brightness for %s: %v", id, err)
			continue
		}

		percent := b.valueToPercent(cap.current, cap.max)

		devices = append(devices, Device{
			Class:          ClassDDC,
			ID:             id,
			Name:           dev.name,
			Current:        cap.current,
			Max:            cap.max,
			CurrentPercent: percent,
			Backend:        "ddc",
		})
	}

	return devices, nil
}

func (b *DDCBackend) SetBrightness(id string, percent int) error {
	b.devicesMutex.RLock()
	dev, ok := b.devices[id]
	b.devicesMutex.RUnlock()

	if !ok {
		return fmt.Errorf("device not found: %s", id)
	}

	if percent < 1 || percent > 100 {
		return fmt.Errorf("percent out of range: %d", percent)
	}

	busPath := fmt.Sprintf("/dev/i2c-%d", dev.bus)

	fd, err := syscall.Open(busPath, syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open i2c device: %w", err)
	}
	defer syscall.Close(fd)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), I2C_SLAVE, uintptr(dev.addr)); errno != 0 {
		return fmt.Errorf("set i2c slave addr: %w", errno)
	}

	cap, err := b.getVCPFeature(fd, VCP_BRIGHTNESS)
	if err != nil {
		return fmt.Errorf("get current brightness: %w", err)
	}

	value := b.percentToValue(percent, cap.max)

	if err := b.setVCPFeature(fd, VCP_BRIGHTNESS, value); err != nil {
		return fmt.Errorf("set vcp feature: %w", err)
	}

	log.Debugf("set %s to %d%% (%d/%d)", id, percent, value, cap.max)

	return nil
}

func (b *DDCBackend) getVCPFeature(fd int, vcp byte) (*ddcCapability, error) {
	request := []byte{
		0x6E | 0x80,
		0x51,
		0x82,
		vcp,
	}

	checksum := byte(DDCCI_ADDR << 1)
	for _, b := range request {
		checksum ^= b
	}
	request = append(request, checksum)

	n, err := syscall.Write(fd, request)
	if err != nil || n != len(request) {
		return nil, fmt.Errorf("write i2c: %w", err)
	}

	time.Sleep(40 * time.Millisecond)

	response := make([]byte, 12)
	n, err = syscall.Read(fd, response)
	if err != nil || n < 8 {
		return nil, fmt.Errorf("read i2c: %w", err)
	}

	if response[0] != 0x6E || response[2] != 0x02 {
		return nil, fmt.Errorf("invalid ddc response")
	}

	resultCode := response[3]
	if resultCode != 0x00 {
		return nil, fmt.Errorf("vcp feature not supported")
	}

	responseVCP := response[4]
	if responseVCP != vcp {
		return nil, fmt.Errorf("vcp mismatch")
	}

	maxHigh := response[5]
	maxLow := response[6]
	currentHigh := response[7]
	currentLow := response[8]

	max := int(binary.BigEndian.Uint16([]byte{maxHigh, maxLow}))
	current := int(binary.BigEndian.Uint16([]byte{currentHigh, currentLow}))

	return &ddcCapability{
		vcp:     vcp,
		max:     max,
		current: current,
	}, nil
}

func (b *DDCBackend) setVCPFeature(fd int, vcp byte, value int) error {
	valueHigh := byte((value >> 8) & 0xFF)
	valueLow := byte(value & 0xFF)

	request := []byte{
		0x6E | 0x80,
		0x51,
		0x84,
		vcp,
		valueHigh,
		valueLow,
	}

	checksum := byte(DDCCI_ADDR << 1)
	for _, b := range request {
		checksum ^= b
	}
	request = append(request, checksum)

	n, err := syscall.Write(fd, request)
	if err != nil || n != len(request) {
		return fmt.Errorf("write i2c: %w", err)
	}

	time.Sleep(40 * time.Millisecond)

	return nil
}

func (b *DDCBackend) percentToValue(percent int, max int) int {
	const minValue = 1

	usableRange := max - minValue
	value := minValue + (percent * usableRange / 100)

	if value < minValue {
		value = minValue
	}
	if value > max {
		value = max
	}

	return value
}

func (b *DDCBackend) valueToPercent(value int, max int) int {
	const minValue = 1

	if value <= minValue {
		return 1
	}

	usableRange := max - minValue
	if usableRange == 0 {
		return 100
	}

	percent := ((value - minValue) * 100) / usableRange

	if percent > 100 {
		percent = 100
	}
	if percent < 1 {
		percent = 1
	}

	return percent
}

func (b *DDCBackend) Close() {
}

var _ = unsafe.Sizeof(0)
var _ = filepath.Join

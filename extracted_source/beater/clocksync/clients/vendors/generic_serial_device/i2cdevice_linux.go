//go:build linux

package generic_serial_device

import (
	"fmt"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
)

// Проверка на этапе компиляции: *I2CDevice реализует I2CDeviceForRegister.
var _ I2CDeviceForRegister = (*I2CDevice)(nil)

// Lock блокирует устройство для RegisterRequest.Execute (по дампу Execute.func1 → Unlock).
func (d *I2CDevice) Lock() {
	if d != nil {
		d.Mu.Lock()
	}
}

// Unlock разблокирует устройство (по дампу I2CDevice.Unlock).
func (d *I2CDevice) Unlock() {
	if d != nil {
		d.Mu.Unlock()
	}
}

const writeThenReadSleepMs = 5

// WriteThenRead по дампу (0x4b8d660): makeslice(readLen), Sleep, Tx(addr, write, read).
func (d *I2CDevice) WriteThenRead(write []byte, readLen int) ([]byte, error) {
	dev := d.i2cDev()
	if dev == nil {
		return nil, fmt.Errorf("no I2C device")
	}
	readBuf := make([]byte, readLen)
	time.Sleep(writeThenReadSleepMs * time.Millisecond)
	if err := dev.Tx(write, readBuf); err != nil {
		return nil, err
	}
	return readBuf, nil
}

// Write по дампу Set100MbpsNoAutoNegotiation (0x4b8d780): Tx(write, nil) — только запись.
func (d *I2CDevice) Write(data []byte) error {
	dev := d.i2cDev()
	if dev == nil {
		return fmt.Errorf("no I2C device")
	}
	return dev.Tx(data, nil)
}

// i2cDev возвращает *i2c.Dev из d.Dev (только на Linux).
func (d *I2CDevice) i2cDev() *i2c.Dev {
	if d == nil || d.Dev == nil {
		return nil
	}
	dev, _ := d.Dev.(*i2c.Dev)
	return dev
}

// NewI2CDevice по дампу (0x4b8d1c0): fixDevicePath; config.Name/Port + config.Addr; NewLogger; driverreg.Init(); i2creg.Open(name); i2c.Dev{Addr, Bus}; при ошибке Open — Logger.Error, return nil.
// Возвращает *I2CDevice, реализующий I2CDeviceForRegister (periph I2C на Linux).
func NewI2CDevice(config interface{}) *I2CDevice {
	if config == nil {
		return nil
	}
	name, addr := getI2CConfigNameAndAddr(config)
	if name == "" {
		return nil
	}
	logger := logging.NewLogger(fmt.Sprintf("i2c-%s-%d", name, addr))
	if fixDevicePath(config) && logger != nil {
		logger.Info("fixDevicePath applied", 0)
	}
	if _, err := driverreg.Init(); err != nil {
		if logger != nil {
			logger.Info("driverreg.Init (periph) skipped", 0)
		}
	}
	bus, err := i2creg.Open(name)
	if err != nil {
		if logger != nil {
			logger.Error(fmt.Sprintf("i2creg.Open %s: %v", name, err))
		}
		return nil
	}
	dev := &i2c.Dev{Addr: uint16(addr), Bus: bus}
	return &I2CDevice{
		Config: config,
		Logger: logger,
		Dev:    dev,
		Bus:    bus,
		Addr:   uint16(addr),
	}
}

func getI2CConfigNameAndAddr(c interface{}) (name string, addr int) {
	if c == nil {
		return "", 0
	}
	if m, ok := c.(map[string]interface{}); ok {
		if n, _ := m["Name"].(string); n != "" {
			name = n
		}
		if name == "" {
			if p, _ := m["Port"].(string); p != "" {
				name = p
			}
		}
		if a, ok := m["Addr"].(int); ok {
			addr = a
		}
		if addr == 0 {
			if a, ok := m["Address"].(int); ok {
				addr = a
			}
		}
	}
	if addr <= 0 {
		addr = 0x5f
	}
	return name, addr
}

func fixDevicePath(c interface{}) bool {
	if c == nil {
		return false
	}
	if m, ok := c.(map[string]interface{}); ok {
		if p, _ := m["Port"].(string); p != "" && (p[0] != '/' || len(p) < 5) {
			m["Port"] = "/dev/i2c-1"
			return true
		}
	}
	return false
}

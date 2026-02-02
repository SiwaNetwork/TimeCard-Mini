package generic_serial_device

import (
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Автоматически извлечено из timebeat-2.2.20

// I2CDevice — общая структура I2C-устройства (по дизассемблеру). Реализация методов — в i2cdevice_linux.go (Lock, Unlock, Write, WriteThenRead).
// На !linux NewI2CDevice возвращает nil; тип объявлен здесь, чтобы сигнатура NewI2CDevice возвращала *I2CDevice.
type I2CDevice struct {
	Config interface{}
	Logger *logging.Logger
	Dev    interface{} // на Linux: *i2c.Dev
	Bus    interface{} // на Linux: i2c.BusCloser
	Addr   uint16
	Mu     sync.Mutex
}

// I2CWriterReader по дизассемблеру VerifyConnectedToDevice: устройство с методом WriteThenRead(write, readLen) (read []byte, err).
type I2CWriterReader interface {
	WriteThenRead(write []byte, readLen int) (read []byte, err error)
}

// I2CWriter по дизассемблеру Set100MbpsNoAutoNegotiation/WriteSGMIIRegister: устройство с методом Write(data []byte).
type I2CWriter interface {
	Write(data []byte) error
}

// I2CDeviceForRegister — устройство с Lock/Unlock и I2C для RegisterRequest.Execute (по дампу 0x4b8e460).
type I2CDeviceForRegister interface {
	Lock()
	Unlock()
	Write(data []byte) error
	WriteThenRead(write []byte, readLen int) (read []byte, err error)
}

// WAIT_TIME_AFTER_I2C — пауза после I2C-операции (по дампу Execute: time.Sleep(WAIT_TIME_AFTER_I2C)).
var WAIT_TIME_AFTER_I2C = 5 * time.Millisecond

// RegisterRequest по дизассемблеру Execute/AddUint16/GetResult: буфер записи, длина чтения, результат, I2C, BigEndian.
// Смещения: 0x08 WriteData, 0x10/0x18 len/cap, 0x20 ReadLen, 0x28/0x30/0x38 Result, 0x40 I2C, 0x48 BigEndian.
type RegisterRequest struct {
	WriteData  []byte
	ReadLen    uint16
	Result    []byte
	I2C       I2CDeviceForRegister
	BigEndian bool
}

// NewRegisterRequest создаёт запрос с заданным I2C-устройством (по дампу CreateVlan).
func NewRegisterRequest(i2c I2CDeviceForRegister) *RegisterRequest {
	if i2c == nil {
		return nil
	}
	return &RegisterRequest{I2C: i2c}
}

// NewI2CDevice возвращает *I2CDevice (на Linux — реализация periph I2C; на !linux — nil).
// Определён в i2cdevice_linux.go (Linux) и i2cdevice_stub.go (!linux).


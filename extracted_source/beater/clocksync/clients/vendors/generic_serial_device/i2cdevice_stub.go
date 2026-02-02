//go:build !linux

package generic_serial_device

// NewI2CDevice на не-Linux: periph I2C недоступен, возвращаем nil (*I2CDevice).
func NewI2CDevice(config interface{}) *I2CDevice {
	return nil
}

package ocptap

// OcpTapDevice по дизассемблеру: OcpTap external device; реализует ExternaLDevice (Run).
type OcpTapDevice struct {
	controller interface{}
	config     string
}

// Run по дизассемблеру: основной цикл OcpTap device. Заглушка до реализации.
func (d *OcpTapDevice) Run() {
	_ = d
}

// NewDevice по дизассемблеру: (controller, config) → OcpTapDevice.
func NewDevice(controller interface{}, config string) interface{} {
	return &OcpTapDevice{controller: controller, config: config}
}

func inittask() {
	// TODO: реконструировать
}


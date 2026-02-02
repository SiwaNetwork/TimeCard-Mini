package orolia

// OroliaDevice по дизассемблеру: Orolia external device; реализует ExternaLDevice (Run).
type OroliaDevice struct {
	controller interface{}
	config     string
}

// Run по дизассемблеру: основной цикл Orolia device. Заглушка до реализации.
func (d *OroliaDevice) Run() {
	_ = d
}

// NewDevice по дизассемблеру: (controller, config) → OroliaDevice.
func NewDevice(controller interface{}, config string) interface{} {
	return &OroliaDevice{controller: controller, config: config}
}

func inittask() {
	// TODO: реконструировать
}


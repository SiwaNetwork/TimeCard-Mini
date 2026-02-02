package arista

// AristaDevice по дизассемблеру: Arista external device; реализует ExternaLDevice (Run).
type AristaDevice struct {
	controller interface{}
	config     string
}

// Run по дизассемблеру: основной цикл Arista device. Заглушка до реализации.
func (d *AristaDevice) Run() {
	_ = d
}

// NewDevice по дизассемблеру (0x4bbc600): (controller, config) → AristaDevice.
func NewDevice(controller interface{}, config string) interface{} {
	return &AristaDevice{controller: controller, config: config}
}

func inittask() {
	// TODO: реконструировать
}


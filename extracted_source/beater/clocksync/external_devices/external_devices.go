package external_devices

import (
	"strings"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/external_devices/arista"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/external_devices/ocptap"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/external_devices/orolia"
	"github.com/shiwa/timecard-mini/extracted-source/config"
)

// ExternaLDevice по дизассемблеру (go:itab AristaDevice/OcpTapDevice/OroliaDevice): интерфейс внешних устройств; Run вызывается из runDevice.
type ExternaLDevice interface {
	Run()
}

// Controller по дизассемблеру: external_devices контроллер (appConfig+0x80).
// Start: loadConfig; затем newproc(func1) для каждого device.
type Controller struct {
	devices []interface{} // по дизассемблеру controller+0x8 slice, +0x10 len
}

var extController *Controller
var extOnce sync.Once

// NewController по дизассемблеру: sync.Once, создаёт Controller.
func NewController() *Controller {
	extOnce.Do(func() {
		extController = &Controller{
			devices: make([]interface{}, 0),
		}
	})
	return extController
}

// Start по дизассемблеру (0x4bc2940): loadConfig; цикл по devices — newproc(func1) для каждого.
func (c *Controller) Start() {
	if c == nil {
		return
	}
	c.loadConfig()
	for _, dev := range c.devices {
		if dev != nil {
			go c.runDevice(dev)
		}
	}
}

// runDevice по дизассемблеру Start.func1 (0x4bc2a40): вызов device.Run() через интерфейс ExternaLDevice.
func (c *Controller) runDevice(dev interface{}) {
	if d, ok := dev.(ExternaLDevice); ok && d != nil {
		d.Run()
	}
}

// loadConfig по дизассемблеру (0x4bc2aa0): итерация по appConfig.ExternalDevices; "arista_*"→arista.NewDevice, "ocptap_*"→ocptap, "orolia_*"→orolia.
func (c *Controller) loadConfig() {
	cfg := config.GetAppConfig()
	if cfg == nil || len(cfg.ExternalDevices) == 0 {
		return
	}
	for _, name := range cfg.ExternalDevices {
		dev := c.configureDevice(name)
		if dev != nil {
			c.devices = append(c.devices, dev)
		}
	}
}

// configureDevice по дизассемблеру: диспетчеризация по типу (arista_eso, ocptap, orolia и т.д.).
func (c *Controller) configureDevice(name string) interface{} {
	switch {
	case strings.HasPrefix(name, "arista"):
		return arista.NewDevice(c, name)
	case strings.HasPrefix(name, "ocptap"):
		return ocptap.NewDevice(c, name)
	case strings.HasPrefix(name, "orolia"):
		return orolia.NewDevice(c, name)
	default:
		return nil
	}
}

func controller() {
	// TODO: реконструировать
}

func func1() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}

func loadConfig() {
	// TODO: реконструировать
}

func once() {
	// TODO: реконструировать
}


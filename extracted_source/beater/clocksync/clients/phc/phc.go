// Package phc — PHC-клиент контроллер. Реконструировано по дизассемблеру:
// NewController (sync.Once), loadConfig (GetStore, Range ConfigureTimeSource),
// ConfigureTimeSource (key "phc" len 3, GetInstance, GetDeviceWithName, NewClient, Swap, Start).
package phc

import (
	"sync"

	phclib "github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/phc"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/phc/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Controller по дизассемблеру: +0x00=logger, +0x10=clients (sync.Map).
type Controller struct {
	logger  interface{}
	clients *sync.Map
}

var (
	phcOnce      sync.Once
	phcController *Controller
)

// NewController по дизассемблеру (clients/phc.NewController@@Base 0x45eed00): sync.Once, doSlow(func1), создаёт Controller с logger и clients.
func NewController() *Controller {
	phcOnce.Do(func() {
		phcController = &Controller{
			clients: &sync.Map{},
		}
	})
	return phcController
}

// GetController возвращает singleton контроллера.
func GetController() *Controller {
	return phcController
}

// loadConfig по дизассемблеру (0x45eeea0): GetStore(), store+8 → Range(ConfigureTimeSource-fm, controller).
func (c *Controller) loadConfig() {
	store := sources.GetStore()
	if store == nil {
		return
	}
	store.GetSources().Range(func(key, value interface{}) bool {
		c.ConfigureTimeSource(key, value)
		return true
	})
}

// Start по дизассемблеру: вызов loadConfig.
func (c *Controller) Start() {
	c.loadConfig()
}

// ConfigureTimeSource по дизассемблеру (0x45eef20): value=*TimeSourceConfig; len 3, cmpw "ph", cmpb 'c' → "phc";
// GetInstance, GetDeviceWithName(name); если device==nil и name!="system" → return;
// clients.Load(name) — уже есть → return; NewClient(config), clients.Swap(name, client), client.Start().
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	cfg, ok := value.(*sources.TimeSourceConfig)
	if !ok || cfg == nil || cfg.Type != "phc" {
		return
	}
	name := cfg.Name
	if name == "" {
		return
	}
	inst := phclib.GetInstance()
	if inst == nil {
		return
	}
	device := inst.GetDeviceWithName(name)
	if device == nil && name != "system" {
		return
	}
	if _, loaded := c.clients.Load(name); loaded {
		return
	}
	cl := client.NewClient(cfg)
	if cl != nil {
		c.clients.Store(name, cl)
		cl.Start()
	}
}

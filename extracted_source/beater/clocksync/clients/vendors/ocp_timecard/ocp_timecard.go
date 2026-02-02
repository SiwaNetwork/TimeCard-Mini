package ocp_timecard

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/ocp_timecard/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Controller по дизассемблеру: OCP Timecard контроллер.
type Controller struct {
	clients *sync.Map
}

var ocpController *Controller
var ocpOnce sync.Once

// NewController по дизассемблеру: sync.Once, создаёт Controller.
func NewController() *Controller {
	ocpOnce.Do(func() {
		ocpController = &Controller{
			clients: &sync.Map{},
		}
	})
	return ocpController
}

// LoadConfig по дизассемблеру (*Controller).loadConfig: GetStore, Range(ConfigureTimeSource).
func (c *Controller) LoadConfig() {
	if c == nil {
		return
	}
	store := sources.GetStore()
	if store == nil {
		return
	}
	store.GetSources().Range(func(key, value interface{}) bool {
		c.ConfigureTimeSource(key, value)
		return true
	})
}

// ConfigureTimeSource по дампу (0x4b6e380): type assert *TimeSourceConfig; DeviceVariantName=="ocp_timecard"; makeIDHashFromConfig; clients.Load; NewClient, Swap, Start.
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	cfg, ok := value.(*sources.TimeSourceConfig)
	if !ok || cfg == nil || cfg.Type != "ocp_timecard" {
		return
	}
	id := makeIDHashFromConfig(cfg)
	if _, loaded := c.clients.Load(id); loaded {
		return
	}
	cl := makeNewClient(cfg)
	if cl != nil {
		c.clients.Store(id, cl)
		cl.Start()
	}
}

func makeIDHashFromConfig(cfg *sources.TimeSourceConfig) string {
	return fmt.Sprintf("%s%d", cfg.Name, cfg.Index)
}

func makeNewClient(cfg *sources.TimeSourceConfig) *client.Client {
	return client.NewClient(cfg)
}

// LoadConfig — package-level для вызова из run_impl (заглушка делегирует в Controller).
func LoadConfig() {
	if ocpController != nil {
		ocpController.LoadConfig()
	}
}


func ConfigureTimeSourceFm() {
	// TODO: реконструировать
}

func Start() {
	// TODO: реконструировать
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


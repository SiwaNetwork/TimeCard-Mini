package timebeat_timecard_mini

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat_timecard_mini/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Controller по дизассемблеру: 0x08=logger, 0x10=clients.
type Controller struct {
	logger  *logging.Logger
	clients *sync.Map
}

var tcmController *Controller
var tcmOnce sync.Once

// NewController по дизассемблеру (0x4ba5ec0): sync.Once, создаёт Controller.
func NewController() *Controller {
	tcmOnce.Do(func() {
		tcmController = &Controller{
			logger:  logging.NewLogger("timebeat-timecard-mini"),
			clients: &sync.Map{},
		}
	})
	return tcmController
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

// ConfigureTimeSource по дампу (0x4ba60e0): type assert *TimeSourceConfig; DeviceVariantName=="timebeat_timecard_mini"; makeIDHashFromConfig; clients.Load; NewClient, Swap, Start.
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	cfg, ok := value.(*sources.TimeSourceConfig)
	if !ok || cfg == nil || cfg.Type != "timebeat_timecard_mini" {
		return
	}
	if cfg.Name == "" && cfg.ID == "" {
		if c.logger != nil {
			c.logger.Error("timebeat_timecard_mini: config missing name and id")
		}
		return
	}
	id := makeIDHashFromConfig(cfg)
	if _, loaded := c.clients.Load(id); loaded {
		return
	}
	cl := c.makeNewClient(cfg)
	if cl != nil {
		c.clients.Store(id, cl)
		cl.Start()
	}
}

func makeIDHashFromConfig(cfg *sources.TimeSourceConfig) string {
	return fmt.Sprintf("%s%d", cfg.Name, cfg.Index)
}

func (c *Controller) makeNewClient(cfg *sources.TimeSourceConfig) *client.Client {
	return client.NewClient(cfg)
}

// LoadConfig — package-level для run_impl.
func LoadConfig() {
	if tcmController != nil {
		tcmController.LoadConfig()
	}
}

func ConfigureTimeSource() {}

func ConfigureTimeSourceFm() {}

func Start() {}


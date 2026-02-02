package open_timecard_mini_v2_pt

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/open_timecard_mini_v2_pt/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Controller по дизассемблеру: Open Timecard Mini v2 PT контроллер.
type Controller struct {
	clients *sync.Map
}

var ocv2Controller *Controller
var ocv2Once sync.Once

// NewController по дизассемблеру: sync.Once, создаёт Controller.
func NewController() *Controller {
	ocv2Once.Do(func() {
		ocv2Controller = &Controller{clients: &sync.Map{}}
	})
	return ocv2Controller
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

// ConfigureTimeSource по дампу (0x4b99720): type assert *TimeSourceConfig; open_timecard_mini_v2_pt; makeIDHashFromConfig; clients.Load; NewClient, Start.
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	cfg, ok := value.(*sources.TimeSourceConfig)
	if !ok || cfg == nil || cfg.Type != "open_timecard_mini_v2_pt" {
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

// LoadConfig — package-level для run_impl.
func LoadConfig() {
	if ocv2Controller != nil {
		ocv2Controller.LoadConfig()
	}
}

func ConfigureTimeSource() {}

func ConfigureTimeSourceFm() {}

func FlashEEProm() {}

func GetClientInstance() {}


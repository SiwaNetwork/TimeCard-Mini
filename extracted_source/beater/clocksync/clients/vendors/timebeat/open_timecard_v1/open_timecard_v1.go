package open_timecard_v1

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/open_timecard_v1/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Controller по дизассемблеру: Open Timecard v1 контроллер.
type Controller struct {
	clients *sync.Map
}

var ocv1Controller *Controller
var ocv1Once sync.Once

// NewController по дизассемблеру: sync.Once, создаёт Controller.
func NewController() *Controller {
	ocv1Once.Do(func() {
		ocv1Controller = &Controller{clients: &sync.Map{}}
	})
	return ocv1Controller
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

// ConfigureTimeSource по дампу (0x4ba50c0): type assert *TimeSourceConfig; open_timecard_v1; makeIDHashFromConfig; clients.Load; NewClient, Start.
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	cfg, ok := value.(*sources.TimeSourceConfig)
	if !ok || cfg == nil || cfg.Type != "open_timecard_v1" {
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
	if ocv1Controller != nil {
		ocv1Controller.LoadConfig()
	}
}

// GetClientInstance по дампу (0x4ba45c0): возвращает первый клиент из контроллера (для ShowDpllStatus и др.).
func GetClientInstance() *client.Client {
	if ocv1Controller == nil || ocv1Controller.clients == nil {
		return nil
	}
	var first *client.Client
	ocv1Controller.clients.Range(func(_, value interface{}) bool {
		if cl, ok := value.(*client.Client); ok {
			first = cl
			return false
		}
		return true
	})
	return first
}

// ShowDpllStatus по дампу (0x4ba49a0): GetClientInstance(); при nil — возврат строки «no client»; иначе client.ShowDpllStatus(phaseStr).
func ShowDpllStatus(phaseStr string) string {
	cl := GetClientInstance()
	if cl == nil {
		return "no open_timecard_v1 client"
	}
	return cl.ShowDpllStatus(phaseStr)
}

// ShowClockgenVersion по дампу: GetClientInstance(); client.ShowClockgenVersion().
func ShowClockgenVersion() string {
	cl := GetClientInstance()
	if cl == nil {
		return "no open_timecard_v1 client"
	}
	return cl.ShowClockgenVersion()
}

// ShowClockgenConfigStatus по дампу: GetClientInstance(); client.ShowClockgenConfigStatus().
func ShowClockgenConfigStatus() string {
	cl := GetClientInstance()
	if cl == nil {
		return "unknown"
	}
	return cl.ShowClockgenConfigStatus()
}

// ShowInputStatus по дампу: GetClientInstance(); client.ShowInputStatus(inputStr).
func ShowInputStatus(inputStr string) string {
	cl := GetClientInstance()
	if cl == nil {
		return "no open_timecard_v1 client"
	}
	return cl.ShowInputStatus(inputStr)
}

func ConfigureTimeSource() {}

func ConfigureTimeSourceFm() {}


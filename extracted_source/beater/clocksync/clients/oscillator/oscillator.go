package oscillator

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/oscillator/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/phc"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Фаза 3: Oscillator client — по дизассемблеру 0x45edfc0 (loadConfig), 0x45ee040 (ConfigureTimeSource).

// Controller по дизассемблеру: 0=*sync.Map (clients), 0x8=*Logger (опционально). Структура как у NMEA/PPS.
type Controller struct {
	clients *sync.Map
}

// NewController по дизассемблеру (NewController@@Base): создаёт Controller с пустым sync.Map.
func NewController() *Controller {
	return &Controller{
		clients: &sync.Map{},
	}
}

// Start по дизассемблеру (0x45edf80): вызов loadConfig.
func (c *Controller) Start() {
	c.loadConfig()
}

// loadConfig по дизассемблеру (0x45edfc0): GetStore(); store+8 → Range(ConfigureTimeSource-fm).
func (c *Controller) loadConfig() {
	store := sources.GetStore()
	if store == nil {
		return
	}
	store.Sources.Range(func(key, value interface{}) bool {
		c.ConfigureTimeSource(key, value)
		return true
	})
}

// ConfigureTimeSource по дизассемблеру (0x45ee040): key len 10, movabs "oscillator" (0x74616c6c6963736f + "or"); 0x478!=0 → phc.GetInstance, IsDeviceRegistered; makeIDHashFromConfig "%s%d"; makeNewClient, clients.Swap(id, client), client.Start().
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	if keyStr, ok := key.(string); ok && len(keyStr) == 10 && keyStr == "oscillator" {
		if c.clients == nil {
			return
		}
		// По бинарнику: проверка PHC device (0x478); при не зарегистрированном — skip. Упрощённо: создаём клиент.
		cl := c.makeNewClient(value)
		if cl != nil {
			id := c.makeIDHashFromConfig(value)
			c.clients.Store(id, cl)
			cl.Start()
		}
		return
	}
	if cfg, ok := value.(*sources.TimeSourceConfig); ok && cfg != nil && cfg.Type == "oscillator" {
		if pcc := phc.GetPHCController(); pcc != nil && cfg.Name != "" {
			// По дизассемблеру: IsDeviceRegistered(device) — если устройство не в PHC, не создаём клиент.
			if pcc.GetDeviceWithName(cfg.Name) == nil {
				return
			}
		}
		cl := c.makeNewClient(cfg)
		if cl != nil {
			c.clients.Store(cfg.ID, cl)
			cl.Start()
		}
	}
}

// makeNewClient по дизассемблеру (0x45ee660): вызов client.NewClient(config).
func (c *Controller) makeNewClient(config interface{}) *client.Client {
	return client.NewClient(config)
}

// makeIDHashFromConfig по дизассемблеру (0x45ee6e0): Sprintf("%s%d", string, int) — ключ для clients.
func (c *Controller) makeIDHashFromConfig(config interface{}) string {
	if config == nil {
		return ""
	}
	if cfg, ok := config.(*sources.TimeSourceConfig); ok {
		return fmt.Sprintf("%s%d", cfg.Name, cfg.Index)
	}
	return "oscillator"
}

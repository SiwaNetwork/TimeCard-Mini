package nmea

import (
	"fmt"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/nmea/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Реконструировано по дизассемблеру бинарника timebeat-2.2.20 (clients/nmea).

// STANDARD_BAUD_RATES по дизассемблеру (ConfigureTimeSource 0x45b438c: 0x7983770) — слайс стандартных скоростей для NMEA; проверка baud в цикле перед nativeOpen.
var STANDARD_BAUD_RATES []int

func init() {
	STANDARD_BAUD_RATES = []int{4800, 9600, 38400, 57600, 115200}
}

// Controller по дизассемблеру NewController.func1: 0=config, 0x8=*Logger, 0x10=*sync.Map.
type Controller struct {
	logger  *logging.Logger
	clients *sync.Map
}

var (
	controllerOnce sync.Once
	controller     *Controller
)

// NewController по дизассемблеру (NewController@@Base 0x45b4080): sync.Once(once), doSlow(func1); func1 создаёт Logger, sync.Map, controller = &Controller{logger, clients}.
func NewController() *Controller {
	controllerOnce.Do(func() {
		logger := logging.NewLogger("nmea-controller")
		controller = &Controller{
			logger:  logger,
			clients: &sync.Map{},
		}
	})
	return controller
}

// Start по дизассемблеру (Controller.Start@@Base 0x45b41e0): вызов loadConfig.
func (c *Controller) Start() {
	c.loadConfig()
}

// loadConfig по дизассемблеру (loadConfig@@Base 0x45b4220): GetStore(), store+8 → Range(ConfigureTimeSource-fm); callback получает (key, value), вызывает c.ConfigureTimeSource(key, value).
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

// ConfigureTimeSource по дизассемблеру (0x45b42a0): type assert *Controller; key len==4, cmpl $0x61656d6e (\"nmea\"); baud в STANDARD_BAUD_RATES; go.bug.st/serial.nativeOpen; при ошибке Logger.Error; makeIDHashFromConfig→key; NewClient, clients.Swap(key, client), client.Start().
// При Range(store) key=ID (hash), value=*TimeSourceConfig — проверяем value.Type \"nmea\"/\"gnss\".
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	// Проверка по key (как в бинарнике: len==4, key=="nmea"); NewClient, Swap(key, client), client.Start().
	if keyStr, ok := key.(string); ok && len(keyStr) == 4 && keyStr == "nmea" {
		cl := c.makeNewClient(value)
		if cl != nil {
			c.clients.Store(c.makeIDHashFromConfig(value), cl)
			cl.Start()
		}
		return
	}
	// Проверка по value.Type для источников из store (GenerateTimeSourcesFromConfig → AddSource(ID, *TimeSourceConfig))
	if cfg, ok := value.(*sources.TimeSourceConfig); ok && (cfg.Type == "nmea" || cfg.Type == "gnss") {
		client := c.makeNewClient(cfg)
		if client != nil {
			c.clients.Store(cfg.ID, client)
			client.Start()
		}
	}
}

// makeNewClient по дизассемблеру (makeNewClient@@Base 0x45b47e0): копирование config, вызов client.NewClient(config).
func (c *Controller) makeNewClient(config interface{}) *client.Client {
	return client.NewClient(config)
}

// makeIDHashFromConfig по дизассемблеру (0x45b48a0): два аргумента — строка (0xe0) и int (0x120); fmt.Sprintf с форматом 4 символа (0x67f8a5 → "%s%d") → ключ для clients.
// В бинарнике: convTstring(device/path), convT64(baud/index), Sprintf("%s%d", s, i), return.
func (c *Controller) makeIDHashFromConfig(config interface{}) string {
	if config == nil {
		return ""
	}
	if cfg, ok := config.(*sources.TimeSourceConfig); ok {
		return fmt.Sprintf("%s%d", cfg.Name, cfg.Index)
	}
	return "nmea"
}

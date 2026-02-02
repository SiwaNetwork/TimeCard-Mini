package pps

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/pps/client"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Controller по дизассемблеру: offsets, clients (sync.Map), eventCh, doneCh, running.
type Controller struct {
	offsets  *servo.Offsets
	clients  *sync.Map
	eventCh  chan client.EventData
	doneCh   chan struct{}
	running  uint32
}

// NewController создаёт PPS-контроллер (offsets для RegisterObservation). По дизассемблеру — аналог NTP/NMEA NewController.
func NewController(offsets *servo.Offsets) *Controller {
	return &Controller{
		offsets: offsets,
		clients: &sync.Map{},
		eventCh: make(chan client.EventData, 64),
		doneCh:  make(chan struct{}),
	}
}

// Start по дизассемблеру (0x45beb80): loadConfig; AfterFunc(5s, func1 — RunSanityTicker); go runEventDataDispatcher.
func (c *Controller) Start() {
	c.loadConfig()
	atomic.StoreUint32(&c.running, 1)
	time.AfterFunc(5*time.Second, func() { runSanityTicker(c) })
	go c.runEventDataDispatcher()
}

// runSanityTicker — заглушка по дизассемблеру Start.func1 (TODO: проверка валидности PPS).
func runSanityTicker(c *Controller) {
	_ = c
}

// runEventDataDispatcher по дизассемблеру (0x45be780): select(eventCh, doneCh); при event и running→dispatchMessage(ev).
func (c *Controller) runEventDataDispatcher() {
	for {
		select {
		case ev := <-c.eventCh:
			if atomic.LoadUint32(&c.running) != 0 {
				c.dispatchMessage(ev)
			}
		case <-c.doneCh:
			return
		}
	}
}

// dispatchMessage по дизассемблеру (0x45be840): clients.Load(ev.ClientKey); client.ProcessEvent(ev.Nsec).
func (c *Controller) dispatchMessage(ev client.EventData) {
	if c.clients == nil {
		return
	}
	key := ev.ClientKey
	if key == "" {
		return
	}
	val, ok := c.clients.Load(key)
	if !ok || val == nil {
		return
	}
	cl, ok := val.(*client.Client)
	if !ok || cl == nil {
		return
	}
	cl.ProcessEvent(ev.Nsec)
}

// loadConfig по дизассемблеру (0x45bece0): GetStore(); store+8 → Range(ConfigureTimeSource-fm).
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

// ConfigureTimeSource по дизассемблеру (0x45bed60): key len 3, cmpw $0x7070 \"pp\", cmpb $0x73 's' → \"pps\"; 0x5a8/0x5a0 != -1; phc.GetInstance, IsDeviceRegistered; makeIDHashFromConfig \"%s%d\"; NewClient, clients.Swap(id, client), client.Start().
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	if keyStr, ok := key.(string); ok && len(keyStr) == 3 && keyStr == "pps" {
		c.makeNewClient(value)
		return
	}
	if cfg, ok := value.(*sources.TimeSourceConfig); ok && cfg.Type == "pps" {
		c.makeNewClient(cfg)
	}
}

func (c *Controller) makeNewClient(config interface{}) {
	if c.clients == nil || c.eventCh == nil {
		return
	}
	cfg, ok := config.(*sources.TimeSourceConfig)
	if !ok || cfg == nil {
		return
	}
	uri := cfg.Name
	if uri == "" {
		uri = "pps:" + cfg.ID
	}
	cl := client.NewClientWithConfig(uri, 0) // TimeSourceConfig не содержит Offset; TODO: передавать из config.ClockSource
	if cl == nil {
		return
	}
	id := cfg.ID
	if id == "" {
		id = uri
	}
	// По дизассемблеру dispatchMessage: key = Sprintf("%s%d", ...) — используем id или uri
	clientKey := id
	if clientKey == "" {
		clientKey = uri
	}
	c.clients.Store(clientKey, cl)
	cl.SetEventChannel(c.eventCh, clientKey)
	cl.Start()
}

// ConfigureTimeSourceFm — closure для Range (по дизассемблеру -fm). Вызов ConfigureTimeSource.
func ConfigureTimeSourceFm() {}

func CreateAndStartNewClient() {}
func EnableExtTs()             {}
func SetCedControl()           {}
func SetCedControlToSecondarySources() {}
func SetMonitorOnly()          {}
func controller()             {}
func dispatchMessage()        {}
func func1()                   {}
func func2()                   {}
func inittask()                {}
func loadConfig()              {}
func makeIDHashFromConfig()    {}
func makeIDHashFromEventData() {}
func makeNewClient()          {}
func once()                    {}
func runEventDataDispatcher()  {}
func ConfigureTimeSource()    {}

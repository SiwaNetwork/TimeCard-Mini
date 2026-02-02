package client

import (
	"fmt"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_gnss_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Реконструировано по дизассемблеру бинарника timebeat-2.2.20 (clients/nmea/client).

// TAIEvent — алиас для generic_gnss_device.TAIEvent (clockName + offset для NotifyTAIOffset).
type TAIEvent = generic_gnss_device.TAIEvent

// Client по дизассемблеру NewClient/Start: 0=device (*GenericGNSSDevice), далее logger; опционально каналы 0x78/0x80/0x88 и поля для decorateObservation (0x268, 0x100, 0x258, 0xf9, 0x170, 0x120, 0x168).
type Client struct {
	device interface{}
	logger *logging.Logger
	// Опциональные каналы от device для runGNSSRunloop (0x78, 0x80, 0x88 по дампу).
	ChGSV chan string      // case 0: строка для NMEAGSVLogEntry.Log
	ChObs chan interface{} // case 1: observation для decorateObservation + RegisterObservation
	ChTAI chan TAIEvent    // case 2: clockName + offset для NotifyTAIOffset
	// Поля для decorateObservation (client+0x268, 0x100, 0x258, 0xf9, 0x170, 0x120, 0x168).
	NMEAGSVLogStr string // client+0x268/0x270 — строка для Sprintf → SourceID
	SourceIDBase  string // client+0x100/0x108 — вторая строка в decorated
	OffsetBase    int64  // client+0x258 → out+0x40
	FlagsByte     byte   // client+0xf9 → out+0x98
	ExtraStr      string // client+0x170/0x178 → out+0xc0/0xc8
	OffsetAdd     int64  // client+0x120 — добавляется к *out+0
	CategoryFlag  byte   // client+0x168: !=0 → out+0x78=5 или 6, иначе 4
}

// NewClient по дизассемблеру (NewClient@@Base 0x45b3400): config копируется; generic_gnss_device.NewDevice(config); logging.NewLogger; return &Client{device, logger}.
func NewClient(config interface{}) *Client {
	device := generic_gnss_device.NewDevice(config)
	logger := logging.NewLogger("nmea-client")
	return &Client{
		device: device,
		logger: logger,
	}
}

// Start по дизассемблеру (Client.Start@@Base 0x45b3720): если client.device==nil return; device.Start(); при GNSSChannels — взять каналы; go runGNSSRunloop(client).
func (c *Client) Start() {
	if c.device == nil {
		return
	}
	if d, ok := c.device.(generic_gnss_device.DeviceInterface); ok {
		d.Start()
	}
	if ch, ok := c.device.(generic_gnss_device.GNSSChannels); ok {
		c.ChObs = ch.GetObservationChan()
		c.ChTAI = ch.GetTaiChan()
	}
	if gsv, ok := c.device.(generic_gnss_device.GNSSChannelsWithGSV); ok {
		c.ChGSV = gsv.GetGSVChan()
	}
	go c.runGNSSRunloop()
}

// runGNSSRunloop по дизассемблеру (0x45b3840): цикл runtime.selectgo по 4 каналам — device 0x78, 0x80, 0x88, done.
// case 0: GSV → NMEAGSVLogEntry.Log; case 1: obs → decorateObservation + RegisterObservation; case 2: TAI → NotifyTAIOffset; case 3 (done): return.
func (c *Client) runGNSSRunloop() {
	chGSV := c.ChGSV
	chObs := c.ChObs
	chTAI := c.ChTAI
	if chGSV == nil && chObs == nil && chTAI == nil {
		return
	}
	for {
		select {
		case msg, ok := <-chGSV:
			if !ok {
				chGSV = nil
				if chObs == nil && chTAI == nil {
					return
				}
				continue
			}
			entry := &logging.NMEAGSVLogEntry{Message: msg}
			entry.Log()
		case obs, ok := <-chObs:
			if !ok {
				chObs = nil
				if chGSV == nil && chTAI == nil {
					return
				}
				continue
			}
			c.runGNSSRunloopSelectCase1(obs)
		case ev, ok := <-chTAI:
			if !ok {
				chTAI = nil
				if chGSV == nil && chObs == nil {
					return
				}
				continue
			}
			hcc := hostclocks.GetController()
			if hcc != nil {
				hcc.NotifyTAIOffset(ev.ClockName, ev.OffsetNs)
			}
		}
		// Без default — select блокируется до получения (по дизассемблеру 4-й case = done → return)
	}
}

// decoratedObservation — структура по дампу decorateObservation: 0x10/0x18 string (Sprintf), 0x20/0x28, 0x40, 0x78 (4/5/6), 0x98, 0xc0/0xc8.
type decoratedObservation struct {
	SourceID string
	Offset   int64
	Flags    byte
	Category int64 // 4, 5 или 6 по дампу: client+0x168 != 0 → 5 или 6, иначе 4
	ExtraStr string
}

// decorateObservation по дизассемблеру (0x45b3ac0): out+0xb8=0; client+0x268/0x270 → Sprintf → out+0x10/0x18 (len 7);
// client+0x100/0x108 → out+0x20/0x28; 0x258 → out+0x40; 0xf9 → out+0x98; 0x170/0x178 → out+0xc0/0xc8; *out+0 += client+0x120;
// если client+0x168 != 0: out+0x78 = 5→6 иначе 4. Возвращает observation для RegisterObservation.
func (c *Client) decorateObservation(obs interface{}) *decoratedObservation {
	out := &decoratedObservation{}
	out.SourceID = fmt.Sprintf("%s", c.NMEAGSVLogStr)
	if out.SourceID == "" {
		out.SourceID = c.SourceIDBase
	}
	out.ExtraStr = c.ExtraStr
	out.Offset = c.OffsetBase + c.OffsetAdd
	out.Flags = c.FlagsByte
	// По дампу: client+0x168 != 0 → если out+0x78 было 5 то 6, иначе 4; client+0x168 == 0 → не трогаем.
	if c.CategoryFlag != 0 {
		out.Category = 4
		// TODO: если obs даёт category 5 → out.Category = 6
	} else {
		out.Category = 4
	}
	_ = obs
	return out
}

// runGNSSRunloopSelectCase1 вызывается из runloop case 1: decorateObservation + RegisterObservation (или RegisterObservationWithCategory при category 5/6).
func (c *Client) runGNSSRunloopSelectCase1(obs interface{}) {
	decorated := c.decorateObservation(obs)
	ctrl := servo.GetController()
	if ctrl == nil || ctrl.GetOffsets() == nil {
		return
	}
	offsets := ctrl.GetOffsets()
	// category 4/5/6 по дампу → 1=primary, 2=secondary для RegisterObservationWithCategory
	if decorated.Category == 5 || decorated.Category == 6 {
		offsets.RegisterObservationWithCategory(decorated.SourceID, decorated.Offset, 2)
	} else {
		offsets.RegisterObservation(decorated.SourceID, decorated.Offset)
	}
}

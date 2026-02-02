package client

import (
	"fmt"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_gnss_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// TAIEvent — алиас для generic_gnss_device.TAIEvent.
type TAIEvent = generic_gnss_device.TAIEvent

// Client по дампу NewClient (0x4ba5600): 0x0=device, 0x8=config, 0x260=logger, 0x268/0x270=NMEAGSVLogStr, 0x78/0x80/0x88=ChGSV/ChObs/ChTAI.
type Client struct {
	device        interface{}
	config        *sources.TimeSourceConfig
	logger        *logging.Logger
	NMEAGSVLogStr string
	ChGSV         chan string
	ChObs         chan interface{}
	ChTAI         chan TAIEvent
	SourceIDBase  string // 0x100/0x108
	OffsetBase    int64  // 0x258
	FlagsByte     byte   // 0xf9
	ExtraStr      string // 0x170/0x178
	OffsetAdd     int64  // 0x120
	CategoryFlag  byte   // 0x168
}

// NewClient по дампу (0x4ba5600): createGNSSDeviceConfig; generic_gnss_device.NewDevice; NewLogger; return Client.
func NewClient(config *sources.TimeSourceConfig) *Client {
	if config == nil {
		return nil
	}
	gnssConfig := createGNSSDeviceConfig(config)
	device := generic_gnss_device.NewDevice(gnssConfig)
	logger := logging.NewLogger("timebeat-timecard-mini")
	gsvStr := "timebeat"
	if config.Name != "" {
		gsvStr = config.Name
	}
	return &Client{
		device:        device,
		config:        config,
		logger:        logger,
		NMEAGSVLogStr: gsvStr,
	}
}

// createGNSSDeviceConfig по дампу: строит config для generic_gnss_device.NewDevice.
func createGNSSDeviceConfig(cfg *sources.TimeSourceConfig) interface{} {
	return cfg
}

// Start по дампу (0x4ba5940): device!=nil → device.Start(); go runGNSSRunloop.
func (c *Client) Start() {
	if c == nil || c.device == nil {
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

// runGNSSRunloop по дампу (0x4ba5a60): select ChGSV/ChObs/ChTAI/done; case obs→decorateObservation+RegisterObservation; case TAI→NotifyTAIOffset.
func (c *Client) runGNSSRunloop() {
	chGSV, chObs, chTAI := c.ChGSV, c.ChObs, c.ChTAI
	if chGSV == nil && chObs == nil && chTAI == nil {
		return
	}
	ctrl := servo.GetController()
	if ctrl == nil {
		return
	}
	offsets := ctrl.GetOffsets()
	if offsets == nil {
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
			decorated := c.decorateObservation(obs)
			if decorated != nil {
				offsets.RegisterObservation(decorated.SourceID, decorated.Offset)
			}
		case ev, ok := <-chTAI:
			if !ok {
				chTAI = nil
				if chGSV == nil && chObs == nil {
					return
				}
				continue
			}
			if hcc := hostclocks.GetController(); hcc != nil {
				hcc.NotifyTAIOffset(ev.ClockName, ev.OffsetNs)
			}
		}
	}
}

type decoratedObservation struct {
	SourceID string
	Offset   int64
	Category int64
}

// decorateObservation по дампу (0x4ba5d20): client+0x268→Sprintf; 0x100→SourceIDBase; 0x258→Offset; 0xf9→Flags; 0x170→ExtraStr; 0x120→OffsetAdd; 0x168→CategoryFlag.
func (c *Client) decorateObservation(obs interface{}) *decoratedObservation {
	out := &decoratedObservation{Category: 3}
	out.SourceID = fmt.Sprintf("%s", c.NMEAGSVLogStr)
	if out.SourceID == "" {
		out.SourceID = c.SourceIDBase
	}
	out.Offset = c.OffsetBase + c.OffsetAdd
	if c.CategoryFlag != 0 {
		if out.Category == 5 {
			out.Category = 6
		} else {
			out.Category = 4
		}
	}
	_ = obs
	_ = c.FlagsByte
	_ = c.ExtraStr
	return out
}

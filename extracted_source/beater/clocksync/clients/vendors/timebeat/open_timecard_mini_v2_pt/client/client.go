package client

import (
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_gnss_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/clock_gen/clock_gen_8A34002E"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// TAIEvent — алиас для generic_gnss_device.TAIEvent.
type TAIEvent = generic_gnss_device.TAIEvent

// Client по дампу open_timecard_mini_v2_pt: 0x0=device, 0x8=clockgen, 0x268=logger; runGNSSRunloop с ChGSV/ChObs/ChTAI.
type Client struct {
	device        interface{}
	clockgen      interface{}
	logger        *logging.Logger
	NMEAGSVLogStr string
	ChGSV         chan string
	ChObs         chan interface{}
	ChTAI         chan TAIEvent
	SourceIDBase  string
	OffsetBase    int64
	FlagsByte     byte
	ExtraStr      string
	OffsetAdd     int64
	CategoryFlag  byte
}

// NewClient по дампу (0x4b989c0): createGNSSDeviceConfig; generic_gnss_device.NewDevice; createClockGenConfig; NewLogger; return Client.
func NewClient(config *sources.TimeSourceConfig) *Client {
	if config == nil {
		return nil
	}
	gnssConfig := createGNSSDeviceConfig(config)
	device := generic_gnss_device.NewDevice(gnssConfig)
	clockgen := createClockGenConfig(config)
	logger := logging.NewLogger("open-timecard-mini-v2-pt")
	gsvStr := "ocv2"
	if config.Name != "" {
		gsvStr = config.Name
	}
	return &Client{
		device:        device,
		clockgen:      clockgen,
		logger:        logger,
		NMEAGSVLogStr: gsvStr,
	}
}

func createGNSSDeviceConfig(cfg *sources.TimeSourceConfig) interface{} {
	return cfg
}

func createClockGenConfig(cfg *sources.TimeSourceConfig) interface{} {
	if cfg == nil {
		return nil
	}
	return clock_gen_8A34002E.NewClockGen8A34012(cfg)
}

// Start по дампу (0x4b98d40): device!=nil → device.Start(); go runGNSSRunloop; clockgen!=nil → clockgen.Start().
func (c *Client) Start() {
	if c == nil {
		return
	}
	if c.device != nil {
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
	if c.clockgen != nil {
		if cg, ok := c.clockgen.(interface{ Start() }); ok {
			cg.Start()
		}
	}
}

// runGNSSRunloop по дампу (0x4b98f40): select ChGSV/ChObs/ChTAI/done; decorateObservation + RegisterObservation; NotifyTAIOffset.
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
		if chGSV == nil && chObs == nil && chTAI == nil {
			return
		}
		select {
		case msg, ok := <-chGSV:
			if !ok {
				chGSV = nil
				continue
			}
			entry := &logging.NMEAGSVLogEntry{Message: msg}
			entry.Log()
		case obs, ok := <-chObs:
			if !ok {
				chObs = nil
				continue
			}
			decorated := c.decorateObservation(obs)
			if decorated != nil {
				offsets.RegisterObservation(decorated.SourceID, decorated.Offset)
			}
		case ev, ok := <-chTAI:
			if !ok {
				chTAI = nil
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
}

func (c *Client) decorateObservation(obs interface{}) *decoratedObservation {
	out := &decoratedObservation{SourceID: c.NMEAGSVLogStr, Offset: c.OffsetBase + c.OffsetAdd}
	if out.SourceID == "" {
		out.SourceID = c.SourceIDBase
	}
	_ = obs
	return out
}

// ExecutePullIn по дампу open_timecard_mini_v2_pt: делегирует clockgen.ExecutePullIn(phase).
func (c *Client) ExecutePullIn(phase int) {
	if c == nil || c.clockgen == nil {
		return
	}
	if cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012); ok {
		cg.ExecutePullIn(phase)
	}
}

// Reset по дампу: client+0x8 = clockgen, clockgen.Reset().
func (c *Client) Reset() error {
	if c == nil || c.clockgen == nil {
		return nil
	}
	if cg, ok := c.clockgen.(interface{ Reset() error }); ok {
		return cg.Reset()
	}
	return nil
}

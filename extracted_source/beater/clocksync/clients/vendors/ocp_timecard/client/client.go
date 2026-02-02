package client

import (
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_gnss_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// TAIEvent — алиас для generic_gnss_device.TAIEvent.
type TAIEvent = generic_gnss_device.TAIEvent

// Client по дампу ocp_timecard: 0x0=device, 0x10=macClient, 0x260=logger; runGNSSRunloop с ChGSV/ChObs/ChTAI.
type Client struct {
	device        interface{}
	macClient     interface{}
	logger        *logging.Logger
	ChGSV         chan string
	ChObs         chan interface{}
	ChTAI         chan TAIEvent
	NMEAGSVLogStr string
	SourceIDBase  string
	OffsetBase    int64
	FlagsByte     byte
	ExtraStr      string
	OffsetAdd     int64
	CategoryFlag  byte
}

// NewClient по дампу (0x4b6c780): createGNSSDeviceConfig; generic_gnss_device.NewDevice; NewOCPTimecardMACClient; NewLogger; return Client.
func NewClient(config *sources.TimeSourceConfig) *Client {
	if config == nil {
		return nil
	}
	gnssConfig := createGNSSDeviceConfig(config)
	device := generic_gnss_device.NewDevice(gnssConfig)
	macClient := newOCPTimecardMACClientStub(config)
	logger := logging.NewLogger("ocp-timecard")
	gsvStr := "ocp"
	if config.Name != "" {
		gsvStr = config.Name
	}
	return &Client{
		device:        device,
		macClient:     macClient,
		logger:        logger,
		NMEAGSVLogStr: gsvStr,
	}
}

// newOCPTimecardMACClientStub — заглушка до полной реконструкции OCPTimecardMACClient.
func newOCPTimecardMACClientStub(cfg *sources.TimeSourceConfig) interface{} {
	return nil
}

// createGNSSDeviceConfig по дампу: строит config для generic_gnss_device.NewDevice.
func createGNSSDeviceConfig(cfg *sources.TimeSourceConfig) interface{} {
	return cfg
}

// configureTimecard по дампу: настройка timecard (заглушка).
func (c *Client) configureTimecard() {}

// Start по дампу (0x4b6caa0): configureTimecard; device!=nil → device.Start(); go runGNSSRunloop; macClient!=nil → macClient.Start.
func (c *Client) Start() {
	c.configureTimecard()
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
	if c.macClient != nil {
		if mc, ok := c.macClient.(interface{ Start() }); ok {
			mc.Start()
		}
	}
}

// runGNSSRunloop по дампу (0x4b6d040): select ChObs/ChTAI; decorateObservation + RegisterObservation; NotifyTAIOffset.
func (c *Client) runGNSSRunloop() {
	chObs, chTAI := c.ChObs, c.ChTAI
	if chObs == nil && chTAI == nil {
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
	for chObs != nil || chTAI != nil {
		select {
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

// decorateObservation по дампу: client+0x268→SourceID; 0x258→Offset.
func (c *Client) decorateObservation(obs interface{}) *decoratedObservation {
	out := &decoratedObservation{}
	out.SourceID = c.NMEAGSVLogStr
	if out.SourceID == "" {
		out.SourceID = c.SourceIDBase
	}
	out.Offset = c.OffsetBase + c.OffsetAdd
	_ = obs
	return out
}

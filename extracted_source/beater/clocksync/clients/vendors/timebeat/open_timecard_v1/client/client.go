package client

import (
	"fmt"
	"strconv"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_gnss_device"
	clock_gen_8A34002E "github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/clock_gen/clock_gen_8A34002E"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/eth_sw/eth_sw_KSZ9567S"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/oscillator/microchip_mac"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/power_sensor/power_sensor_INA230"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// TAIEvent — алиас для generic_gnss_device.TAIEvent.
type TAIEvent = generic_gnss_device.TAIEvent

// Client по дампу open_timecard_v1: 0x0=device, 0x288=logger; StartGNSSClient → device.Start, go runGNSSRunloop.
type Client struct {
	device        interface{}
	clockgen      interface{}
	ethSwitch     interface{}
	oscillator    interface{}
	powerSensor   interface{}
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

// NewClient по дампу (0x4ba2f80): createGNSSDeviceConfig; generic_gnss_device.NewDevice; createClockGenConfig; createEthSwitchConfig; createOscLffoClient; createPowerSensorConfig; NewLogger; return Client.
func NewClient(config *sources.TimeSourceConfig) *Client {
	if config == nil {
		return nil
	}
	gnssConfig := createGNSSDeviceConfig(config)
	device := generic_gnss_device.NewDevice(gnssConfig)
	logger := logging.NewLogger("open-timecard-v1")
	gsvStr := "ocv1"
	if config.Name != "" {
		gsvStr = config.Name
	}
	return &Client{
		device:        device,
		clockgen:      createClockGenConfig(config),
		ethSwitch:     createEthSwitchConfig(config),
		oscillator:    createOscLffoClient(config),
		powerSensor:   createPowerSensorConfig(config),
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

func createEthSwitchConfig(cfg *sources.TimeSourceConfig) interface{} {
	if cfg == nil {
		return nil
	}
	return eth_sw_KSZ9567S.NewEthSwKSZ9567S(cfg)
}

func createOscLffoClient(cfg *sources.TimeSourceConfig) interface{} {
	if cfg == nil {
		return nil
	}
	return microchip_mac.NewMicrochipMac(cfg)
}

func createPowerSensorConfig(cfg *sources.TimeSourceConfig) interface{} {
	if cfg == nil {
		return nil
	}
	return power_sensor_INA230.NewPowerSensorINA230(cfg)
}

// Start по дампу (0x4ba3720): StartGNSSClient, StartClockGenClient, StartEthernetSwitchClient, StartOscillatorClient, StartPowerSensorClient.
func (c *Client) Start() {
	c.StartGNSSClient()
	c.StartClockGenClient()
	c.StartEthernetSwitchClient()
	c.StartOscillatorClient()
	c.StartPowerSensorClient()
}

// StartGNSSClient по дампу (0x4ba3b80): device!=nil → device.Start(); go runGNSSRunloop.
func (c *Client) StartGNSSClient() {
	if c == nil || c.device == nil {
		if c != nil && c.logger != nil {
			c.logger.Warn("open_timecard_v1: GNSS device nil")
		}
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

// StartClockGenClient по дампу: clockgen.Start() (заглушка).
func (c *Client) StartClockGenClient() {
	if c != nil && c.clockgen != nil {
		if cg, ok := c.clockgen.(interface{ Start() }); ok {
			cg.Start()
		}
	}
}

// StartEthernetSwitchClient по дампу: ethSwitch.Start() (заглушка).
func (c *Client) StartEthernetSwitchClient() {
	if c != nil && c.ethSwitch != nil {
		if es, ok := c.ethSwitch.(interface{ Start() }); ok {
			es.Start()
		}
	}
}

// StartOscillatorClient по дампу: oscillator.Start() (заглушка).
func (c *Client) StartOscillatorClient() {
	if c != nil && c.oscillator != nil {
		if osc, ok := c.oscillator.(interface{ Start() }); ok {
			osc.Start()
		}
	}
}

// StartPowerSensorClient по дампу: powerSensor.Start() (заглушка).
func (c *Client) StartPowerSensorClient() {
	if c != nil && c.powerSensor != nil {
		if ps, ok := c.powerSensor.(interface{ Start() }); ok {
			ps.Start()
		}
	}
}

// ExecutePullIn по дампу (0x4ba1900): парсинг аргументов (Atoi, ParseInt base 16), clockgen+0x8 → clockgen.ExecutePullIn(phase).
func (c *Client) ExecutePullIn(phase int) {
	if c == nil || c.clockgen == nil {
		return
	}
	if cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012); ok {
		cg.ExecutePullIn(phase)
	}
}

// Reset по дампу (0x4ba1180): client+0x8 = clockgen, clockgen.Reset(); при ошибке — возврат строки.
func (c *Client) Reset() error {
	if c == nil || c.clockgen == nil {
		return nil
	}
	if cg, ok := c.clockgen.(interface{ Reset() error }); ok {
		return cg.Reset()
	}
	return nil
}

// ShowDpllStatus по дампу (0x4ba1680): strconv.Atoi(phaseStr) → idx; GetDPLLState(cg, idx); DPLLStatusToString; GetDPLLRefState(cg, idx); DPLLRefStatusToString; конкатенация строк.
func (c *Client) ShowDpllStatus(phaseStr string) string {
	if c == nil || c.clockgen == nil {
		return ""
	}
	phase, err := strconv.Atoi(phaseStr)
	if err != nil {
		return phaseStr + " invalid"
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return ""
	}
	state, err := cg.GetDPLLState(phase)
	if err != nil {
		return err.Error()
	}
	statusStr := clock_gen_8A34002E.DPLLStatusToString(state)
	refState, err := cg.GetDPLLRefState(phase)
	if err != nil {
		return err.Error()
	}
	refStr := clock_gen_8A34002E.DPLLRefStatusToString(refState)
	return statusStr + fmt.Sprintf(" %s", refStr)
}

// ShowClockgenVersion по дампу: clockgen.GetMajorRelease, GetMinorRelease, GetHotfixRelease; "major.minor.hotfix".
func (c *Client) ShowClockgenVersion() string {
	if c == nil || c.clockgen == nil {
		return "0.0.0"
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return "0.0.0"
	}
	major, err := cg.GetMajorRelease()
	if err != nil {
		return "0.0.0"
	}
	minor, err := cg.GetMinorRelease()
	if err != nil {
		return "0.0.0"
	}
	hotfix, err := cg.GetHotfixRelease()
	if err != nil {
		return "0.0.0"
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, hotfix)
}

// ShowClockgenConfigStatus по дампу: clockgen.GetTimebeatClockgenConfigVersion().
func (c *Client) ShowClockgenConfigStatus() string {
	if c == nil || c.clockgen == nil {
		return "unknown"
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return "unknown"
	}
	s, err := cg.GetTimebeatClockgenConfigVersion()
	if err != nil {
		return "unknown"
	}
	return s
}

// ShowInputStatus по дампу: strconv.Atoi(inputStr) → idx; GetInputMonStatus(cg, idx); InputMonStatusToString.
func (c *Client) ShowInputStatus(inputStr string) string {
	if c == nil || c.clockgen == nil {
		return ""
	}
	idx, err := strconv.Atoi(inputStr)
	if err != nil {
		return inputStr + " invalid"
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return ""
	}
	status, err := cg.GetInputMonStatus(idx)
	if err != nil {
		return err.Error()
	}
	return clock_gen_8A34002E.InputMonStatusToString(status)
}

// SetDpllFodFreq по дампу: Atoi(idxStr), ParseUint base 10 (val64Str), ParseUint base 16 (val16Str); clockgen.SetDpllFodFreq(idx, high, low).
func (c *Client) SetDpllFodFreq(idxStr, val64Str, val16Str string) error {
	if c == nil || c.clockgen == nil {
		return fmt.Errorf("no clockgen")
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return fmt.Errorf("clockgen not ClockGen8A34012")
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return err
	}
	high, err := strconv.ParseUint(val64Str, 10, 64)
	if err != nil {
		return err
	}
	low, err := strconv.ParseUint(val16Str, 16, 16)
	if err != nil {
		return err
	}
	return cg.SetDpllFodFreq(idx, high, uint16(low))
}

// SetOutputDiv по дампу: Atoi(idxStr), ParseUint base 10 (valStr); clockgen.SetOutputDiv(idx, value).
func (c *Client) SetOutputDiv(idxStr, valStr string) error {
	if c == nil || c.clockgen == nil {
		return fmt.Errorf("no clockgen")
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return fmt.Errorf("clockgen not ClockGen8A34012")
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return err
	}
	value, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return err
	}
	return cg.SetOutputDiv(idx, value)
}

// SetOutputDutyCycleHigh по дампу: Atoi(idxStr), ParseUint base 10; clockgen.SetOutputDutyCycleHigh(idx, value).
func (c *Client) SetOutputDutyCycleHigh(idxStr, valStr string) error {
	if c == nil || c.clockgen == nil {
		return fmt.Errorf("no clockgen")
	}
	cg, ok := c.clockgen.(*clock_gen_8A34002E.ClockGen8A34012)
	if !ok {
		return fmt.Errorf("clockgen not ClockGen8A34012")
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return err
	}
	value, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return err
	}
	return cg.SetOutputDutyCycleHigh(idx, value)
}

// runGNSSRunloop по дампу (0x4ba3d20): select ChGSV/ChObs/ChTAI; decorateObservation + RegisterObservation; NotifyTAIOffset.
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

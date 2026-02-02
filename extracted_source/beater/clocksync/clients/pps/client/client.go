package client

import (
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/pps/events_linux"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// EventData — пакет для отправки в контроллер (дубликат из pps для избежания циклического импорта).
type EventData struct {
	ClientKey string
	Nsec      int64
}

// Client по дизассемблеру PPS: 0x8/0x10=URI, 0x88=Timer (Reset 10s в ProcessEvent), 0x308=counter, 0x310=flag, 0x321=enablePPSPending (Start.func1 сбрасывает), 0x328=enablePPSDelay.
type Client struct {
	URI             string
	Timer           *time.Timer
	Counter         int64
	WarnFlag        bool
	Logger          *logging.Logger
	baseOffset      int64
	eventCh         chan EventData
	clientKey       string
	enablePPSDelay  time.Duration // 0x328 — задержка до EnablePPS; AfterFunc вызывает Start.func1
	enablePPSPending bool         // 0x321 — флаг, сбрасывается в Start.func1
}

const (
	ppsTimerReset   = 10 * time.Second
	ppsWarnThresholdNs = 500_000_000 // 0x1dcd6500
	ppsWarnMaxCount = 30
	oneSecondNs     = 1_000_000_000  // 0x3b9aca00
)

// ProcessEvent по дизассемблеру (0x45bd7e0): client+0x88 Timer.Reset(10s); GetController(), GetClockWithURI(client+0x8/0x10), PPSRegistered(clock); если client+0x310: разбор event (nsec); вызов SubmitOffset (call 45bdca0 в дампе).
func (c *Client) ProcessEvent(nsec int64) {
	if c.Timer != nil {
		c.Timer.Reset(ppsTimerReset)
	}
	hcc := hostclocks.GetController()
	if hcc != nil {
		hc := hcc.GetClockWithURI(c.URI)
		if hc != nil {
			hc.PPSRegistered()
		}
	}
	// TODO: при c.WarnFlag — разбор nsec: sec = nsec/1e9, nsec = nsec%1e9, запись в слайс тройок (sec, nsec, *time.Location)
	_ = nsec
	c.SubmitOffset()
}

// SubmitOffset по дизассемблеру (0x45bdca0): getSecondaryOffset(); если |offset|+0x1dcd6500 > 1e9: client+0x308 inc (max 30), Logger.Warn; иначе если counter>0 dec; observation: offset = clamp(-1e9..1e9)+client+0x148; category=5; RegisterObservation. При counter>=15 — Logger.Warn иначе 0x98=1.
func (c *Client) SubmitOffset() {
	offset, err := c.getSecondaryOffset()
	if err != nil {
		return
	}
	absOffset := offset
	if absOffset < 0 {
		absOffset = -absOffset
	}
	if absOffset+ppsWarnThresholdNs > oneSecondNs {
		if c.Counter < ppsWarnMaxCount {
			c.Counter++
		}
		if c.Logger != nil {
			c.Logger.Warn("PPS offset exceeds threshold")
		}
		return
	}
	if c.Counter > 0 {
		c.Counter--
	}
	// clamp offset to ±1e9, add client.baseOffset
	clamped := offset
	if clamped > oneSecondNs {
		clamped = oneSecondNs
	}
	if clamped < -oneSecondNs {
		clamped = -oneSecondNs
	}
	offsetNs := clamped + c.baseOffset
	ctrl := servo.GetController()
	if ctrl != nil && ctrl.GetOffsets() != nil {
		ctrl.GetOffsets().RegisterObservationWithCategory(c.URI, offsetNs, 5)
	}
}

// getSecondaryOffset по дизассемблеру (0x45be0a0): servo.GetController().GetSecondarySourcesOffset(); hostclocks.GetController(), GetClockWithURI(client.URI); если clock==nil — return offset; если clock.isMaster — GetClockWithURI("phc.0"), ref; иначе ref=clock.ref; return offset - ref.
func (c *Client) getSecondaryOffset() (int64, error) {
	ctrl := servo.GetController()
	if ctrl == nil {
		return 0, nil
	}
	offset, err := ctrl.GetSecondarySourcesOffset()
	if err != nil {
		return 0, err
	}
	hcc := hostclocks.GetController()
	if hcc == nil {
		return offset, nil
	}
	hc := hcc.GetClockWithURI(c.URI)
	if hc == nil {
		return offset, nil
	}
	var ref int64
	if hc.IsMaster() {
		refHC := hcc.GetClockWithURI("phc.0")
		if refHC != nil {
			ref = refHC.GetOffset()
		}
	} else {
		ref = hc.GetOffset()
	}
	return offset - ref, nil
}

func EnableExtTs() {}

func NewClient() *Client {
	return &Client{Logger: logging.NewLogger("pps-client")}
}

// NewClientWithConfig создаёт PPS client с URI и baseOffset из config.
func NewClientWithConfig(uri string, baseOffset int64) *Client {
	c := NewClient()
	if c == nil {
		return nil
	}
	c.URI = uri
	c.baseOffset = baseOffset
	return c
}

func PPSWatchDogTimerExpired() {}

func PPSWatchDogTimerExpiredFm() {}

// onPPSWatchDogExpired — callback по истечении watchdog (по дампу Logger.Warn при отсутствии PPS).
func (c *Client) onPPSWatchDogExpired() {
	if c != nil && c.Logger != nil {
		c.Logger.Warn("PPS watchdog expired")
	}
}

// RunSanityTicker по дизассемблеру (0x45bd360): тикер/проверка валидности PPS. Заглушка.
func RunSanityTicker() {}

func SetCedControlToSecondarySources() {}

func SetMonitorOnly() {}

// SetEventChannel задаёт канал и ключ для отправки PPS событий в контроллер.
func (c *Client) SetEventChannel(ch chan EventData, key string) {
	c.eventCh = ch
	c.clientKey = key
}

// Start по дизассемблеру (0x45bd4a0): PPSWatchDogTimerExpired AfterFunc(10s); client+0x88=Timer; EnableExtTs; Logger.Info/Warn; при enablePPSDelay>0 — AfterFunc(enablePPSDelay, Start.func1); go runEventLoop.
func (c *Client) Start() {
	// По дампу: closure PPSWatchDogTimerExpired-fm с client, time.AfterFunc(10s)
	timer := time.AfterFunc(ppsTimerReset, func() {
		c.onPPSWatchDogExpired()
	})
	c.Timer = timer
	// EnableExtTs — по дампу call 45bd740
	extTsOk := c.enableExtTs()
	if !extTsOk && c.Logger != nil {
		c.Logger.Warn("EnableExtTs failed")
	} else if c.Logger != nil {
		c.Logger.Info("PPS client started", 0)
	}
	// client+0x328: при enablePPSDelay>0 — AfterFunc(enablePPSDelay, Start.func1)
	if c.enablePPSDelay > 0 {
		time.AfterFunc(c.enablePPSDelay, func() {
			c.startFunc1()
		})
	}
	// runEventLoop — наш цикл чтения PPS (events_linux или симуляция)
	if c.eventCh != nil && c.clientKey != "" {
		go c.runEventLoop()
	}
}

func (c *Client) enableExtTs() bool {
	EnableExtTs()
	return true
}

// startFunc1 по дампу (0x45bd600): Logger.Info; client+0x321=0.
func (c *Client) startFunc1() {
	if c.Logger != nil {
		c.Logger.Info("enablePPS delay elapsed", 0)
	}
	c.enablePPSPending = false
}

// runEventLoop по дампу: цикл чтения PPS. Linux — events_linux.RunPPSPollLoop (PPS_FETCH ioctl); иначе — симуляция тикером 1s.
func (c *Client) runEventLoop() {
	send := func(nsec int64) {
		select {
		case c.eventCh <- EventData{ClientKey: c.clientKey, Nsec: nsec}:
		default:
			if c.Logger != nil {
				c.Logger.Warn("PPS event channel full, dropping")
			}
		}
	}
	// По дампу phc.KernelPPSSource.TimePPSFetch — опрос /dev/pps0. ppsIndex=0 для первого устройства.
	events_linux.RunPPSPollLoop(0, time.Second, send)
}

func inittask() {
	// TODO: реконструировать
}

func resetEnablePPSTimer() {
	// TODO: реконструировать
}


package source_watchdog

import (
	"fmt"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Strategy по дампу: 0=не задан, 1=NoGroupMembers, 2=SourceRangeExceeded.
const (
	StrategyNone                 = 0
	StrategyNoGroupMembers       = 1
	StrategySourceRangeExceeded  = 2
)

// StrategyNames для логирования.
var StrategyNames = []string{"none", "no_group_members", "source_range_exceeded"}

// WatchdogConfig — конфиг для CreateModuleInstance (по дампу config+0x8, 0x10, 0x28-0x58, 0x40, 0x48).
type WatchdogConfig struct {
	GroupMembersCheck interface{} // +0x8: не nil → strategy 1
	DoneCh            chan struct{}
	// Поля для NewSteadyState (0x28-0x58)
	Threshold   float64
	WindowSize  int
	SteadyMin   float64
	SteadyMax   float64
	RangeMin    float64
	RangeMax    float64
}

// WatchDogEvents — события watchdog, map key → chan (Subscribe).
type WatchDogEvents struct {
	mu   sync.Mutex
	subs map[string]chan interface{}
}

// NewWatchDogEvents создаёт WatchDogEvents.
func NewWatchDogEvents() *WatchDogEvents {
	return &WatchDogEvents{subs: make(map[string]chan interface{})}
}

// Subscribe по дампу (0x4b6f8c0): возвращает chan для key; создаёт при отсутствии.
func (e *WatchDogEvents) Subscribe(key string) chan interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch, ok := e.subs[key]; ok {
		return ch
	}
	ch := make(chan interface{}, 8)
	e.subs[key] = ch
	return ch
}

// Unsubscribe по дампу: удаляет подписку.
func (e *WatchDogEvents) Unsubscribe(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch, ok := e.subs[key]; ok {
		close(ch)
		delete(e.subs, key)
	}
}

// Publish отправляет событие подписчикам.
func (e *WatchDogEvents) Publish(key string, ev interface{}) {
	e.mu.Lock()
	ch := e.subs[key]
	e.mu.Unlock()
	if ch != nil {
		select {
		case ch <- ev:
		default:
		}
	}
}

// WatchdogModule по дампу: 0=strategy, 0x18=logger, 0x20=holdoverCounter, 0x30=tickCounter, 0x38=config, 0x40=steadyState, 0x48=events.
type WatchdogModule struct {
	Strategy        int64
	Logger          *logging.Logger
	HoldoverCounter int64
	TickCounter     int64
	Config          *WatchdogConfig
	SteadyState     *algos.SteadyState
	Events          *WatchDogEvents
	ModuleID        string
}

// CreateModuleInstance по дампу (0x4b6fb40): WatchDogEvents, WatchdogModule, configureStrategy.
func CreateModuleInstance(config *WatchdogConfig, partID string) *WatchdogModule {
	if config == nil {
		config = &WatchdogConfig{}
	}
	moduleID := "watchdog"
	if partID != "" {
		moduleID = "watchdog" + partID
	}
	logger := logging.NewLogger(moduleID)
	events := NewWatchDogEvents()
	mod := &WatchdogModule{
		Logger:   logger,
		Config:   config,
		Events:   events,
		ModuleID: moduleID,
	}
	mod.configureStrategy()
	return mod
}

// configureStrategy по дампу (0x4b6ffc0): config+0x8 → strategy 1; config+0x40,0x48 → strategy 2; NewSteadyState; логирование.
func (m *WatchdogModule) configureStrategy() {
	if m.Config == nil {
		return
	}
	if m.Config.GroupMembersCheck != nil {
		m.Strategy = StrategyNoGroupMembers
	}
	if m.Config.Threshold != 0 || m.Config.WindowSize != 0 {
		if m.Config.SteadyMin != 0 || m.Config.SteadyMax != 0 || m.Config.RangeMin != 0 || m.Config.RangeMax != 0 {
			if m.Strategy == StrategyNoGroupMembers {
				m.Strategy = StrategySourceRangeExceeded
			}
		}
	}
	if m.Config.WindowSize > 0 {
		m.SteadyState = algos.NewSteadyState(m.Config.Threshold, m.Config.WindowSize)
	} else {
		m.SteadyState = algos.NewSteadyState(0.01, 60)
	}
	if m.Strategy > 0 && m.Strategy < int64(len(StrategyNames)) {
		name := StrategyNames[m.Strategy]
		if m.Logger != nil {
			m.Logger.Info(fmt.Sprintf("watchdog strategy: %s", name), 0)
		}
	}
}

// Start по дампу (0x4b6fca0): Sleep 60s; если strategy==0 → Warn; иначе go runloop.
func (m *WatchdogModule) Start() {
	time.Sleep(60 * time.Second)
	if m.Strategy == StrategyNone {
		if m.Logger != nil {
			m.Logger.Warn("watchdog not started: no strategy configured")
		}
		return
	}
	go m.runloop()
}

// runloop по дампу (0x4b6fe80): ticker 1s, select ticker vs DoneCh; на tick → processPeriodic.
func (m *WatchdogModule) runloop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	doneCh := m.Config.DoneCh
	if doneCh == nil {
		doneCh = make(chan struct{})
	}
	for {
		select {
		case <-ticker.C:
			m.processPeriodic()
		case <-doneCh:
			return
		}
	}
}

// processPeriodic по дампу (0x4b6ff40): tickCounter++; если >= 60 → execute по strategy.
func (m *WatchdogModule) processPeriodic() {
	m.TickCounter++
	if m.TickCounter < 60 {
		return
	}
	m.TickCounter = 0
	switch m.Strategy {
	case StrategyNoGroupMembers:
		m.executeHoldoverOnNoGroupMembers()
	case StrategySourceRangeExceeded:
		m.executeHoldoverOnSourceRangeExceeded()
	}
}

// executeHoldoverOnNoGroupMembers по дампу (0x4b70180): enterHoldover.
func (m *WatchdogModule) executeHoldoverOnNoGroupMembers() {
	m.enterHoldover()
}

// executeHoldoverOnSourceRangeExceeded по дампу (0x4b70200): проверки, enterHoldover/leaveHoldover.
func (m *WatchdogModule) executeHoldoverOnSourceRangeExceeded() {
	if m.Config == nil || m.SteadyState == nil {
		return
	}
	// Упрощённая логика: при необходимости enter/leave holdover
	if m.SteadyState.IsSteady() {
		if m.HoldoverCounter > 0 {
			m.HoldoverCounter--
		}
		m.leaveHoldover()
	} else {
		if m.HoldoverCounter < 10 {
			m.HoldoverCounter++
		}
		m.enterHoldover()
	}
}

// enterHoldover по дампу: логирование/действия при входе в holdover.
func (m *WatchdogModule) enterHoldover() {
	if m.Logger != nil {
		m.Logger.Warn("watchdog: enter holdover")
	}
}

// leaveHoldover по дампу: логирование при выходе из holdover.
func (m *WatchdogModule) leaveHoldover() {
	if m.Logger != nil {
		m.Logger.Info("watchdog: leave holdover", 0)
	}
}

// AddAlgoOutputs по дампу: добавляет algo outputs (заглушка).
func (m *WatchdogModule) AddAlgoOutputs(_ interface{}) {}

// SetMovingMinimum по дампу: задаёт moving minimum (заглушка).
func (m *WatchdogModule) SetMovingMinimum(_ float64) {}

// CardConfigPartsToWatchdogConfig — большой конвертер (12KB в дампе); заглушка.
func CardConfigPartsToWatchdogConfig(_ interface{}) *WatchdogConfig {
	return &WatchdogConfig{}
}

// Package servo — контроллер синхронизации.
// RunPeriodicAdjustSlaveClocks по дизассемблеру: NewTicker(0x3b9aca00)=1s, Sleep(50ms+rand(0..150ms)), select;
// затем StepSlaveClocksIfNecessary(hostClockCtrl,-1), SlewSlaveClocks(hostClockCtrl). См. code_analysis/disassembly/Controller_RunPeriodicAdjustSlaveClocks.txt.
package servo

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos"
	"github.com/shiwa/timecard-mini/extracted-source/beater/utility"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/config"
)

// Controller — главный контроллер servo
type Controller struct {
	mu                 sync.Mutex
	offsets            *Offsets
	clockQuality       *ClockQuality
	algo               Algo
	masterClock        string
	running            bool
	stepEnabled        bool
	stepAndExitEnabled bool
	adjustClock        bool  // false = только мониторинг
	stepLimitNs        int64 // порог step (наносекунды)
	manualOverride     bool  // SetManualOverride: передаётся в HostClock по masterClock URI
	aliveTime          time.Time
	lastServoTime      time.Time
	servoInterval      time.Duration
	servoIntervalStr    string  // по дизассемблеру RunWakeupServoLoop: appConfig.servo_interval → ParseTimeString; пусто = использовать servoInterval
	// fineTuneServoTicker по дизассемблеру: 0x18(controller).0x68=counter, 0x58=scale, 0x70=durationNs, 0x78=flag
	fineTuneCounter  int64
	fineTuneScale    byte
	fineTuneDuration int64
	fineTuneAdjusted bool
}

var controllerInstance *Controller
var controllerOnce sync.Once

// GetController возвращает singleton контроллер
func GetController() *Controller {
	controllerOnce.Do(func() {
		controllerInstance = NewController()
	})
	return controllerInstance
}

// NewController создаёт контроллер (по умолчанию PID, adjust_clock=true, step 500ms)
func NewController() *Controller {
	return &Controller{
		offsets:       NewOffsets(),
		clockQuality:  NewClockQuality(),
		algo:          NewAlgo("pid", 0.5, algos.DefaultAlgoCoefficients.Ki, algos.DefaultAlgoCoefficients.Kd),
		adjustClock:   true,
		stepLimitNs:   500 * int64(time.Millisecond),
		aliveTime:     time.Now(),
		lastServoTime: time.Now(),
		servoInterval: time.Second,
	}
}

// SetConfig задаёт параметры из конфига (вызывается из clocksync.RunWithConfig).
func (c *Controller) SetConfig(adjustClock bool, stepLimitNs int64, interval time.Duration, algorithm string, kp, ki, kd float64) {
	c.SetConfigWithIntervalStr(adjustClock, stepLimitNs, interval, "", algorithm, kp, ki, kd)
}

// SetConfigWithIntervalStr задаёт параметры и строку интервала (servo_interval из appConfig для RunWakeupServoLoop).
func (c *Controller) SetConfigWithIntervalStr(adjustClock bool, stepLimitNs int64, interval time.Duration, intervalStr string, algorithm string, kp, ki, kd float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adjustClock = adjustClock
	if stepLimitNs > 0 {
		c.stepLimitNs = stepLimitNs
	}
	if interval > 0 {
		c.servoInterval = interval
	}
	if intervalStr != "" {
		c.servoIntervalStr = intervalStr
	}
	c.algo = NewAlgo(algorithm, kp, ki, kd)
}

// SetStepAndExitEnabled задаёт step_and_exit из конфига (Run.func2 EnableStepAndExitDieTimeout).
func (c *Controller) SetStepAndExitEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stepAndExitEnabled = enabled
}

// Run запускает контроллер. По дизассемблеру (0x45a07e0): 6 goroutines — func1..func6.
// func1: Offsets.RunProcessObservationsLoop; func2 (условно stepAndExit): EnableStepAndExitDieTimeout;
// func3: RunRTCSetLoop; func4: RunTimers; func5: RunWakeupServoLoop; func6 (условно HTTP): Offsets.RunSetLogEntriesLoop.
func (c *Controller) Run(ctx context.Context) error {
	c.mu.Lock()
	c.running = true
	stepAndExit := c.stepAndExitEnabled
	c.mu.Unlock()

	// func1: RunProcessObservationsLoop
	if c.offsets != nil {
		go c.offsets.RunProcessObservationsLoop()
	}
	// func2: условно stepAndExit — EnableStepAndExitDieTimeout
	if stepAndExit {
		go c.EnableStepAndExitDieTimeout()
	}
	// func3: RunRTCSetLoop
	go c.RunRTCSetLoop()
	// func4: RunTimers
	go c.RunTimers(ctx)
	// func5: RunWakeupServoLoop
	go c.RunWakeupServoLoop(ctx)
	// func6: условно HTTP — RunSetLogEntriesLoop
	if cfg := config.GetAppConfig(); cfg != nil && cfg.HTTPEnabled {
		if c.offsets != nil {
			go c.offsets.RunSetLogEntriesLoop()
		}
	}

	<-ctx.Done()

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	return nil
}

// RunTimers запускает таймеры
func (c *Controller) RunTimers(ctx context.Context) {
	ticker := time.NewTicker(c.servoInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.doPeriodicTasks()
		}
	}
}

// RunWakeupServoLoop запускает servo loop (по дизассемблеру: appConfig servo_interval → ParseTimeString; цикл с RunPeriodicAdjustSlaveClocks; fineTuneServoTicker вызывается внутри RunPeriodicAdjustSlaveClocks).
func (c *Controller) RunWakeupServoLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.RunPeriodicAdjustSlaveClocks()
			// Интервал сна: по дизассемблеру appConfig.servo_interval парсится через ParseTimeString; при пустой строке — default 1s.
			sleepDur := c.servoInterval
			if c.servoIntervalStr != "" {
				sleepDur = utility.ParseTimeString(c.servoIntervalStr, time.Second)
			}
			if sleepDur < time.Millisecond {
				sleepDur = time.Second
			}
			time.Sleep(sleepDur)
		}
	}
}

// fineTuneServoTicker по дизассемблеру (__Controller_.fineTuneServoTicker 0x45a1ca0): GetUTCTimeFromMasterClock, delta=now.Sub(last); если delta<=100ms — обнулить counter, return (delta>100ms); иначе counter++; если counter>3 — pow(10,scale), обновить duration, flag=1, Logger.Warn; return (delta>100ms).
const fineTuneDeltaThresholdNs = 100_000_000 // 0x5f5e100 = 100 ms

func (c *Controller) fineTuneServoTicker(now, last time.Time) bool {
	delta := now.Sub(last).Nanoseconds()
	if delta <= fineTuneDeltaThresholdNs {
		c.mu.Lock()
		c.fineTuneCounter = 0
		c.mu.Unlock()
		return false
	}
	c.mu.Lock()
	c.fineTuneCounter++
	if c.fineTuneCounter > 3 {
		// В бинарнике: pow(10, scale), scale duration, 0x70=durationNs, 0x78=1, Logger.Warn
		c.fineTuneAdjusted = true
		c.mu.Unlock()
		return true
	}
	c.mu.Unlock()
	return true
}

// RunPeriodicAdjustSlaveClocks корректирует slave часы: offset → algo → adjusttime (или step). По бинарнику вызывается fineTuneServoTicker(now, last).
func (c *Controller) RunPeriodicAdjustSlaveClocks() {
	c.mu.Lock()
	now := time.Now()
	last := c.lastServoTime
	c.lastServoTime = now
	dt := now.Sub(last)
	offsetNs := c.bestOffsetNs()
	adjustClock := c.adjustClock
	stepLimitNs := c.stepLimitNs
	c.mu.Unlock()

	// По дизассемблеру doPeriodicTasks/RunWakeupServoLoop: вызов fineTuneServoTicker(now, last); при true — интервал отстаёт от master.
	_ = c.fineTuneServoTicker(now, last)

	if !adjustClock {
		return
	}
	if dt <= 0 {
		return
	}
	offsetAbs := offsetNs
	if offsetAbs < 0 {
		offsetAbs = -offsetAbs
	}
	if stepLimitNs > 0 && int64(offsetAbs) > stepLimitNs {
		refTime := time.Now().UTC().Add(time.Duration(int64(offsetNs)))
		_ = adjusttime.StepClockUsingSetTimeSyscall(0, refTime) // clockID 0 = CLOCK_REALTIME
		return
	}
	freq := c.algo.Calculate(offsetNs, dt)
	_ = adjusttime.SetFrequency(0, freq) // clockID 0 = CLOCK_REALTIME
}

// bestOffsetNs возвращает текущий offset для коррекции (медиана активных источников или 0)
func (c *Controller) bestOffsetNs() float64 {
	candidates := c.offsets.GetSourceCandidates()
	if len(candidates) == 0 {
		return 0
	}
	vals := make([]int64, 0, len(candidates))
	for _, ts := range candidates {
		vals = append(vals, ts.offset)
	}
	return float64(medianInt64(vals))
}

func medianInt64(a []int64) int64 {
	if len(a) == 0 {
		return 0
	}
	b := make([]int64, len(a))
	copy(b, a)
	for i := 0; i < len(b); i++ {
		for j := i + 1; j < len(b); j++ {
			if b[j] < b[i] {
				b[i], b[j] = b[j], b[i]
			}
		}
	}
	k := len(b) / 2
	if len(b)%2 == 0 {
		return (b[k-1] + b[k]) / 2
	}
	return b[k]
}

// RunPeriodicMasterClockElection выбирает master по приоритету (по умолчанию — первый активный источник).
func (c *Controller) RunPeriodicMasterClockElection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	candidates := c.offsets.GetSourceCandidates()
	if len(candidates) > 0 && c.masterClock == "" {
		c.masterClock = candidates[0].id
	}
}

// RunPeriodicLogForAllClocks логирует состояние всех часов (вызов hostclocks.LogForSlaveClocks при наличии контроллера).
func (c *Controller) RunPeriodicLogForAllClocks() {
	hcc := hostclocks.GetHostClockController()
	if hcc != nil {
		hcc.LogForSlaveClocks()
	}
}

// RunRTCSetLoop по дизассемблеру (0x45a8320): если appConfig.flag — NewTicker(controller.0x18.0x20), в цикле StepRTCClock(), select; иначе лог и return.
func (c *Controller) RunRTCSetLoop() {
	c.mu.Lock()
	interval := c.servoInterval
	c.mu.Unlock()
	if interval < time.Millisecond {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		_ = adjusttime.StepRTCClock()
	}
}

// updateServoIntervalIfRequired по дизассемблеру (0x45a1bc0): если controller.0x58==scale — return; иначе 0x58=scale, 0x59=1, GetHostClockController().UpdatePPS(scale); при adjustClock — Fprintf (лог).
func (c *Controller) updateServoIntervalIfRequired(scale byte) {
	c.mu.Lock()
	if c.fineTuneScale == scale {
		c.mu.Unlock()
		return
	}
	c.fineTuneScale = scale
	c.fineTuneAdjusted = true
	hcc := hostclocks.GetHostClockController()
	adjustClock := c.adjustClock
	c.mu.Unlock()
	if hcc != nil {
		hcc.UpdatePPS(scale)
	}
	if adjustClock {
		// В бинарнике: fmt.Fprintf(os.Stdout, ...) с scale
		_ = scale
	}
}

// getSourcesSnapshot по дизассемблеру (0x45a8860): controller+0x10=offsets; GetSourcesSnapshot(1); если doWeHaveSourcesForType(snapshot,1) — return snapshot; иначе GetSourcesSnapshot(2), return.
func (c *Controller) getSourcesSnapshot() map[string]*TimeSource {
	snap := c.offsets.GetSourcesSnapshotForCategory(1)
	if c.doWeHaveSourcesForType(snap, 1) {
		return snap
	}
	return c.offsets.GetSourcesSnapshotForCategory(2)
}

// doWeHaveSourcesForType по дизассемблеру (doWeHaveSourcesForType@@Base): проверка, есть ли источники данного типа в снимке; return len>0.
func (c *Controller) doWeHaveSourcesForType(snapshot map[string]*TimeSource, category int) bool {
	return len(snapshot) > 0
}

// addClockNamesToSourcesSnapshot по дизассемблеру (0x45a2720): итерация по снимку; для каждого источника GetClockWithURI(id); при найденном clock и category>=2 — PHC.GetDeviceName → ts.clockName; иначе "system".
func (c *Controller) addClockNamesToSourcesSnapshot(snapshot map[string]*TimeSource) {
	hcc := hostclocks.GetHostClockController()
	if hcc == nil {
		return
	}
	for _, ts := range snapshot {
		ts.mu.Lock()
		cat := ts.category
		ts.mu.Unlock()
		hc := hcc.GetClockWithURI(ts.id)
		if hc != nil && cat >= 2 {
			name, _ := hc.GetPHCDeviceName()
			ts.mu.Lock()
			ts.clockName = name
			ts.mu.Unlock()
		} else {
			ts.mu.Lock()
			ts.clockName = "system"
			ts.mu.Unlock()
		}
	}
}

// doPeriodicTasks выполняет периодические задачи (по дизассемблеру: getSourcesSnapshot, addClockNamesToSourcesSnapshot, ..., updateServoIntervalIfRequired(scale)).
func (c *Controller) doPeriodicTasks() {
	snap := c.getSourcesSnapshot()
	c.addClockNamesToSourcesSnapshot(snap)
	c.RunPeriodicMasterClockElection()
	c.RunPeriodicLogForAllClocks()
	c.mu.Lock()
	scale := c.fineTuneScale
	c.mu.Unlock()
	c.updateServoIntervalIfRequired(scale)
}

// ChangeMasterClock меняет master
func (c *Controller) ChangeMasterClock(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.masterClock = name
}

// HoldMasterClockElection удерживает выбор master (однократный вызов RunPeriodicMasterClockElection).
func (c *Controller) HoldMasterClockElection() {
	c.RunPeriodicMasterClockElection()
}

// ResetAllServos сбрасывает все servo (algo.ResetServo при наличии интерфейса).
func (c *Controller) ResetAllServos() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.algo.(interface{ ResetServo() }); ok {
		r.ResetServo()
	}
}

// CheckIfWeShouldStep проверяет необходимость step: true если |offsetNs| > stepLimitNs.
func (c *Controller) CheckIfWeShouldStep(offsetNs int64) bool {
	c.mu.Lock()
	limit := c.stepLimitNs
	c.mu.Unlock()
	if limit <= 0 {
		return false
	}
	if offsetNs < 0 {
		offsetNs = -offsetNs
	}
	return offsetNs > limit
}

// DoWeRun проверяет работу контроллера
func (c *Controller) DoWeRun() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// EnableStepAndExitDieTimeout по дизассемблеру (0x45a9e40): NewTimer(servoInterval), select(timer.C, ...); при timer — AnnotationLogEntry.Log, Logger.Critical, Sleep(2s), os.Exit(255).
func (c *Controller) EnableStepAndExitDieTimeout() {
	c.mu.Lock()
	interval := c.servoInterval
	c.mu.Unlock()
	if interval < time.Millisecond {
		interval = time.Second
	}
	timer := time.NewTimer(interval)
	<-timer.C
	timer.Stop()
	entry := &logging.AnnotationLogEntry{Source: "step_and_exit", Message: "step and exit die timeout"}
	entry.Log()
	if lg := logging.GetErrorLogger(); lg != nil {
		lg.Critical("step and exit die timeout", 0)
	}
	time.Sleep(2 * time.Second)
	os.Exit(255)
}

// GetAliveTimeInSeconds возвращает время работы
func (c *Controller) GetAliveTimeInSeconds() float64 {
	return time.Since(c.aliveTime).Seconds()
}

// GetClockQuality возвращает ClockQuality
func (c *Controller) GetClockQuality() *ClockQuality {
	return c.clockQuality
}

// GetOffsets возвращает Offsets
func (c *Controller) GetOffsets() *Offsets {
	return c.offsets
}

// GetUTCTimeFromMasterClock по дизассемблеру (GetUTCTimeFromMasterClock@@Base 0x45a1ac0):
// IsRPIComputeModule() → branch; при !IsRPI или (RPI и master PHC name=="eth0" и phc+0x108!=0) — master.GetTimeNow().Add(-master.staticPHCOffset); иначе GetConstructedUTCTimeFromMasterClock.
func (c *Controller) GetUTCTimeFromMasterClock() time.Time {
	hcc := hostclocks.GetHostClockController()
	if hcc == nil {
		return time.Time{}
	}
	master := hcc.GetMasterHostClock()
	if master == nil {
		return time.Time{}
	}
	useRealPath := !adjusttime.IsRPIComputeModule()
	if adjusttime.IsRPIComputeModule() {
		name, n := master.GetPHCDeviceName()
		if n == 4 && name == "eth0" {
			useRealPath = true // в бинарнике дополнительно проверяется phc+0x108
		}
	}
	if useRealPath {
		t := master.GetTimeNow()
		return t.Add(-time.Duration(master.GetStaticPHCOffset()))
	}
	return c.GetConstructedUTCTimeFromMasterClock()
}

// GetConstructedUTCTimeFromMasterClock по дизассемблеру (0x45a1a00): GetMasterHostClock, phc=0x40(master), GetDeviceName; если len==6 и name=="system" — time.Now(); иначе GetClockWithURI("system"), clock.offset, time.Now().Add(offset).
func (c *Controller) GetConstructedUTCTimeFromMasterClock() time.Time {
	hcc := hostclocks.GetHostClockController()
	if hcc == nil {
		return time.Now().UTC()
	}
	master := hcc.GetMasterHostClock()
	if master == nil {
		return time.Now().UTC()
	}
	name, n := master.GetPHCDeviceName()
	if n == 6 && name == "system" {
		return time.Now().UTC()
	}
	hc := hcc.GetClockWithURI("system")
	if hc == nil {
		return time.Now().UTC()
	}
	return time.Now().UTC().Add(time.Duration(hc.GetOffset()))
}

// GetSecondarySourcesOffset по дизассемблеру (0x45a80e0): controller+0x10=Offsets; GetSourcesSnapshotForCategory(2); если snapshot nil/пустой — return (0, error "no secondary sources"); итерация по map → слайс; determineSnapshotKeyIndicators(slice) → offset; return (offset, nil).
func (c *Controller) GetSecondarySourcesOffset() (int64, error) {
	if c.offsets == nil {
		return 0, errNoSecondarySources
	}
	snap := c.offsets.GetSourcesSnapshotForCategory(2)
	if len(snap) == 0 {
		return 0, errNoSecondarySources
	}
	slice := make([]*TimeSource, 0, len(snap))
	for _, ts := range snap {
		slice = append(slice, ts)
	}
	offset := determineSnapshotKeyIndicators(slice)
	return offset, nil
}

// determineSnapshotKeyIndicators по дизассемблеру (0x45a9300): приём слайса *TimeSource; для каждого 0x48/0x50, 0x60/0x68 (offset), арифметика с 0x3b9aca00 (1e9); накопление. Минимальная реконструкция: медиана offset из снимка.
func determineSnapshotKeyIndicators(slice []*TimeSource) int64 {
	if len(slice) == 0 {
		return 0
	}
	vals := make([]int64, 0, len(slice))
	for _, ts := range slice {
		ts.mu.Lock()
		vals = append(vals, ts.offset)
		ts.mu.Unlock()
	}
	return medianInt64(vals)
}

var errNoSecondarySources = errors.New("no secondary sources")

// SetManualOverride по дизассемблеру (SetManualOverride@@Base): GetClockWithURI(controller.0x30), затем HostClock.SetManualOverride(enabled).
func (c *Controller) SetManualOverride(enabled bool) {
	c.mu.Lock()
	uri := c.masterClock
	c.mu.Unlock()
	hc := hostclocks.GetHostClockController().GetClockWithURI(uri)
	if hc != nil {
		hc.SetManualOverride(enabled)
	}
	c.mu.Lock()
	c.manualOverride = enabled
	c.mu.Unlock()
}

// ShouldT3BeAppTimestamp проверяет T3 timestamp
func (c *Controller) ShouldT3BeAppTimestamp() bool {
	return false
}

// StepAndExitInPeace по дизассемблеру (0x45a9f80): StepRTCClock(), Sleep, Log, Info, Sleep, os.Exit(0).
func (c *Controller) StepAndExitInPeace() {
	_ = adjusttime.StepRTCClock()
	os.Exit(0)
}

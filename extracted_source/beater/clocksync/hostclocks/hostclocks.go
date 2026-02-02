// Package hostclocks — управление системными часами
package hostclocks

import (
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/filters"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/statistics"
)

// PHCDeviceInterface — минимальный интерфейс по дизассемблеру (GetDeviceName, GetClockID 0x64, DeterminePHCOffset).
type PHCDeviceInterface interface {
	GetDeviceName() (string, int)
	GetClockID() int   // phc.0x64 по дизассемблеру
	DeterminePHCOffset() int64
}

// OffsetFilterInterface — по дизассемблеру 0x18: фильтр с IsFiltered(offset int64) bool.
type OffsetFilterInterface interface {
	IsFiltered(offset int64) bool
}

// algoScaleUpdater — по дизассемблеру updateAlgoCoefficients: метод с одним аргументом scale (0 или 1).
type algoScaleUpdater interface {
	UpdateScaleFromStore(scale byte)
}

// algoClockFreqUpdater — по дизассемблеру SetHoldoverFrequency: вызов 0x20/0x28(hc) с freq (float64).
type algoClockFreqUpdater interface {
	UpdateClockFreq(freq float64)
}

// algoPPSUpdater — по дизассемблеру UpdatePPS: метод с аргументом scale (byte); вызывается на каждом slave.
type algoPPSUpdater interface {
	UpdatePPS(scale byte)
}

// HostClock — системные часы (по дизассемблеру: 0x40=*PHCDevice, 0x91=isMaster, 0xf0=enabled, 0x100=staticPHCOffset, 0x110=phcMu).
type HostClock struct {
	mu               sync.Mutex
	name             string
	ppsRegistered    bool   // по дампу PPSRegistered: отмечает часы как зарегистрированные для PPS
	clockID          int
	frequency        float64
	offset           int64   // *clock.0 по дизассемблеру — сюда пишем итоговый offset
	phcDevice        interface{} // *phc.PHCDevice по дизассемблеру 0x40
	phcMu            sync.Mutex   // по дизассемблеру 0x110
	filter           interface{} // 0x18 — OffsetFilterInterface для IsFiltered
	staticPHCOffset  int64        // 0x100 — статический PHC offset
	useFilter        bool        // 0xf1 — вызывать IsFiltered перед записью
	flag48                    byte   // 0x48 — флаг (1 если offset отфильтрован)
	enabled                   bool   // 0xf0 — по дизассемблеру StepClock/SlewClockPossiblyAsync
	interferenceMon           *InterferenceMonitor // 0x10 — по дизассемблеру getState/triggerInterference/commitFrequency
	frequencyPpm              float64              // 0x70 — частота для SetFrequency в StepClock
	isMaster                  bool                 // 0x91 — по дизассемблеру SlewClockPossiblyAsync
	suppressWouldHaveSteppedLog byte                // 0x58 — по дизассемблеру LogWouldHaveSteppedMessage: если !=0, не логировать
	// maintainExponentiallyWeightedMovingAveragesClientStore: 0x8 = clockData (clientStore 0x8, ema 0x0, value 0x10); rawData 0x0 (value 0x0), 0x18 = ema2.
	clientStore interface{}       // 0x8.0x8 — куда пишется аргумент clientStore
	ema1        *statistics.EMA   // 0x8.0 — первый EMA.AddValue(value из 0x0)
	ema2        *statistics.EMA   // 0x0.0x18 — второй EMA.AddValue(value из 0x8.0x10)
	// По дизассемблеру NewHostClock: 0x20=itab Algorithm, 0x28=impl (AlgoPID/LinReg/Pi в зависимости от имени устройства).
	algoType        string  // "pid"|"linreg"|"pi" — ветка по GetDeviceName (rho->PID, system/alpha->Pi, beta/gamma/sigma->LinReg/Pi).
	bestFitFiltered *algos.BestFitFiltered // 0x30 по дизассемблеру SetHoldoverFrequency — GetLeastSquaresGradientFiltered
	algorithm       interface{}            // 0x20/0x28 — для UpdateScaleFromStore и UpdateClockFreq (LinReg.UpdateClockFreq и т.д.)
	// 0x108 по дизассемблеру StepClock (4592e05–4592e22): после успешного step — NewKalmanFilter(GetDeviceName), запись в hc+0x108.
	kalmanFilter interface{} // *filters.KalmanFilter
}

// Порог для step (по дизассемблеру StepSlaveClocksIfNecessary: 0x1dcd6500 = 500_000_000 ns = 500 ms).
const stepClockThresholdNs int64 = 500_000_000

// HostClockController — контроллер часов (phcMu в бинарнике на HostClock 0x110, не здесь).
// По дизассемблеру updateRelevantSlaveClocks: 0x8/0x10=slaveClocks, 0x20/0x28/0x30=relevantSlaveClocks (ptr/len/cap), 0x48=includeAllRelevantSlaves.
type HostClockController struct {
	mu                      sync.Mutex
	hostClocks              map[string]*HostClock
	slaveClocks             []*HostClock // по дизассемблеру: 0x8=ptr, 0x10=len
	relevantSlaveClocks     []*HostClock // 0x20/0x28/0x30 — подмножество slave (name=="system" или includeAllRelevantSlaves)
	includeAllRelevantSlaves byte       // 0x48 — если !=0, в relevant попадают все slave
	masterClock             *HostClock
	algoCoeffKp             float64
	algoCoeffKi             float64
	algoCoeffKd             float64
	ppsScale                byte   // 0x68 по дизассемблеру UpdatePPS(scale) — передаётся в HostClock.UpdatePPS
	taiOffsetNs             int64  // 0x78 по дампу processTAISubmission — последний TAI offset
	taiSourceName           string // 0x80/0x88 — имя источника TAI
}

var controller *HostClockController
var once sync.Once

// GetHostClockController возвращает контроллер
func GetHostClockController() *HostClockController {
	once.Do(func() {
		controller = newHostClockController()
	})
	return controller
}

func newHostClockController() *HostClockController {
	return &HostClockController{
		hostClocks:  make(map[string]*HostClock),
		slaveClocks: nil,
	}
}

// systemClockDevice — фиктивное PHC-устройство для часов "system" (createSystemClock по дизассемблеру: 0x8=-1, 0x64=0, name "system" len 6).
type systemClockDevice struct {
	clockID int // по дизассемблеру 0x8=-1, 0x64=0 — используем -1 как признак системных часов
}

func (d *systemClockDevice) GetDeviceName() (string, int) { return "system", 6 }
func (d *systemClockDevice) GetClockID() int               { return d.clockID }
func (d *systemClockDevice) DeterminePHCOffset() int64     { return 0 }

// algoTypeByDeviceName по дизассемблеру NewHostClock: выбор алгоритма по имени PHC (0x6872=rho->AlgoPID; beta/alpha/gamma/sigma->LinReg/Pi; system->Pi).
func algoTypeByDeviceName(name string) string {
	switch name {
	case "system":
		return "pi"
	case "eth0", "rho":
		return "pid"
	case "alpha", "beta":
		return "pi"
	case "gamma", "sigma":
		return "linreg"
	default:
		return "pid"
	}
}

// NewHostClock по дизассемблеру (NewHostClock@@Base): создаёт *HostClock по phc-устройству и флагу enabled.
// В бинарнике: ветки по GetDeviceName — rho->AlgoPID; beta/alpha/gamma/sigma->LinReg или Pi; system->Pi (0x20/0x28 = itab+impl).
// Создаём algorithm (AlgoPID/LinReg/Pi), bestFitFiltered для SetHoldoverFrequency, ema1/ema2 для maintainExponentiallyWeightedMovingAverages.
func NewHostClock(phc PHCDeviceInterface, enabled bool) *HostClock {
	name, _ := phc.GetDeviceName()
	algoType := algoTypeByDeviceName(name)
	var algorithm interface{}
	switch algoType {
	case "pid":
		algorithm = algos.NewAlgoPID()
	case "linreg":
		algorithm = algos.NewLinReg()
	case "pi":
		algorithm = algos.NewPi()
	default:
		algorithm = algos.NewAlgoPID()
	}
	hc := &HostClock{
		name:             name,
		clockID:          phc.GetClockID(),
		phcDevice:        phc,
		enabled:          enabled,
		algoType:         algoType,
		algorithm:        algorithm,
		bestFitFiltered:  algos.NewBestFitFiltered(algos.DefaultWindowSize),
		ema1:             statistics.NewEMA(),
		ema2:             statistics.NewEMA(),
	}
	return hc
}

// CreateSystemClock по дизассемблеру (createSystemClock@@Base 0x45988e0):
// 1) Создать объект с 0x8=-1, 0x64=0 и [1]string "system".
// 2) Перебор appConfig (0x178, 0x180), поиск строки "system".
// 3) NewHostClock(config); 0x60(clock)=0 (interferenceMon=nil); AddSlaveHostClock(clock).
func (hcc *HostClockController) CreateSystemClock() {
	dev := &systemClockDevice{clockID: -1}
	hc := NewHostClock(dev, true)
	hc.interferenceMon = nil
	hcc.AddSlaveHostClock(hc)
}

// GetHostClock возвращает часы по имени
func (hcc *HostClockController) GetHostClock(name string) *HostClock {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	return hcc.hostClocks[name]
}

// GetMasterHostClock возвращает master часы
func (hcc *HostClockController) GetMasterHostClock() *HostClock {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	return hcc.masterClock
}

// SetMasterHostClock устанавливает master часы
func (hcc *HostClockController) SetMasterHostClock(hc *HostClock) {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	hcc.masterClock = hc
}

// AddHostClock добавляет часы
func (hcc *HostClockController) AddHostClock(hc *HostClock) {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	hcc.hostClocks[hc.name] = hc
}

// AddSlaveHostClock по дизассемблеру (AddSlaveHostClock@@Base):
// 1) Lock(0x38); inc len(0x10); growslice if needed; append hc to slice 0x8; updateRelevantSlaveClocks; Unlock.
// 2) Если controller.0 != nil: AddCurrentPHCOffset(hc); DetermineSlaveClockOffsets(controller).
func (hcc *HostClockController) AddSlaveHostClock(hc *HostClock) {
	hcc.mu.Lock()
	hcc.slaveClocks = append(hcc.slaveClocks, hc)
	hcc.updateRelevantSlaveClocks()
	hcc.mu.Unlock()

	// В бинарнике: проверка controller.0 != nil (controller.masterClock?)
	// Если не nil: AddCurrentPHCOffset(hc), DetermineSlaveClockOffsets
	if hcc.masterClock != nil {
		hc.AddCurrentPHCOffset()
		hcc.DetermineSlaveClockOffsets()
	}
}

// updateRelevantSlaveClocks по дизассемблеру (updateRelevantSlaveClocks@@Base 0x45952e0):
// Цикл по slaveClocks; для каждого slave (!isMaster): если PHC.0xda!=0 — в результат; иначе GetDeviceName:
// если name=="system" (len 6) — в результат; иначе если controller.0x48!=0 — в результат.
// Итог: relevantSlaveClocks = slice (ptr 0x20, len 0x28, cap 0x30).
// Реконструировано по дизассемблеру бинарника timebeat-2.2.20.
func (hcc *HostClockController) updateRelevantSlaveClocks() {
	var relevant []*HostClock
	for _, hc := range hcc.slaveClocks {
		if hc == nil || hc.isMaster {
			continue
		}
		include := hcc.includeAllRelevantSlaves != 0
		if !include {
			name, _ := hc.getDeviceNameFromPHC()
			include = name == "system"
		}
		if include {
			relevant = append(relevant, hc)
		}
	}
	hcc.relevantSlaveClocks = relevant
}

// getDeviceNameFromPHC возвращает имя устройства из phcDevice (0x40); по дизассемблеру GetDeviceName(phc).
func (hc *HostClock) getDeviceNameFromPHC() (string, int) {
	if hc.phcDevice == nil {
		return "", 0
	}
	if dev, ok := hc.phcDevice.(PHCDeviceInterface); ok {
		return dev.GetDeviceName()
	}
	return hc.name, len(hc.name)
}

// GetPHCDeviceName возвращает имя PHC-устройства (для GetUTCTimeFromMasterClock в servo).
func (hc *HostClock) GetPHCDeviceName() (string, int) {
	return hc.getDeviceNameFromPHC()
}

// GetStaticPHCOffset возвращает статический PHC offset (0x100 по дизассемблеру).
func (hc *HostClock) GetStaticPHCOffset() int64 {
	return hc.staticPHCOffset
}

// GetOffset возвращает текущий offset (по дизассемблеру GetConstructedUTCTimeFromMasterClock: clock.offset).
func (hc *HostClock) GetOffset() int64 {
	return hc.offset
}

// RemoveHostClock удаляет часы
func (hcc *HostClockController) RemoveHostClock(name string) {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	delete(hcc.hostClocks, name)
}

// getClockWithURILocked по дизассемблеру GetClockWithURI: цикл по slice 0x8/0x10; для каждого clock 0x40=PHCDevice;
// GetDeviceName(phc); memequal(name, uri) → return clock; DoesDeviceHaveName(phc, uri) → return clock; иначе nil.
func (hcc *HostClockController) getClockWithURILocked(uri string) *HostClock {
	for _, hc := range hcc.slaveClocks {
		if hc == nil {
			continue
		}
		if hc.phcDevice == nil {
			if hc.name == uri {
				return hc
			}
			continue
		}
		if p, ok := hc.phcDevice.(PHCDeviceInterface); ok {
			name, _ := p.GetDeviceName()
			if name == uri {
				return hc
			}
		}
		if d, ok := hc.phcDevice.(interface{ DoesDeviceHaveName(string) bool }); ok && d.DoesDeviceHaveName(uri) {
			return hc
		}
	}
	for _, hc := range hcc.hostClocks {
		if hc != nil && hc.name == uri {
			return hc
		}
	}
	return nil
}

// GetClockWithURI по дизассемблеру (0x45963c0): цикл по slice 0x8(ptr)/0x10(len); для каждого clock 0x40=*PHCDevice; если phc!=nil — GetDeviceName, memequal(name,uri) → return clock; DoesDeviceHaveName(phc,uri) → return clock; иначе nil. Return nil если не найден.
func (hcc *HostClockController) GetClockWithURI(uri string) *HostClock {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	return hcc.getClockWithURILocked(uri)
}

// GetController по дизассемблеру — возвращает глобальный контроллер (для master, masterOffset).
func GetController() *HostClockController {
	return GetHostClockController()
}

// GetAllClockOffsets по дизассемблеру (api.GetAllClockOffsets 0x4bec480): возвращает срез данных об офсетах всех часов для API. Минимальная реконструкция: обход slaveClocks/hostClocks, сбор name+offset.
func GetAllClockOffsets() []interface{} {
	hcc := GetHostClockController()
	if hcc == nil {
		return nil
	}
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	var out []interface{}
	for _, hc := range hcc.slaveClocks {
		if hc != nil {
			hc.mu.Lock()
			out = append(out, map[string]interface{}{"name": hc.name, "offset": hc.offset})
			hc.mu.Unlock()
		}
	}
	for name, hc := range hcc.hostClocks {
		if hc != nil {
			hc.mu.Lock()
			out = append(out, map[string]interface{}{"name": name, "offset": hc.offset})
			hc.mu.Unlock()
		}
	}
	return out
}

// NotifyTAIOffset по дизассемблеру (0x4597d40): cmp len(clockName) 2/3/4; при len==2 — цикл slaveClocks, staticPHCOffset=offsetNs; при 3/4 — processTAISubmission.
func (hcc *HostClockController) NotifyTAIOffset(clockName string, offsetNs int64) {
	switch len(clockName) {
	case 2:
		hcc.mu.Lock()
		for _, hc := range hcc.slaveClocks {
			if hc != nil {
				hc.mu.Lock()
				hc.staticPHCOffset = offsetNs
				hc.mu.Unlock()
			}
		}
		hcc.mu.Unlock()
	case 3, 4:
		hcc.processTAISubmission(int64(len(clockName)), clockName, offsetNs)
	}
}

// processTAISubmission по дизассемблеру (0x4597f60): controller+0x80=cmp category; 0x88=cmp 0; 0x78=offset; time.Duration.String; Logger.Warn при изменении; selectnbsend.
func (hcc *HostClockController) processTAISubmission(category int64, sourceName string, offsetNs int64) {
	hcc.mu.Lock()
	prevOffset := hcc.taiOffsetNs
	prevName := hcc.taiSourceName
	if hcc.taiOffsetNs != offsetNs || hcc.taiSourceName != sourceName {
		hcc.taiOffsetNs = offsetNs
		hcc.taiSourceName = sourceName
		if lg := logging.GetErrorLogger(); lg != nil {
			dur := time.Duration(offsetNs)
			lg.Warn("TAI offset updated: " + sourceName + " " + dur.String())
		}
	}
	hcc.mu.Unlock()
	_ = category
	_ = prevOffset
	_ = prevName
}

// GetTimeOfLastMasterClockAdjustment по дизассемблеру (runReferenceUpdateLoop 0x45bbde5): возвращает время последней коррекции master-часов для NTP reference timestamp.
func (hcc *HostClockController) GetTimeOfLastMasterClockAdjustment() time.Time {
	if master := hcc.GetMasterHostClock(); master != nil {
		return master.GetTimeNow()
	}
	return time.Now()
}

// PPSRegistered по дизассемблеру (ProcessEvent 0x45bd83a): вызов на *HostClock после GetClockWithURI; отмечает часы как зарегистрированные для PPS.
func (hc *HostClock) PPSRegistered() {
	if hc != nil {
		hc.mu.Lock()
		hc.ppsRegistered = true
		hc.mu.Unlock()
	}
}

// getClockID возвращает clockID из phc.0x64 (0 если phc nil или не реализует GetClockID).
func (hc *HostClock) getClockID() int {
	if hc.phcDevice == nil {
		return 0
	}
	p, ok := hc.phcDevice.(PHCDeviceInterface)
	if !ok {
		return 0
	}
	return p.GetClockID()
}

// GetClockID по дизассемблеру (GetClockID@@Base): возврат phc.0x64.
// Дизассемблер: mov 0x40(%rax),%rcx; mov 0x64(%rcx),%eax; ret
func (hc *HostClock) GetClockID() int {
	return hc.getClockID()
}

// GetClockName по дизассемблеру (GetClockName@@Base): возврат GetDeviceName(phc.0x40).
// Дизассемблер: mov 0x40(%rax),%rax; test %rax,%rax; je -> (ret 0,0); call GetDeviceName; ret.
func (hc *HostClock) GetClockName() (string, int) {
	if hc.phcDevice == nil {
		return "", 0
	}
	if p, ok := hc.phcDevice.(PHCDeviceInterface); ok {
		return p.GetDeviceName()
	}
	return "", 0
}

// AddCurrentPHCOffset по дизассемблеру (__HostClock_.AddCurrentPHCOffset 0x4593120):
// 1) controller; masterOffset = controller.masterClock.0x100 или 0.
// 2) phc=hc.0x40; name,len=GetDeviceName(phc). Если len==6 и name=="system": GetController; lockPHCFunctions(master); master.0x40.DeterminePHCOffset(); unlock; offset = -phcOffset - masterOffset.
// 3) Иначе: lockPHCFunctions(hc); phcOffset=hc.0x40.DeterminePHCOffset(); unlock; offset = phcOffset + hc.0x100.
// 4) 0xf1(hc): если !=0 — filter=hc.0x18, IsFiltered(offset); при true — flag=1; иначе flag=предыдущее (в бинарнике из len). hc.0x48=flag; *hc.0=offset.
func (hc *HostClock) AddCurrentPHCOffset() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.addCurrentPHCOffsetLocked()
}

// addCurrentPHCOffsetLocked — логика AddCurrentPHCOffset без взятия hc.mu (вызывать при уже удержанном lock).
func (hc *HostClock) addCurrentPHCOffsetLocked() {

	var masterOffset int64
	if hcc := GetController(); hcc != nil {
		if master := hcc.GetMasterHostClock(); master != nil {
			masterOffset = master.staticPHCOffset
		}
	}

	phc := hc.phcDevice
	if phc == nil {
		return
	}
	p, ok := phc.(PHCDeviceInterface)
	if !ok {
		return
	}
	deviceName, nameLen := p.GetDeviceName()

	var offset int64
	if nameLen == 6 && deviceName == "system" {
		hcc := GetController()
		if hcc == nil {
			return
		}
		master := hcc.GetMasterHostClock()
		if master == nil {
			return
		}
		lockPHCFunctions(master)
		var phcOffset int64
		if masterDev, ok := master.phcDevice.(PHCDeviceInterface); ok {
			phcOffset = masterDev.DeterminePHCOffset()
		}
		unlockPHCFunctions(master)
		offset = -phcOffset - masterOffset
	} else {
		lockPHCFunctions(hc)
		phcOffset := p.DeterminePHCOffset()
		unlockPHCFunctions(hc)
		offset = phcOffset + hc.staticPHCOffset
	}

	// По дизассемблеру 0xf1: cmp 0xf1(rcx); 0x18(rcx)=filter; IsFiltered → при true mov $1, flag; иначе сохраняют bl (len). Семантика: flag48=1 = offset отфильтрован.
	flag := byte(0)
	if hc.useFilter && hc.filter != nil {
		if f, ok := hc.filter.(OffsetFilterInterface); ok && f.IsFiltered(offset) {
			flag = 1
		}
	}
	hc.flag48 = flag
	hc.offset = offset
}

// addOffset по дизассемблеру (addOffset@@Base 0x4593b40): 0x48(rax)=cl (flag); mov (%rax),%rax → *receiver.0; mov %rbx,(%rax) — запись value по указателю 0x0.
// Минимальная реконструкция: flag48=flag, offset=value (запись в поле offset как в «итоговый offset»).
func (hc *HostClock) addOffset(flag byte, value int64) {
	hc.flag48 = flag
	hc.offset = value
}

// GetTimeNow по дизассемблеру (GetTimeNow@@Base 0x45946a0): lockPHCFunctions; 0x40(hc)=phcDevice, 0x64(phc)=clockID;
// GetClockUsingGetTimeSyscall(clockID); unlockPHCFunctions; return time.
func (hc *HostClock) GetTimeNow() time.Time {
	lockPHCFunctions(hc)
	defer unlockPHCFunctions(hc)
	clockID := hc.getClockID()
	return adjusttime.GetClockUsingGetTimeSyscall(clockID)
}

// GetRawData по дизассемблеру (GetRawData@@Base): возврат *hc.0 (первое поле структуры — rawData или clockData).
// Дизассемблер: mov (%rax),%rax; ret
// TODO: когда будет доступна структура RawData/ClockData, вернуть её из поля hc.
func (hc *HostClock) GetRawData() interface{} {
	return nil // заглушка, пока rawData не определён
}

// IsMaster по дизассемблеру (IsMaster@@Base): возврат 0x91(hc).
// Дизассемблер: movzbl 0x91(%rax),%eax; ret
func (hc *HostClock) IsMaster() bool {
	return hc.isMaster
}

// SetMaster по дизассемблеру (SetMaster@@Base): запись bl в 0x91(hc).
// Дизассемблер: mov %bl,0x91(%rax); ret
func (hc *HostClock) SetMaster(isMaster bool) {
	hc.isMaster = isMaster
}

// maintainExponentiallyWeightedMovingAveragesClientStore по дизассемблеру (maintainExponentiallyWeightedMovingAveragesClientStore@@Base):
// 1) *hc.0x8.0x8 = clientStore.
// 2) EMA.AddValue(*hc.0.0, *hc.0x8.0) — первый вызов: value из rawData.0x0, receiver ema из clockData.0x0.
// 3) EMA.AddValue(*hc.0x8.0x10, *hc.0.0x18) — второй: value из clockData.0x10, receiver ema из rawData.0x18.
// Реконструировано по дизассемблеру бинарника timebeat-2.2.20.
func (hc *HostClock) maintainExponentiallyWeightedMovingAveragesClientStore(clientStore interface{}) {
	hc.clientStore = clientStore
	// В бинарнике: value1 = *hc.0.0, value2 = *hc.0x8.0x10; используем hc.offset как значение при отсутствии RawData/ClockData.
	v := hc.offset
	if hc.ema1 != nil {
		hc.ema1.AddValue(v)
	}
	if hc.ema2 != nil {
		hc.ema2.AddValue(v)
	}
}

// StepClock по дизассемблеру (__HostClock_.StepClock 0x4592980):
// 1) 0xf0(hc)==0 → return. 2) phc=0x40; GetDeviceName; Logger(0x38).Warn(concat). 3) GetClockUsingGetTimeSyscall(phc.0x64). 4) newTime = *hc.0 + time.Add(offset).
// 5) InterferenceMonitor(0x10).getState(); при state!=0 → Warn, LogWouldHaveSteppedMessage, return. 6) |offset| vs порог: при |offset|<=порог → Warn, LogWouldHaveSteppedMessage, return.
// 7) lockPHCFunctions; SetFrequency(phc.0x64, hc.0x70); GetClockFrequency; commitFrequency(im); StepClockUsingSetTimeSyscall; unlock. 8) При err: Logger.Error; при успехе: Logger.Info, LogSteppedMessage, NewKalmanFilter(GetDeviceName)→hc.0x108.
func (hc *HostClock) StepClock(t time.Time) error {
	if !hc.enabled {
		return nil
	}
	clockID := hc.getClockID()
	now := adjusttime.GetClockUsingGetTimeSyscall(clockID)
	newTime := now.Add(time.Duration(hc.offset))
	if hc.interferenceMon != nil && hc.interferenceMon.getState() {
		logWouldHaveSteppedMessage(hc)
		return nil
	}
	// Порог: по дизассемблеру |offset| vs порог; при |offset|<=порог — LogWouldHaveSteppedMessage и return
	if hc.offset <= stepClockThresholdNs && hc.offset >= -stepClockThresholdNs {
		logWouldHaveSteppedMessage(hc)
		return nil
	}
	lockPHCFunctions(hc)
	_ = adjusttime.SetFrequency(clockID, hc.frequencyPpm)
	freq, _ := adjusttime.GetClockFrequency(clockID)
	if hc.interferenceMon != nil {
		hc.interferenceMon.commitFrequency(freq)
	}
	err := adjusttime.StepClockUsingSetTimeSyscall(clockID, newTime)
	unlockPHCFunctions(hc)
	if err != nil {
		// Logger.Error — заглушка до подключения logging
		return err
	}
	// Успех: по дизассемблеру Logger.Info(0x13 bytes), LogSteppedMessage(hc), NewKalmanFilter(GetDeviceName)→hc.0x108
	hc.logSteppedMessage()
	deviceName, _ := hc.GetClockName()
	hc.kalmanFilter = filters.NewKalmanFilter(deviceName, nil)
	return nil
}

// StepFromMasterClock по дизассемблеру (__HostClock_.StepFromMasterClock@@Base):
// phc=0x40(hc); если phc==nil или GetDeviceName=="system" (len 6) → return false.
// lockPHCFunctions; phcOffset=DeterminePHCOffset(); unlockPHCFunctions;
// если (phcOffset+0x1dcd6500) > 0x3b9aca00 (500 ms): hc.flag48=0, hc.offset=phcOffset, StepClock(hc), return true; иначе return false.
func (hc *HostClock) StepFromMasterClock() bool {
	if hc.phcDevice == nil {
		return false
	}
	name, nameLen := hc.getDeviceNameFromPHC()
	if nameLen == 6 && name == "system" {
		return false
	}
	lockPHCFunctions(hc)
	var phcOffset int64
	if p, ok := hc.phcDevice.(PHCDeviceInterface); ok {
		phcOffset = p.DeterminePHCOffset()
	}
	unlockPHCFunctions(hc)
	// (phcOffset + 500_000_000) > 1_000_000_000 → step
	if phcOffset+stepClockThresholdNs <= 1_000_000_000 {
		return false
	}
	hc.flag48 = 0
	hc.offset = phcOffset
	_ = hc.StepClock(time.Now())
	return true
}

// SetHoldoverFrequency по дизассемблеру (__HostClock_.SetHoldoverFrequency@@Base 0x45943c0):
// lockPHCFunctions; defer func1 (unlockPHCFunctions); Logger.Warn(0x11); hc.0x30=BestFitFiltered, hc.0x40=phc; clockID=phc.0x64;
// GetLeastSquaresGradientFiltered() → gradient (ppb); SetFrequency(clockID, gradient→ppm); GetClockFrequency(clockID);
// 0x20/0x28(hc)=algorithm, вызов method(freq); commitFrequency(im).
func (hc *HostClock) SetHoldoverFrequency() {
	lockPHCFunctions(hc)
	defer func() { unlockPHCFunctions(hc) }()
	// Logger.Warn (0x38(hc)) — заглушка до подключения logging
	clockID := hc.getClockID()
	if hc.bestFitFiltered != nil {
		gradientPpb := hc.bestFitFiltered.GetLeastSquaresGradientFiltered()
		ppm := gradientPpb / 1000
		_ = adjusttime.SetFrequency(clockID, ppm)
	}
	freq, _ := adjusttime.GetClockFrequency(clockID)
	if u, ok := hc.algorithm.(algoClockFreqUpdater); ok {
		u.UpdateClockFreq(freq)
	}
	if hc.interferenceMon != nil {
		hc.interferenceMon.commitFrequency(freq)
	}
}

// SlewClock по дизассемблеру (__HostClock_.SlewClock.txt): 0x91 — skip; 0x40=PHCDevice; 0xc0==2 или name=="system" и master 0xc0==2 — GetPreciseTime; иначе если (time&0x3fffffff) > 0x989680 (10 ms) — async; иначе SlewClockPossiblyAsync.
//
// SlewClockPossiblyAsync — полный путь по дизассемблеру (SlewClockPossiblyAsync.txt):
//  1. Если hc.0xf0==0: AddCurrentPHCOffset(), return.
//  2. state := InterferenceMonitor(0x10).getState(). Если state==0 (false): идём в п.3. Иначе: если state!=3 — конец; если config.appConfig+0xe0==0 — конец; иначе AlgoLogEntry.Log(), return.
//  3. lockPHCFunctions(hc); GetClockFrequency(phc.0x64); GetClockUsingGetTimeSyscall(phc.0x64); isFrequencyUnchanged(im,freq)? иначе triggerInterference(im); unlockPHCFunctions(hc).
//  4. Если hc.0x91 (isMaster): использовать время из GetClockUsingGetTimeSyscall (сохранённое в 0x48(rsp)); иначе AddCurrentPHCOffset().
//  5. Если hc.0x48 (addOffset flag): return.
//  6. Иначе: вызов time.Add(offset) через интерфейс (0x20/0/0x28, метод 0x18(r9)); затем switch по типу (r11): 1 — path system: lockPHCFunctions, GetController, сравнение offset с порогом 0x60(controller), SetOffset/SetFrequency(-offset), unlock; 2 — path PHC: lockPHCFunctions, SetFrequency(0xc8), расчёт времени (0x3b9aca00*..., 0x3fffffff), append в slice, unlock; иначе — без branch.
//  7. lockPHCFunctions(hc); GetClockFrequency → 0xc8(rsp); unlockPHCFunctions(hc); GetDeviceName; Log; commitFrequency(im); return.
func (hc *HostClock) SlewClockPossiblyAsync(offsetNs int64) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.offset = offsetNs
	if !hc.enabled {
		hc.AddCurrentPHCOffset()
		return nil
	}
	if hc.interferenceMon != nil && hc.interferenceMon.getState() {
		// state == 3 и config — AlgoLogEntry.Log; иначе просто return
		return nil
	}
	clockID := hc.getClockID()
	lockPHCFunctions(hc)
	freq, _ := adjusttime.GetClockFrequency(clockID)
	now := adjusttime.GetClockUsingGetTimeSyscall(clockID)
	if hc.interferenceMon != nil && !hc.interferenceMon.isFrequencyUnchanged(freq) {
		hc.interferenceMon.triggerInterference()
	}
	unlockPHCFunctions(hc)
	if hc.isMaster {
		_ = now // использовать сохранённое время (в бинарнике из 0x48(rsp))
	} else {
		hc.addCurrentPHCOffsetLocked()
	}
	if hc.flag48 != 0 {
		return nil
	}
	// time.Add(offset), switch type 1/2 (system/PHC), SetOffset/SetFrequency или SetFrequency+append
	// Упрощённо: slew через adjtimex
	_ = adjusttime.SlewClock(offsetNs)
	lockPHCFunctions(hc)
	freq2, _ := adjusttime.GetClockFrequency(clockID)
	if hc.interferenceMon != nil {
		hc.interferenceMon.commitFrequency(freq2)
	}
	unlockPHCFunctions(hc)
	return nil
}

// logWouldHaveSteppedMessage по дизассемблеру (LogWouldHaveSteppedMessage@@Base): если 0x58(hc)!=0 — return; иначе GetDeviceName(phc), WouldHaveSteppedMessage.Send(...), go TriggerWouldHaveSteppedMessageTimer. Заглушка до подключения logging.
func logWouldHaveSteppedMessage(hc *HostClock) {
	if hc.suppressWouldHaveSteppedLog != 0 {
		return
	}
	hc.triggerWouldHaveSteppedMessageTimer()
}

// logSteppedMessage по дизассемблеру (LogSteppedMessage@@Base 0x4593e00): вызывается после успешного StepClock; Logger.Info, логирование факта step. Заглушка до подключения logging.
func (hc *HostClock) logSteppedMessage() {
	_ = hc
}

// triggerWouldHaveSteppedMessageTimer по дизассемблеру (TriggerWouldHaveSteppedMessageTimer@@Base): вызывается из closure в LogWouldHaveSteppedMessage. Заглушка до подключения таймера/логирования.
func (hc *HostClock) triggerWouldHaveSteppedMessageTimer() {
	_ = hc
}

// SetFrequency устанавливает частоту; по дизассемблеру — вызов adjtimex ADJ_FREQUENCY (adjusttime.SetFrequency).
func (hc *HostClock) SetFrequency(ppm float64) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.frequency = ppm
	hc.frequencyPpm = ppm
	clockID := hc.getClockID()
	if clockID != 0 {
		_ = adjusttime.SetFrequency(clockID, ppm)
	}
	return nil
}

// GetFrequency возвращает частоту
func (hc *HostClock) GetFrequency() float64 {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return hc.frequency
}

// LogForSlaveClocks по дизассемблеру (LogForSlaveClocks@@Base): Lock(0x38), defer unlock; цикл по 0x20/0x28 (slaveClocks); для каждого hc вызывается LogRawAndEMAData(hc).
func (hcc *HostClockController) LogForSlaveClocks() {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	for _, hc := range hcc.slaveClocks {
		if hc != nil {
			hc.LogRawAndEMAData(nil) // clientStore=nil по умолчанию
		}
	}
}

// LogRawAndEMAData по дизассемблеру (LogRawAndEMAData@@Base):
// 1) lockPHCFunctions; PerformGranularityMeasurement(phc.0x64); *hc.0+0x18=granularity; unlockPHCFunctions.
// 2) Если 0x91(isMaster)==0: *hc.0+0x10=0, xor ebx (clientStore=nil), call maintainEMA(...,0).
//    Иначе: clientStore из аргумента (rbx), call maintainEMA(hc,clientStore).
// 3) Если 0x60(hc)!=nil и 0x40(0x60)!=nil → GetDeviceName(phc); иначе (nil,0).
// 4) Вызов LogRawAndEMAEntry или HostClockLogEntry.Send с параметрами (rdx=deviceName, cl=flag, rax=hc, rbx=clientStore).
//
// Реализация (заглушка до logging):
func (hc *HostClock) LogRawAndEMAData(clientStore interface{}) {
	lockPHCFunctions(hc)
	clockID := hc.getClockID()
	granularity := adjusttime.PerformGranularityMeasurement(clockID)
	if hc.phcDevice != nil {
		// В бинарнике: *hc.0+0x18 = granularity (поле rawData.0x18 или clockData.granularity)
		// TODO: записать granularity в rawData/clockData, когда будут доступны структуры логирования.
		_ = granularity
	}
	unlockPHCFunctions(hc)

	if !hc.isMaster {
		// В бинарнике: *hc.0+0x10 = 0 (reset rawData?), clientStore = nil
		clientStore = nil
	}
	// call maintainExponentiallyWeightedMovingAveragesClientStore(hc, clientStore)
	// Заглушка: метод уже существует и вызывается по дизассемблеру
	hc.maintainExponentiallyWeightedMovingAveragesClientStore(clientStore)

	// Получить deviceName из phc для логирования
	var deviceName string
	var nameLen int
	if hc.phcDevice != nil {
		if p, ok := hc.phcDevice.(PHCDeviceInterface); ok {
			deviceName, nameLen = p.GetDeviceName()
		}
	}
	_ = deviceName
	_ = nameLen

	// В бинарнике: создание LogRawAndEMAEntry или HostClockLogEntry и вызов Send.
	// Заглушка до подключения logging пакета.
	_ = hc.flag48 // по дизассемблеру читается 0x48
}

// algoTypeToInt по дизассемблеру GetCoefficientsForTypeInt: 0=pid, 1=linreg, 2=pi (типы в CoefficientStore).
func algoTypeToInt(algoType string) int {
	switch algoType {
	case "pid":
		return 0
	case "linreg":
		return 1
	case "pi":
		return 2
	default:
		return 0
	}
}

// updateStoreForClock по дизассемблеру updateAlgoCoefficients: для одного clock получаем coeffs по scale и записываем в store.
func updateStoreForClock(hc *HostClock, scale byte) {
	store := algos.GetCoefficientStore()
	algoTypeInt := algoTypeToInt(hc.algoType)
	c := store.GetCoefficientsForTypeInt(algoTypeInt, scale)
	if c != nil {
		store.ChangeSteeringCoefficientsInt(algoTypeInt, c)
	}
}

// UpdateScaleFromStore по дизассемблеру (updateAlgoCoefficients): вызов algorithm.UpdateScaleFromStore(scale) или обновление store по algoType+scale.
func (hc *HostClock) UpdateScaleFromStore(scale byte) {
	if u, ok := hc.algorithm.(algoScaleUpdater); ok {
		u.UpdateScaleFromStore(scale)
		return
	}
	updateStoreForClock(hc, scale)
}

// updateAlgoCoefficients по дизассемблеру (__HostClockController_.updateAlgoCoefficients@@Base 0x4595020):
// Читает appConfig (строка "gamma" len 5 или "rho" len 3). master = controller.masterClock; name = GetDeviceName(master).
// Если name == "system" (len 6): для каждого slave вызывается method(0). Иначе: method(master, 1); для каждого slave method(1).
// method = UpdateScaleFromStore(scale). Без доступа к config — ветка по master name: system → scale 0 для slave; иначе scale 1 для master и slave.
func (hcc *HostClockController) updateAlgoCoefficients() {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	master := hcc.masterClock
	if master == nil {
		return
	}
	name, nameLen := master.getDeviceNameFromPHC()
	scale := byte(1)
	if nameLen == 6 && name == "system" {
		scale = 0
		for _, slave := range hcc.slaveClocks {
			if slave != nil {
				slave.UpdateScaleFromStore(scale)
			}
		}
		return
	}
	master.UpdateScaleFromStore(scale)
	for _, slave := range hcc.slaveClocks {
		if slave != nil {
			slave.UpdateScaleFromStore(scale)
		}
	}
}

// ElectMasterClock по дизассемблеру (__HostClockController_.ElectMasterClock@@Base 0x4594c00):
// Lock(0x38); current = *(controller) (masterClock); если current==nil → promoteToMaster(newMaster);
// иначе если current!=newMaster → demoteCurrentMaster(), promoteToMaster(newMaster);
// updateRelevantSlaveClocks(); updateAlgoCoefficients(); Unlock(); movzbl 0x68(controller), UpdatePPS(scale).
func (hcc *HostClockController) ElectMasterClock(newMaster *HostClock) {
	hcc.mu.Lock()
	current := hcc.masterClock
	if current == nil {
		hcc.promoteToMaster(newMaster)
	} else if current != newMaster {
		hcc.demoteCurrentMaster()
		hcc.promoteToMaster(newMaster)
	}
	hcc.updateRelevantSlaveClocks()
	hcc.updateAlgoCoefficients()
	hcc.mu.Unlock()
	hcc.UpdatePPS(hcc.ppsScale)
}

// demoteCurrentMaster по дизассемблеру (demoteCurrentMaster@@Base): current=masterClock; GetDeviceName(current.phcDevice); Logger.Info; если name=="system" — SetManualOverride/иное; иначе current.SetMaster(false).
func (hcc *HostClockController) demoteCurrentMaster() {
	current := hcc.masterClock
	if current == nil {
		return
	}
	name, nameLen := current.getDeviceNameFromPHC()
	_ = name
	_ = nameLen
	// Logger.Info — заглушка до подключения logging
	if nameLen == 6 && name == "system" {
		if current.interferenceMon != nil {
			current.interferenceMon.setStateByte(0)
		}
	} else {
		current.SetMaster(false)
	}
}

// promoteToMaster по дизассемблеру (promoteToMaster@@Base): newMaster.0x60=0 (interferenceMon или иное); newMaster.0x91=1 (isMaster); GetDeviceName; Logger.Info; *(controller)=newMaster (masterClock=newMaster).
func (hcc *HostClockController) promoteToMaster(newMaster *HostClock) {
	if newMaster == nil {
		return
	}
	// 0x60(clock)=0 по дизассемблеру — не трогаем interferenceMon, т.к. может быть nil
	newMaster.isMaster = true
	// Logger.Info(GetDeviceName) — заглушка
	hcc.masterClock = newMaster
}

// UpdatePPS по дизассемблеру (__HostClockController_.UpdatePPS@@Base 0x4596500): Lock; 0x68(controller)=scale; цикл по slaveClocks: если clock.isMaster — clock.UpdatePPS(scale), иначе clock.UpdatePPS(0); Unlock.
func (hcc *HostClockController) UpdatePPS(scale byte) {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	hcc.ppsScale = scale
	for _, clock := range hcc.slaveClocks {
		if clock == nil {
			continue
		}
		if clock.isMaster {
			clock.UpdatePPS(scale)
		} else {
			clock.UpdatePPS(0)
		}
	}
}

// UpdatePPS по дизассемблеру (__HostClock_.UpdatePPS@@Base): вызов algorithm.UpdatePPS(scale) если реализует algoPPSUpdater.
func (hc *HostClock) UpdatePPS(scale byte) {
	if u, ok := hc.algorithm.(algoPPSUpdater); ok {
		u.UpdatePPS(scale)
	}
}

// InterferenceMonitor по дизассемблеру (getState 0x4596e80, setState 0x4596d40, runTimer 0x4597360):
// 0x0 — byte, в runTimer после срабатывания таймера записывается 1; 0x8 — duration для NewTimer; 0x10 — state byte (0/1/2/3); 0x28 — mutex; 0x38/0x40 — savedFrequencyPpm.
type InterferenceMonitor struct {
	mu                sync.Mutex   // 0x28 по дизассемблеру getState/setState: lock cmpxchg
	state             string
	stateByte         byte         // 0x10 по дизассемблеру: 0/1/2/3 (setState(bl), getState возвращает этот байт)
	loggerName        string       // 0x8 по NewInterferenceMonitor (runTimer читает im.0x8 для NewTimer — duration)
	savedFrequencyPpm float64      // 0x38/0x40 — последняя закоммиченная частота (ppm)
	hasSavedFrequency bool
}

// getState по дизассемблеру (getState.txt): Lock(0x28), defer unlock; return 0x10(rcx) как byte.
func (im *InterferenceMonitor) getState() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.stateByte == 1
}

// setState по дизассемблеру (setState.txt): аргумент — byte (bl). Lock(0x28) через lock cmpxchg; defer func1 (unlock); 0x10(rcx)=bl; defer run.
// В бинарнике сигнатура setState(im, state byte): только байт записывается в 0x10. Вызовы: SetManualOverride — setState(0) или setState(3).
func (im *InterferenceMonitor) setState(state string) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.state = state
	if state != "" {
		im.stateByte = 1
	} else {
		im.stateByte = 0
	}
}

// setStateByte устанавливает state byte (0x10) — по дизассемблеру setState(im, state byte).
func (im *InterferenceMonitor) setStateByte(b byte) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.stateByte = b
}

// SetManualOverride по дизассемблеру (Controller.SetManualOverride вызывает GetClockWithURI(masterClock), затем HostClock.SetManualOverride): setState(0) или setState(3).
func (hc *HostClock) SetManualOverride(enabled bool) {
	if hc.interferenceMon == nil {
		return
	}
	if enabled {
		hc.interferenceMon.setStateByte(3)
	} else {
		hc.interferenceMon.setStateByte(0)
	}
}

// triggerInterference по вызовам из SlewClockPossiblyAsync: при смене частоты вызывается для перевода в состояние interference.
func (im *InterferenceMonitor) triggerInterference() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.stateByte = 1
}

// isFrequencyUnchanged по вызовам из SlewClockPossiblyAsync(im, freq): сравнивает текущую частоту с сохранённой (0x38/0x40), возвращает true если не изменилась.
func (im *InterferenceMonitor) isFrequencyUnchanged(freq float64) bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	if !im.hasSavedFrequency {
		return true
	}
	return im.savedFrequencyPpm == freq
}

// commitFrequency по вызовам из StepClock/SlewClockPossiblyAsync: сохраняет частоту после SetFrequency/GetClockFrequency (0x38/0x40).
func (im *InterferenceMonitor) commitFrequency(freq float64) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.savedFrequencyPpm = freq
	im.hasSavedFrequency = true
}

// lockPHCFunctions по дизассемблеру (lockPHCFunctions.txt): аргумент = *HostClock.
// IsRPIComputeModule(); если !RPI — return. phc := 0x40(hc); если phc==nil — return. GetDeviceName(phc) → (ptr, len); если len!=4 или name!="eth0" — return. Если 0x108(phc)!=0 — return. lock 0x110(hc).
func lockPHCFunctions(hc *HostClock) {
	if !isRPIComputeModule() {
		return
	}
	if hc.phcDevice == nil {
		return
	}
	if phc, ok := hc.phcDevice.(PHCDeviceInterface); ok {
		name, n := phc.GetDeviceName()
		if n != 4 || name != "eth0" {
			return
		}
	}
	hc.phcMu.Lock()
}

// unlockPHCFunctions по дизассемблеру (unlockPHCFunctions.txt): те же проверки RPI, 0x40, GetDeviceName=="eth0", 0x108==0. Затем unlock 0x110(hc).
func unlockPHCFunctions(hc *HostClock) {
	if !isRPIComputeModule() {
		return
	}
	if hc.phcDevice == nil {
		return
	}
	if phc, ok := hc.phcDevice.(PHCDeviceInterface); ok {
		name, n := phc.GetDeviceName()
		if n != 4 || name != "eth0" {
			return
		}
	}
	hc.phcMu.Unlock()
}

func isRPIComputeModule() bool {
	return adjusttime.IsRPIComputeModule()
}

// NewInterferenceMonitor по дизассемблеру (NewInterferenceMonitor.txt): NewLogger(concat), new struct; 0x8=name, 0(rax)=1, 0x30=logger; 0x10=0 если enable иначе 1.
func NewInterferenceMonitor(enable bool, name string) *InterferenceMonitor {
	im := &InterferenceMonitor{loggerName: name}
	if enable {
		im.stateByte = 0
	} else {
		im.stateByte = 1
	}
	return im
}

// RunTimer по дизассемблеру (0x4597360): NewTimer(im.0x8), chanrecv1 (ожидание); movb $1,(im+0); setState(0); getState(); если state < 4 — StateNames[state], logInterferenceStateChange(logger, stateName). Заглушка: setStateByte(0), logInterferenceStateChange.
func (im *InterferenceMonitor) RunTimer() {
	im.setStateByte(0)
	im.logInterferenceStateChange(im.getStateAsString())
}

// logInterferenceStateChange по дизассемблеру (0x45974c0): Logger с сообщением о смене состояния (concat string). Заглушка до подключения logging.
func (im *InterferenceMonitor) logInterferenceStateChange(stateName string) {
	_ = stateName
}

// getStateAsString по дизассемблеру (getStateAsString 0x45972a0): по state byte (0x10) возвращает имя из StateNames (0x79919c0). Заглушка: возврат строки по stateByte.
func (im *InterferenceMonitor) getStateAsString() string {
	im.mu.Lock()
	b := im.stateByte
	im.mu.Unlock()
	switch b {
	case 0:
		return "normal"
	case 1:
		return "interference"
	case 2:
		return "manual"
	case 3:
		return "manual-override"
	default:
		return "unknown"
	}
}

// DetermineSlaveClockOffsets по дизассемблеру (DetermineSlaveClockOffsets.txt): Lock(0x38), defer unlock; цикл по slice 0x20 len 0x28 — для каждого HostClock вызывается AddCurrentPHCOffset(clock).
func (hcc *HostClockController) DetermineSlaveClockOffsets() {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	hcc.determineSlaveClockOffsetsLocked()
}

func (hcc *HostClockController) determineSlaveClockOffsetsLocked() {
	for _, hc := range hcc.slaveClocks {
		if hc != nil {
			hc.AddCurrentPHCOffset()
		}
	}
}

// StepSlaveClocksIfNecessary по дизассемблеру (StepSlaveClocksIfNecessary.txt): GetMasterHostClock; Lock(0x38);
// если имя master == "system" (len 6) — пропустить блок step; иначе GetClockWithURI("system"); если |offset| <= 0x1dcd6500 — пропуск;
// StepClock(clock); DetermineSlaveClockOffsets; цикл по slaveClocks (0x20, 0x28): если имя != "system" и |offset| > 500 ms — StepClock(slave).
func (hcc *HostClockController) StepSlaveClocksIfNecessary() {
	master := hcc.GetMasterHostClock()
	if master == nil {
		return
	}
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	// Если master — системные часы ("system"), блок step не выполняем (переходим к циклу по slave).
	if master.name == "system" {
		// по дизассемблеру: jmp к циклу по slaveClocks
	} else {
		clock := hcc.getClockWithURILocked("system")
		if clock != nil {
			absOffset := clock.offset
			if absOffset < 0 {
				absOffset = -absOffset
			}
			if absOffset > stepClockThresholdNs {
				_ = clock.StepClock(time.Now())
				hcc.determineSlaveClockOffsetsLocked()
			}
		}
	}
	for _, hc := range hcc.slaveClocks {
		if hc == nil {
			continue
		}
		if hc.name == "system" {
			continue
		}
		absOffset := hc.offset
		if absOffset < 0 {
			absOffset = -absOffset
		}
		if absOffset > stepClockThresholdNs {
			_ = hc.StepClock(time.Now())
		}
	}
}

// SlewSlaveClocks по дизассемблеру (SlewSlaveClocks.txt): Lock(0x38), defer unlock; цикл по slice 0x20 len 0x28 — SlewClock(clock).
func (hcc *HostClockController) SlewSlaveClocks() {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	for _, hc := range hcc.slaveClocks {
		if hc != nil {
			_ = hc.SlewClockPossiblyAsync(hc.offset)
		}
	}
}

package phc

import (
	"encoding/binary"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// Автоматически извлечено из timebeat-2.2.20

// Константы по дизассемблеру DeterminePHCOffset (case 2: порог и граница).
const (
	determinePHCOffsetThreshold = 0x4c4b3f + 1 // 5000000: если Basic() >= 5e6, return 0
	determinePHCOffsetBound     = 0x98967f      // ~10e6
)

// StrategyType — тип стратегии определения offset (0xc0 в бинарнике).
const (
	StrategyBasic    = 1 // return slice[0]
	StrategyBasicFallback = 2 // Basic(); если ret >= 5e6 return 0, иначе return FallbackOffset
	StrategyExtended = 3 // return Extended()
	StrategyEFX      = 4 // EFX(); sort.Slice; return slice[len/2]
	StrategyPrecise  = 5 // Precise(); sort.Slice; return slice[len/2]
)

// ptpSysOffsetData — результат GetPHCToSysClockSamplesBasic по дизассемблеру: (rax)=n_samples, буфер заполняется ядром; элемент i: sec at 0x10+i*32, nsec at 0x18+i*32.
type ptpSysOffsetData struct {
	N_samples uint32
	_         [12]byte
	Buf       [maxPTPOffsetSamples * 32]byte
}

const maxPTPOffsetSamples = 32 // по дизассемблеру cmp $0x33; элементов по 32 байт

// sampleSec/sampleNsec читают из буфера: после заголовка 16 байт элемент i — Buf[i*32] (sec 8 байт, nsec 4 байта).
func (p *ptpSysOffsetData) sampleSec(i int) int64 {
	if p == nil || i < 0 || i*32+8 > len(p.Buf) {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(p.Buf[i*32:]))
}

func (p *ptpSysOffsetData) sampleNsec(i int) int32 {
	if p == nil || i < 0 || i*32+12 > len(p.Buf) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(p.Buf[i*32+8:]))
}

// PHCDevice по дизассемблеру: 0x30/0x38/0x40 = имя, 0x60=FD, 0x64=clockID, 0xc0=StrategyType, 0xc8=FallbackOffset, 0xf0=*len (makeslice), 0x100=mutex.
type PHCDevice struct {
	DeviceNames    []string   // 0x30/0x38 — список имён устройств (по GetDeviceNames)
	DeviceName     string     // для GetDeviceName (первое имя из DeviceNames)
	FD             int        // 0x60 — file descriptor для Ioctl (GetPHCToSysClockSamplesBasic)
	ClockID        int        // 0x64
	StrategyType   int        // 0xc0: 1=Basic, 2=Basic+fallback, 3=Extended, 4=EFX, 5=Precise
	FallbackOffset int64      // 0xc8: возврат при case 2 если Basic() < 5e6
	NumSamples     int        // длина слайса (из *0xf0 в бинарнике: makeslice(len(0xf0), int64))
	mu             sync.Mutex // 0x100
}

// GetDeviceName по дизассемблеру (GetDeviceName@@Base): lock 0x100, defer unlock(func1), если len(0x38)==1 — (ptr, len) первого элемента, иначе strings.Join(0x30, ""); возврат (name, len).
func (d *PHCDevice) GetDeviceName() (string, int) {
	if d == nil {
		return "", 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.getDeviceNameLocked()
}

// GetDeviceNameLocked по дизассемблеру (GetDeviceNameLocked@@Base): без блокировки; 0x30/0x38/0x40 — если len==1 возврат (ptr,len) первого, иначе strings.Join(sep=0x54d6ef0 "").
func (d *PHCDevice) GetDeviceNameLocked() (string, int) {
	if d == nil {
		return "", 0
	}
	return d.getDeviceNameLocked()
}

func (d *PHCDevice) getDeviceNameLocked() (string, int) {
	return d.DeviceName, len(d.DeviceName)
}

// DoesDeviceHaveName по дизассемблеру (__PHCDevice_.DoesDeviceHaveName.txt): lock 0x100; цикл по 0x30/0x38 (DeviceNames); memequal(element, name); при совпадении return true; defer unlock.
func (d *PHCDevice) DoesDeviceHaveName(name string) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, n := range d.DeviceNames {
		if n == name {
			return true
		}
	}
	return false
}

// GetDeviceNames по дизассемблеру (GetDeviceNames@@Base 0x4587860): lock 0x100; defer unlock; 0x38(device)=len; makeslice(len); typedslicecopy(0x30/0x38 → новый slice); return (slice, len, cap).
// Реконструировано по дизассемблеру бинарника timebeat-2.2.20.
func (d *PHCDevice) GetDeviceNames() ([]string, int, int) {
	if d == nil {
		return nil, 0, 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.DeviceNames)
	if n == 0 {
		return nil, 0, 0
	}
	result := make([]string, n)
	copy(result, d.DeviceNames)
	return result, n, n
}

// GetClockID по дизассемблеру 0x64(phc).
func (d *PHCDevice) GetClockID() int {
	if d == nil {
		return 0
	}
	return d.ClockID
}

// DeterminePHCOffset по дизассемблеру (__PHCDevice_.DeterminePHCOffset.txt):
// 0x64==0 → return 0; makeslice(len(*0xf0), int64); switch 0xc0: 1→Basic, return slice[0]; 2→Basic, если ret>=5e6 return 0 иначе 0xc8; 3→Extended return; 4/5→EFX/Precise, sort.Slice(func1), medianIdx=len/2+1 (1-based), return slice[medianIdx].
func (d *PHCDevice) DeterminePHCOffset() int64 {
	if d == nil {
		return 0
	}
	if d.ClockID == 0 {
		return 0
	}
	n := d.NumSamples
	if n <= 0 {
		n = 1
	}
	slice := make([]int64, n)
	switch d.StrategyType {
	case StrategyBasic:
		return d.DeterminePTPOffsetBasic(slice)
	case StrategyBasicFallback:
		ret := d.DeterminePTPOffsetBasic(slice)
		if ret+determinePHCOffsetThreshold >= determinePHCOffsetBound {
			return 0
		}
		return d.FallbackOffset
	case StrategyExtended:
		return d.DeterminePTPOffsetExtended(slice)
	case StrategyEFX:
		d.DeterminePTPOffsetEFX(slice)
		return medianOfSlice(slice)
	case StrategyPrecise:
		d.DeterminePTPOffsetPrecise(slice)
		return medianOfSlice(slice)
	default:
		return 0
	}
}

// medianOfSlice по дизассемблеру: sort.Slice(slice, func1: slice[i]>slice[j]); medianIdx = len/2+1 (1-based); return slice[medianIdx]. func1 — setg, т.е. descending; median = slice[len/2].
func medianOfSlice(s []int64) int64 {
	if len(s) == 0 {
		return 0
	}
	if len(s) == 1 {
		return s[0]
	}
	sort.Slice(s, func(i, j int) bool { return s[i] > s[j] }) // descending по func1
	medianIdx := len(s) / 2
	return s[medianIdx]
}

// getPHCToSysClockSamplesBasicImpl подставляется из phc_linux.go при сборке с linux; иначе nil.
var getPHCToSysClockSamplesBasicImpl func(d *PHCDevice, nSamples int) *ptpSysOffsetData

// GetPHCToSysClockSamplesBasic по дизассемблеру: newobject, (struct).n_samples = arg2, Ioctl(phc.0x60, PTP_SYS_OFFSET, &struct); при ошибке return nil.
func (d *PHCDevice) GetPHCToSysClockSamplesBasic(nSamples int) *ptpSysOffsetData {
	if d == nil {
		return nil
	}
	if getPHCToSysClockSamplesBasicImpl != nil {
		return getPHCToSysClockSamplesBasicImpl(d, nSamples)
	}
	return nil
}

// Константа по дизассемблеру DeterminePTPOffsetBasic: добавка к offset перед записью в slice (округление).
const determinePTPOffsetBasicRound = 0x5e4dfc14c2e60000

// nsPerSec — 1e9, по дизассемблеру 0x3b9aca00.
const nsPerSec int64 = 1e9

// sampleToNs по дизассемблеру: sec*1e9 + (nsec & 0x3fffffff).
func sampleToNs(sec int64, nsec int32) int64 {
	return sec*nsPerSec + int64(nsec&0x3fffffff)
}

// DeterminePTPOffsetBasic по дизассемблеру (DeterminePTPOffsetBasic@@Base): GetPHCToSysClockSamplesBasic(d); если nil return 0; цикл i: сэмплы по 32 байт (0x10=sec, 0x18=nsec), триплет (i, 2*i+1, 2*i+2); offset = (t1+t2)/2 - t3; slice[i] = offset + 0x5e4dfc14c2e60000; return slice[0] (или по phc.0xf0.0x28+rbx).
func (d *PHCDevice) DeterminePTPOffsetBasic(slice []int64) int64 {
	if d == nil || len(slice) == 0 {
		return 0
	}
	nWant := len(slice) * 2
	if nWant < 3 {
		nWant = 3
	}
	data := d.GetPHCToSysClockSamplesBasic(nWant)
	if data == nil {
		return 0
	}
	n := int(data.N_samples)
	if n < 3 {
		return 0
	}
	var first int64
	// Цикл по дизассемблеру: индексы i, 2*i+1, 2*i+2; offset = (t1+t2)/2 - t3.
	for i := 0; 2*i+2 < n && i < len(slice); i++ {
		t1 := sampleToNs(data.sampleSec(i), data.sampleNsec(i))
		t2 := sampleToNs(data.sampleSec(2*i+1), data.sampleNsec(2*i+1))
		t3 := sampleToNs(data.sampleSec(2*i+2), data.sampleNsec(2*i+2))
		offset := (t1+t2)/2 - t3
		slice[i] = offset + determinePTPOffsetBasicRound
		if i == 0 {
			first = slice[i]
		}
	}
	return first
}

// DeterminePTPOffsetExtended по дизассемблеру: возврат одного значения. Пока fallback на Basic (полная реализация — по дизассемблеру Extended).
func (d *PHCDevice) DeterminePTPOffsetExtended(slice []int64) int64 {
	if d == nil || len(slice) == 0 {
		return 0
	}
	return d.DeterminePTPOffsetBasic(slice)
}

// DeterminePTPOffsetEFX по дизассемблеру: заполняет slice. Пока fallback на Basic (полная реализация — по дизассемблеру EFX).
func (d *PHCDevice) DeterminePTPOffsetEFX(slice []int64) {
	if d == nil || len(slice) == 0 {
		return
	}
	d.DeterminePTPOffsetBasic(slice)
	// EFX в бинарнике может использовать другой ioctl/расчёт; median берётся в DeterminePHCOffset через sort.Slice
}

// DeterminePTPOffsetPrecise по дизассемблеру: заполняет slice. Пока fallback на Basic (полная реализация — по дизассемблеру Precise).
func (d *PHCDevice) DeterminePTPOffsetPrecise(slice []int64) {
	if d == nil || len(slice) == 0 {
		return
	}
	d.DeterminePTPOffsetBasic(slice)
}

// Константы ioctl для PPS/pin (подставляются в phc_linux при сборке с linux).
var (
	PTP_ENABLE_PPS  uintptr = 0
	PTP_PIN_SETFUNC uintptr = 0
)

// ptpPinFuncStruct — структура для PTP_PIN_SETFUNC (pin, func, channel по дизассемблеру 0x68, 0x6c, 0x70).
type ptpPinFuncStruct struct {
	Pin     int32
	Func    int32
	Channel int32
}

// ptpPeroutRequest — структура для PTP_PEROUT_REQUEST (SetPerOut@@Base 0x458e2a0): start_sec/start_nsec, period_sec/period_nsec.
type ptpPeroutRequest struct {
	StartSec   int64
	StartNsec  int64
	PeriodSec  int64
	PeriodNsec int64
}

// PTP_PEROUT_REQUEST — ioctl для периодического выхода (подставляется в phc_linux).
var PTP_PEROUT_REQUEST uintptr = 0

// Ioctl вызывает системный ioctl; реализация подставляется из phc_linux при сборке с linux.
var Ioctl = func(fd int, request uintptr, ptr unsafe.Pointer) error { return nil }

// SetPPS по дизассемблеру (__PHCDevice_.SetPPS@@Base): Ioctl(phc.FD, PTP_ENABLE_PPS, &scale).
func (d *PHCDevice) SetPPS(scale byte) error {
	if d == nil || d.FD <= 0 {
		return nil
	}
	return Ioctl(d.FD, PTP_ENABLE_PPS, unsafe.Pointer(&scale))
}

// EnablePPS по дизассемблеру (__PHCDevice_.EnablePPS@@Base): вызов SetPPS(); при успехе конкатенация строк/логирование; возврат.
func (d *PHCDevice) EnablePPS() error {
	if d == nil {
		return nil
	}
	return d.SetPPS(1)
}

// SetPinFunction по дизассемблеру (__PHCDevice_.SetPinFunction@@Base): структура pin/func/channel по 0x68/0x6c/0x70; Ioctl(phc.FD, PTP_PIN_SETFUNC, &struct).
func (d *PHCDevice) SetPinFunction(pin, funcIndex, channel int32) error {
	if d == nil || d.FD <= 0 {
		return nil
	}
	st := ptpPinFuncStruct{Pin: pin, Func: funcIndex, Channel: channel}
	return Ioctl(d.FD, PTP_PIN_SETFUNC, unsafe.Pointer(&st))
}

// SetPerOut по дизассемблеру (__PHCDevice_.SetPerOut@@Base 0x458e2a0): time.Now(); структура start_sec/start_nsec, period_sec/period_nsec по 0x28/0x40/0x48; Ioctl(phc.FD, PTP_PEROUT_REQUEST, &struct).
func (d *PHCDevice) SetPerOut(periodNs int64) error {
	if d == nil || d.FD <= 0 {
		return nil
	}
	if PTP_PEROUT_REQUEST == 0 {
		return nil
	}
	now := time.Now()
	req := ptpPeroutRequest{
		StartSec:   now.Unix(),
		StartNsec:  int64(now.Nanosecond()),
		PeriodSec:  periodNs / 1e9,
		PeriodNsec: periodNs % 1e9,
	}
	return Ioctl(d.FD, PTP_PEROUT_REQUEST, unsafe.Pointer(&req))
}

// SetSysfsE810PPS по дизассемблеру (__PHCDevice_.SetSysfsE810PPS@@Base 0x458f340): Intel E810 sysfs PPS; полная реализация по дизассемблеру — заглушка.
func (d *PHCDevice) SetSysfsE810PPS(channel int) error {
	_ = d
	_ = channel
	return nil
}

// EnablePPSOut по дизассемблеру (__PHCDevice_.EnablePPSOut@@Base 0x458db00): enable — вызов EnablePPSOut(0), SetPinFunction(0,2,0), SetPerOut(0x3b9aca00); !enable — SetPerOut(0), SetPinFunction(0,0,0). Возврат nil (в бинарнике — slice ошибок).
func (d *PHCDevice) EnablePPSOut(enable bool) error {
	if d == nil {
		return nil
	}
	if enable {
		_ = d.EnablePPSOut(false)
		if err := d.SetPinFunction(0, 2, 0); err != nil {
			return err
		}
		return d.SetPerOut(0x3b9aca00) // 1e9 ns = 1 s
	}
	_ = d.SetPerOut(0)
	return d.SetPinFunction(0, 0, 0)
}

// EnablePPSOutOnChannel по дизассемблеру (__PHCDevice_.EnablePPSOutOnChannel@@Base 0x458d680): channel!=0 — рекурсия EnablePPSOutOnChannel(0), SetPerOut(0x3b9aca00); channel==0 — SetPerOut(0); затем SetSysfsE810PPS(channel).
func (d *PHCDevice) EnablePPSOutOnChannel(channel int) error {
	if d == nil {
		return nil
	}
	if channel != 0 {
		_ = d.EnablePPSOutOnChannel(0)
		_ = d.SetPerOut(0x3b9aca00)
	} else {
		_ = d.SetPerOut(0)
	}
	return d.SetSysfsE810PPS(channel)
}

// PHCController по дизассемблеру (GetDeviceWithName@@Base): 0x00=devices ptr, 0x08=len; цикл по devices, lock device.0x100, DoesDeviceHaveName(name), unlock; при совпадении return device.
// ppsEntries — опциональный список из appConfig+0x238/0x240 для полного EnablePPSIfRequired ("ifName:idx:channel" или "channel:ifName:channelNum").
type PHCController struct {
	devices    []*PHCDevice
	ppsEntries []string // appConfig+0x238/0x240; при вызове EnablePPSIfRequired используется, если не пусто
}

var defaultPHCController *PHCController
var phcControllerOnce sync.Once

// NewController по дизассемблеру (phc.NewController@@Base): sync.Once, создаёт PHCController, SetPHCController; loadConfig читает store.Range(ConfigureTimeSource).
func NewController() {
	phcControllerOnce.Do(func() {
		defaultPHCController = &PHCController{devices: nil, ppsEntries: nil}
	})
}

// GetInstance по дизассемблеру (phc.GetInstance@@Base): sync.Once, создаёт PHCController; используется clients/phc.ConfigureTimeSource.
func GetInstance() *PHCController {
	NewController()
	return defaultPHCController
}

// GetPHCController возвращает глобальный PHCController (если задан через SetPHCController); для вызова EnablePPSIfRequired из RunWithConfig.
func GetPHCController() *PHCController {
	return defaultPHCController
}

// SetPHCController задаёт глобальный контроллер (вызывается при инициализации hostclocks/устройств).
func SetPHCController(c *PHCController) {
	defaultPHCController = c
}

// GetDeviceWithName по дизассемблеру (__PHCController_.GetDeviceWithName.txt): цикл по 0x00/0x08 (devices); lock device.0x100; device.DoesDeviceHaveName(name); unlock; при совпадении return device.
func (c *PHCController) GetDeviceWithName(name string) *PHCDevice {
	if c == nil {
		return nil
	}
	for _, d := range c.devices {
		if d == nil {
			continue
		}
		if d.DoesDeviceHaveName(name) {
			return d
		}
	}
	return nil
}

// SetPPSConfig задаёт список записей из appConfig+0x238/0x240 для EnablePPSIfRequired (формат "ifName:idx:channel" или "channel:ifName:channelNum").
func (c *PHCController) SetPPSConfig(entries []string) {
	if c == nil {
		return
	}
	c.ppsEntries = entries
}

// EnablePPSIfRequired по дизассемблеру (__PHCController_.EnablePPSIfRequired 0x4584f00): обход slice из appConfig+0x238/0x240 (список строк "ifName:idx:channel");
// strings.Split(sep=":"); при len!=3 — Logger.Error; иначе GetDeviceWithName(ifName); если part0=="channel" — EnablePPSOutOnChannel(device, channel); иначе EnablePPSOut(device).
// Если ppsEntries пуст — fallback: обход c.devices и вызов EnablePPS() для каждого.
func (c *PHCController) EnablePPSIfRequired() {
	if c == nil {
		return
	}
	if len(c.ppsEntries) > 0 {
		for _, entry := range c.ppsEntries {
			parts := strings.Split(entry, ":")
			if len(parts) != 3 {
				log.Printf("phc: EnablePPSIfRequired: invalid entry (expected ifName:idx:channel), got %q", entry)
				continue
			}
			var device *PHCDevice
			var channel int
			if parts[0] == "channel" {
				device = c.GetDeviceWithName(parts[1])
				channel, _ = strconv.Atoi(parts[2])
			} else {
				device = c.GetDeviceWithName(parts[0])
			}
			if device == nil {
				continue
			}
			if parts[0] == "channel" {
				_ = device.EnablePPSOutOnChannel(channel)
			} else {
				_ = device.EnablePPSOut(true)
			}
		}
		return
	}
	for _, d := range c.devices {
		if d == nil {
			continue
		}
		_ = d.EnablePPS()
	}
}

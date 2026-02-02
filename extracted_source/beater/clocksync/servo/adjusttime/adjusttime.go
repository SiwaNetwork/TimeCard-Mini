//go:build linux

// Package adjusttime — коррекция системных часов (Linux adjtimex).
// Реконструировано по дизассемблеру SetFrequency@@Base (code_analysis/disassembly/adjusttime_SetFrequency.txt):
// freq (int64) * rodata 0x54de698 -> adjtimex offset, mode 0x4002 (ADJ_FREQUENCY|ADJ_STATUS), syscall.Adjtimex.
package adjusttime

import (
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Константы из linux/timex.h
const (
	ADJ_OFFSET    = 0x0001
	ADJ_FREQUENCY = 0x0002
	ADJ_MAXERROR  = 0x0004
	ADJ_ESTERROR  = 0x0008
	ADJ_STATUS    = 0x0010
	ADJ_TIMECONST = 0x0020
	ADJ_TICK      = 0x4000
	ADJ_OFFSET_SINGLESHOT = 0x8001
	ADJ_OFFSET_SS_READ    = 0xa001
	// ADJ_OFFSET mode из бинарника SetOffset: 0x2100
	ADJ_OFFSET_MODE = 0x2100

	STA_PLL     = 0x0001
	STA_PPSFREQ = 0x0002
	STA_PPSTIME = 0x0004
	STA_FLL     = 0x0008
)

// SYS_clock_adjtime = 305 (0x131) на linux/amd64 — по дизассемблеру GetClockFrequency/SetOffset.
const syscallClockAdjtime = 305

// ScaledPPM — 1 ppm = 65536 в ядре (timex.freq).
const scaledPPM = 65536

// timex структура для adjtimex/clock_adjtime (смещения по дизассемблеру: Freq at 0x70(rsp) от buf 0x60, time at 0xa8/0xb0).
type timex struct {
	Modes     uint32
	_         [4]byte
	Offset    int64
	Freq      int64
	Maxerror  int64
	Esterror  int64
	Status    int32
	_         [4]byte
	Constant  int64
	Precision int64
	Tolerance int64
	_         [4]byte   // до offset 72 (0x48)
	TimeSec   int64     // 0xa8-0x60 в бинарнике SetOffset
	TimeNsec  int64     // 0xb0-0x60
	Tick      int64
	Ppsfreq   int64
	Jitter    int64
	Shift     int32
	Stabil    int64
	Jitcnt    int64
	Calcnt    int64
	Errcnt    int64
	Stbcnt    int64
	Tai       int32
	_         [44]byte
}

// GetClockUsingGetTimeSyscall по дизассемблеру: syscall 228 (SYS_clock_gettime)(clockID, &ts). По вызовам из StepClock/SlewClock — clockID = phc.0x64.
func GetClockUsingGetTimeSyscall(clockID int) time.Time {
	var ts syscall.Timespec
	_, _, err := syscall.Syscall(228, uintptr(clockID), uintptr(unsafe.Pointer(&ts)), 0)
	if err != 0 {
		return time.Now()
	}
	return time.Unix(ts.Sec, ts.Nsec)
}

// SYS_clock_settime = 227 на linux/amd64 (по дизассемблеру StepClockUsingSetTimeSyscall: mov 0xe3 = 227).
const syscallClockSettime = 227

// StepClockUsingSetTimeSyscall по дизассемблеру (adjusttime_StepClockUsingSetTimeSyscall.txt): конвертация time в timespec (0x3b9aca00, 0x3fffffff); syscall 0xe3 (227 = clock_settime)(clockID, &ts); при errno != 0 — Logger.Error, return error.
func StepClockUsingSetTimeSyscall(clockID int, t time.Time) error {
	sec := t.Unix()
	nsec := int64(t.Nanosecond() & 0x3fffffff)
	var ts syscall.Timespec
	ts.Sec = sec
	ts.Nsec = nsec
	_, _, errno := syscall.Syscall(uintptr(syscallClockSettime), uintptr(clockID), uintptr(unsafe.Pointer(&ts)), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// SetFrequency устанавливает частоту (ppm). В бинарнике: Freq = freq_int64 * const (0x54de698), Modes = 0x4002. clockID по вызовам из StepClock/SlewClock (phc.0x64).
func SetFrequency(clockID int, ppm float64) error {
	_ = clockID
	tx := &timex{
		Modes: ADJ_FREQUENCY,
		Freq:  int64(ppm * 65536), // kernel: scaled ppm (1 ppm = 65536)
	}
	return adjtimex(tx)
}

// GetClockFrequency по дизассемблеру (0x4418840): duffzero timex (buf 0x60); clockID==0 → Adjtimex(ptr); иначе Syscall(0x131, clockID, &tx, 0); при err — Logger.Error; 0x70(rsp)=Freq, div 0x54de698 (scaledPPM), return ppm.
func GetClockFrequency(clockID int) (float64, error) {
	tx := &timex{}
	if clockID == 0 {
		if err := adjtimex(tx); err != nil {
			return 0, err
		}
	} else {
		_, _, errno := syscall.Syscall(syscallClockAdjtime, uintptr(clockID), uintptr(unsafe.Pointer(tx)), 0)
		if errno != 0 {
			return 0, errno
		}
	}
	return float64(tx.Freq) / float64(scaledPPM), nil
}

// GetFrequency — алиас для совместимости (без clockID, CLOCK_REALTIME).
func GetFrequency() (float64, error) {
	return GetClockFrequency(0)
}

// GetSystemClockMaxFrequency по дизассемблеру (GetSystemClockMaxFrequency@@Base 0x4418a40): duffzero timex; Adjtimex(ptr);
// при err != 0 — Logger.Error, return 0x7a120 (500000 ppm); иначе return timex.Freq/65536 (поле по 0x98(rsp), divsd 0x54de698).
func GetSystemClockMaxFrequency() int64 {
	tx := &timex{}
	if adjtimex(tx) != nil {
		return 0x7a120 // 500000 ppm — default max при ошибке (по дизассемблеру)
	}
	return tx.Freq / int64(scaledPPM)
}

// SlewClock плавно корректирует время
func SlewClock(offsetNs int64) error {
	tx := &timex{
		Modes:  ADJ_OFFSET,
		Offset: offsetNs / 1000, // микросекунды
	}
	return adjtimex(tx)
}

// GetPreciseTime по дизассемблеру (GetPreciseTime.txt): time.Now(); маска 0x3fffffff на младшую часть; возврат (low30bits, high, 0).
func GetPreciseTime() time.Time {
	t := time.Now()
	// В бинарнике: and 0x3fffffff с наносекундами, возврат (sec, nsec&0x3fffffff, 0)
	return t
}

// IsRPIComputeModule по дизассемблеру (IsRPIComputeModule@@Base): sync.Once(onceDetectBCM54210PE), doSlow(func1); возврат detectedBCM54210PE. func1: os.Stat/ReadFile пути 0xd байт, strings.Index подстрок 0x10; при совпадении — detectedBCM54210PE=1, computeModuleType=4 или 5.
var (
	onceDetectBCM54210PE  sync.Once
	detectedBCM54210PE    bool
	computeModuleType     int64
	deviceTreeModelPath   = "/proc/device-tree/model" // 13 байт по бинарнику
	substringCM4         = "Compute Module 4"
	substringCM5         = "Compute Module 5"
)

func IsRPIComputeModule() bool {
	onceDetectBCM54210PE.Do(detectBCM54210PE)
	return detectedBCM54210PE
}

func detectBCM54210PE() {
	_, err := os.Stat(deviceTreeModelPath)
	if err != nil {
		return
	}
	data, err := os.ReadFile(deviceTreeModelPath)
	if err != nil {
		return
	}
	s := strings.TrimSpace(string(data))
	if strings.Contains(s, substringCM4) {
		detectedBCM54210PE = true
		computeModuleType = 4
	}
	if strings.Contains(s, substringCM5) {
		detectedBCM54210PE = true
		computeModuleType = 5
	}
}

// SetOffset по дизассемблеру (0x4418e40): Modes=0x2100 в 0x60(rsp); offsetNs<0 → 0xa8=-1, 0xb0=offsetNs+0x3b9aca00; иначе 0xb0=offsetNs; clockID==0 → Adjtimex; иначе Syscall(0x131, clockID, &tx, 0); при err — Logger.Error.
func SetOffset(clockID int, offsetNs int64) error {
	tx := &timex{Modes: ADJ_OFFSET_MODE}
	if offsetNs < 0 {
		tx.TimeSec = -1
		tx.TimeNsec = offsetNs + 1e9
	} else {
		tx.TimeSec = 0
		tx.TimeNsec = offsetNs
	}
	if clockID == 0 {
		return adjtimex(tx)
	}
	_, _, errno := syscall.Syscall(syscallClockAdjtime, uintptr(clockID), uintptr(unsafe.Pointer(tx)), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// PerformGranularityMeasurement по дизассемблеру (PerformGranularityMeasurement@@Base): GetClockUsingGetTimeSyscall(clockID) дважды, возврат t2.Sub(t1).
func PerformGranularityMeasurement(clockID int) time.Duration {
	t1 := GetClockUsingGetTimeSyscall(clockID)
	t2 := GetClockUsingGetTimeSyscall(clockID)
	return t2.Sub(t1)
}

// adjtimex вызывает syscall
func adjtimex(tx *timex) error {
	_, _, errno := syscall.Syscall(syscall.SYS_ADJTIMEX,
		uintptr(unsafe.Pointer(tx)), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// DetectBCM54210PE проверяет наличие BCM54210PE (то же, что IsRPIComputeModule по бинарнику).
func DetectBCM54210PE() bool {
	return detectedBCM54210PE
}

// StepRTCClock по дизассемблеру (0x4419260): openat(rax=0xffffff9c=AT_FDCWD, path 8 bytes rodata, edi=1, esi=0x180=O_RDWR|O_CLOEXEC);
// GetPreciseTime(0x4419720); time.abs/time.date — компоненты в 0x9c(sec), 0xa0(min), 0xa4(hour), 0xa8(mday), 0xac(mon-1), 0xb0(year-1900);
// ioctlPtr(fd, 0x4024700a=RTC_SET_TIME, &rtc); при err — Logger.Error; close(fd). Путь /dev/rtc0.
const (
	rtcDevicePath     = "/dev/rtc0"
	syscallIoctl      = 16
	linuxO_RDWR       = 0x2
	linuxO_CLOEXEC    = 0x80000
	linuxAT_FDCWD     = -100
	linuxRTC_SET_TIME = 0x4024700a // _IOW('p', 0x0a, struct rtc_time)
)

// rtcTime — аналог struct rtc_time из linux/rtc.h (поля int32).
type rtcTime struct {
	Sec   int32
	Min   int32
	Hour  int32
	Mday  int32
	Mon   int32
	Year  int32
	Wday  int32
	Yday  int32
	Isdst int32
}

// StepRTCClock синхронизирует аппаратные часы RTC с текущим системным временем (GetPreciseTime).
// По бинарнику: открыть /dev/rtc0, взять время, записать в RTC через RTC_SET_TIME, закрыть.
func StepRTCClock() error {
	fd, err := openRTCDevice()
	if err != nil {
		return err
	}
	defer func() { _, _, _ = syscall.Syscall(syscall.SYS_CLOSE, uintptr(fd), 0, 0) }()
	t := GetPreciseTime()
	rt := timeToRtcTime(t)
	_, _, errno := syscall.Syscall(syscallIoctl, uintptr(fd), uintptr(linuxRTC_SET_TIME), uintptr(unsafe.Pointer(&rt)))
	if errno != 0 {
		return errno
	}
	return nil
}

func openRTCDevice() (int, error) {
	path, err := syscall.BytePtrFromString(rtcDevicePath)
	if err != nil {
		return -1, err
	}
	// openat(AT_FDCWD, path, O_RDWR|O_CLOEXEC, 0) — по дизассемблеру; Linux x86_64 openat = 257
	const syscallOpenat = 257
	atFdcwd := int64(linuxAT_FDCWD)
	fd, _, errno := syscall.Syscall6(syscallOpenat, uintptr(atFdcwd), uintptr(unsafe.Pointer(path)), uintptr(linuxO_RDWR|linuxO_CLOEXEC), 0, 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func timeToRtcTime(t time.Time) rtcTime {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return rtcTime{
		Sec:   int32(sec),
		Min:   int32(min),
		Hour:  int32(hour),
		Mday:  int32(day),
		Mon:   int32(month - 1),
		Year:  int32(year - 1900),
		Wday:  int32(t.Weekday()),
		Yday:  int32(t.YearDay() - 1),
		Isdst: -1,
	}
}

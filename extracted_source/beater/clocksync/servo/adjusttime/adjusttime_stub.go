//go:build !linux

package adjusttime

import "time"

// Stub для не-Linux: adjtimex/clock_settime недоступен.

func GetClockUsingGetTimeSyscall(clockID int) time.Time { _ = clockID; return time.Now() }
func StepClockUsingSetTimeSyscall(clockID int, t time.Time) error { _ = clockID; _ = t; return nil }
func SetFrequency(clockID int, ppm float64) error { _ = clockID; _ = ppm; return nil }
func GetClockFrequency(clockID int) (float64, error) { _ = clockID; return 0, nil }
func GetFrequency() (float64, error) { return 0, nil }
func GetSystemClockMaxFrequency() int64 { return 500000 }
func SetOffset(clockID int, offsetNs int64) error { _ = clockID; _ = offsetNs; return nil }
func SlewClock(offsetNs int64) error { _ = offsetNs; return nil }
func GetPreciseTime() time.Time { return time.Now() }
func IsRPIComputeModule() bool { return false }
func PerformGranularityMeasurement(clockID int) time.Duration { _ = clockID; return 0 }
func DetectBCM54210PE() bool { return false }
func StepRTCClock() error { return nil }
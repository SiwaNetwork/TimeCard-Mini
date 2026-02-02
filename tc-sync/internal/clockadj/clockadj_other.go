//go:build !linux

package clockadj

import "time"

// Slew — заглушка на не-Linux (коррекция не выполняется).
func Slew(offsetNs int64) error {
	_ = offsetNs
	return nil
}

// SetFrequency — заглушка на не-Linux.
func SetFrequency(ppm float64) error {
	_ = ppm
	return nil
}

// Step — заглушка на не-Linux.
func Step(t time.Time) error {
	_ = t
	return nil
}

// GetFrequency — заглушка на не-Linux.
func GetFrequency() (ppm float64, err error) {
	return 0, nil
}

// GranularityNs — заглушка на не-Linux.
func GranularityNs() int64 {
	return 0
}

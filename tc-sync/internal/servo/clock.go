package servo

import "time"

// ClockSource — источник времени (системные часы; на Linux можно добавить adjtimex/PHC)
type ClockSource interface {
	Now() (time.Time, error)
	Set(time.Time) error
	SetOffset(offset time.Duration) error
	Frequency() (float64, error)
	SetFrequency(ppm float64) error
}

// SystemClock — реализация через стандартный time (без прав root — только чтение)
type SystemClock struct{}

// Now возвращает текущее системное время
func (SystemClock) Now() (time.Time, error) {
	return time.Now(), nil
}

// Set — заглушка: установка времени требует clock_settime (root)
func (SystemClock) Set(t time.Time) error {
	// На Linux: unix.Settimeofday() или clock_settime(CLOCK_REALTIME)
	_ = t
	return nil
}

// SetOffset — заглушка: установка смещения (adjtime/adjtimex)
func (SystemClock) SetOffset(offset time.Duration) error {
	_ = offset
	return nil
}

// Frequency — заглушка: 0 ppm (нет коррекции)
func (SystemClock) Frequency() (float64, error) {
	return 0, nil
}

// SetFrequency — заглушка: установка частоты через adjtimex (root)
func (SystemClock) SetFrequency(ppm float64) error {
	_ = ppm
	return nil
}

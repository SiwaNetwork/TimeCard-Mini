package source

import "time"

// TimeSource — источник времени (аналог Timebeat: GNSS, NTP, PTP, PPS)
type TimeSource interface {
	// Name возвращает имя источника для логов
	Name() string
	// Protocol возвращает протокол: gnss, ntp, ptp, pps
	Protocol() string
	// GetTime возвращает текущее время по источнику и статус
	GetTime() (time.Time, Status)
	// Close освобождает ресурсы
	Close() error
}

// Status — состояние источника (как в Timebeat: active, unavailable, etc.)
type Status int

const (
	StatusUnavailable Status = iota
	StatusUnlocked    // есть данные, но не locked (например GNSS без fix)
	StatusLocked      // источник пригоден для синхронизации
)

func (s Status) String() string {
	switch s {
	case StatusUnavailable:
		return "unavailable"
	case StatusUnlocked:
		return "unlocked"
	case StatusLocked:
		return "locked"
	default:
		return "unknown"
	}
}

// IsUsable возвращает true, если источник можно использовать для коррекции часов
func (s Status) IsUsable() bool {
	return s == StatusLocked
}

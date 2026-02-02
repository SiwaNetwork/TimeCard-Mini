package clockselect

import (
	"time"

	"github.com/shiwa/timecard-mini/tc-sync/internal/source"
)

// Election — выбор активного источника времени (аналог Timebeat: primary → secondary)
type Election struct {
	primary   []source.TimeSource
	secondary []source.TimeSource
	active    source.TimeSource
}

// NewElection создаёт выборщик из списков primary и secondary
func NewElection(primary, secondary []source.TimeSource) *Election {
	return &Election{
		primary:   primary,
		secondary: secondary,
	}
}

// Select выбирает лучший доступный источник: сначала primary, при недоступности — secondary
func (e *Election) Select() source.TimeSource {
	for _, s := range e.primary {
		if _, st := s.GetTime(); st.IsUsable() {
			e.active = s
			return s
		}
	}
	for _, s := range e.secondary {
		if _, st := s.GetTime(); st.IsUsable() {
			e.active = s
			return s
		}
	}
	e.active = nil
	return nil
}

// Active возвращает текущий активный источник (после Select)
func (e *Election) Active() source.TimeSource {
	return e.active
}

// GetTimeFromActive возвращает время от активного источника; если активного нет — (zero, false)
func (e *Election) GetTimeFromActive() (time.Time, bool) {
	if e.active == nil {
		e.Select()
	}
	if e.active == nil {
		return time.Time{}, false
	}
	t, st := e.active.GetTime()
	return t, st.IsUsable()
}

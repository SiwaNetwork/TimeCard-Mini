package clockselect

import (
	"testing"
	"time"

	"github.com/shiwa/timecard-mini/tc-sync/internal/source"
)

// mockSource реализует source.TimeSource для тестов.
type mockSource struct {
	name     string
	protocol string
	t        time.Time
	st       source.Status
}

func (m *mockSource) Name() string              { return m.name }
func (m *mockSource) Protocol() string          { return m.protocol }
func (m *mockSource) GetTime() (time.Time, source.Status) { return m.t, m.st }
func (m *mockSource) Close() error              { return nil }

func TestElection_Select(t *testing.T) {
	lockedTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	unavail := &mockSource{"u", "u", time.Time{}, source.StatusUnavailable}
	unlocked := &mockSource{"ul", "ul", lockedTime, source.StatusUnlocked}
	locked1 := &mockSource{"p1", "gnss", lockedTime, source.StatusLocked}
	locked2 := &mockSource{"s1", "ntp", lockedTime.Add(time.Second), source.StatusLocked}

	t.Run("no sources", func(t *testing.T) {
		e := NewElection(nil, nil)
		if e.Select() != nil {
			t.Error("expected nil with no sources")
		}
	})

	t.Run("primary usable", func(t *testing.T) {
		e := NewElection([]source.TimeSource{locked1}, []source.TimeSource{locked2})
		got := e.Select()
		if got != locked1 {
			t.Errorf("expected primary locked source, got %v", got)
		}
		if e.Active() != locked1 {
			t.Error("Active() should return selected source")
		}
	})

	t.Run("primary unavailable fallback to secondary", func(t *testing.T) {
		e := NewElection([]source.TimeSource{unavail, unlocked}, []source.TimeSource{locked2})
		got := e.Select()
		if got != locked2 {
			t.Errorf("expected secondary locked, got %v", got)
		}
	})

	t.Run("none usable", func(t *testing.T) {
		e := NewElection([]source.TimeSource{unavail, unlocked}, []source.TimeSource{unavail})
		got := e.Select()
		if got != nil {
			t.Errorf("expected nil when none usable, got %v", got)
		}
		if e.Active() != nil {
			t.Error("Active() should be nil")
		}
	})
}

func TestElection_GetTimeFromActive(t *testing.T) {
	lockedTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	locked := &mockSource{"p", "gnss", lockedTime, source.StatusLocked}

	t.Run("no active calls Select", func(t *testing.T) {
		e := NewElection([]source.TimeSource{locked}, nil)
		tm, ok := e.GetTimeFromActive()
		if !ok {
			t.Error("expected ok after GetTimeFromActive triggers Select")
		}
		if !tm.Equal(lockedTime) {
			t.Errorf("got %v want %v", tm, lockedTime)
		}
	})

	t.Run("no sources", func(t *testing.T) {
		e := NewElection(nil, nil)
		tm, ok := e.GetTimeFromActive()
		if ok {
			t.Error("expected !ok when no sources")
		}
		if !tm.IsZero() {
			t.Errorf("expected zero time, got %v", tm)
		}
	})

	t.Run("after Select", func(t *testing.T) {
		e := NewElection([]source.TimeSource{locked}, nil)
		e.Select()
		tm, ok := e.GetTimeFromActive()
		if !ok || !tm.Equal(lockedTime) {
			t.Errorf("GetTimeFromActive: ok=%v tm=%v", ok, tm)
		}
	})
}

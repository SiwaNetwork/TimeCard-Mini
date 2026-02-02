package clocksync

import (
	"testing"
)

func TestParseStepLimit(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"", 500_000_000},
		{"500ms", 500_000_000},
		{"15m", 15 * 60 * 1e9},
		{"1s", 1e9},
		{"invalid", 500_000_000},
	}
	for _, tt := range tests {
		got := ParseStepLimit(tt.in)
		if got != tt.want {
			t.Errorf("ParseStepLimit(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

package ubx

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestParseNAVPVTTime(t *testing.T) {
	// Минимальный payload 92 байта: год/месяц/день/час/мин/сек/valid в начале, nano на offset 16
	makePayload := func(valid byte, year int, month, day, hour, min, sec int, nano int32) []byte {
		p := make([]byte, NAVPVTSize)
		binary.LittleEndian.PutUint16(p[navPvtYear:], uint16(year))
		p[navPvtMonth] = byte(month)
		p[navPvtDay] = byte(day)
		p[navPvtHour] = byte(hour)
		p[navPvtMin] = byte(min)
		p[navPvtSec] = byte(sec)
		p[navPvtValid] = valid
		if len(p) > navPvtNano+4 {
			binary.LittleEndian.PutUint32(p[navPvtNano:], uint32(nano))
		}
		return p
	}

	t.Run("valid time", func(t *testing.T) {
		p := makePayload(NavPVTValidTime, 2025, 1, 15, 12, 30, 45, 123456789)
		got, ok := ParseNAVPVTTime(p)
		if !ok {
			t.Fatal("expected ok")
		}
		want := time.Date(2025, 1, 15, 12, 30, 45, 123456789, time.UTC)
		if !got.Equal(want) {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("no validTime flag", func(t *testing.T) {
		p := makePayload(0, 2025, 1, 15, 12, 30, 45, 0)
		_, ok := ParseNAVPVTTime(p)
		if ok {
			t.Error("expected !ok when validTime not set")
		}
	})

	t.Run("short payload", func(t *testing.T) {
		p := make([]byte, 50)
		_, ok := ParseNAVPVTTime(p)
		if ok {
			t.Error("expected !ok for short payload")
		}
	})

	t.Run("nano clamp negative", func(t *testing.T) {
		p := makePayload(NavPVTValidTime, 2025, 1, 1, 0, 0, 0, -1)
		got, ok := ParseNAVPVTTime(p)
		if !ok {
			t.Fatal("expected ok")
		}
		if got.Nanosecond() != 0 {
			t.Errorf("nano should be clamped to 0, got %d", got.Nanosecond())
		}
	})

	t.Run("nano clamp over 999999999", func(t *testing.T) {
		p := makePayload(NavPVTValidTime, 2025, 1, 1, 0, 0, 0, 2000000000)
		got, ok := ParseNAVPVTTime(p)
		if !ok {
			t.Fatal("expected ok")
		}
		if got.Nanosecond() != 999999999 {
			t.Errorf("nano should be clamped to 999999999, got %d", got.Nanosecond())
		}
	})
}

func TestIsNAVPVTPacket(t *testing.T) {
	// Пакет с 8-байтным заголовком (как ожидает NAVPVTPayload/IsNAVPVTPacket): sync(2)+class+id+len(2)+2 = 8, затем payload 92
	full := make([]byte, 8+NAVPVTSize)
	full[0], full[1] = Sync1, Sync2
	full[2], full[3] = ClassNAV, IDNAVPVT
	binary.LittleEndian.PutUint16(full[4:6], NAVPVTSize)

	t.Run("valid", func(t *testing.T) {
		if !IsNAVPVTPacket(full) {
			t.Error("expected true for valid NAV-PVT packet")
		}
	})
	t.Run("wrong sync", func(t *testing.T) {
		b := append([]byte{0x00, 0x00}, full[2:]...)
		if IsNAVPVTPacket(b) {
			t.Error("expected false for wrong sync")
		}
	})
	t.Run("wrong class", func(t *testing.T) {
		b := make([]byte, len(full))
		copy(b, full)
		b[2] = 0xff
		if IsNAVPVTPacket(b) {
			t.Error("expected false for wrong class")
		}
	})
	t.Run("too short", func(t *testing.T) {
		if IsNAVPVTPacket(full[:20]) {
			t.Error("expected false for short packet")
		}
	})
}

func TestNAVPVTPayload(t *testing.T) {
	// Пакет с payload начиная с байта 8 (как в NAVPVTPayload)
	pkt := make([]byte, 8+NAVPVTSize)
	pkt[0], pkt[1] = Sync1, Sync2
	pkt[2], pkt[3] = ClassNAV, IDNAVPVT
	binary.LittleEndian.PutUint16(pkt[4:6], NAVPVTSize)
	pkt[8] = 1 // первый байт payload

	t.Run("valid", func(t *testing.T) {
		got := NAVPVTPayload(pkt)
		if len(got) != NAVPVTSize || got[0] != 1 {
			t.Errorf("got payload len=%d first=%d", len(got), got[0])
		}
	})
	t.Run("short packet", func(t *testing.T) {
		got := NAVPVTPayload(pkt[:5])
		if got != nil {
			t.Error("expected nil for short packet")
		}
	})
}

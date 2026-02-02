package source

import (
	"fmt"
	"time"

	"github.com/shiwa/timecard-mini/tc-sync/internal/ubx"
)

// NAV-PVT read timeout (u-blox часто шлёт NAV-PVT 1 Hz)
const gnssReadTimeout = 1500 * time.Millisecond

// GNSS — источник времени по GNSS (UBX/Timecard Mini).
// GetTime() читает UBX-NAV-PVT с приёмника и возвращает UTC время приёмника при валидном fix.
type GNSS struct {
	port    *ubx.Port
	device  string
	baud    int
	lastNow time.Time
	lastOk  bool
}

// NewGNSS создаёт источник GNSS по последовательному порту
func NewGNSS(device string, baud int) (*GNSS, error) {
	if baud == 0 {
		baud = 9600
	}
	port, err := ubx.Open(device, baud)
	if err != nil {
		return nil, err
	}
	return &GNSS{
		port:   port,
		device: device,
		baud:   baud,
	}, nil
}

// Name возвращает имя источника
func (g *GNSS) Name() string {
	return fmt.Sprintf("gnss:%s", g.device)
}

// Protocol возвращает протокол
func (g *GNSS) Protocol() string {
	return "gnss"
}

// GetTime возвращает текущее время по приёмнику.
// Читает UBX-NAV-PVT с порта; при валидном времени (validTime) возвращает UTC приёмника и StatusLocked.
// Если NAV-PVT не пришёл или время невалидно — возвращает последнее известное время (или time.Now()) и StatusUnlocked.
func (g *GNSS) GetTime() (time.Time, Status) {
	deadline := time.Now().Add(gnssReadTimeout)
	for time.Now().Before(deadline) {
		packet, err := g.port.ReadUBX(gnssReadTimeout / 2)
		if err != nil {
			if g.lastOk {
				return g.lastNow, StatusLocked
			}
			return time.Now().UTC(), StatusUnlocked
		}
		if !ubx.IsNAVPVTPacket(packet) {
			continue
		}
		payload := ubx.NAVPVTPayload(packet)
		t, ok := ubx.ParseNAVPVTTime(payload)
		if !ok {
			continue
		}
		g.lastNow = t
		g.lastOk = true
		return t, StatusLocked
	}
	if g.lastOk {
		return g.lastNow, StatusLocked
	}
	return time.Now().UTC(), StatusUnlocked
}

// Close закрывает порт
func (g *GNSS) Close() error {
	if g.port == nil {
		return nil
	}
	return g.port.Close()
}

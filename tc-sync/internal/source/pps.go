package source

import (
	"fmt"
	"time"
)

// PPS — источник времени по PPS (1 pulse-per-second).
// Время секунды берётся с linked_device (GNSS); начало секунды + cable_delay даёт опорное время.
// На Linux при наличии /dev/pps можно дополнительно проверять наличие сигнала (pps_linux.go).
type PPS struct {
	interfaceName string
	pin           int
	linkedDevice  string
	cableDelayNs  int64
	ppsIndex     int   // индекс /dev/pps{N} на Linux (для опционального fetch)
	gnss         *GNSS // источник секунды (linked_device)
}

// NewPPS создаёт PPS источник.
// linkedDevice — путь к GNSS (например /dev/ttyS0) для получения секунды; при пустом GetTime() вернёт Unavailable.
// cableDelayNs — задержка кабеля в наносекундах (добавляется к началу секунды).
// linkedBaud — скорость linked_device (0 = 115200).
// ppsIndex — индекс /dev/pps{N} на Linux для опциональной проверки сигнала (0 = /dev/pps0); <0 не использовать.
func NewPPS(interfaceName string, pin int, linkedDevice string, cableDelayNs int, linkedBaud int, ppsIndex int) (*PPS, error) {
	p := &PPS{
		interfaceName: interfaceName,
		pin:           pin,
		linkedDevice:  linkedDevice,
		cableDelayNs:  int64(cableDelayNs),
		ppsIndex:     ppsIndex,
	}
	if linkedDevice == "" {
		return p, nil
	}
	if linkedBaud == 0 {
		linkedBaud = 115200
	}
	gnss, err := NewGNSS(linkedDevice, linkedBaud)
	if err != nil {
		return nil, fmt.Errorf("pps linked_device %s: %w", linkedDevice, err)
	}
	p.gnss = gnss
	return p, nil
}

// Name возвращает имя источника
func (p *PPS) Name() string {
	return fmt.Sprintf("pps:%s pin%d linked=%s", p.interfaceName, p.pin, p.linkedDevice)
}

// Protocol возвращает протокол
func (p *PPS) Protocol() string {
	return "pps"
}

// getPPSSubSecond если задана, возвращает подсекунду с /dev/pps (только Linux).
var getPPSSubSecond func(ppsIndex int) (nsec int32, ok bool)

// GetTime возвращает опорное время: секунда с linked_device (GNSS), начало секунды + cable_delay.
// На Linux при заданном ppsIndex и getPPSSubSecond подсекунда может браться с /dev/pps{N}.
// Без linked_device возвращает StatusUnavailable.
func (p *PPS) GetTime() (time.Time, Status) {
	if p.gnss == nil {
		return time.Time{}, StatusUnavailable
	}
	t, st := p.gnss.GetTime()
	if st != StatusLocked {
		return time.Time{}, st
	}
	sec := t.Truncate(time.Second)
	ref := sec.Add(time.Duration(p.cableDelayNs))
	if getPPSSubSecond != nil && p.ppsIndex >= 0 {
		if nsec, ok := getPPSSubSecond(p.ppsIndex); ok {
			ref = sec.Add(time.Duration(nsec)).Add(time.Duration(p.cableDelayNs))
		}
	}
	return ref, StatusLocked
}

// Close закрывает linked GNSS
func (p *PPS) Close() error {
	if p.gnss != nil {
		return p.gnss.Close()
	}
	return nil
}

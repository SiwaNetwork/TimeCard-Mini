package source

import (
	"fmt"
	"net"
	"time"
)

// NTP — источник времени по NTP (клиент)
// Минимальный NTP client (один запрос — время с сервера)
type NTP struct {
	host    string
	timeout time.Duration
}

// NewNTP создаёт NTP источник
func NewNTP(host string, timeout time.Duration) *NTP {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &NTP{host: host, timeout: timeout}
}

// Name возвращает имя источника
func (n *NTP) Name() string {
	return fmt.Sprintf("ntp:%s", n.host)
}

// Protocol возвращает протокол
func (n *NTP) Protocol() string {
	return "ntp"
}

// GetTime запрашивает время у NTP сервера (RFC 5905, упрощённо)
func (n *NTP) GetTime() (time.Time, Status) {
	addr, err := net.ResolveUDPAddr("udp", n.host+":123")
	if err != nil {
		return time.Time{}, StatusUnavailable
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return time.Time{}, StatusUnavailable
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(n.timeout)); err != nil {
		return time.Time{}, StatusUnavailable
	}
	// NTP request: 48 bytes, first byte = 0x1b (version 4, client)
	req := make([]byte, 48)
	req[0] = 0x1b
	if _, err := conn.Write(req); err != nil {
		return time.Time{}, StatusUnavailable
	}
	resp := make([]byte, 48)
	if _, err := conn.Read(resp); err != nil {
		return time.Time{}, StatusUnavailable
	}
	// Seconds in bytes 40-43 (big-endian), fraction 44-47
	sec := uint32(resp[40])<<24 | uint32(resp[41])<<16 | uint32(resp[42])<<8 | uint32(resp[43])
	frac := uint32(resp[44])<<24 | uint32(resp[45])<<16 | uint32(resp[46])<<8 | uint32(resp[47])
	// NTP epoch = 1900-01-01
	ntpEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	t := ntpEpoch.Add(time.Duration(sec)*time.Second + time.Duration(frac)*time.Second/0x100000000)
	return t.UTC(), StatusLocked
}

// Close не требует освобождения ресурсов
func (n *NTP) Close() error {
	return nil
}

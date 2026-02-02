package ubx

import (
	"fmt"
	"io"
	"time"

	"github.com/tarm/serial"
)

// Port — обёртка над последовательным портом для UBX
type Port struct {
	port *serial.Port
}

// Open открывает последовательный порт
func Open(device string, baud int) (*Port, error) {
	c := &serial.Config{
		Name: device,
		Baud: baud,
	}
	p, err := serial.OpenPort(c)
	if err != nil {
		return nil, fmt.Errorf("serial open %s: %w", device, err)
	}
	return &Port{port: p}, nil
}

// WritePacket отправляет готовый UBX пакет
func (p *Port) WritePacket(packet []byte) error {
	_, err := p.port.Write(packet)
	return err
}

// ConfigureTimePulse отправляет CFG-TP5 на устройство
func (p *Port) ConfigureTimePulse(c TP5Config) error {
	packet := BuildCFGTP5(c)
	return p.WritePacket(packet)
}

// readerWithDeadline — интерфейс для установки таймаута чтения (есть у *os.File, нет у tarm/serial на Windows).
type readerWithDeadline interface {
	SetReadDeadline(t time.Time) error
}

// ReadUBX читает один UBX пакет (ждёт sync, затем length, затем payload+checksum)
func (p *Port) ReadUBX(timeout time.Duration) ([]byte, error) {
	if rd, ok := interface{}(p.port).(readerWithDeadline); ok {
		deadline := time.Now().Add(timeout)
		if err := rd.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
	}
	buf := make([]byte, 0, 512)
	// Читаем до sync
	for {
		var b [1]byte
		if _, err := io.ReadFull(p.port, b[:]); err != nil {
			return nil, err
		}
		buf = append(buf, b[0])
		if len(buf) >= 2 && buf[len(buf)-2] == Sync1 && buf[len(buf)-1] == Sync2 {
			break
		}
		if len(buf) > 2 {
			buf = buf[len(buf)-2:]
		}
	}
	// Читаем оставшиеся 6 байт заголовка (class, id, length[2])
	header := make([]byte, 6)
	if _, err := io.ReadFull(p.port, header); err != nil {
		return nil, err
	}
	buf = append(buf, header...)
	length := uint16(header[2]) | uint16(header[3])<<8
	rest := make([]byte, int(length)+2)
	if _, err := io.ReadFull(p.port, rest); err != nil {
		return nil, err
	}
	buf = append(buf, rest...)
	if !VerifyChecksum(buf) {
		return buf, fmt.Errorf("ubx checksum mismatch")
	}
	return buf, nil
}

// Close закрывает порт
func (p *Port) Close() error {
	if p.port == nil {
		return nil
	}
	return p.port.Close()
}

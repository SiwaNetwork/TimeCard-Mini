package source

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// NMEA read timeout
const nmeaReadTimeout = 2 * time.Second

// NMEA — источник времени по NMEA RMC (GPRMC/GNRMC) с последовательного порта.
// Парсит время и дату из RMC; опционально применяет offset (нс).
type NMEA struct {
	port   *serial.Port
	device string
	baud   int
	offset int64 // статическое смещение в наносекундах (как в shiwatime)
	lastOk bool
	lastT  time.Time
}

// NewNMEA создаёт источник NMEA по последовательному порту.
func NewNMEA(device string, baud int, offsetNs int64) (*NMEA, error) {
	if baud == 0 {
		baud = 9600
	}
	c := &serial.Config{Name: device, Baud: baud}
	port, err := serial.OpenPort(c)
	if err != nil {
		return nil, fmt.Errorf("nmea open %s: %w", device, err)
	}
	return &NMEA{
		port:   port,
		device: device,
		baud:   baud,
		offset: offsetNs,
	}, nil
}

// Name возвращает имя источника
func (n *NMEA) Name() string {
	return fmt.Sprintf("nmea:%s", n.device)
}

// Protocol возвращает протокол
func (n *NMEA) Protocol() string {
	return "nmea"
}

// GetTime читает строки NMEA, ищет GPRMC/GNRMC, парсит UTC время и дату; возвращает время + offset.
func (n *NMEA) GetTime() (time.Time, Status) {
	deadline := time.Now().Add(nmeaReadTimeout)
	rd := bufio.NewReader(n.port)
	for time.Now().Before(deadline) {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				continue
			}
			if n.lastOk {
				return n.lastT, StatusLocked
			}
			return time.Time{}, StatusUnavailable
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "$GP") && !strings.HasPrefix(line, "$GN") {
			continue
		}
		if !strings.Contains(line, "RMC") {
			continue
		}
		t, ok := parseRMC(line)
		if !ok {
			continue
		}
		if n.offset != 0 {
			t = t.Add(time.Duration(n.offset))
		}
		n.lastT = t
		n.lastOk = true
		return t, StatusLocked
	}
	if n.lastOk {
		return n.lastT, StatusLocked
	}
	return time.Time{}, StatusUnlocked
}

// parseRMC парсит $GPRMC или $GNRMC: поле 1 = hhmmss.ss, поле 2 = A/V, поле 9 = ddmmyy.
func parseRMC(line string) (time.Time, bool) {
	// Убрать checksum *xx
	if i := strings.Index(line, "*"); i >= 0 {
		line = line[:i]
	}
	parts := strings.Split(line, ",")
	if len(parts) < 10 {
		return time.Time{}, false
	}
	// parts[0] = $GPRMC, [1] = time, [2] = status, [9] = date
	if parts[2] != "A" {
		return time.Time{}, false
	}
	timeStr := parts[1]
	dateStr := parts[9]
	// time: hhmmss.ss
	if len(timeStr) < 6 {
		return time.Time{}, false
	}
	hh, _ := strconv.Atoi(timeStr[0:2])
	mm, _ := strconv.Atoi(timeStr[2:4])
	ss, _ := strconv.Atoi(timeStr[4:6])
	nsec := 0
	if len(timeStr) >= 8 && timeStr[6] == '.' {
		fracStr := timeStr[7:]
		frac, _ := strconv.Atoi(fracStr)
		// frac = дробная часть (1–9 цифр) -> наносекунды
		digits := len(fracStr)
		if digits > 9 {
			digits = 9
		}
		for i := 0; i < 9-digits; i++ {
			frac *= 10
		}
		nsec = frac
		if nsec > 999999999 {
			nsec = 999999999
		}
	}
	// date: ddmmyy
	if len(dateStr) < 6 {
		return time.Time{}, false
	}
	day, _ := strconv.Atoi(dateStr[0:2])
	month, _ := strconv.Atoi(dateStr[2:4])
	year, _ := strconv.Atoi(dateStr[4:6])
	if year < 80 {
		year += 2000
	} else {
		year += 1900
	}
	t := time.Date(year, time.Month(month), day, hh, mm, ss, nsec, time.UTC)
	return t, true
}

// Close закрывает порт
func (n *NMEA) Close() error {
	if n.port == nil {
		return nil
	}
	return n.port.Close()
}

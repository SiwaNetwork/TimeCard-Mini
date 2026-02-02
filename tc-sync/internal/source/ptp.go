package source

import (
	"fmt"
	"time"
)

// PTP — источник времени по PTP (IEEE 1588).
// На Linux: чтение времени из PHC (/dev/ptpN), синхронизированного ptp4l (linuxptp).
// phcDevice — путь к PHC, например /dev/ptp0; при пустом используется /dev/ptp0 на Linux.
type PTP struct {
	domain    int
	iface     string
	masters   []string
	phcDevice string // путь к PHC (/dev/ptp0), для чтения времени после ptp4l
}

// getTimeFromPHC если задана, читает время из PHC (только Linux).
var getTimeFromPHC func(phcDevice string) (time.Time, bool)

// NewPTP создаёт PTP источник.
// phcDevice — путь к PHC для чтения времени (например /dev/ptp0); при пустом на Linux используется /dev/ptp0.
func NewPTP(domain int, iface string, masters []string, phcDevice string) *PTP {
	if phcDevice == "" {
		phcDevice = "/dev/ptp0"
	}
	return &PTP{
		domain:    domain,
		iface:     iface,
		masters:   masters,
		phcDevice: phcDevice,
	}
}

// Name возвращает имя источника
func (p *PTP) Name() string {
	return fmt.Sprintf("ptp:domain%d %s phc=%s", p.domain, p.iface, p.phcDevice)
}

// Protocol возвращает протокол
func (p *PTP) Protocol() string {
	return "ptp"
}

// GetTime возвращает время из PHC (Linux, ptp4l) или StatusUnavailable.
func (p *PTP) GetTime() (time.Time, Status) {
	if getTimeFromPHC != nil {
		if t, ok := getTimeFromPHC(p.phcDevice); ok {
			return t, StatusLocked
		}
	}
	return time.Time{}, StatusUnavailable
}

// Close не требует освобождения ресурсов (PHC читается по требованию)
func (p *PTP) Close() error {
	return nil
}

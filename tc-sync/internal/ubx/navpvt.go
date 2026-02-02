package ubx

import (
	"encoding/binary"
	"time"
)

// NAV class и ID (u-blox)
const (
	ClassNAV   = 0x01
	IDNAVPVT   = 0x07 // NAV-PVT: position, velocity, time
	NAVPVTSize = 92   // минимальный размер payload NAV-PVT
)

// NAV-PVT time offsets в payload (после 8-байтного header пакета: payload = packet[8:8+length])
const (
	navPvtYear  = 4  // uint16
	navPvtMonth = 6  // uint8
	navPvtDay   = 7  // uint8
	navPvtHour  = 8  // uint8
	navPvtMin   = 9  // uint8
	navPvtSec   = 10 // uint8
	navPvtValid = 11 // uint8: bit0 validDate, bit1 validTime, bit2 fullyResolved
	navPvtNano  = 16 // int32, наносекунды
)

// Valid flags NAV-PVT
const (
	NavPVTValidDate = 1 << 0
	NavPVTValidTime = 1 << 1
	NavPVTValidFullyResolved = 1 << 2
)

// ParseNAVPVTTime парсит UTC время из payload UBX-NAV-PVT (92+ байт).
// Возвращает (time.Time в UTC, true) если valid указывает на пригодное время.
func ParseNAVPVTTime(payload []byte) (time.Time, bool) {
	if len(payload) < NAVPVTSize {
		return time.Time{}, false
	}
	valid := payload[navPvtValid]
	if valid&NavPVTValidTime == 0 {
		return time.Time{}, false
	}
	year := int(binary.LittleEndian.Uint16(payload[navPvtYear:]))
	month := int(payload[navPvtMonth])
	day := int(payload[navPvtDay])
	hour := int(payload[navPvtHour])
	min := int(payload[navPvtMin])
	sec := int(payload[navPvtSec])
	nano := 0
	if len(payload) > navPvtNano+4 {
		n := int32(binary.LittleEndian.Uint32(payload[navPvtNano : navPvtNano+4]))
		if n < 0 {
			nano = 0
		} else if n > 999999999 {
			nano = 999999999
		} else {
			nano = int(n)
		}
	}
	t := time.Date(year, time.Month(month), day, hour, min, sec, nano, time.UTC)
	return t, true
}

// IsNAVPVTPacket возвращает true, если пакет — UBX-NAV-PVT (class 0x01, id 0x07).
func IsNAVPVTPacket(packet []byte) bool {
	if len(packet) < 8+NAVPVTSize {
		return false
	}
	if packet[0] != Sync1 || packet[1] != Sync2 {
		return false
	}
	return packet[2] == ClassNAV && packet[3] == IDNAVPVT
}

// NAVPVTPayload возвращает payload NAV-PVT из полного пакета (без header и checksum).
func NAVPVTPayload(packet []byte) []byte {
	if len(packet) < 8 {
		return nil
	}
	payloadLen := int(binary.LittleEndian.Uint16(packet[4:6]))
	if len(packet) < 8+payloadLen {
		return nil
	}
	return packet[8 : 8+payloadLen]
}

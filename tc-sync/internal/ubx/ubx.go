package ubx

import "encoding/binary"

// Sync bytes для UBX протокола
const (
	Sync1 = 0xB5
	Sync2 = 0x62
)

// Классы и ID сообщений
const (
	ClassCFG = 0x06
	IDTP5    = 0x31 // CFG-TP5 Time Pulse
	IDACK    = 0x05 // CFG-ACK
	IDNAK    = 0x00 // CFG-NAK
)

// Header — заголовок UBX сообщения (8 байт)
type Header struct {
	Sync1  uint8
	Sync2  uint8
	Class  uint8
	ID     uint8
	Length uint16
}

// Checksum вычисляет UBX контрольную сумму (без sync bytes)
func Checksum(data []byte) (ckA, ckB uint8) {
	for _, b := range data {
		ckA += b
		ckB += ckA
	}
	return ckA, ckB
}

// EncodePacket собирает полный UBX пакет: header + payload + checksum
func EncodePacket(class, id uint8, payload []byte) []byte {
	length := uint16(len(payload))
	buf := make([]byte, 0, 8+len(payload)+2)
	buf = append(buf, Sync1, Sync2, class, id)
	buf = binary.LittleEndian.AppendUint16(buf, length)
	buf = append(buf, payload...)
	ckA, ckB := Checksum(buf[2:])
	buf = append(buf, ckA, ckB)
	return buf
}

// ParseHeader парсит заголовок из буфера (минимум 8 байт)
func ParseHeader(buf []byte) (h Header, ok bool) {
	if len(buf) < 8 || buf[0] != Sync1 || buf[1] != Sync2 {
		return Header{}, false
	}
	h.Sync1 = buf[0]
	h.Sync2 = buf[1]
	h.Class = buf[2]
	h.ID = buf[3]
	h.Length = binary.LittleEndian.Uint16(buf[4:6])
	return h, true
}

// VerifyChecksum проверяет контрольную сумму пакета (header + payload + 2 байта checksum)
func VerifyChecksum(packet []byte) bool {
	if len(packet) < 10 {
		return false
	}
	ckA, ckB := Checksum(packet[2 : len(packet)-2])
	return packet[len(packet)-2] == ckA && packet[len(packet)-1] == ckB
}

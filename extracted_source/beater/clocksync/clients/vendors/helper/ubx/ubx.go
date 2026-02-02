// Package ubx — протокол u-blox UBX
package ubx

// UBX_HEADER константы
const (
	Sync1     byte = 0xB5
	Sync2     byte = 0x62
	ClassCFG  byte = 0x06
	ClassNAV  byte = 0x01
	ClassMON  byte = 0x0A
	IDCFGTP5  byte = 0x31
	IDNAVPVT  byte = 0x07
)

// UBX_HEADER структура заголовка
var UBX_HEADER = []byte{Sync1, Sync2}

// TP5Message — структура CFG-TP5
type TP5Message struct {
	TPIdx           uint8
	Reserved0       uint8
	Reserved1       uint16
	AntCableDelay   int16
	RFGroupDelay    int16
	FreqPeriod      uint32
	FreqPeriodLock  uint32
	PulseLenRatio   uint32
	PulseLenRatioLock uint32
	UserConfigDelay int32
	Flags           uint32
}

// NAVPVTMessage — структура NAV-PVT (92 байта)
type NAVPVTMessage struct {
	ITOW        uint32
	Year        uint16
	Month       uint8
	Day         uint8
	Hour        uint8
	Min         uint8
	Sec         uint8
	Valid       uint8
	TAcc        uint32
	Nano        int32
	FixType     uint8
	Flags       uint8
	Flags2      uint8
	NumSV       uint8
	Lon         int32
	Lat         int32
	Height      int32
	HMSL        int32
	HAcc        uint32
	VAcc        uint32
	VelN        int32
	VelE        int32
	VelD        int32
	GSpeed      int32
	HeadMot     int32
	SAcc        uint32
	HeadAcc     uint32
	PDOP        uint16
	Flags3      uint16
	Reserved1   [4]byte
	HeadVeh     int32
	MagDec      int16
	MagAcc      uint16
}

// EncodePacket создаёт UBX пакет
func EncodePacket(class, id byte, payload []byte) []byte {
	length := len(payload)
	pkt := make([]byte, 8+length)
	pkt[0] = Sync1
	pkt[1] = Sync2
	pkt[2] = class
	pkt[3] = id
	pkt[4] = byte(length & 0xFF)
	pkt[5] = byte(length >> 8)
	copy(pkt[6:], payload)
	// Checksum
	var ckA, ckB byte
	for i := 2; i < 6+length; i++ {
		ckA += pkt[i]
		ckB += ckA
	}
	pkt[6+length] = ckA
	pkt[7+length] = ckB
	return pkt
}

// DecodePacket декодирует UBX пакет
func DecodePacket(data []byte) (class, id byte, payload []byte, ok bool) {
	if len(data) < 8 {
		return 0, 0, nil, false
	}
	if data[0] != Sync1 || data[1] != Sync2 {
		return 0, 0, nil, false
	}
	class = data[2]
	id = data[3]
	length := int(data[4]) | int(data[5])<<8
	if len(data) < 8+length {
		return 0, 0, nil, false
	}
	payload = data[6 : 6+length]
	// Verify checksum
	var ckA, ckB byte
	for i := 2; i < 6+length; i++ {
		ckA += data[i]
		ckB += ckA
	}
	if ckA != data[6+length] || ckB != data[7+length] {
		return 0, 0, nil, false
	}
	return class, id, payload, true
}

// BuildCFGTP5 создаёт пакет CFG-TP5
func BuildCFGTP5(tp5 *TP5Message) []byte {
	payload := make([]byte, 32)
	payload[0] = tp5.TPIdx
	// ... заполнение полей
	return EncodePacket(ClassCFG, IDCFGTP5, payload)
}

// IsEntireUBXMessageReceived по дизассемблеру runReadloop (0x4402680): проверяет, что в buf есть полный UBX-пакет.
// Сигнатура по вызову: (buf slice, off, length) → bool. Используется перед ToUBXMessage.
func IsEntireUBXMessageReceived(buf []byte, off, length int) bool {
	if off < 0 || length < 8 || off+length > len(buf) {
		return false
	}
	payloadLen := int(buf[off+4]) | int(buf[off+5])<<8
	if length < 8+payloadLen {
		return false
	}
	// Checksum at 6+payloadLen, 7+payloadLen
	end := off + 8 + payloadLen
	if end > len(buf) {
		return false
	}
	var ckA, ckB byte
	for i := off + 2; i < off+6+payloadLen; i++ {
		ckA += buf[i]
		ckB += ckA
	}
	return ckA == buf[off+6+payloadLen] && ckB == buf[off+7+payloadLen]
}

// ToUBXMessage по дизассемблеру runReadloop (0x4402160): парсит буфер в UBX-сообщение; возвращает (msg, consumed).
// Заглушка: возвращает (nil, 0); полная реконструкция — разбор class/id и заполнение *NAVPVTMessage и др.
func ToUBXMessage(buf []byte, off, length int) (msg interface{}, consumed int) {
	if !IsEntireUBXMessageReceived(buf, off, length) {
		return nil, 0
	}
	payloadLen := int(buf[off+4]) | int(buf[off+5])<<8
	consumed = 8 + payloadLen
	class, id := buf[off+2], buf[off+3]
	if class == ClassNAV && id == IDNAVPVT && payloadLen >= 92 {
		p := buf[off+6 : off+6+92]
		nav := &NAVPVTMessage{}
		nav.ITOW = uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24
		nav.Year = uint16(p[4]) | uint16(p[5])<<8
		nav.Month = p[6]
		nav.Day = p[7]
		nav.Hour = p[8]
		nav.Min = p[9]
		nav.Sec = p[10]
		nav.Valid = p[11]
		nav.TAcc = uint32(p[12]) | uint32(p[13])<<8 | uint32(p[14])<<16 | uint32(p[15])<<24
		nav.Nano = int32(uint32(p[16]) | uint32(p[17])<<8 | uint32(p[18])<<16 | uint32(p[19])<<24)
		nav.FixType = p[20]
		nav.NumSV = p[23]
		return nav, consumed
	}
	return nil, consumed
}

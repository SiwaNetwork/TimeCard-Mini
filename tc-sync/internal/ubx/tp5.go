package ubx

import "encoding/binary"

// CFG-TP5 payload layout (32 bytes, u-blox spec)
// Offset 0:   tpIdx (1)
// Offset 1:   version (1)
// Offset 2-4: reserved (2)
// Offset 4-6: antCableDelay (2, int16)
// Offset 6-8: rfGroupDelay (2, int16)
// Offset 8:   freqPeriod (4), freqPeriodLock (4)
// Offset 16:  pulseLenRatio (4), pulseLenRatioLock (4) — длина импульса в нс
// Offset 24:  userConfigDelay (4, int32)
// Offset 28:  flags (4)

const TP5PayloadSize = 32

// TP5Flags — биты флагов CFG-TP5
const (
	TP5Active        = 0x01
	TP5LockGnssFreq  = 0x02
	TP5LockedOtherSet = 0x04
	TP5IsLength      = 0x10 // использовать длительность импульса (length), а не ratio
	TP5AlignToTow    = 0x20
	TP5Polarity      = 0x40
)

// TP5Config — параметры Time Pulse 5
type TP5Config struct {
	TPIdx             uint8
	AntCableDelayNs   int16
	RfGroupDelayNs    int16
	FreqPeriod        uint32 // мкГц, 1 Гц = 1000000
	FreqPeriodLock    uint32
	PulseLenRatioNs   uint32 // длительность импульса в наносекундах
	PulseLenRatioLock uint32
	UserConfigDelayNs int32
	Active            bool
	LockGnssFreq      bool
	LockedOtherSet    bool
	IsLength          bool
	AlignToTow        bool
	Polarity          bool
}

// DefaultTP5 возвращает конфиг по умолчанию (1 Гц, 5 мс импульс)
func DefaultTP5() TP5Config {
	return TP5Config{
		TPIdx:             0,
		FreqPeriod:        1000000,
		FreqPeriodLock:    1000000,
		PulseLenRatioNs:   5000000, // 5 ms
		PulseLenRatioLock: 5000000,
		Active:            true,
		LockGnssFreq:      true,
		LockedOtherSet:    true,
		IsLength:          true,
		AlignToTow:        true,
	}
}

// Marshal сериализует TP5Config в 32-байтный payload
func (c TP5Config) Marshal() []byte {
	payload := make([]byte, TP5PayloadSize)
	payload[0] = c.TPIdx
	payload[1] = 0 // version
	// 2-4 reserved
	binary.LittleEndian.PutUint16(payload[4:6], uint16(c.AntCableDelayNs))
	binary.LittleEndian.PutUint16(payload[6:8], uint16(c.RfGroupDelayNs))
	binary.LittleEndian.PutUint32(payload[8:12], c.FreqPeriod)
	binary.LittleEndian.PutUint32(payload[12:16], c.FreqPeriodLock)
	binary.LittleEndian.PutUint32(payload[16:20], c.PulseLenRatioNs)
	binary.LittleEndian.PutUint32(payload[20:24], c.PulseLenRatioLock)
	binary.LittleEndian.PutUint32(payload[24:28], uint32(c.UserConfigDelayNs))
	var flags uint32
	if c.Active {
		flags |= TP5Active
	}
	if c.LockGnssFreq {
		flags |= TP5LockGnssFreq
	}
	if c.LockedOtherSet {
		flags |= TP5LockedOtherSet
	}
	if c.IsLength {
		flags |= TP5IsLength
	}
	if c.AlignToTow {
		flags |= TP5AlignToTow
	}
	if c.Polarity {
		flags |= TP5Polarity
	}
	binary.LittleEndian.PutUint32(payload[28:32], flags)
	return payload
}

// BuildCFGTP5 собирает полный UBX CFG-TP5 пакет
func BuildCFGTP5(c TP5Config) []byte {
	return EncodePacket(ClassCFG, IDTP5, c.Marshal())
}

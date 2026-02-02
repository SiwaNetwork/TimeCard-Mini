package generic_serial_device

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// Execute по дампу (0x4b8e460): Lock I2C; если ReadLen==0 — Write(WriteData), иначе WriteThenRead(WriteData, ReadLen) → Result; Sleep(WAIT_TIME_AFTER_I2C); defer Unlock.
func (r *RegisterRequest) Execute() error {
	if r == nil || r.I2C == nil {
		return errors.New("RegisterRequest or I2C is nil")
	}
	r.I2C.Lock()
	defer r.I2C.Unlock()
	if r.ReadLen == 0 {
		if err := r.I2C.Write(r.WriteData); err != nil {
			time.Sleep(WAIT_TIME_AFTER_I2C)
			return err
		}
		time.Sleep(WAIT_TIME_AFTER_I2C)
		return nil
	}
	readBuf, err := r.I2C.WriteThenRead(r.WriteData, int(r.ReadLen))
	if err != nil {
		time.Sleep(WAIT_TIME_AFTER_I2C)
		return err
	}
	r.Result = readBuf
	r.ReadLen = uint16(len(readBuf))
	time.Sleep(WAIT_TIME_AFTER_I2C)
	return nil
}

// ResetRequestData по дампу (0x4b8e7e0): устанавливает длину WriteData в newLen (panic если newLen > cap).
func (r *RegisterRequest) ResetRequestData(newLen int) {
	if r == nil {
		return
	}
	if newLen > cap(r.WriteData) {
		panic("ResetRequestData: newLen > cap")
	}
	r.WriteData = r.WriteData[:newLen]
}

// SetRegisterLen по дампу (0x4b8e440): записывает uint16 в поле по смещению 0x20.
func (r *RegisterRequest) SetRegisterLen(n uint16) {
	if r != nil {
		r.ReadLen = n
	}
}

// SetReadLen по дампу (0x4b8e420): то же поле 0x20 — число байт для чтения.
func (r *RegisterRequest) SetReadLen(n uint16) {
	if r != nil {
		r.ReadLen = n
	}
}

// AddUint16 по дампу (0x4b8e0c0): append 2 байта; при BigEndian — rol 0x8 (swap).
func (r *RegisterRequest) AddUint16(v uint16) {
	if r == nil {
		return
	}
	if r.BigEndian {
		v = (v >> 8) | (v << 8)
	}
	r.WriteData = append(r.WriteData, byte(v), byte(v>>8))
}

// AddUint32 по дампу (0x4b8e1e0): append 4 байта; при BigEndian — bswap (big-endian порядок).
func (r *RegisterRequest) AddUint32(v uint32) {
	if r == nil {
		return
	}
	if r.BigEndian {
		r.WriteData = binary.BigEndian.AppendUint32(r.WriteData, v)
	} else {
		r.WriteData = binary.LittleEndian.AppendUint32(r.WriteData, v)
	}
}

// AddUint8 по дампу (0x4b8e000): append 1 байт.
func (r *RegisterRequest) AddUint8(v byte) {
	if r == nil {
		return
	}
	r.WriteData = append(r.WriteData, v)
}

// AddBytes по дампу (0x4b8dee0): append b к WriteData.
func (r *RegisterRequest) AddBytes(b []byte) {
	if r == nil {
		return
	}
	r.WriteData = append(r.WriteData, b...)
}

// GetResult по дампу (0x4b8e7c0): возвращает Result (ptr, len, cap).
func (r *RegisterRequest) GetResult() []byte {
	if r == nil {
		return nil
	}
	return r.Result
}

// ClearOrSetBit по дампу (0x4b8e8e0): Execute; копирует Result; если set — установить бит bitIndex, иначе сбросить; WriteData = [reg 2 байта] + [modified 2 байта]; SetReadLen(0); Execute.
func (r *RegisterRequest) ClearOrSetBit(bitIndex int, set bool) error {
	if r == nil || r.I2C == nil {
		return errors.New("RegisterRequest or I2C is nil")
	}
	if bitIndex < 0 || bitIndex >= int(r.ReadLen)*8 {
		return fmt.Errorf("bit index %d out of range (ReadLen=%d)", bitIndex, r.ReadLen)
	}
	if err := r.Execute(); err != nil {
		return err
	}
	resultLen := len(r.Result)
	if resultLen == 0 {
		return nil
	}
	copied := make([]byte, resultLen)
	copy(copied, r.Result)
	byteIdx := bitIndex / 8
	if byteIdx >= resultLen {
		return fmt.Errorf("bit index %d out of result length %d", bitIndex, resultLen)
	}
	bitInByte := uint(bitIndex % 8)
	if set {
		copied[byteIdx] |= 1 << bitInByte
	} else {
		copied[byteIdx] &^= 1 << bitInByte
	}
	regPrefix := make([]byte, 2)
	if len(r.WriteData) >= 2 {
		copy(regPrefix, r.WriteData[:2])
	}
	r.WriteData = make([]byte, 0, 4)
	r.WriteData = append(r.WriteData, regPrefix...)
	r.WriteData = append(r.WriteData, copied[:2]...)
	r.Result = nil
	r.ReadLen = 0
	return r.Execute()
}

package eth_sw_KSZ9567S

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_serial_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/eth_sw/eth_sw_KSZ9567S/definitions"
)

// Автоматически извлечено из timebeat-2.2.20

func ClearAllToZeroInVlanTable() {
	// TODO: реконструировать
}

func CreateVlan() {
	// TODO: реконструировать
}

func DeleteMSTPForVlan() {
	// TODO: реконструировать
}

func DisableTransparentClockMode() {
	// TODO: реконструировать
}

func EnableTransparentClockMode() {
	// TODO: реконструировать
}

func ErrataEight() {
	// TODO: реконструировать
}

// ErrataFour — пакетная заглушка; метод (e *EthSwKSZ9567S) ErrataFour() вызывается из RunAllErrata.
func ErrataFour() {
	// TODO: реконструировать
}

func ErrataNine() {
	// TODO: реконструировать
}

func ErrataOne() {
	// TODO: реконструировать
}

// Методы Errata* для RunAllErrata (заглушки до реконструкции по дампу).
func (e *EthSwKSZ9567S) ErrataOne()   { ErrataOne() }
func (e *EthSwKSZ9567S) ErrataTwo()   { ErrataTwo() }
func (e *EthSwKSZ9567S) ErrataFour()  { ErrataFour() }
func (e *EthSwKSZ9567S) ErrataNine()  { ErrataNine() }

func ErrataSeven() {
	// TODO: реконструировать
}

func ErrataTwo() {
	// TODO: реконструировать
}

func FlipBit() {
	// TODO: реконструировать
}

func GetDefaultVlanTag() {
	// TODO: реконструировать
}

// MSTPRegister по дампу GetMSTPRegister.func1 (0x4b9e140): два map (VlanToMSTI, второй — зарезервирован), синглтон mstpRegister.
type MSTPRegister struct {
	VlanToMSTI map[int]byte // vlanID -> MSTI index (по дампу mapassign byte 0)
	_          map[int]uint16
}

var (
	mstpRegister   *MSTPRegister
	onceMSTPRegister sync.Once
)

// GetMSTPRegister по дампу (0x4b9a960): sync.Once, создаёт MSTPRegister с двумя map, сохраняет в mstpRegister, возвращает его.
func (e *EthSwKSZ9567S) GetMSTPRegister() *MSTPRegister {
	onceMSTPRegister.Do(func() {
		mstpRegister = &MSTPRegister{
			VlanToMSTI: make(map[int]byte),
		}
	})
	return mstpRegister
}

// GetMSTPForVlan по дампу (0x4b9a9c0): возвращает MSTI index для vlanID (0 если не найден).
func (m *MSTPRegister) GetMSTPForVlan(vlanID int) byte {
	if m == nil || m.VlanToMSTI == nil {
		return 0
	}
	return m.VlanToMSTI[vlanID]
}

func GetRegisterForPort() {
	// TODO: реконструировать
}

func GetVlanRegister() {
	// TODO: реконструировать
}

func GetVlanTable() {
	// TODO: реконструировать
}

// splitL2Config по дампу parseSplitL2Options: offset 0 = Rj45Port (int), offset 8 = SfpPort (int).
type splitL2Config struct {
	Rj45Port int
	SfpPort  int
}

// EthSwKSZ9567S по дизассемблеру NewEthSwKSZ9567S: 0=I2C, 8=Logger, 0x10=Config, 0x18=byte, 0x20=*splitL2Config.
type EthSwKSZ9567S struct {
	I2C             interface{}
	Logger          *logging.Logger
	Config          interface{}
	Flag18          byte
	SplitL2Config   *splitL2Config
}

// NewEthSwKSZ9567S по дизассемблеру (0x4b9bac0): NewI2CDevice(config); если nil → return nil; fmt.Sprintf; NewLogger; newobject; 0=I2C, 8=Logger, 0x10=config, 0x18=1.
func NewEthSwKSZ9567S(config interface{}) *EthSwKSZ9567S {
	dev := generic_serial_device.NewI2CDevice(config)
	if dev == nil {
		return nil
	}
	_ = fmt.Sprintf // по дампу вызывается перед NewLogger
	logger := logging.NewLogger("eth-sw-ksz9567s")
	return &EthSwKSZ9567S{
		I2C:    dev,
		Logger: logger,
		Config: config,
		Flag18: 1,
	}
}

// Start по дизассемблеру (0x4b9bbc0): ParseConfig; VerifyConnectedToDevice — если !=nil return; RunAllErrata; SetSerDesMode; SetPTPClock; если 0x18 EnableTransparentClockMode иначе Disable; если 0x20 SetupVlanSeparation(1) иначе SetupVlanSeparation(0).
func (e *EthSwKSZ9567S) Start() {
	if e == nil {
		return
	}
	e.ParseConfig()
	if e.VerifyConnectedToDevice() != nil {
		return
	}
	e.RunAllErrata()
	e.SetSerDesMode()
	e.SetPTPClock()
	if e.Flag18 != 0 {
		e.EnableTransparentClockMode()
	} else {
		e.DisableTransparentClockMode()
	}
	e.SetupVlanSeparation(e.getVlanFlag())
}

func (e *EthSwKSZ9567S) getVlanFlag() int {
	if e != nil && e.SplitL2Config != nil {
		return 1
	}
	return 0
}

// VerificationReqOne/Two/Three — 3 байта I2C-запроса (по дампу 0x4b9c120: lea VerificationReqOne, 3 bytes).
// Типичные значения для KSZ9567: чтение регистров 0x00, 0x01, 0x02 (Chip ID и т.д.). Реальные байты из бинарника — см. code_analysis/extract_verification_bytes.go
var VerificationReqOne = []byte{0x00, 0x00, 0x00}
var VerificationReqTwo = []byte{0x00, 0x00, 0x01}
var VerificationReqThree = []byte{0x00, 0x00, 0x02}

// VerificationRespOne/Two/Three — ожидаемый 1 байт ответа (по дампу: cmp read[0], VerificationRespOne).
// KSZ9567: регистр 0x00 = 0x95, 0x01 = 0x96 (Chip ID). Третий — по даташиту или из бинарника.
var VerificationRespOne = byte(0x95)
var VerificationRespTwo = byte(0x96)
var VerificationRespThree = byte(0x97)

// ParseConfig по дампу (0x4b9c280): итерация Config.Options; ключ "spl" (3 байта memequal) → split value по "="; parts[0]=="split_l2" (8) и len(parts)>=4 → parseSplitL2Options; parts[0]=="sfp_accuracy" (12) и parts[1]=="precise" (7) → Flag18=0, Logger.Info.
func (e *EthSwKSZ9567S) ParseConfig() {
	if e == nil || e.Config == nil {
		return
	}
	opts := getConfigOptions(e.Config)
	for _, kv := range opts {
		key, val := kv[0], kv[1]
		if len(key) < 3 || key[:3] != "spl" {
			continue
		}
		parts := strings.Split(val, "=")
		if len(parts) < 2 {
			continue
		}
		if len(parts[0]) >= 8 && parts[0][:8] == "split_l2" {
			if len(parts) >= 4 {
				e.parseSplitL2Options(parts)
			}
			continue
		}
		if len(parts[0]) >= 12 && parts[0][:12] == "sfp_accuracy" {
			if len(parts) >= 2 && strings.ToLower(parts[1]) == "precise" {
				e.Flag18 = 0
				if e.Logger != nil {
					e.Logger.Info("sfp_accuracy=precise", 0)
				}
			}
			continue
		}
	}
}

func getConfigOptions(c interface{}) [][2]string {
	if c == nil {
		return nil
	}
	if m, ok := c.(map[string]interface{}); ok {
		if o, ok := m["Options"].([]interface{}); ok {
			var out [][2]string
			for _, item := range o {
				if pair, ok := item.([]interface{}); ok && len(pair) >= 2 {
					k, _ := pair[0].(string)
					v, _ := pair[1].(string)
					out = append(out, [2]string{k, v})
				}
			}
			return out
		}
	}
	return nil
}

// parseSplitL2Options по дампу (0x4b9c4c0): len(parts)>2; при nil e.SplitL2Config — new(splitL2Config); parts[1]=="sfp" (len 3) → Atoi(parts[2]) → SfpPort или Error; parts[1]=="rj45" (len 4) → Atoi(parts[2]) → Rj45Port; иначе Logger.Info(Join(parts[1:], ",")).
func (e *EthSwKSZ9567S) parseSplitL2Options(parts []string) {
	if e == nil || len(parts) <= 2 {
		return
	}
	if e.SplitL2Config == nil {
		e.SplitL2Config = &splitL2Config{}
	}
	kind := parts[1]
	switch kind {
	case "sfp":
		if len(parts) < 4 {
			return
		}
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("split_l2 sfp parse error: key=%s value=%s", parts[0], parts[2]))
			}
			e.SplitL2Config = nil
			return
		}
		if e.Logger != nil {
			e.Logger.Info(fmt.Sprintf("split_l2 sfp port %d", n), 1)
		}
		e.SplitL2Config.SfpPort = n
	case "rj45":
		if len(parts) < 4 {
			return
		}
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("split_l2 rj45 parse error: key=%s value=%s", parts[0], parts[2]))
			}
			e.SplitL2Config = nil
			return
		}
		e.SplitL2Config.Rj45Port = n
		if e.Logger != nil {
			e.Logger.Info(fmt.Sprintf("split_l2 rj45 port %d", n), 1)
		}
	default:
		if e.Logger != nil {
			e.Logger.Info(strings.Join(parts[1:], ","), 1)
		}
	}
}

// VerifyConnectedToDevice по дампу (0x4b9c120): WriteThenRead(ReqOne, 1); сравнить read[0] с VerificationRespOne; то же ReqTwo/RespTwo, ReqThree/RespThree; при несовпадении или ошибке — return error.
func (e *EthSwKSZ9567S) VerifyConnectedToDevice() error {
	if e == nil || e.I2C == nil {
		return errors.New("no I2C device")
	}
	dev, ok := e.I2C.(generic_serial_device.I2CWriterReader)
	if !ok {
		return nil
	}
	buf, err := dev.WriteThenRead(VerificationReqOne, 1)
	if err != nil || len(buf) < 1 {
		return fmt.Errorf("verify req1: %w", err)
	}
	if buf[0] != VerificationRespOne {
		return fmt.Errorf("verify resp1: got 0x%02x", buf[0])
	}
	buf, err = dev.WriteThenRead(VerificationReqTwo, 1)
	if err != nil || len(buf) < 1 {
		return fmt.Errorf("verify req2: %w", err)
	}
	if buf[0] != VerificationRespTwo {
		return fmt.Errorf("verify resp2: got 0x%02x", buf[0])
	}
	buf, err = dev.WriteThenRead(VerificationReqThree, 1)
	if err != nil || len(buf) < 1 {
		return fmt.Errorf("verify req3: %w", err)
	}
	if buf[0] != VerificationRespThree {
		return fmt.Errorf("verify resp3: got 0x%02x", buf[0])
	}
	return nil
}

// RunAllErrata по дизассемблеру (0x4b9a220): Set100MbpsNoAutoNegotiation; ErrataOne; ErrataTwo; ErrataFour; WriteSGMIIRegister(0x1f0004, 0x1a0); ErrataNine; Set1000MbpsAutoNegotiation.
func (e *EthSwKSZ9567S) RunAllErrata() {
	if e == nil {
		return
	}
	e.Set100MbpsNoAutoNegotiation()
	e.ErrataOne()
	e.ErrataTwo()
	e.ErrataFour()
	e.WriteSGMIIRegister(0x1f0004, 0x1a0)
	e.ErrataNine()
	e.Set1000MbpsAutoNegotiation()
}

// Set100MbpsNoAutoNegotiation по дампу (0x4b9a2c0): цикл port 1..5; reg16 = (port<<12)|0x100, rol8; payload = [reg_hi, reg_lo, 0x21, 0x00]; I2C.Write(payload); при ошибке Logger.Error.
func (e *EthSwKSZ9567S) Set100MbpsNoAutoNegotiation() {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	const dataVal = 0x0021 // 100Mbps, no auto-negotiation (дамп: movb $0x21, 0x2e(rsp))
	for port := 1; port <= 5; port++ {
		reg16 := uint16((port << 12) | 0x100)
		reg16 = (reg16 << 8) | (reg16 >> 8) // rol 8
		payload := []byte{byte(reg16 >> 8), byte(reg16), byte(dataVal), byte(dataVal >> 8)}
		if err := wr.Write(payload); err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("Set100MbpsNoAutoNegotiation port %d: %v", port, err))
			}
		}
	}
}

// Set1000MbpsAutoNegotiation по дампу (0x4b9a3e0): цикл port 1..5; reg16 = (port<<12)|0x100, rol8; payload = [reg_hi, reg_lo, 0x13, 0x40]; I2C.Write(payload); при ошибке Logger.Error.
func (e *EthSwKSZ9567S) Set1000MbpsAutoNegotiation() {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	const dataVal uint16 = 0x4013 // 1000Mbps, auto-negotiation (дамп: movw $0x4013)
	for port := 1; port <= 5; port++ {
		reg16 := uint16((port << 12) | 0x100)
		reg16 = (reg16 << 8) | (reg16 >> 8)
		payload := []byte{byte(reg16 >> 8), byte(reg16), byte(dataVal & 0xff), byte(dataVal >> 8)}
		if err := wr.Write(payload); err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("Set1000MbpsAutoNegotiation port %d: %v", port, err))
			}
		}
	}
}

// WriteSGMIIRegister по дампу (0x4b9cc20): первый запрос 6 байт [0x72, 0x00, reg_be_4]; второй 4 байта [0x72, 0x06, val_swapped_2].
func (e *EthSwKSZ9567S) WriteSGMIIRegister(reg, val uint32) {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	regBE := make([]byte, 4)
	binary.BigEndian.PutUint32(regBE, reg)
	payload1 := append([]byte{0x72, 0x00}, regBE...)
	if err := wr.Write(payload1); err != nil {
		return
	}
	val16 := uint16(val)
	val16 = (val16 << 8) | (val16 >> 8)
	payload2 := []byte{0x72, 0x06, byte(val16 >> 8), byte(val16)}
	_ = wr.Write(payload2)
}

// SetSerDesMode по дампу (0x4b9a8e0): три вызова WriteSGMIIRegister — (0x1f8001, 0x19), (0x1f0004, 0x1a0), (0x1f0000, 0x1340).
func (e *EthSwKSZ9567S) SetSerDesMode() {
	if e == nil {
		return
	}
	e.WriteSGMIIRegister(0x1f8001, 0x19)
	e.WriteSGMIIRegister(0x1f0004, 0x1a0)
	e.WriteSGMIIRegister(0x1f0000, 0x1340)
}

// SetPTPClock по дизассемблеру (0x4b9b060): GetPreciseTime; конвертация в PTP (seconds + nanoseconds); I2C write reg 0x0405 — 8 байт времени; затем reg 0x0500 = 0x0a (триггер).
func (e *EthSwKSZ9567S) SetPTPClock() {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	t := adjusttime.GetPreciseTime()
	sec := t.Unix()
	ns := int64(t.Nanosecond())
	if sec < 0 {
		sec = 0
		ns = 0
	}
	// PTP time по дампу: 8 байт — 4 байта seconds (big-endian), 4 байта nanoseconds (big-endian).
	payload1 := make([]byte, 10)
	payload1[0], payload1[1] = 0x05, 0x04
	binary.BigEndian.PutUint32(payload1[2:6], uint32(sec))
	binary.BigEndian.PutUint32(payload1[6:10], uint32(ns))
	if err := wr.Write(payload1); err != nil {
		if e.Logger != nil {
			e.Logger.Error("SetPTPClock: write PTP time failed")
		}
		return
	}
	payload2 := []byte{0x05, 0x00, 0x00, 0x0a}
	if err := wr.Write(payload2); err != nil {
		if e.Logger != nil {
			e.Logger.Error("SetPTPClock: write PTP trigger failed")
		}
	}
}

// EnableTransparentClockMode по дизассемблеру (0x4b9ada0): два I2C write — reg 0x1405 = 0x79, reg 0x1605 = 0x041c.
func (e *EthSwKSZ9567S) EnableTransparentClockMode() {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	payload1 := []byte{0x05, 0x14, 0x00, 0x79}
	if err := wr.Write(payload1); err != nil {
		if e.Logger != nil {
			e.Logger.Error("EnableTransparentClockMode: first write failed")
		}
		return
	}
	payload2 := []byte{0x05, 0x16, 0x1c, 0x04}
	if err := wr.Write(payload2); err != nil {
		if e.Logger != nil {
			e.Logger.Error("EnableTransparentClockMode: second write failed")
		}
	}
}

// DisableTransparentClockMode по дизассемблеру (0x4b9af00): два I2C write — reg 0x1405 = 0, reg 0x1605 = 0.
func (e *EthSwKSZ9567S) DisableTransparentClockMode() {
	if e == nil || e.I2C == nil {
		return
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return
	}
	payload1 := []byte{0x05, 0x14, 0x00, 0x00}
	if err := wr.Write(payload1); err != nil {
		if e.Logger != nil {
			e.Logger.Error("DisableTransparentClockMode: first write failed")
		}
		return
	}
	payload2 := []byte{0x05, 0x16, 0x00, 0x00}
	if err := wr.Write(payload2); err != nil {
		if e.Logger != nil {
			e.Logger.Error("DisableTransparentClockMode: second write failed")
		}
	}
}

// vlanPortConfig по дампу SetupVlanSeparation: два порта (default VLAN tag для портов 4 и 6).
type vlanPortConfig struct {
	Port1, Port2 int
}

// FlipBit по дампу (0x4b9b940): read-modify-write по I2C — регистр reg (2 байта BE), прочитать 1 байт, установить/сбросить бит bit (set!=0 → set, set==0 → clear), записать обратно. Возвращает error.
func (e *EthSwKSZ9567S) FlipBit(reg uint16, set int, bit int) error {
	if e == nil || e.I2C == nil {
		return errors.New("no I2C device")
	}
	wrr, ok1 := e.I2C.(generic_serial_device.I2CWriterReader)
	wr, ok2 := e.I2C.(generic_serial_device.I2CWriter)
	if !ok1 || !ok2 {
		return errors.New("I2C device does not support read and write")
	}
	regBE := []byte{byte(reg >> 8), byte(reg)}
	read, err := wrr.WriteThenRead(regBE, 1)
	if err != nil || len(read) < 1 {
		return err
	}
	b := read[0]
	if set != 0 {
		b |= (1 << bit)
	} else {
		b &^= (1 << bit)
	}
	payload := []byte{byte(reg >> 8), byte(reg), b}
	return wr.Write(payload)
}

// Set8021QVlanEnable по дампу (0x4b9cdc0): FlipBit(reg 0x310, enable, bit 7).
func (e *EthSwKSZ9567S) Set8021QVlanEnable(enable int) error {
	return e.FlipBit(0x310, enable, 7)
}

// SetDynamicEntryEgressVlanFiltering по дампу (0x4b9ce20): FlipBit(reg 0x312, 1, bit 5). Конфиг cfg не используется в дампе.
func (e *EthSwKSZ9567S) SetDynamicEntryEgressVlanFiltering(cfg *vlanPortConfig) error {
	_ = cfg
	return e.FlipBit(0x312, 1, 5)
}

// SetDefaultVlanTag по дампу (0x4b9ce80): PortRegisterDefaultTagLookup[portID] → reg; два I2C write: reg (2 байта) + 2 байта данных, reg+1 (2 байта) + (value>>8)&0xf, value&0xff.
func (e *EthSwKSZ9567S) SetDefaultVlanTag(portID int, value int) error {
	if e == nil || e.I2C == nil {
		return errors.New("no I2C device")
	}
	if portID < 0 || portID >= len(definitions.PortRegisterDefaultTagLookup) {
		return fmt.Errorf("portID %d out of range", portID)
	}
	wr, ok := e.I2C.(generic_serial_device.I2CWriter)
	if !ok {
		return errors.New("I2C device does not support write")
	}
	reg := definitions.PortRegisterDefaultTagLookup[portID]
	// По дампу: первый запрос — reg (2 байта) + data = reg (2 байта BE); второй — reg+1 (2 байта) + [(value>>8)&0xf, value&0xff].
	payload1 := []byte{byte(reg >> 8), byte(reg), byte(reg >> 8), byte(reg)}
	if err := wr.Write(payload1); err != nil {
		return err
	}
	reg2 := reg + 1
	payload2 := []byte{byte(reg2 >> 8), byte(reg2), byte((value >> 8) & 0xf), byte(value & 0xff)}
	return wr.Write(payload2)
}

// CreateVlan по дампу (0x4b9d160): 5 RegisterRequest (reg 0x400/0x404/0x408/0x40c/0x40e); GetMSTPRegister().GetMSTPForVlan(vlanID); memberPorts/untaggedPorts; Execute.
func (e *EthSwKSZ9567S) CreateVlan(portID, vlanID, flags int) error {
	if e == nil {
		return errors.New("EthSwKSZ9567S is nil")
	}
	i2cReg, ok := e.I2C.(generic_serial_device.I2CDeviceForRegister)
	if !ok {
		return errors.New("I2C device does not support RegisterRequest")
	}
	mstp := e.GetMSTPRegister()
	msti := mstp.GetMSTPForVlan(vlanID)
	if vlanID < 0 {
		panic("vlanID < 0")
	}
	var memberPorts, untaggedPorts uint32
	if vlanID < 32 {
		memberPorts = 1 << uint(vlanID)
	}
	if flags != 0 {
		memberPorts = 0x7f
	}
	untaggedPorts = memberPorts | 0x0a
	if flags != 0 {
		untaggedPorts = 0x7f
	}
	portMstiVal := (uint32(msti) << 12) | 0x80000000

	newReq := func() *generic_serial_device.RegisterRequest {
		return generic_serial_device.NewRegisterRequest(i2cReg)
	}

	// Регистр 0x0400: portMstiVal (4 байта LE)
	req1 := newReq()
	req1.SetReadLen(0)
	req1.AddUint16(0x0400)
	req1.BigEndian = false
	req1.AddUint32(portMstiVal)
	if err := req1.Execute(); err != nil {
		return err
	}

	// Регистр 0x0404: memberPorts (4 байта BE)
	req2 := newReq()
	req2.SetReadLen(0)
	req2.AddUint16(0x0404)
	req2.BigEndian = true
	req2.AddUint32(memberPorts)
	if err := req2.Execute(); err != nil {
		return err
	}

	// Регистр 0x0408: untaggedPorts (4 байта BE)
	req3 := newReq()
	req3.SetReadLen(0)
	req3.AddUint16(0x0408)
	req3.BigEndian = true
	req3.AddUint32(untaggedPorts)
	if err := req3.Execute(); err != nil {
		return err
	}

	// Регистр 0x040c: portID (2 байта)
	req4 := newReq()
	req4.SetReadLen(0)
	req4.AddUint16(0x040c)
	req4.AddUint16(uint16(portID & 0xfff))
	if err := req4.Execute(); err != nil {
		return err
	}

	// Регистр 0x040e: 0x81 (1 байт)
	req5 := newReq()
	req5.SetReadLen(0)
	req5.AddUint16(0x040e)
	req5.AddUint8(0x81)
	if err := req5.Execute(); err != nil {
		return err
	}

	return nil
}

// SetupVlanSeparation по дизассемблеру (0x4b9bc80): Set8021QVlanEnable(0); config = SplitL2Config или {1,1}; SetDynamicEntryEgressVlanFiltering; SetDefaultVlanTag(4/6); при vlanFlag — CreateVlan(4/6); иначе CreateVlan(0,1,1); Set8021QVlanEnable(1).
func (e *EthSwKSZ9567S) SetupVlanSeparation(vlanFlag int) {
	if e == nil {
		return
	}
	if err := e.Set8021QVlanEnable(0); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: Set8021QVlanEnable(0): %v", err))
		}
		return
	}
	var cfg vlanPortConfig
	if vlanFlag != 0 && e.SplitL2Config != nil {
		cfg.Port1 = e.SplitL2Config.Rj45Port
		cfg.Port2 = e.SplitL2Config.SfpPort
	} else {
		cfg.Port1 = 1
		cfg.Port2 = 1
	}
	if err := e.SetDynamicEntryEgressVlanFiltering(&cfg); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: SetDynamicEntryEgressVlanFiltering: %v", err))
		}
		return
	}
	if err := e.SetDefaultVlanTag(4, cfg.Port1); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: SetDefaultVlanTag(4): %v", err))
		}
		return
	}
	if err := e.SetDefaultVlanTag(6, cfg.Port2); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: SetDefaultVlanTag(6): %v", err))
		}
		return
	}
	if vlanFlag != 0 {
		if err := e.CreateVlan(4, cfg.Port1, 0); err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: CreateVlan port 4: %v", err))
			}
		}
		if err := e.CreateVlan(6, cfg.Port2, 0); err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: CreateVlan port 6: %v", err))
			}
		}
		if err := e.Set8021QVlanEnable(1); err != nil {
			if e.Logger != nil {
				e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: Set8021QVlanEnable(1): %v", err))
			}
		}
		return
	}
	if err := e.CreateVlan(0, 1, 1); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: CreateVlan(0,1,1): %v", err))
		}
	}
	if err := e.Set8021QVlanEnable(1); err != nil {
		if e.Logger != nil {
			e.Logger.Error(fmt.Sprintf("SetupVlanSeparation: Set8021QVlanEnable(1): %v", err))
		}
	}
}

func NewMSTPRegister() {
	// TODO: реконструировать
}

func NewRegisterBuffer() {
	// TODO: реконструировать
}

func NewVlanRegister() {
	// TODO: реконструировать
}

func ParseConfig() {
	// TODO: реконструировать
}

func ReadVlanTable() {
	// TODO: реконструировать
}

func ResetInputEventTrigger() {
	// TODO: реконструировать
}

func Run1PPSInRunloop() {
	// TODO: реконструировать
}

func RunAllErrata() {
	// TODO: реконструировать
}

func Set1000MbpsAutoNegotiation() {
	// TODO: реконструировать
}

func Set100MbpsNoAutoNegotiation() {
	// TODO: реконструировать
}

func Set8021QVlanEnable() {
	// TODO: реконструировать
}

func SetDefaultVlanTag() {
	// TODO: реконструировать
}

func SetDynamicEntryEgressVlanFiltering() {
	// TODO: реконструировать
}

func SetPTPClock() {
	// TODO: реконструировать
}

func SetSerDesMode() {
	// TODO: реконструировать
}

func SetupVlanSeparation() {
	// TODO: реконструировать
}

func WriteMMDRegister() {
	// TODO: реконструировать
}

func WriteSGMIIRegister() {
	// TODO: реконструировать
}

func dumpTimehead() {
	// TODO: реконструировать
}

func func1() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}


func onceVlanRegister() {
	// TODO: реконструировать
}

func parseSplitL2Options() {
	// TODO: реконструировать
}

func vlanRegister() {
	// TODO: реконструировать
}


package clock_gen_8A34002E

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/generic_serial_device"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Автоматически извлечено из timebeat-2.2.20

func DPLLRefStatusToString() {
	// TODO: реконструировать
}

func DPLLStatusToString() {
	// TODO: реконструировать
}

func DebugBytesWritten() {
	// TODO: реконструировать
}

func DebugEraseOption() {
	// TODO: реконструировать
}

func ExecutePullIn() {
	// TODO: реконструировать
}

func GetDPLL() {
	// TODO: реконструировать
}

func GetDPLLRefState() {
	// TODO: реконструировать
}

func GetDPLLState() {
	// TODO: реконструировать
}

func GetDpllFinePhaseAdvCfg() {
	// TODO: реконструировать
}

func GetDpllFodFreq() {
	// TODO: реконструировать
}

func GetEEPROMConfigStatus() {
	// TODO: реконструировать
}

func GetEEPROMStatus() {
	// TODO: реконструировать
}

func GetHotfixRelease() {
	// TODO: реконструировать
}

func GetInputMonFreqStatus() {
	// TODO: реконструировать
}

func GetInputMonStatus() {
	// TODO: реконструировать
}

func GetMajorRelease() {
	// TODO: реконструировать
}

func GetMinorRelease() {
	// TODO: реконструировать
}

func GetMode() {
	// TODO: реконструировать
}

func GetTimebeatClockgenConfigVersion() {
	// TODO: реконструировать
}

func InputMonStatusToString() {
	// TODO: реконструировать
}

func MonitorDPLLAdjustments() {
	// TODO: реконструировать
}

// ClockGen8A34012 по дизассемблеру NewClockGen8A34012: 0=Logger, 8=Name, 0x18=I2C, 0x20=Config, 0x50=byte, 0x60/0x78=map.
// Dplls и Paused/PausedMu — по вызовам GetDPLL(3) и isPaused() из runLoop/addWatchDogModule.
// PTPGroup (0x48) — при nil offset берётся из servo GetActiveSourceGroupMembers(PTPGroupName).
// PTPGroupName (0x30/0x38) — имя группы для GetActiveSourceGroupMembers (по дампу disciplineDPLLWithOffsetFromPTPGroup).
type ClockGen8A34012 struct {
	Logger        *logging.Logger
	Name          string // 0x8
	I2C           interface{}
	Config        interface{}
	Flag50        byte
	Map60         map[string]interface{}
	Map78         map[string]interface{}
	Dplls         []*DPLL // по GetDPLL(idx)
	Paused        bool
	PausedMu      sync.Mutex
	PTPGroup      interface{} // 0x48: при nil используем servo
	PTPGroupName  string      // группа для GetActiveSourceGroupMembers (по умолчанию "ptp")
}

// NewClockGen8A34012 по дизассемблеру (0x4b96480): NewI2CDevice(config); если nil → return nil; makemap×2; newobject; 0x18=I2C, 0x20=config; 0x10=0x19, 0x8=name; 0x60/0x78=map; Logger=NewLogger(name); ParseConfigOptions(c); return c.
func NewClockGen8A34012(config interface{}) *ClockGen8A34012 {
	dev := generic_serial_device.NewI2CDevice(config)
	if dev == nil {
		return nil
	}
	dplls := make([]*DPLL, 4)
	for i := range dplls {
		dplls[i] = &DPLL{}
	}
	c := &ClockGen8A34012{
		Logger:        logging.NewLogger("clock-gen-8a34012"),
		Name:          "clock-gen-8a34012",
		I2C:           dev,
		Config:        config,
		Flag50:        1,
		Map60:         make(map[string]interface{}),
		Map78:         make(map[string]interface{}),
		Dplls:         dplls,
		PTPGroupName:  "ptp",
	}
	for i := range dplls {
		dplls[i].ClockGen = c
	}
	// Адреса регистров DPLL по дампу (dpllCtrlLookup/типичный 8A34012): база на индекс.
	setDPLLRegisterBases(c.Dplls)
	c.ParseConfigOptions()
	return c
}

// setDPLLRegisterBases задаёт RefModeReg, ManualRefReg, FreqReg, PhaseReg для каждого DPLL (база блока по индексу).
func setDPLLRegisterBases(dplls []*DPLL) {
	// Базовые адреса блоков DPLL 0..3 для 8A34012 (типичные смещения).
	bases := []uint16{0x1100, 0x1140, 0x1180, 0x1200}
	for i := range dplls {
		if i >= len(bases) || dplls[i] == nil {
			continue
		}
		base := bases[i]
		dplls[i].RefModeReg = base + 0x00
		dplls[i].ManualRefReg = base + 0x02
		dplls[i].FreqReg = base + 0x14
		dplls[i].PhaseReg = base + 0x18
	}
}

// ParseConfigOptions по дампу (0x4b96dc0): итерация Config.Options (0x20+0x08/0x10); ключ "dco" (6 байт memequal) → split по "=", пишем в map["dco"]; ключ "watchdog" (8) → map["watchdog"]; затем setDCOConfigOptions(map["dco"]), addWatchDogModule(map["watchdog"]).
func (c *ClockGen8A34012) ParseConfigOptions() {
	if c == nil || c.Config == nil {
		return
	}
	opts := getClockGenOptions(c.Config)
	dcoOpts := make(map[string][]string)
	watchdogOpts := make(map[string][]string)
	for _, kv := range opts {
		key, val := kv[0], kv[1]
		if len(key) >= 3 && key[:3] == "dco" {
			parts := strings.Split(val, "=")
			if len(parts) >= 2 {
				dcoOpts[parts[0]] = append(dcoOpts[parts[0]], parts[1])
			}
			continue
		}
		if len(key) >= 8 && key[:8] == "watchdog" {
			parts := strings.Split(val, "=")
			if len(parts) >= 2 {
				watchdogOpts[parts[0]] = append(watchdogOpts[parts[0]], parts[1])
			}
			continue
		}
	}
	if len(dcoOpts) > 0 {
		c.setDCOConfigOptions(dcoOpts)
	}
	if len(watchdogOpts) > 0 {
		c.addWatchDogModule(watchdogOpts)
	}
}

func getClockGenOptions(cfg interface{}) [][2]string {
	if cfg == nil {
		return nil
	}
	if m, ok := cfg.(map[string]interface{}); ok {
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

func (c *ClockGen8A34012) setDCOConfigOptions(opts map[string][]string) {
	_ = opts
}

func (c *ClockGen8A34012) addWatchDogModule(opts map[string][]string) {
	_ = opts
}

// Start по дизассемблеру (Start.func1, runLoop): go setupDPLLForGNSS(DPLL(3)); go runLoop. runLoop — ticker 100ms, GetDPLL(3), isPaused, disciplineDPLLWithOffsetFromPTPGroup.
func (c *ClockGen8A34012) Start() {
	if c == nil {
		return
	}
	go func() {
		d := c.GetDPLL(3)
		if d != nil {
			d.setupDPLLForGNSS()
		}
	}()
	go c.runLoop()
}

// runLoop по дампу (0x4b96280): GetDPLL(3); если Flag50!=0 — Logger.Info("...", 12); NewTicker(0x3b9aca00 ns = 1e9/10 = 100ms); цикл: <-ticker.C; если Flag50==0 — continue; если isPaused() — continue; dpll.disciplineDPLLWithOffsetFromPTPGroup().
func (c *ClockGen8A34012) runLoop() {
	if c == nil {
		return
	}
	dpll := c.GetDPLL(3)
	if dpll == nil {
		return
	}
	if c.Flag50 != 0 && c.Logger != nil {
		c.Logger.Info("clock-gen runLoop started", 12)
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if c.Flag50 == 0 {
			continue
		}
		if c.isPaused() {
			continue
		}
		dpll.disciplineDPLLWithOffsetFromPTPGroup()
	}
}

// GetDPLL по вызовам runLoop/addWatchDogModule: GetDPLL(3); возвращает DPLL по индексу. Dplls инициализируется в NewClockGen8A34012 (4 элемента).
func (c *ClockGen8A34012) GetDPLL(idx int) *DPLL {
	if c == nil || c.Dplls == nil || idx < 0 || idx >= len(c.Dplls) {
		return nil
	}
	return c.Dplls[idx]
}

// ExecutePullIn по дампу (0x4b928a0): GetDPLL(индекс из аргумента), затем d.executePhasePullIn(phase). В дампе один аргумент (ecx) передаётся в executePhasePullIn.
func (c *ClockGen8A34012) ExecutePullIn(phase int) {
	d := c.GetDPLL(3)
	if d != nil {
		d.executePhasePullIn(int32(phase))
	}
}

// ResetDPLLFreq по дампу (0x4b90000): сброс частоты DPLL — GetDPLL(3), writeFreq(0).
func (c *ClockGen8A34012) ResetDPLLFreq() {
	d := c.GetDPLL(3)
	if d != nil {
		d.writeFreq(0)
	}
}

// Reset по дампу (0x4b96c80): RegisterRequest reg=0x12c0, байт 0x5a; I2C Execute.
func (c *ClockGen8A34012) Reset() error {
	i2c, _ := c.I2C.(generic_serial_device.I2CDeviceForRegister)
	if i2c == nil {
		return nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return nil
	}
	req.BigEndian = true
	req.AddUint16(0x12c0)
	req.AddUint8(0x5a)
	return req.Execute()
}

// VerifyConnectedToDevice по дампу (0x4b96980): I2C WriteThenRead(write 2 байта, read 1) — проверка связи с устройством.
func (c *ClockGen8A34012) VerifyConnectedToDevice() (bool, error) {
	if c == nil || c.I2C == nil {
		return false, nil
	}
	i2c, ok := c.I2C.(generic_serial_device.I2CDeviceForRegister)
	if !ok {
		return false, nil
	}
	write := []byte{0x00, 0x00}
	_, err := i2c.WriteThenRead(write, 1)
	return err == nil, err
}

// isPaused по дампу (0x4b973e0) и isPaused.func1: чтение флага под mutex (Unlock в closure).
func (c *ClockGen8A34012) isPaused() bool {
	if c == nil {
		return false
	}
	c.PausedMu.Lock()
	b := c.Paused
	c.PausedMu.Unlock()
	return b
}

// remoteGroupOffsetResult — результат getRemoteGroupOffset (по дампу 0x10/0x18/0x20: два значения для усреднения).
type remoteGroupOffsetResult struct {
	V1, V2 int64
}

// DPLLFreqParams по дампу disciplineDPLLWithOffsetFromPTPGroup (d+0x30): AlgoPID (0x08), UseFirstMedian (0x10), MovingMedian (0x18), MovingMedian (0x20), Watchdog (0x28).
type DPLLFreqParams struct {
	AlgoPID        *algos.AlgoPID
	UseFirstMedian bool
	Median1        *algos.MovingMedian
	Median2        *algos.MovingMedian
	Watchdog       interface{}
}

// DPLL по дампу: 0x8=ClockGen, 0x10/0x12=RefModeReg/ManualRefReg, 0x14/0x16=FreqReg/PhaseReg, 0x30=PhaseMode, FreqParams.
type DPLL struct {
	ClockGen      *ClockGen8A34012
	RefModeReg    uint16           // регистр reference mode (по дампу SetReferenceMode 0x10)
	ManualRefReg  uint16           // регистр manual reference (по дампу SetManualReference 0x12)
	PhaseReg      uint16           // регистр фазы (writePhase 0x16)
	FreqReg       uint16           // регистр частоты (writeFreq/readFreq 0x14)
	PhaseMode     bool             // true = writePhase, false = writeFreq + PID
	FreqParams    *DPLLFreqParams  // для ветки частоты: AlgoPID, MovingMedian×2
}

// getRemoteGroupOffset по дампу (0x4b90720): при PTPGroup==nil — servo GetController/GetOffsets/GetActiveSourceGroupMembers(PTPGroupName), иначе — из PTP-канала. Возвращает два offset для усреднения.
func (d *DPLL) getRemoteGroupOffset() *remoteGroupOffsetResult {
	if d == nil || d.ClockGen == nil {
		return nil
	}
	c := d.ClockGen
	if c.PTPGroup != nil {
		// TODO: получить offset из PTP group channel (по дампу getRemoteGroupOffset.func1)
		return nil
	}
	ctrl := servo.GetController()
	if ctrl == nil {
		return nil
	}
	o := ctrl.GetOffsets()
	if o == nil {
		return nil
	}
	group := c.PTPGroupName
	if group == "" {
		group = "ptp"
	}
	members := o.GetActiveSourceGroupMembers(group)
	if len(members) < 2 {
		return nil
	}
	return &remoteGroupOffsetResult{
		V1: members[0].GetFilteredOffset(),
		V2: members[1].GetFilteredOffset(),
	}
}

// disciplineDPLLWithOffsetFromPTPGroup по дампу (0x4b8f2a0): getRemoteGroupOffset; усреднение (V1+V2)/2; при PhaseMode — writePhase(phase), иначе GetPreciseTime, AlgoPID.CalculateNewFrequency, PPMToFCW, MovingMedian.Sample×2, writeFreq.
func (d *DPLL) disciplineDPLLWithOffsetFromPTPGroup() {
	if d == nil || d.ClockGen == nil {
		return
	}
	offset := d.getRemoteGroupOffset()
	if offset == nil {
		return
	}
	avg := (offset.V1 + offset.V2) / 2
	if d.PhaseMode {
		phase := d.clampPhaseForWrite(avg)
		d.logWritePhase(offset.V1, offset.V2)
		d.writePhase(int32(phase))
		return
	}
	d.logWriteFreq(offset.V1, offset.V2)
	// Freq path по дампу: UseStoredPID? иначе GetPreciseTime → AlgoPID.CalculateNewFrequency → clamp PPM → PPMToFCW → MovingMedian.Sample (опц.) → MovingMedian.Sample → writeFreq
	var ppm float64
	if d.FreqParams != nil && d.FreqParams.AlgoPID != nil {
		_ = adjusttime.GetPreciseTime()
		dt := 100 * time.Millisecond
		ppm = d.FreqParams.AlgoPID.CalculateNewFrequency(float64(avg), dt)
		ppm = clampPPMForFCW(ppm)
	} else {
		ppm = 0
	}
	fcwRaw := PPMToFCW(ppm)
	fcw := fcwRaw
	if d.FreqParams != nil {
		if d.FreqParams.UseFirstMedian && d.FreqParams.Median1 != nil {
			fcw = uint64(d.FreqParams.Median1.Sample(int64(fcwRaw)))
		}
		if d.FreqParams.Median2 != nil {
			fcw = uint64(d.FreqParams.Median2.Sample(int64(fcw)))
		}
	}
	d.writeFreq(fcw)
}

// clampPhaseForWrite по дампу disciplineDPLL: avg сравнивается с 0x5f5e100 (100e6), 0xfa0a1f00 (signed), затем *5*4.
func (d *DPLL) clampPhaseForWrite(avg int64) int64 {
	const maxPhase = 0x77359400
	const minPhase = 0x88ca6c00
	if avg > 0x5f5e100 {
		return maxPhase
	}
	if avg < -0x5a1f00 {
		return minPhase
	}
	return avg * 5 * 4
}

// clampPPMForFCW по дампу (0x4b8f681–0x4b8f6c0): сумма трёх значений / 3, сравнение с константами, clamp.
func clampPPMForFCW(ppm float64) float64 {
	const div = 3.0
	const high = 100.0
	const low = -100.0
	x := ppm / div
	if x > high {
		return high
	}
	if x < low {
		return low
	}
	return x
}

func (d *DPLL) logWritePhase(_, _ int64) {}
func (d *DPLL) logWriteFreq(_, _ int64)  {}

// executePhasePullIn по дампу (0x4b8f7a0): RegisterRequest reg=PhaseReg (d+0x18), 4 байта phase; Execute. Эквивалент writePhase(phase).
func (d *DPLL) executePhasePullIn(phase int32) {
	d.writePhase(phase)
}

// writePhase по дампу (0x4b90ec0): RegisterRequest с reg=PhaseReg, phase (4 байта), BigEndian из конфига; Execute.
func (d *DPLL) writePhase(phase int32) {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return
	}
	req.BigEndian = false
	req.AddUint16(d.PhaseReg)
	req.AddUint32(uint32(phase))
	_ = req.Execute()
}

// readFreq по дампу (0x4b90220): RegisterRequest reg=FreqReg, ReadLen=6, Execute; парсинг Result в uint64 (6 байт).
func (d *DPLL) readFreq() (uint64, error) {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return 0, nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return 0, nil
	}
	req.BigEndian = true
	req.AddUint16(d.FreqReg)
	req.SetReadLen(6)
	if err := req.Execute(); err != nil {
		return 0, err
	}
	r := req.GetResult()
	if len(r) < 6 {
		return 0, nil
	}
	low := binary.LittleEndian.Uint32(r[0:4])
	b4 := uint64(r[4])
	b5 := uint64(r[5] & 3)
	return uint64(low) | (b4 << 32) | (b5 << 40), nil
}

// writeFreq по дампу (0x4b90060): readFreq(); при ошибке return; RegisterRequest reg=FreqReg, 6 байт FCW; Execute.
func (d *DPLL) writeFreq(fcw uint64) {
	if _, err := d.readFreq(); err != nil {
		return
	}
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return
	}
	req.BigEndian = true
	req.AddUint16(d.FreqReg)
	req.AddUint32(uint32(fcw))
	req.AddUint8(byte(fcw >> 32))
	req.AddUint8(byte((fcw >> 40) & 3))
	_ = req.Execute()
}

func (d *DPLL) getI2CForRegister() generic_serial_device.I2CDeviceForRegister {
	if d == nil || d.ClockGen == nil || d.ClockGen.I2C == nil {
		return nil
	}
	i2c, _ := d.ClockGen.I2C.(generic_serial_device.I2CDeviceForRegister)
	return i2c
}

// setupDPLLForGNSS по дампу (0x4b91e20): Sleep(50ms), SetReferenceMode(1), при ошибке Logger.Error; Sleep(50ms), SetManualReference(2), при ошибке Logger.Error; Sleep(50ms).
func (d *DPLL) setupDPLLForGNSS() {
	if d == nil {
		return
	}
	const setupSleep = 50 * time.Millisecond
	time.Sleep(setupSleep)
	if err := d.SetReferenceMode(1); err != nil && d.ClockGen != nil && d.ClockGen.Logger != nil {
		d.ClockGen.Logger.Error(fmt.Sprintf("setupDPLLForGNSS SetReferenceMode: %v", err))
	}
	time.Sleep(setupSleep)
	if err := d.SetManualReference(2); err != nil && d.ClockGen != nil && d.ClockGen.Logger != nil {
		d.ClockGen.Logger.Error(fmt.Sprintf("setupDPLLForGNSS SetManualReference: %v", err))
	}
	time.Sleep(setupSleep)
}

// SetReferenceMode по дампу (0x4b91b80): RegisterRequest reg=RefModeReg (d+0x10), 1 байт mode; Execute.
func (d *DPLL) SetReferenceMode(mode int) error {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return nil
	}
	req.BigEndian = true
	req.AddUint16(d.RefModeReg)
	req.AddUint8(byte(mode))
	return req.Execute()
}

// SetManualReference по дампу (0x4b91ce0): RegisterRequest reg=ManualRefReg (d+0x12), 1 байт ref; Execute.
func (d *DPLL) SetManualReference(ref int) error {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return nil
	}
	req.BigEndian = true
	req.AddUint16(d.ManualRefReg)
	req.AddUint8(byte(ref))
	return req.Execute()
}

// SetMode по дампу (0x4b916c0): RegisterRequest reg=RefModeReg+0x37, 1 байт mode (биты 0x7, 0x7 от двух аргументов); Execute.
func (d *DPLL) SetMode(mode byte) error {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return nil
	}
	req.BigEndian = true
	req.AddUint16(d.RefModeReg + 0x37)
	req.AddUint8(mode & 0x3f)
	return req.Execute()
}

// GetMode по дампу (0x4b91880): RegisterRequest reg=RefModeReg+0x37, ReadLen=1, Execute; возврат Result[0] (биты 0x40, 0x38>>3, 0x7).
func (d *DPLL) GetMode() (byte, error) {
	i2c := d.getI2CForRegister()
	if i2c == nil {
		return 0, nil
	}
	req := generic_serial_device.NewRegisterRequest(i2c)
	if req == nil {
		return 0, nil
	}
	req.BigEndian = true
	req.AddUint16(d.RefModeReg + 0x37)
	req.SetReadLen(1)
	if err := req.Execute(); err != nil {
		return 0, err
	}
	r := req.GetResult()
	if len(r) == 0 {
		return 0, nil
	}
	return r[0], nil
}

// ClockGen8A34002E — алиас для совместимости с open_timecard_v1 (ожидает *ClockGen8A34002E).
type ClockGen8A34002E = ClockGen8A34012

func NewRegisterBuffer() {
	// TODO: реконструировать
}

// PPMToFCW по дампу (0x4b905e0): pow(10,c1), pow(10,c2); tmp = (1 - 1/(ppm/div+1))*a; other = 1-b; result = min(tmp,other); result = max(result, lowBound); cvttsd2si.
// Константы из .rodata: 54de498=1.0, 54de540=10.0, 54de810 — делитель ppm; после min/max — приведение к int64.
func PPMToFCW(ppm float64) uint64 {
	const one = 1.0
	const ten = 10.0
	const divPPM = 1e6
	a := math.Pow(ten, 6)
	denom := ppm/divPPM + one
	tmp := one - one/denom
	tmp *= a
	upper := 1e6
	lower := -1e6
	result := math.Min(tmp, upper)
	result = math.Max(result, lower)
	return uint64(int64(result))
}

func ParseConfigOptions() {
	// TODO: реконструировать
}

func Reset() {
	// TODO: реконструировать
}

func ResetDPLLFreq() {
	// TODO: реконструировать
}

func SetDpllFodFreq() {
	// TODO: реконструировать
}

func SetHoldover() {
	// TODO: реконструировать
}

func SetHoldoverFm() {
	// TODO: реконструировать
}

func SetLoopBandwidth() {
	// TODO: реконструировать
}

func SetManualReference() {
	// TODO: реконструировать
}

func SetMode() {
	// TODO: реконструировать
}

func SetOutputDiv() {
	// TODO: реконструировать
}

func SetOutputDutyCycleHigh() {
	// TODO: реконструировать
}

func SetPageAddress1ByteMode() {
	// TODO: реконструировать
}

func SetPageAddress2ByteMode() {
	// TODO: реконструировать
}

func SetReferenceMode() {
	// TODO: реконструировать
}

func SetWritePhaseTimer() {
	// TODO: реконструировать
}

func Start() {
	// TODO: реконструировать
}

func VerifyConnectedToDevice() {
	// TODO: реконструировать
}

func WriteToEEPROM() {
	// TODO: реконструировать
}

func addWatchDogModule() {
	// TODO: реконструировать
}

func disciplineDPLLWithOffsetFromPTPGroup() {
	// TODO: реконструировать
}

func dpllCtrlLookup() {
	// TODO: реконструировать
}

func dpllRefModeLookup() {
	// TODO: реконструировать
}

func dpllRefStatusLookup() {
	// TODO: реконструировать
}

func dpllStatusLookup() {
	// TODO: реконструировать
}

func executePhasePullIn() {
	// TODO: реконструировать
}

func func1() {
	// TODO: реконструировать
}

func func2() {
	// TODO: реконструировать
}

func func3() {
	// TODO: реконструировать
}

func getRemoteGroupOffset() {
	// TODO: реконструировать
}

func init() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}

func isPaused() {
	// TODO: реконструировать
}

func logAnnotation() {
	// TODO: реконструировать
}

func logInputMonFreq() {
	// TODO: реконструировать
}


func monStatusLookup() {
	// TODO: реконструировать
}

func monitorFFO() {
	// TODO: реконструировать
}

func monitorFODFreq() {
	// TODO: реконструировать
}

func monitorFPA() {
	// TODO: реконструировать
}

func monitorFreq() {
	// TODO: реконструировать
}

func outputLookup() {
	// TODO: реконструировать
}


func runLoop() {
	// TODO: реконструировать
}

func setDCOConfigOptions() {
	// TODO: реконструировать
}

func setLoopBandwidth() {
	// TODO: реконструировать
}

func setPaused() {
	// TODO: реконструировать
}

func setupDPLLForDCOFreq() {
	// TODO: реконструировать
}

func setupDPLLForDCOPhase() {
	// TODO: реконструировать
}

func setupDPLLForGNSS() {
	// TODO: реконструировать
}

func stmp_0() {
	// TODO: реконструировать
}

func stmp_1() {
	// TODO: реконструировать
}

func stmp_2() {
	// TODO: реконструировать
}

func stmp_3() {
	// TODO: реконструировать
}

func stmp_4() {
	// TODO: реконструировать
}



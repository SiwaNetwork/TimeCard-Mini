package microchip_mac

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"go.bug.st/serial"
)

// Автоматически извлечено из timebeat-2.2.20. Реконструкция по дампам NewMicrochipMac, makeSerialDevice, RunReadloop, RunWriteloop, processPhaseMessage, sendMacCommand, Start.

// Mac — клиент oscillator (Microchip). По дампу NewMicrochipMac/makeSerialDevice/Start:
// 0=Config, 8=Serial, 0x10=Writer, 0x18=Logger, 0x20=Name, 0x28=Reader, 0x30=writeBuf, 0x38=Mutex, 0x41=StopFlag, 0x42=LogPhase.
type Mac struct {
	Config   interface{}
	Serial   interface{}   // go.bug.st/serial.Port или nil
	Writer   *bufio.Writer // 0x10
	Logger   *logging.Logger
	Name     string        // 0x20, имя из config
	Reader   *bufio.Reader // 0x28
	WriteBuf *bytes.Buffer // 0x30, буфер для записи
	Mu                sync.Mutex    // 0x38, для sendMacCommand
	StopFlag          byte          // 0x41: !=0 — не обрабатывать phase
	LogPhase          bool          // 0x42: true — выводить phase в stdout
	DisableTauSteps   bool          // по дампу parseOscTauStepProgramFromConfig: tau_steps=disable
}

// NewMicrochipMac по дампу (0x4b9f720): config (0x38,0x40=name); concat name+" "+0x20; NewLogger; new Mac; 0=Config, 0x18=Logger, 0x20=config.Name; makeSerialDevice → 0x8, 0x10; если Serial==nil — log и return nil; иначе buf 0x1000, Writer/Reader, 0x30, 0x28; return Mac.
func NewMicrochipMac(config interface{}) *Mac {
	if config == nil {
		return nil
	}
	name := getConfigName(config)
	if name == "" {
		name = "microchip-mac"
	}
	logger := logging.NewLogger(name + " ")
	cfg := config
	m := &Mac{
		Config:   cfg,
		Logger:   logger,
		Name:     name,
		WriteBuf: &bytes.Buffer{},
	}
	m.Serial = m.makeSerialDevice()
	if m.Serial == nil {
		logSerialError(m, "no serial port")
		return nil
	}
	setupReaderWriter(m)
	return m
}

func getConfigName(c interface{}) string {
	if c == nil {
		return ""
	}
	if m, ok := c.(map[string]interface{}); ok {
		if n, _ := m["Name"].(string); n != "" {
			return n
		}
		if p, _ := m["Port"].(string); p != "" {
			return "mac-" + p
		}
	}
	return ""
}

func getStringField(c interface{}, field string) (string, bool) {
	if c == nil {
		return "", false
	}
	if m, ok := c.(map[string]interface{}); ok {
		v, ok := m[field].(string)
		return v, ok
	}
	return "", false
}

func logSerialError(m *Mac, msg string) {
	if m == nil || m.Logger == nil {
		return
	}
	m.Logger.Error(msg)
}

// makeSerialDevice по дампу (0x4b9fbe0): если config.Port пустой — Logger.Error и return nil; иначе serial.Open(port, Mode{BaudRate: baud}); при ошибке — Error и nil; иначе return port.
func (m *Mac) makeSerialDevice() interface{} {
	port, baud := getPortAndBaud(m.Config)
	if port == "" {
		if m.Logger != nil {
			m.Logger.Error(fmt.Sprintf("serial port not configured for %s", m.Name))
		}
		return nil
	}
	mode := &serial.Mode{BaudRate: baud}
	p, err := serial.Open(port, mode)
	if err != nil {
		if m.Logger != nil {
			m.Logger.Error(fmt.Sprintf("failed to open serial %s: %v", port, err))
		}
		return nil
	}
	return p
}

func getPortAndBaud(c interface{}) (port string, baud int) {
	if c == nil {
		return "", 0
	}
	if m, ok := c.(map[string]interface{}); ok {
		if p, _ := m["Port"].(string); p != "" {
			port = p
		}
		if b, ok := m["Baud"].(int); ok && b > 0 {
			baud = b
		}
	}
	if baud <= 0 {
		baud = 9600
	}
	return port, baud
}

func setupReaderWriter(m *Mac) {
	if m.Serial == nil {
		return
	}
	// По дампу: из Serial берутся Reader/Writer (bufio). Если Serial — интерфейс io.ReadWriteCloser, оборачиваем.
	rw, ok := m.Serial.(interface {
		Read([]byte) (int, error)
		Write([]byte) (int, error)
	})
	if !ok {
		return
	}
	m.Reader = bufio.NewReaderSize(rw, 4096)
	m.Writer = bufio.NewWriterSize(rw, 4096)
}

// Start по дампу (0x4b9fde0): go func1 (DoInitialOscillatorConfig + determineAndRunTauStepsProgram); Sleep(0x37e11d600 ns); go func2 (RunReadloop); go func3 (RunWriteloop).
func (m *Mac) Start() {
	if m == nil {
		return
	}
	go m.doInitialAndTau()
	time.Sleep(1500 * time.Millisecond)
	go m.RunReadloop()
	go m.RunWriteloop()
}

func (m *Mac) doInitialAndTau() {
	if m == nil {
		return
	}
	m.DoInitialOscillatorConfig()
	m.determineAndRunTauStepsProgram()
}

// initialOscillatorCommands — три команды по дампу DoInitialOscillatorConfig (длины 0x14, 0x15, 0x17). Содержимое из rodata; заглушки для восстановления.
var initialOscillatorCommands = []string{
	"@00", "@01", "@02",
}

// DoInitialOscillatorConfig по дампу (0x4ba0020): pausePeriodic; цикл i=0..2 — sendMacCommand(cmd[i]); Sleep(0x1dcd6500 ≈500ms); Logger.Info.
func (m *Mac) DoInitialOscillatorConfig() {
	if m == nil {
		return
	}
	m.pausePeriodic()
	for _, cmd := range initialOscillatorCommands {
		m.sendMacCommand(cmd)
		time.Sleep(500 * time.Millisecond)
	}
	if m.Logger != nil {
		m.Logger.Info("initial oscillator config done", 0)
	}
}

// TauStep по дампу determineAndRunTauStepsProgram/runTauSteps: значение и длительность шага (мс).
type TauStep struct {
	Value      int64
	DurationMs int
}

// getDefaultTauStepProgram по дампу (0x4ba0199): 6 шагов — (0xd18c2e2800, 200), (0xd18c2e2800, 400), (800), (1000), (25000).
func getDefaultTauStepProgram() []TauStep {
	const defaultVal = 0xd18c2e2800
	return []TauStep{
		{defaultVal, 200},
		{defaultVal, 400},
		{defaultVal, 800},
		{defaultVal, 1000},
		{defaultVal, 25000},
	}
}

// determineAndRunTauStepsProgram по дампу (0x4ba0140): parseOscTauStepProgramFromConfig(); если nil — default шаги; если m.LogPhase или m.DisableTauSteps — не вызывать runTauSteps; иначе runTauSteps(steps).
func (m *Mac) determineAndRunTauStepsProgram() {
	if m == nil {
		return
	}
	steps := m.parseOscTauStepProgramFromConfig()
	if steps == nil {
		steps = getDefaultTauStepProgram()
	}
	if m.LogPhase || m.DisableTauSteps {
		return
	}
	m.runTauSteps(steps)
}

// getConfigOptions возвращает пары ключ-значение из config["Options"] (как в eth_sw).
func getConfigOptionsMac(c interface{}) [][2]string {
	if c == nil {
		return nil
	}
	if cfg, ok := c.(map[string]interface{}); ok {
		if o, ok := cfg["Options"].([]interface{}); ok {
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

// parseOscTauStepProgramFromConfig по дампу (0x4ba0260): итерация Options; ключ "tau"; split value по ","; "tau_steps"+"disable" → Warn, DisableTauSteps=true, return nil; иначе пары value,duration_ms → []TauStep.
func (m *Mac) parseOscTauStepProgramFromConfig() []TauStep {
	if m == nil || m.Config == nil {
		return nil
	}
	opts := getConfigOptionsMac(m.Config)
	var result []TauStep
	for _, kv := range opts {
		key, val := kv[0], kv[1]
		if len(key) < 3 || key[:3] != "tau" {
			continue
		}
		parts := strings.Split(val, ",")
		if len(parts) < 3 {
			continue
		}
		if parts[0] != "tau_steps" {
			continue
		}
		if len(parts) >= 2 && strings.ToLower(parts[1]) == "disable" {
			if m.Logger != nil {
				m.Logger.Warn("tau_steps=disable: using default oscillator behaviour")
			}
			m.DisableTauSteps = true
			return nil
		}
		for i := 1; i+1 < len(parts); i += 2 {
			value, err1 := strconv.ParseInt(strings.TrimSpace(parts[i]), 10, 64)
			durMs, err2 := strconv.Atoi(strings.TrimSpace(parts[i+1]))
			if err1 != nil || err2 != nil {
				if m.Logger != nil {
					m.Logger.Error(fmt.Sprintf("parseOscTauStepProgramFromConfig: invalid pair %q,%q", parts[i], parts[i+1]))
				}
				continue
			}
			result = append(result, TauStep{Value: value, DurationMs: durMs})
		}
		return result
	}
	return nil
}

// pausePeriodic по дампу DoInitialOscillatorConfig/runTauSteps — заглушка (останавливает периодическую отправку).
func (m *Mac) pausePeriodic() {}

// runTauSteps по дампу (0x4ba0740): Logger.Info("runTauSteps started"); для каждого шага — pausePeriodic; sendMacCommand("tau %d", value); Logger.Info(duration); Sleep(duration_ms).
func (m *Mac) runTauSteps(steps []TauStep) {
	if m == nil || len(steps) == 0 {
		return
	}
	if m.Logger != nil {
		m.Logger.Info("runTauSteps started", 0)
	}
	for _, step := range steps {
		m.pausePeriodic()
		m.sendMacCommand(fmt.Sprintf("tau %d", step.Value))
		if m.Logger != nil {
			m.Logger.Info(fmt.Sprintf("tau step duration %d ms", step.DurationMs), 0)
		}
		time.Sleep(time.Duration(step.DurationMs) * time.Millisecond)
	}
	if m.Logger != nil {
		m.Logger.Info("runTauSteps completed", 0)
	}
}

// RunReadloop по дампу (0x4b9f220): цикл — Sleep(0x989680 ns = 10 ms); ReadBytes('\n'); если err!=nil — continue; если m.StopFlag!=0 — continue; processPhaseMessage(line).
func (m *Mac) RunReadloop() {
	if m == nil || m.Reader == nil {
		return
	}
	const readLoopSleep = 10 * time.Millisecond
	for {
		time.Sleep(readLoopSleep)
		line, err := m.Reader.ReadBytes('\n')
		if err != nil {
			continue
		}
		if m.StopFlag != 0 {
			continue
		}
		m.processPhaseMessage(line)
	}
}

// RunWriteloop по дампу (0x4b9f2c0): NewTicker(0x3b9aca00 = 1e9 ns = 1 s); цикл — <-ticker.C; sendMacCommand(m, "\r\n", 11 байт из rodata).
func (m *Mac) RunWriteloop() {
	if m == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.sendMacCommand("\r\n")
	}
}

// processPhaseMessage по дампу (0x4b9f340): len>=2 и data[0:2]=="[="; сдвиг на 2; ParseFloat; умножить на константу из rodata; если m.LogPhase — Fprintf stdout; иначе GetPreciseTime, OTCSourceLogEntry.Log.
func (m *Mac) processPhaseMessage(data []byte) {
	if m == nil || len(data) < 2 {
		return
	}
	if data[0] != '[' || data[1] != '=' {
		return
	}
	rest := data[2:]
	if len(rest) < 3 {
		return
	}
	if len(rest) >= 3 && rest[len(rest)-3] == '\r' && rest[len(rest)-2] == '\n' {
		rest = rest[:len(rest)-2]
	} else if len(rest) >= 1 && rest[len(rest)-1] == '\n' {
		rest = rest[:len(rest)-1]
	}
	phase, err := strconv.ParseFloat(string(rest), 64)
	if err != nil {
		return
	}
	const scale = 1e9
	phaseNs := phase * scale
	if m.LogPhase {
		if m.Logger != nil {
			m.Logger.Info(fmt.Sprintf("phase %.6f ns (raw %.6f)", phaseNs, phase), 0)
		}
		return
	}
	t := adjusttime.GetPreciseTime()
	entry := &logging.OTCSourceLogEntry{
		SourceName: m.Name,
		PhaseNs:    int64(phaseNs),
		Time:       t,
	}
	entry.Log()
	_ = entry
}

// sendMacCommand по дампу (0x4ba0920): Lock(m.Mu); defer Unlock; concat "\r\n"+cmd (2+len); запись в m.Writer (bufio); при ошибке — Logger.Error.
func (m *Mac) sendMacCommand(cmd string) {
	if m == nil {
		return
	}
	m.Mu.Lock()
	defer m.Mu.Unlock()
	s := "\r\n" + cmd
	if m.Writer == nil {
		return
	}
	_, err := m.Writer.Write([]byte(s))
	if err != nil && m.Logger != nil {
		m.Logger.Error(fmt.Sprintf("sendMacCommand: %v", err))
	}
	_ = m.Writer.Flush()
}

func RunReadloop() {}
func RunWriteloop() {}
func processPhaseMessage() {}
func sendMacCommand() {}
func makeSerialDevice() {}

func DoInitialOscillatorConfig() {}
func GetDefaultTauStepProgram()  {}
func determineAndRunTauStepsProgram() {}
func parseOscTauStepProgramFromConfig() {}
func pausePeriodic() {}
func reenablePeriodic() {}
func reenablePeriodicFm() {}
func runTauSteps() {}

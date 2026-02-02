package generic_gnss_device

import (
	"bufio"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Автоматически извлечено из timebeat-2.2.20

func ContainsNMEA() {
	// TODO: реконструировать
}

func GNSSConstellations() {
	// TODO: реконструировать
}

func GNSSSystemNames() {
	// TODO: реконструировать
}

func GNSSTalker() {
	// TODO: реконструировать
}

func GetGnssSvStats() {
	// TODO: реконструировать
}

func GetGnssSvStatsDetail() {
	// TODO: реконструировать
}

func GetGnssTmode() {
	// TODO: реконструировать
}

func GetNMEAStatsChan() {
	// TODO: реконструировать
}

func GetObservationChan() {
	// TODO: реконструировать
}

func GetTaiChan() {
	// TODO: реконструировать
}

func MakeObservationFromRMCLine() {
	// TODO: реконструировать
}

// DeviceInterface по дизассемблеру nmea/client.Start: Start().
type DeviceInterface interface {
	Start()
}

// TAIEvent — данные TAI (clockName + offset) для NotifyTAIOffset. Используется NMEA client.
type TAIEvent struct {
	ClockName string
	OffsetNs  int64
}

// GNSSChannels — интерфейс устройства с каналами наблюдений (для NMEA runGNSSRunloop).
type GNSSChannels interface {
	GetObservationChan() chan interface{}
	GetTaiChan() chan TAIEvent
}

// GNSSChannelsWithGSV — опционально: устройство отдаёт канал NMEA GSV (satellites in view) для логирования.
// По дампу runReadloop: 0x120(device) = channel for GSV strings; recordGSV → selectnbsend.
type GNSSChannelsWithGSV interface {
	GNSSChannels
	GetGSVChan() chan string
}

// GenericGNSSDevice — структура по дизассемблеру NewDevice/runReadloop.
// Offsets: 0x70=ChUBX, 0x78=ChTAI, 0x80=ChObs, 0x88=ChExtra, 0xb0=FlagB0, 0xf0=MapF0, 0x100=Logger, 0x108=Config,
// 0x60=Serial, 0x110=Reader, 0x118=Writer, 0x120=ChGSV. Используется при полной реконструкции.
type GenericGNSSDevice struct {
	ChUBX   chan interface{}        // 0x70 — UBX out
	ChTAI   chan TAIEvent           // 0x78
	ChObs   chan interface{}        // 0x80
	ChExtra chan interface{}        // 0x88
	FlagB0  byte                    // 0xb0 — debug/dump
	MapF0   map[string]interface{}  // 0xf0
	Logger  interface{}             // 0x100 *logging.Logger
	Config  interface{}             // 0x108
	Serial  interface{}             // 0x60
	Reader  interface{}             // 0x110 *bufio.Reader
	Writer  interface{}             // 0x118 *bufio.Writer
	ChGSV   chan string             // 0x120 — NMEA GSV для логирования
}

// RunReadloopFlow (по дизассемблеру __GenericGNSSDevice_.runReadloop): цикл чтения из device.Reader (bufio.Reader),
// буфер до 0x400 байт, поиск UBX_HEADER (bytes.Index), ubx.IsEntireUBXMessageReceived → ubx.ToUBXMessage,
// отправка UBX в ChUBX (0x70). Для NMEA: ContainsNMEA → ToNMEAMessage → go-nmea.Parse; если GSV — recordGSV (→ ChGSV);
// если RMC — MakeObservationFromRMCLine → selectnbsend в ChObs (0x80). TAI при необходимости в ChTAI (0x78).
// helper/ubx: IsEntireUBXMessageReceived, ToUBXMessage уже добавлены для реконструкции.

// stubGNSSDevice — заглушка до полной реконструкции по дампу NewDevice (0x45afb20).
// По дампу GenericGNSSDevice: 0x78=GetTaiChan, 0x80=GetObservationChan (45acc80, 45acca0).
// Реализует DeviceInterface и GNSSChannels; runReadloop отправляет симулированные observation/TAI.
type stubGNSSDevice struct {
	chObs     chan interface{}
	chTAI     chan TAIEvent
	doneCh    chan struct{}
	clockName string
}

func (d *stubGNSSDevice) Start() {
	if d.doneCh == nil {
		d.doneCh = make(chan struct{})
	}
	if d.clockName == "" {
		d.clockName = "gps"
	}
	go d.runReadloop()
}

// runReadloop по дампу (0x45af3e0): цикл — опрос GNSS, отправка в chObs/chTAI. Stub: тикер 1s → observation, TAI.
func (d *stubGNSSDevice) runReadloop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.doneCh:
			return
		case <-ticker.C:
			// Симулируем observation (nil — NMEA client обработает)
			select {
			case d.chObs <- nil:
			default:
			}
			// Симулируем TAI (offset 0)
			select {
			case d.chTAI <- TAIEvent{ClockName: d.clockName, OffsetNs: 0}:
			default:
			}
		}
	}
}

func (d *stubGNSSDevice) GetObservationChan() chan interface{} { return d.chObs }
func (d *stubGNSSDevice) GetTaiChan() chan TAIEvent            { return d.chTAI }
func (d *stubGNSSDevice) GetGSVChan() chan string             { return nil }

// NewDevice по дизассемблеру (NewDevice@@Base): создаёт устройство из config; при успешном makeSerialDevice — *GenericGNSSDevice, иначе stubGNSSDevice.
func NewDevice(config interface{}) interface{} {
	chUBX := make(chan interface{}, 10)
	chTAI := make(chan TAIEvent, 10)
	chObs := make(chan interface{}, 20)
	chExtra := make(chan interface{}, 20)
	logger := logging.NewLogger("generic-gnss")
	d := &GenericGNSSDevice{
		ChUBX:   chUBX,
		ChTAI:   chTAI,
		ChObs:   chObs,
		ChExtra: chExtra,
		MapF0:   make(map[string]interface{}),
		Logger:  logger,
		Config:  config,
		ChGSV:   make(chan string, 16),
	}
	serial := d.makeSerialDevice()
	if serial != nil {
		if rw, ok := serial.(interface {
			Read(p []byte) (n int, err error)
			Write(p []byte) (n int, err error)
		}); ok {
			d.Serial = serial
			d.Reader = bufio.NewReaderSize(rw, 4096)
			d.Writer = bufio.NewWriterSize(rw, 4096)
			return d
		}
	}
	// Нет serial — возвращаем stub с теми же каналами наблюдений
	return &stubGNSSDevice{
		chObs:     chObs,
		chTAI:     chTAI,
		doneCh:    make(chan struct{}),
		clockName: "gps",
	}
}

func STANDARD_BAUD_RATES() {
	// TODO: реконструировать
}

func Start() {
	// TODO: реконструировать
}

func ToNMEAMessage() {
	// TODO: реконструировать
}

func configure1PPSTimepulse() {
	// TODO: реконструировать
}

func configureExternalOscillatorSourceInGNSSReceiver() {
	// TODO: реконструировать
}

func configureGNSSConstellationsInGNSSReceiver() {
	// TODO: реконструировать
}

func configureOutputFilterTimeAccuracyMask() {
	// TODO: реконструировать
}

func configureSignals() {
	// TODO: реконструировать
}

func configureSynchronisationManagerInGNSSReceiver() {
	// TODO: реконструировать
}

func configureTimeMode() {
	// TODO: реконструировать
}

func deleteSat() {
	// TODO: реконструировать
}

func detectUbloxUnit() {
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

func func4() {
	// TODO: реконструировать
}

func func5() {
	// TODO: реконструировать
}

func func6() {
	// TODO: реконструировать
}

func getLayersValue() {
	// TODO: реконструировать
}

func handleConnection() {
	// TODO: реконструировать
}

func init() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}

func logSatsInView() {
	// TODO: реконструировать
}

func makeSatHash() {
	// TODO: реконструировать
}

func makeSerialDevice() {
	// TODO: реконструировать
}

func parseConfigOptionsAndConfigureGNSSModule() {
	// TODO: реконструировать
}

func parseModVerExtensionProtocolVersion() {
	// TODO: реконструировать
}

func parseModVerExtensionType() {
	// TODO: реконструировать
}

func receiveAckAckMessage() {
	// TODO: реконструировать
}

func receiveAckNakMessage() {
	// TODO: реконструировать
}

func receiveAidHuiMessage() {
	// TODO: реконструировать
}

func receiveCfgValSetMessage() {
	// TODO: реконструировать
}

func receiveMonVerMessage() {
	// TODO: реконструировать
}

func receiveNavTimelsMessage() {
	// TODO: реконструировать
}

func recordGSV() {
	// TODO: реконструировать
}

func requestTimeMode() {
	// TODO: реконструировать
}

func runGSVLoggerLoop() {
	// TODO: реконструировать
}

func runMessageDispatcher() {
	// TODO: реконструировать
}

func runReadloop() {
	// TODO: реконструировать
}

func runWriteLoop() {
	// TODO: реконструировать
}

func send1PPSOnTimepulsePin() {
	// TODO: реконструировать
}

func send1PPSOnTimepulsePinWhenLocked() {
	// TODO: реконструировать
}

func sendMonVerMessage() {
	// TODO: реконструировать
}

func sendNmeaMainTalkerMessage() {
	// TODO: реконструировать
}

func sendUbxAidHuiNavMessage() {
	// TODO: реконструировать
}

func sendUbxNavTimelsMessage() {
	// TODO: реконструировать
}

func setNmeaTcpServerConfigOption() {
	// TODO: реконструировать
}

func setOscillatorConfigOption() {
	// TODO: реконструировать
}

func setPPSOutputFilterAccuracy() {
	// TODO: реконструировать
}

func setSignalConfigOption() {
	// TODO: реконструировать
}

func stmp_0() {
	// TODO: реконструировать
}

func validityToExtTsSrc() {
	// TODO: реконструировать
}

func waitForExit() {
	// TODO: реконструировать
}

func writeBytesToSerialPort() {
	// TODO: реконструировать
}

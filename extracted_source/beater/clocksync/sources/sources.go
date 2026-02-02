package sources

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/config"
)

// Автоматически извлечено из timebeat-2.2.20

// TimeSourceStore по дизассемблеру newTimeSourceStore: store+8 = Sources (sync.Map).
// GetSourcesForCLI: Lock(0x98), обнуление слайсов, Range(parseSourceForCLI-fm) → cliKeys/cliValues.
// GetNewIndex: store+0x80 — счётчик, инкремент и return. GenerateTimeSourcesFromConfig: store+0x90 = map (PTP по ключу).
type TimeSourceStore struct {
	Pad       [8]byte
	Sources   sync.Map
	cliMu     sync.Mutex        // по дизассемблеру GetSourcesForCLI 0x98
	cliKeys   []interface{}     // результат Range для CLI
	cliValues []interface{}
	// nextIndex по дизассемблеру GetNewIndex 0x80: load, inc, store, return
	NextIndex int64
	// ptpByKey по дизассемблеру GenerateTimeSourcesFromConfig 0x90: makemap, для ptp+enabled записей
	ptpByKey map[string]interface{}
	// clockProtocolMap по дизассемблеру IsClockProtocolEnabled/enableClockProtocol 0x68: mapaccess2_fast64/mapassign_fast64
	clockProtocolMap map[int64]bool
	// deviceVariantMap по дизассемблеру store+0x70: ocp_timecard(2), timebeat_timecard_mini(3), open_timecard_v1(4), open_timecard_mini_v2_pt(5)
	deviceVariantMap map[int64]bool
	// ptpPeerDelayMulticastEnabled по дизассемблеру IsPTPPeerDelayMulticastEnabled/enablePTPPeerDelayMulticast: movzbl (rax) / movb $1,(rax)
	ptpPeerDelayMulticastEnabled bool
}

// AdjustFor_* по дизассемблеру adjustForProfile (0x4414900): вызываются по имени профиля; в бинарнике меняют конфиг/коэффициенты. Минимальная реконструкция: no-op.
func AdjustFor_Enterprise_Draft() {}

func AdjustFor_G_8265_1() {}

func AdjustFor_G_8275_1() {}

func AdjustFor_G_8275_2() {}

func AdjustFor_IEC_IEEE_61850_9_3() {}

// AutoCreatePTPSource по дизассемблеру: создание PTP-источника при автообнаружении. Заглушка.
func AutoCreatePTPSource() {}

// TimeSourceConfig — минимальная структура источника по дизассемблеру CreateSource (поля 0x4c0..0x538, ID hash).
// Category: 1=primary, 2=secondary (для NTP ConfigureTimeSource).
type TimeSourceConfig struct {
	Type     string
	Name     string
	Index    int64
	ID       string // SHA1 от конкатенации полей (по дизассемблеру: concatstrings 8, sha1.Sum)
	Category int64  // 1=primary, 2=secondary — для NTP loadConfig
}

// CreateSource по дизассемблеру (0x44159c0): NewLogger("sources"); strings.Join([type,name,...], sep);
// strconv.FormatInt(index,10) и др.; сборка структуры; concatstrings(8); crypto/sha1.Sum; сравнение sourceType с таблицей ProtocolName (7997f20+0x10..+0x88, switch по типу).
// Реконструкция: логгер, конкатенация полей для ID, SHA1, возврат TimeSourceConfig; ветки по sourceType соответствуют protocolNames (nmea/ntp/ptp/gnss и др.).
func (s *TimeSourceStore) CreateSource(sourceType string, name string, index int64) interface{} {
	_ = logging.NewLogger("sources")
	joined := strings.Join([]string{sourceType, name}, "|")
	indexStr := strconv.FormatInt(index, 10)
	toHash := strings.Join([]string{joined, indexStr, sourceType, name}, "")
	sum := sha1.Sum([]byte(toHash))
	idHex := hex.EncodeToString(sum[:])
	cfg := &TimeSourceConfig{Type: sourceType, Name: name, Index: index, ID: idHex}
	switch sourceType {
	case "nmea", "ntp", "ptp", "gnss", "ptp-gm", "ptp-slave":
		return cfg
	case "pps", "gps", "oscillator", "phc":
		return cfg
	case "ocp_timecard", "timebeat_timecard_mini", "open_timecard_v1", "open_timecard_mini_v2_pt":
		return cfg
	default:
		_ = fmt.Sprint(cfg)
		return cfg
	}
}

// DeviceVariantName по дизассемблеру: имя варианта устройства. Заглушка.
func DeviceVariantName() {}

// GenerateTimeSourcesFromConfig по дизассемблеру (0x4417180): копирование слайсов конфига (appConfig+0x48/0x60),
// makemap → store+0x90, цикл по элементам: type=="ptp" и byte+0x11!=0 → mapassign; затем generateClockSourceForType(primary), generateClockSourceForType(secondary).
func (s *TimeSourceStore) GenerateTimeSourcesFromConfig(cfg *config.ClockSyncConfig) {
	if cfg == nil {
		return
	}
	s.ptpByKey = make(map[string]interface{})
	for i := range cfg.PrimaryClocks {
		ent := &cfg.PrimaryClocks[i]
		if ent.Protocol == "ptp" && !ent.Disable {
			key := ent.Interface
			if key == "" {
				key = fmt.Sprintf("ptp:%d", ent.Domain)
			}
			s.ptpByKey[key] = ent
		}
	}
	for i := range cfg.SecondaryClocks {
		ent := &cfg.SecondaryClocks[i]
		if ent.Protocol == "ptp" && !ent.Disable {
			key := ent.Interface
			if key == "" {
				key = fmt.Sprintf("ptp:%d", ent.Domain)
			}
			s.ptpByKey[key] = ent
		}
	}
	s.generateClockSourceForType(cfg.PrimaryClocks, 1)   // 1=primary
	s.generateClockSourceForType(cfg.SecondaryClocks, 2) // 2=secondary
}

// adjustForProfile по дизассемблеру (0x4414900): (rbx)+0x58/0x60 = profile string; по len/содержимому вызываются AdjustFor_*.
// len 6 "hybrid", len 16 "enterprise-draft" → Enterprise_Draft; len 8 "G.8265.1"/"G.8275.1"/"G.8275.2" → соответствующий AdjustFor_; len 18 → IEC_IEEE_61850_9_3.
func (s *TimeSourceStore) adjustForProfile(profileName string) {
	switch len(profileName) {
	case 6:
		if profileName == "hybrid" {
			AdjustFor_Enterprise_Draft()
		}
	case 8:
		switch profileName {
		case "G.8265.1":
			AdjustFor_G_8265_1()
		case "G.8275.1":
			AdjustFor_G_8275_1()
		case "G.8275.2":
			AdjustFor_G_8275_2()
		}
	case 16:
		if profileName == "enterprise-draft" {
			AdjustFor_Enterprise_Draft()
		}
	case 18:
		if profileName == "IEC_IEEE_61850_9_3" {
			AdjustFor_IEC_IEEE_61850_9_3()
		}
	}
}

// protocolToKey — маппинг protocol string → key для enableClockProtocol (по дизассемблеру Controller.Run: 1=PTP, 2=NTP, 3=PPS, 4=NMEA, 6=PHC, 7=oscillator).
func protocolToKey(protocol string) int64 {
	switch protocol {
	case "ptp", "ptp-slave", "ptp-gm":
		return 1
	case "ntp":
		return 2
	case "pps":
		return 3
	case "gnss", "nmea", "gps":
		return 4
	case "phc":
		return 6
	case "oscillator":
		return 7
	default:
		return -1
	}
}

// protocolToDeviceVariantKey — маппинг protocol string → key для EnableDeviceVariant (store+0x70): 2=ocp_timecard, 3=timebeat_timecard_mini, 4=open_timecard_v1, 5=open_timecard_mini_v2_pt.
func protocolToDeviceVariantKey(protocol string) int64 {
	switch protocol {
	case "ocp_timecard":
		return 2
	case "timebeat_timecard_mini":
		return 3
	case "open_timecard_v1":
		return 4
	case "open_timecard_mini_v2_pt":
		return 5
	default:
		return -1
	}
}

// generateClockSourceForType по дизассемблеру (0x4417400): цикл по слайсу; если не disabled — GetNewIndex, CreateSource, enableClockProtocol, adjustForProfile, AddSource.
// category: 1=primary, 2=secondary — для NTP ConfigureTimeSource.
func (s *TimeSourceStore) generateClockSourceForType(slice []config.ClockSource, category int64) {
	for i := range slice {
		ent := &slice[i]
		if ent.Disable {
			continue
		}
		if key := protocolToKey(ent.Protocol); key >= 0 {
			s.EnableClockProtocol(key)
		}
		if dvKey := protocolToDeviceVariantKey(ent.Protocol); dvKey >= 0 {
			s.EnableDeviceVariant(dvKey)
		}
		if ent.Profile != "" {
			s.adjustForProfile(ent.Profile)
		}
		idx := s.GetNewIndex()
		name := ent.IP
		if name == "" {
			name = ent.Interface
		}
		if name == "" {
			name = ent.Device
		}
		if name == "" {
			name = fmt.Sprintf("source-%d", idx)
		}
		created := s.CreateSource(ent.Protocol, name, idx)
		if cfg, ok := created.(*TimeSourceConfig); ok {
			cfg.Category = category
			s.AddSource(cfg.ID, created)
		}
	}
}

// GetNewIndex по дизассемблеру (0x44179a0): store+0x80 load, inc, store, return incremented value.
func (s *TimeSourceStore) GetNewIndex() int64 {
	s.NextIndex++
	return s.NextIndex
}

// GetPTPTransmissionLayerInformation по дизассемблеру: сведения о PTP transport. Заглушка.
func GetPTPTransmissionLayerInformation() {}

// GetSourcesForCLI по дизассемблеру (0x44122e0): Lock(0x98); обнуление 0xa0, 0xa8/0xb0; defer Unlock; Range(store+8, parseSourceForCLI-fm); return store.a8/a0, store.b0/b8 (cliKeys, cliValues).
func (s *TimeSourceStore) GetSourcesForCLI() (keys []interface{}, values []interface{}) {
	s.cliMu.Lock()
	defer s.cliMu.Unlock()
	s.cliKeys = nil
	s.cliValues = nil
	s.Sources.Range(func(key, value interface{}) bool {
		s.parseSourceForCLI(key, value)
		return true
	})
	return s.cliKeys, s.cliValues
}

// parseSourceForCLI по дизассемблеру (0x44124e0): коллбэк для Range(key, value); type assert value; convT64, fmt.Sprintf(формат 0x16); growslice; append в store+0xa0 (cliKeys) и 0xa8/0xb0 (cliValues).
func (s *TimeSourceStore) parseSourceForCLI(key, value interface{}) {
	s.cliKeys = append(s.cliKeys, key)
	s.cliValues = append(s.cliValues, value)
}

var (
	storeOnce sync.Once
	storeVar  *TimeSourceStore
)

// GetStore по дизассемблеру (0x4416fe0): sync.Once(once 0x7e7b3e8); doSlow(GetStore.func1); return store (0x7e2c268). func1 вызывает newTimeSourceStore.
func GetStore() *TimeSourceStore {
	storeOnce.Do(func() {
		storeVar = newTimeSourceStore()
	})
	return storeVar
}

// newTimeSourceStore по дизассемблеру (newTimeSourceStore@@Base 0x4417040): NewLogger, makemap x2, newobject(TimeSourceStore), 0x78=logger, 0x68/0x70=map, store=rax.
func newTimeSourceStore() *TimeSourceStore {
	_ = logging.NewLogger("sources-store")
	return &TimeSourceStore{Sources: sync.Map{}, ptpByKey: make(map[string]interface{})}
}

// GetSources по дизассемблеру (0x4417160): test (rax); add $0x8, rax; ret — возврат &store.Sources (sync.Map для Range).
func (s *TimeSourceStore) GetSources() *sync.Map {
	return &s.Sources
}

// AddSource по дизассемблеру (0x4417b80): store+8 = Sources; key type [20]uint8 (SHA1), value type *TimeSourceConfig (K0rE3Tf5); sync.(*Map).Swap(store+8, key, value).
func (s *TimeSourceStore) AddSource(key, value interface{}) {
	s.Sources.Swap(key, value)
}

// IsClockProtocolEnabled по дизассемблеру (0x4417a20): store+0x68 = map, mapaccess2_fast64(key=rbx), return value (bool).
func (s *TimeSourceStore) IsClockProtocolEnabled(protocolKey int64) bool {
	if s.clockProtocolMap == nil {
		return false
	}
	return s.clockProtocolMap[protocolKey]
}

// EnableClockProtocol по дизассемблеру (0x44179c0): store+0x68 mapassign_fast64(key=rbx), movb $1,(rax).
func (s *TimeSourceStore) EnableClockProtocol(protocolKey int64) {
	if s.clockProtocolMap == nil {
		s.clockProtocolMap = make(map[int64]bool)
	}
	s.clockProtocolMap[protocolKey] = true
}

// deviceVariantMap по дизассемблеру store+0x70: map[int64]bool для ocp_timecard(2), timebeat_timecard_mini(3), open_timecard_v1(4), open_timecard_mini_v2_pt(5).
func (s *TimeSourceStore) deviceVariantMapInit() {
	if s.deviceVariantMap == nil {
		s.deviceVariantMap = make(map[int64]bool)
	}
}

// IsDeviceVariantEnabled по дизассемблеру store+0x70: mapaccess2_fast64(key) — возврат value (bool).
func (s *TimeSourceStore) IsDeviceVariantEnabled(deviceVariantKey int64) bool {
	if s.deviceVariantMap == nil {
		return false
	}
	return s.deviceVariantMap[deviceVariantKey]
}

// EnableDeviceVariant по дизассемблеру: mapassign по ключу, value=true.
func (s *TimeSourceStore) EnableDeviceVariant(deviceVariantKey int64) {
	s.deviceVariantMapInit()
	s.deviceVariantMap[deviceVariantKey] = true
}

// IsPTPPeerDelayMulticastEnabled по дизассемблеру (0x4417b60): movzbl (rax), return byte (receiver+0).
func (s *TimeSourceStore) IsPTPPeerDelayMulticastEnabled() bool {
	return s.ptpPeerDelayMulticastEnabled
}

// EnablePTPPeerDelayMulticast по дизассемблеру (0x4417b40): movb $0x1,(rax) — запись в receiver+0.
func (s *TimeSourceStore) EnablePTPPeerDelayMulticast() {
	s.ptpPeerDelayMulticastEnabled = true
}

// protocolNames — глобальная таблица имён протоколов по дизассемблеру (7997f20 = ProtocolName@@Base).
// makeTimeSourceKey: index 0..8 (cmp $0x9), 16 байт на запись (string header). CreateSource/generateClockSourceForType сравнивают sourceType с этими строками.
var protocolNames = []string{"nmea", "ntp", "ptp", "gnss", "ptp-gm", "ptp-slave", "oscillator", "pps", "gps"}

// typeNames — глобальная таблица имён типов для логов по дизассемблеру (798bd40 = TypeName@@Base).
// doWeHaveSourcesForType: index 0..2 (cmp $0x3), 16 байт на запись; используется в NewTimeSourceLogEntry, LogTimesourceExpired, LogNewTimesourceReporting.
var typeNames = []string{"primary", "secondary", "reference"}

// ProtocolName возвращает имя протокола по индексу (по дизассемблеру makeTimeSourceKey: ProtocolName@@Base+index*16).
// Индекс 0..8; при выходе за границу возвращает пустую строку.
func ProtocolName(i int) string {
	if i < 0 || i >= len(protocolNames) {
		return ""
	}
	return protocolNames[i]
}

// TypeName возвращает имя типа источника по индексу (по дизассемблеру doWeHaveSourcesForType: TypeName@@Base+index*16).
// Индекс 0..2; при выходе за границу возвращает пустую строку.
func TypeName(i int) string {
	if i < 0 || i >= len(typeNames) {
		return ""
	}
	return typeNames[i]
}

// addMulticastInterface по дизассемблеру: добавление multicast-интерфейса. Заглушка.
func addMulticastInterface() {}

// addSource (standalone) — по дизассемблеру; метод AddSource на TimeSourceStore реализован выше.
func addSource() {}

// enableClockProtocol — см. EnableClockProtocol (метод выше).
// enableDeviceVariant — см. EnableDeviceVariant.
// enablePTPPeerDelayMulticast — см. EnablePTPPeerDelayMulticast.

func func1() {}

// generateClockSourceForType (standalone) — метод (s *TimeSourceStore).generateClockSourceForType реализован выше.
func generateClockSourceForType() {}

// GetNextAvailablePTPAutoDomain по дизассемблеру (0x4418600): store+0x90 map, цикл по индексу 0..2, mapaccess2 по ключу store+0x88+idx; при отсутствии возврат byte из 0x88+idx. Минимально: возврат 0.
func (s *TimeSourceStore) GetNextAvailablePTPAutoDomain() int {
	if s.ptpByKey == nil {
		return 0
	}
	for d := 0; d < 3; d++ {
		if _, ok := s.ptpByKey[fmt.Sprintf("domain:%d", d)]; !ok {
			return d
		}
	}
	return 0
}

// GetPTPDomainMap по дизассемблеру (0x4418460): makemap → store+0x90, цикл по слайсу записей; type=="ptp" и enabled → mapassign. Возврат ptpByKey.
func (s *TimeSourceStore) GetPTPDomainMap(entries []config.ClockSource) map[string]interface{} {
	if s.ptpByKey == nil {
		s.ptpByKey = make(map[string]interface{})
	}
	for i := range entries {
		ent := &entries[i]
		if ent.Protocol == "ptp" && !ent.Disable {
			key := ent.Interface
			if key == "" {
				key = fmt.Sprintf("ptp:%d", ent.Domain)
			}
			s.ptpByKey[key] = ent
		}
	}
	return s.ptpByKey
}

func inittask() {}

// newTimeSourceStore реализован выше (по дизассемблеру).

func once() {}

// parseSourceForCLI — метод выше (s *TimeSourceStore).parseSourceForCLI(key, value).
// parseSourceForCLI-fm в дизассемблере — closure, вызывающий parseSourceForCLI; в GetSourcesForCLI это анонимный func в Range.

// setEnablePTPMulticast по дизассемблеру: включение PTP multicast. Заглушка.
func setEnablePTPMulticast() {}

// setPTPTransmissionLayerInformation по дизассемблеру: установка сведений о PTP transport. Заглушка.
func setPTPTransmissionLayerInformation() {}

func store() {}


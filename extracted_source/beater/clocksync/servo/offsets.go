package servo

import (
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/statistics"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// OffsetFilterInterface — фильтр для offset (по дизассемблеру IsFilteredCache/IsFiltered в ProcessObservation).
type OffsetFilterInterface interface {
	IsFiltered(offset int64) bool
}

// Offsets — управление источниками времени и их offset'ами.
// В бинарнике RegisterObservation: selectnbsend в канал +0x28; при неудаче — Logger.Warn (code_analysis/disassembly/Offsets_RegisterObservation.txt).
// RunProcessObservationsLoop (0x45a2c80): select на канале 0x28, при приходе sourceID — ProcessObservation(sourceID).
type Offsets struct {
	mu              sync.RWMutex
	sources         map[string]*TimeSource
	filter          OffsetFilterInterface // опционально: при updateTimeSourceFilteredOffset вызывается IsFiltered(ts.offset)
	ema             *statistics.EMA       // 0x38 по дизассемблеру maintainEMA; при ProcessObservation вызывается EMA.AddValue
	observationChan chan string          // 0x28 по дизассемблеру RunProcessObservationsLoop: канал sourceID для ProcessObservation
	ttl             time.Duration
	holdover        bool
	emaEnabled      bool
	flag68          byte // 0x68 по дизассемблеру maintainEMA: or 0x8 после AddValue
}

// Category по дизассемблеру GetSourceCandidates: 0x48(ts)==1 → primary list, ==2 → secondary list.
const (
	CategoryPrimary   = 1
	CategorySecondary = 2
)

// TimeSource — источник времени (по дизассемблеру GetSourceCandidates: 0x48=category). addClockNamesToSourcesSnapshot: 0x80/0x88 = clockName (string).
type TimeSource struct {
	mu              sync.Mutex
	id              string
	protocol        string
	offset          int64
	filteredOffset  int64
	lastUpdate      time.Time
	active          bool
	sourceGroup     string
	rmsAccumulator  float64
	rmsCount        int
	category        int    // 0x48 по дизассемблеру: 1=primary, 2=secondary
	clockName       string // 0x80/0x88 по дизассемблеру addClockNamesToSourcesSnapshot: имя PHC или "system"
}

// TimeSourceMap — карта источников
type TimeSourceMap struct {
	mu      sync.RWMutex
	sources map[string]*TimeSource
}

// NewOffsets создаёт Offsets
func NewOffsets() *Offsets {
	return &Offsets{
		sources:         make(map[string]*TimeSource),
		ema:             statistics.NewEMA(),
		observationChan: make(chan string, 64), // по дизассемблеру RunProcessObservationsLoop: канал для select
		ttl:             10 * time.Second,
	}
}

// makeTimeSourceKey по дизассемблеру (makeTimeSourceKey@@Base 0x45a4360): ключ для карты источников; полная реализация — нормализация/хеш; здесь — идентичность.
func makeTimeSourceKey(sourceID string) string {
	return sourceID
}

// getTimeSource по дизассемблеру (__TimeSourceMap_.getTimeSource 0x45a3820): ключ = makeTimeSourceKey(sourceID); lookup в map; если найден и resetIfExists — resetTimeSource(ts); иначе если не найден — addTimesource (новый ts, вставка в map). Возврат *TimeSource.
func (o *Offsets) getTimeSource(sourceID string, resetIfExists bool) *TimeSource {
	key := makeTimeSourceKey(sourceID)
	ts, ok := o.sources[key]
	if ok {
		if resetIfExists {
			ts.resetTimeSource()
		}
		return ts
	}
	ts = &TimeSource{id: sourceID, category: CategoryPrimary}
	o.sources[key] = ts
	return ts
}

// RegisterObservation по дизассемблеру (0x45a2b60): getTimeSource, updateTimeSourceUnfilteredOffset; selectnbsend(0x28, sourceID); при неудаче — Logger.Warn.
func (o *Offsets) RegisterObservation(sourceID string, offsetNs int64) {
	o.mu.Lock()
	ts := o.getTimeSource(sourceID, false)
	ts.updateTimeSourceUnfilteredOffset(offsetNs)
	o.mu.Unlock()
	select {
	case o.observationChan <- sourceID:
	default:
		if lg := logging.GetErrorLogger(); lg != nil {
			lg.Warn("RegisterObservation: observationChan full, dropping " + sourceID)
		}
	}
}

// RegisterObservationWithCategory регистрирует наблюдение и задаёт category (1=primary, 2=secondary) для GetSourceCandidates.
// По дизассемблеру: как RegisterObservation — selectnbsend(0x28, sourceID) для ProcessObservation.
func (o *Offsets) RegisterObservationWithCategory(sourceID string, offsetNs int64, category int) {
	o.mu.Lock()
	ts := o.getTimeSource(sourceID, false)
	ts.mu.Lock()
	ts.category = category
	ts.mu.Unlock()
	ts.updateTimeSourceUnfilteredOffset(offsetNs)
	o.mu.Unlock()
	select {
	case o.observationChan <- sourceID:
	default:
		if lg := logging.GetErrorLogger(); lg != nil {
			lg.Warn("RegisterObservationWithCategory: observationChan full, dropping " + sourceID)
		}
	}
}

// ProcessObservation обрабатывает наблюдение.
// По дизассемблеру (__Offsets_.ProcessObservation 0x45a2d60): Lock(0x20), getTimeSource(0x10, sourceID); при ts.f0 — skip;
// ts.0x30=offset, ts.0x50=1, ts.0xc0=0; updateTimeSourceInternals(ts); ts.0x28=0; при filter — IsFilteredCache/IsFiltered, ts.0x28=offset;
// maintainEMA(ts); флаг 0x68|=0x8; selectnbsend; при неудаче Logger.Warn.
func (o *Offsets) ProcessObservation(sourceID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	ts := o.getTimeSource(sourceID, false)
	ts.mu.Lock()
	active := ts.active
	ts.mu.Unlock()
	if !active {
		return
	}
	ts.updateTimeSourceInternals()
	if o.filter != nil {
		o.updateTimeSourceFilteredOffsetFor(ts)
	} else {
		ts.updateTimeSourceFilteredOffset()
	}
	o.maintainEMA(ts)
}

// GetSourceCandidates по дизассемблеру (__Offsets_.GetSourceCandidates 0x45a51c0): Lock(0x20), defer unlock; mapiterinit 0x10; для каждого ts: 0x48(ts)==1 → append в slice primary (0x80/0x88/0x90), ==2 → append в secondary (0x98/0xa0/0xa8); return primary если len>0 иначе secondary.
func (o *Offsets) GetSourceCandidates() []*TimeSource {
	o.mu.Lock()
	defer o.mu.Unlock()
	var primary, secondary []*TimeSource
	for _, ts := range o.sources {
		if !ts.active {
			continue
		}
		ts.mu.Lock()
		cat := ts.category
		ts.mu.Unlock()
		if cat == CategoryPrimary || cat == 0 {
			primary = append(primary, ts)
		} else if cat == CategorySecondary {
			secondary = append(secondary, ts)
		}
	}
	if len(primary) > 0 {
		return primary
	}
	return secondary
}

// GetSourcesSnapshot возвращает снимок источников (все).
// GetSourcesSnapshot(category) по дизассемблеру getSourcesSnapshot: вызов с category 1 (primary) или 2 (secondary); возврат снимка по категории.
func (o *Offsets) GetSourcesSnapshot() map[string]*TimeSource {
	return o.GetSourcesSnapshotForCategory(0)
}

// GetSourcesSnapshotForCategory возвращает снимок источников категории category (0=все, 1=primary, 2=secondary).
func (o *Offsets) GetSourcesSnapshotForCategory(category int) map[string]*TimeSource {
	o.mu.RLock()
	defer o.mu.RUnlock()
	snapshot := make(map[string]*TimeSource)
	for k, v := range o.sources {
		if category == 0 {
			snapshot[k] = v
			continue
		}
		v.mu.Lock()
		cat := v.category
		v.mu.Unlock()
		if cat == category {
			snapshot[k] = v
		}
	}
	return snapshot
}

// GetSourcesCount возвращает количество источников
func (o *Offsets) GetSourcesCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.sources)
}

// GetTimeSourceCount возвращает количество активных источников
func (o *Offsets) GetTimeSourceCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	count := 0
	for _, ts := range o.sources {
		if ts.active {
			count++
		}
	}
	return count
}

// GetActiveSourceGroupMembers возвращает активных членов группы
func (o *Offsets) GetActiveSourceGroupMembers(group string) []*TimeSource {
	o.mu.RLock()
	defer o.mu.RUnlock()
	var members []*TimeSource
	for _, ts := range o.sources {
		if ts.active && ts.sourceGroup == group {
			members = append(members, ts)
		}
	}
	return members
}

// SetFilter задаёт фильтр для offset (используется в updateTimeSourceFilteredOffset).
func (o *Offsets) SetFilter(f OffsetFilterInterface) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.filter = f
}

// NotifyOffsetUsed уведомляет об использовании offset (обновляет lastUpdate источника, чтобы не считать его устаревшим).
func (o *Offsets) NotifyOffsetUsed(sourceID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if ts, ok := o.sources[sourceID]; ok {
		ts.mu.Lock()
		ts.lastUpdate = time.Now()
		ts.mu.Unlock()
	}
}

// EnterHoldoverIfThisWasLastTimeSourceToExpire входит в holdover
func (o *Offsets) EnterHoldoverIfThisWasLastTimeSourceToExpire() {
	o.mu.Lock()
	defer o.mu.Unlock()
	activeCount := 0
	for _, ts := range o.sources {
		if ts.active {
			activeCount++
		}
	}
	if activeCount == 0 {
		o.holdover = true
	}
}

// ExitHoldoverIfWeHaveAnyActiveTimeSources выходит из holdover
func (o *Offsets) ExitHoldoverIfWeHaveAnyActiveTimeSources() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, ts := range o.sources {
		if ts.active {
			o.holdover = false
			return
		}
	}
}

// DebugTTL для отладки TTL
func (o *Offsets) DebugTTL() {}

// DisableFilterForPeerID отключает фильтр
func (o *Offsets) DisableFilterForPeerID(peerID string) {}

// EnableFilterForPeerID включает фильтр
func (o *Offsets) EnableFilterForPeerID(peerID string) {}

// LogNewTimesourceReporting логирует новый источник
func (o *Offsets) LogNewTimesourceReporting(sourceID string) {}

// LogTimesourceExpired логирует истечение источника
func (o *Offsets) LogTimesourceExpired(sourceID string) {}

// RunProcessObservationsLoop по дизассемблеру (0x45a2c80): цикл select на канале 0x28(Offsets); при приходе sourceID — ProcessObservation(sourceID).
func (o *Offsets) RunProcessObservationsLoop() {
	for sourceID := range o.observationChan {
		o.ProcessObservation(sourceID)
	}
}

// RunSetLogEntriesLoop по дизассемблеру (0x45a5cc0): SetLogEntriesLoop, NewTicker(10s), цикл select(ticker.C, controller); при ticker — SetLogEntriesLoop снова.
func (o *Offsets) RunSetLogEntriesLoop() {
	o.SetLogEntriesLoop()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		o.SetLogEntriesLoop()
	}
}

// SetLogEntriesLoop по дизассемблеру (0x45a5d80): Lock, итерация по o.sources, сбор slice, вызов logging.SetHTTPTimeSourcesStatus(slice).
func (o *Offsets) SetLogEntriesLoop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	var entries []interface{}
	for _, ts := range o.sources {
		if ts == nil {
			continue
		}
		entries = append(entries, ts.SourceSnapshot())
	}
	logging.SetHTTPTimeSourcesStatus(entries)
}

// TemporarilyDisableClientFilterForSourceGroupMembers временно отключает фильтр
func (o *Offsets) TemporarilyDisableClientFilterForSourceGroupMembers(group string) {}

// getNMEAExtTsValueForLinkedDevice получает NMEA timestamp
func (o *Offsets) getNMEAExtTsValueForLinkedDevice(device string) int64 { return 0 }

// maintainEMA по дизассемблеру (__Offsets_.maintainEMA 0x45a3760): 0x38(Offsets)=EMA, 0x50(ts)=value; EMA.AddValue(ts.filteredOffset); 0x68(Offsets)|=0x8.
func (o *Offsets) maintainEMA(ts *TimeSource) {
	if o.ema != nil {
		o.ema.AddValue(ts.filteredOffset)
		o.flag68 |= 8
	}
}

// TimeSource methods

// SourceSnapshot возвращает снимок
func (ts *TimeSource) SourceSnapshot() map[string]interface{} {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return map[string]interface{}{
		"id":       ts.id,
		"offset":   ts.offset,
		"filtered": ts.filteredOffset,
		"active":   ts.active,
	}
}

// GetFilteredOffset возвращает отфильтрованный offset источника (для PTP/DPLL discipline).
func (ts *TimeSource) GetFilteredOffset() int64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.filteredOffset
}

// updateTimeSourceUnfilteredOffset по дизассемблеру: +0x30=offset, +0x50=1 (active), +0xc0=0 (TimeSource_updateTimeSourceUnfilteredOffset.txt).
func (ts *TimeSource) updateTimeSourceUnfilteredOffset(offsetNs int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.offset = offsetNs
	ts.lastUpdate = time.Now()
	ts.active = true
}

// updateTimeSourceFilteredOffset по дизассемблеру ProcessObservation: при obs.60!=0 — ветка с IsFilteredCache/IsFiltered; иначе filteredOffset = offset.
func (ts *TimeSource) updateTimeSourceFilteredOffset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.filteredOffset = ts.offset
}

// updateTimeSourceFilteredOffsetFor вызывается из ProcessObservation при наличии фильтра; вызывающий уже держит o.mu.
func (o *Offsets) updateTimeSourceFilteredOffsetFor(ts *TimeSource) {
	f := o.filter
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if f != nil && f.IsFiltered(ts.offset) {
		// Отфильтрованное наблюдение: не обновляем filteredOffset (оставляем предыдущее)
		return
	}
	ts.filteredOffset = ts.offset
}

// updateTimeSourceInternals по дизассемблеру (__TimeSource_.updateTimeSourceInternals 0x45a35e0): GetController, GetUTCTimeFromMasterClock; ts.0x50=1 (active); копирование полей source→ts; offset² в 0x58, inc 0x60 (RMS).
func (ts *TimeSource) updateTimeSourceInternals() {
	_ = GetController().GetUTCTimeFromMasterClock()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.active = true
	ts.rmsAccumulator += float64(ts.offset) * float64(ts.offset)
	ts.rmsCount++
}

// maintainRMS по дизассемблеру (__TimeSource_.maintainRMS.txt): sum_sq += value² (0x58), count++ (0x60).
func (ts *TimeSource) maintainRMS(value int64) {
	ts.rmsAccumulator += float64(value) * float64(value)
	ts.rmsCount++
}

// modifyObservationIfRequired по дизассемблеру (__TimeSource_.modifyObservationIfRequired.txt): если ts+0xf0 != 0, то obs+0x99 = 0.
func (ts *TimeSource) modifyObservationIfRequired(offset int64) int64 { return offset }
func (ts *TimeSource) resetTimeSource() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.offset = 0
	ts.filteredOffset = 0
	ts.active = false
}

// TimeSourceMap methods

func (tsm *TimeSourceMap) addTimesource(ts *TimeSource) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	tsm.sources[ts.id] = ts
}

func (tsm *TimeSourceMap) deleteTimeSource(id string) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	delete(tsm.sources, id)
}

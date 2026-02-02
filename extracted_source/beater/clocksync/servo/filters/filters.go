package filters

import (
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Автоматически извлечено из timebeat-2.2.20

// Константы типов фильтра по дизассемблеру NewFilterConfig (cmp $0x1, $0x2, $0x4, $0x5).
const (
	FilterTypeNoneGaussian1 = 1
	FilterTypeNoneGaussian2 = 2
	FilterTypeNoneGaussian4 = 4
	FilterTypeNoneGaussian5 = 5
)

// NoneGaussianFilter по дизассемблеру (__NoneGaussianFilter_.IsFiltered.txt): 0x00=count (incq в IsFiltered),
// 0x08/0x10 — slice, 0x40=config, 0x60=mutex; 0x48=median, 0x50=predicted; 0x58=counter.
type NoneGaussianFilter struct {
	count     int64               // 0x00 — кол-во в буфере; в IsFilteredCache если count>=config.divisor → false
	slicePtr  *[]int64            // 0x08
	sliceLen  int                 // 0x10
	slice2    []int64             // 0x20 — второй буфер по NewNoneGausianFilter
	logger    interface{}         // 0x38 — по дизассемблеру NewNoneGausianFilter
	config    *noneGaussianConfig // 0x40
	mu        sync.Mutex          // 0x60
	medianIdx int64               // 0x48
	medianVal int64               // 0x50
	counter   int64               // 0x58
}

type noneGaussianConfig struct {
	filterType int64   // 0x00 — 1,2,4,5
	divisor    int64   // 0x08 — размер окна
	mult       float64 // 0x10 — множитель границы (rodata 54de570 / 54de5e8 / 54de660)
	idx        int64   // 0x18 — 89 или 17
	extra      int64   // 0x20 — 10 или 2
	enabled    byte    // 0x28 — если 0, IsFiltered всегда false
}

// IsFiltered по дизассемблеру: offset (rbx); config.0x28==0→false; lock; буфер/sort; (offset+10)<=20 и counter<5→true, иначе counter=0.
func (f *NoneGaussianFilter) IsFiltered(offset int64) bool {
	if f == nil {
		return false
	}
	if f.config != nil && f.config.enabled == 0 {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if offset >= -10 && offset <= 10 && f.counter < 5 {
		f.counter++
		f.count++ // по дизассемблеру incq 0x00(rcx)
		return true
	}
	f.counter = 0
	return false
}

// IsFilteredCache по дизассемблеру __NoneGaussianFilter_.IsFilteredCache.txt: config.0x28; (rax).0 >= config.0x08→false;
// lock; band [predicted-diff, median+diff]; in band→false; (offset+10)<=20→false; counter>=5→false; else true.
func (f *NoneGaussianFilter) IsFilteredCache(offset int64) bool {
	if f == nil || f.config == nil {
		return false
	}
	if f.config.enabled == 0 {
		return false
	}
	if f.count >= f.config.divisor {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	diff := int64(float64(f.medianIdx-f.medianVal) * f.config.mult)
	lower := f.medianVal - diff
	upper := f.medianIdx + diff
	if offset >= lower && offset <= upper {
		return false
	}
	if offset >= -10 && offset <= 10 {
		return false
	}
	if f.counter >= 5 {
		return false
	}
	return true
}

// KalmanFilter по дизассемблеру (__KalmanFilter_.AddValue 0x4579da0): 0x10/0x18 = state (interface с методом по +0x18),
// 0x20 = другой интерфейс (метод по +0x20). AddValue(value int64): cvtsi2sd value→float64; newobject(slice или state);
// makeslice(float64,1) если nil; newobject(state): 1,1,1,1,slice; call state.Update(measurement); call result.Store().
type KalmanFilter struct {
	state      interface{} // 0x10 — объект с Update(measurement float64), по дизассемблеру call *0x18(rdx)
	storeFunc  interface{} // 0x20 — call *0x20(rax) после Update (сохранение результата)
}

// AddValue по дизассемблеру (__KalmanFilter_.AddValue): value (rbx)→float64; создание state с measurement;
// вызов 0x10(receiver).Update(state); вызов 0x20(result).Store(). Минимальная реконструкция: передаём value в state если есть метод AddValue.
func (k *KalmanFilter) AddValue(value int64) {
	if k == nil {
		return
	}
	measurement := float64(value)
	if upd, ok := k.state.(interface{ Update(float64) }); ok {
		upd.Update(measurement)
	}
}

func GetFilterFromConfig() {
	// TODO: реконструировать
}

func IsFiltered() {
	// TODO: реконструировать
}

func IsFilteredCache() {
	// TODO: реконструировать
}

// NewFilterConfig по дизассемблеру (NewFilterConfig@@Base 0x4579260): (filterType в AX, enabled в BL);
// switch rax 1/2/4/5; newobject(config); заполнение +0 type, +8 divisor, +0x10 mult (rodata), +0x18 idx, +0x20 extra, +0x28 enabled.
// Тип 1: divisor=100, mult=54de570, idx=89, extra=10. Тип 2: divisor=20, mult=54de570, idx=17, extra=2.
// Тип 4: divisor=100, mult=54de5e8, idx=89, extra=10. Тип 5: divisor=200, mult=54de660, idx=89, extra=10.
func NewFilterConfig(filterType int, enabled bool) *noneGaussianConfig {
	en := byte(0)
	if enabled {
		en = 1
	}
	switch filterType {
	case FilterTypeNoneGaussian1:
		return &noneGaussianConfig{filterType: 1, divisor: 100, mult: 0.02, idx: 89, extra: 10, enabled: en}
	case FilterTypeNoneGaussian2:
		return &noneGaussianConfig{filterType: 2, divisor: 20, mult: 0.02, idx: 17, extra: 2, enabled: en}
	case FilterTypeNoneGaussian4:
		return &noneGaussianConfig{filterType: 4, divisor: 100, mult: 0.05, idx: 89, extra: 10, enabled: en}
	case FilterTypeNoneGaussian5:
		return &noneGaussianConfig{filterType: 5, divisor: 200, mult: 0.05, idx: 89, extra: 10, enabled: en}
	default:
		return nil
	}
}

// NewKalmanFilter по дизассемблеру (NewKalmanFilter@@Base 0x4579400): создаёт config (0x30/0x38), state (матрицы 2x2, буферы),
// присваивает полям 0x30/0x38. Очень большая функция (2464 байт). Заглушка: возвращает пустой KalmanFilter.
func NewKalmanFilter(config interface{}, _ interface{}) *KalmanFilter {
	return &KalmanFilter{}
}

// NewNoneGausianFilter по дизассемблеру (NewNoneGausianFilter@@Base 0x4579ec0): NewFilterConfig(filterType);
// makeslice(int64, config.divisor) x2; concatstring2(prefix, "-filter"); NewLogger; newobject; заполнение 0x8/0x10/0x18, 0x20/0x28/0x30, 0x38=logger, 0x40=config.
func NewNoneGausianFilter(filterType int, loggerPrefix string) *NoneGaussianFilter {
	cfg := NewFilterConfig(filterType, true)
	if cfg == nil {
		return nil
	}
	n := int(cfg.divisor)
	if n <= 0 {
		n = 64
	}
	slice1 := make([]int64, n)
	slice2 := make([]int64, n)
	name := loggerPrefix + "-filter"
	logger := logging.NewLogger(name)
	return &NoneGaussianFilter{
		slicePtr: &slice1,
		sliceLen: n,
		slice2:   slice2,
		logger:   logger,
		config:   cfg,
	}
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

func inittask() {
	// TODO: реконструировать
}

// isOutlier по дизассемблеру __NoneGaussianFilter_.isOutlier.txt: 0x40=config, 0x48=median, 0x50=predicted;
// diff=(median-predicted)*config.0x10; band [predicted-diff, median+diff]; outside band && |offset|>10 → true.
func (f *NoneGaussianFilter) isOutlier(offset int64) bool {
	if f == nil || f.config == nil {
		return false
	}
	median := f.medianIdx   // 0x48
	predicted := f.medianVal // 0x50
	diff := int64(float64(median-predicted) * f.config.mult)
	upper := predicted + diff
	lower := predicted - diff
	if offset <= upper && offset >= lower {
		return false
	}
	// (offset+10) > 20 → offset > 10; для отрицательных (offset+10) как unsigned > 20 при offset < -10
	if offset > 10 || offset < -10 {
		return true
	}
	return false
}


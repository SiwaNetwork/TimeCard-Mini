package algos

import (
	"math"
	"sync"
	"time"
)

// Реконструировано по дизассемблеру linreg_servo.regress@@Base (code_analysis/disassembly/linreg_regress.txt).
// Окно 64 (0x40), проверка 0x620(rax)<0x40. Формула: gradient=(sum_xy-sum_x*sum_y/n)/(sum_xx-sum_x²/n),
// intercept=sum_y/n-gradient*sum_x/n. EMA с константой из .rodata 0x54de2a8 (≈0.02).

// LinRegEMAAlpha — коэффициент EMA для сглаживания slope (бинарник 0x54de2a8)
const LinRegEMAAlpha = 0.02

// LinReg — линейная регрессия (linreg_sample, regress)
type LinReg struct {
	mu           sync.Mutex
	windowSize   int
	samples      []float64
	timestamps   []float64
	index        int
	count        int
	lastSlope    float64
	lastTime     time.Time
	syncInterval float64
}

// linreg_servo — внутренняя структура
type linreg_servo struct {
	data          []float64
	timestamps    []float64
	refTime       float64
	refOffset     float64
	count         int
	index         int
	windowSize    int
	syncInterval  float64
	freqRatio     float64
	lastFrequency float64
}

// NewLinReg создаёт LinReg с окном 64
func NewLinReg() *LinReg {
	return &LinReg{
		windowSize: DefaultWindowSize,
		samples:    make([]float64, DefaultWindowSize),
		timestamps: make([]float64, DefaultWindowSize),
	}
}

// linreg_servo_create создаёт linreg_servo
func linreg_servo_create(windowSize int) *linreg_servo {
	if windowSize <= 0 {
		windowSize = DefaultWindowSize
	}
	return &linreg_servo{
		data:       make([]float64, windowSize),
		timestamps: make([]float64, windowSize),
		windowSize: windowSize,
	}
}

// CalculateFrequency — расчёт коррекции через slope
func (lr *LinReg) CalculateFrequency(offsetNs float64, dt time.Duration) float64 {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Добавляем sample
	ts := float64(time.Now().UnixNano())
	lr.samples[lr.index] = offsetNs
	lr.timestamps[lr.index] = ts
	lr.index = (lr.index + 1) % lr.windowSize
	if lr.count < lr.windowSize {
		lr.count++
	}

	if lr.count < 2 {
		return 0
	}

	// Регрессия
	slope := lr.regress()
	lr.lastSlope = slope
	lr.lastTime = time.Now()

	// slope в ns/ns, конвертируем в частоту
	return slope
}

// regress — линейная регрессия по окну
func (lr *LinReg) regress() float64 {
	n := float64(lr.count)
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < lr.count; i++ {
		x := lr.timestamps[i]
		y := lr.samples[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// slope = (n*Σxy - Σx*Σy) / (n*Σx² - (Σx)²)
	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-15 {
		return 0
	}
	slope := (n*sumXY - sumX*sumY) / denom
	return slope
}

// linreg_sample добавляет sample
func (ls *linreg_servo) linreg_sample(offset, localTs float64) float64 {
	ls.add_sample(offset, localTs)
	if ls.count < 2 {
		return 0
	}
	return ls.regress()
}

// add_sample добавляет точку
func (ls *linreg_servo) add_sample(offset, ts float64) {
	ls.data[ls.index] = offset
	ls.timestamps[ls.index] = ts
	ls.index = (ls.index + 1) % ls.windowSize
	if ls.count < ls.windowSize {
		ls.count++
	}
}

// regress выполняет регрессию. По дизассемблеру (linreg_servo.regress 0x457d240): 0x620(rax)=count, cmp 0x40 → panic если count>=64; 0x608=refTime, данные (ts, offset) по 0x8+idx*24; gradient=(sum_xy-sum_x*sum_y/n)/(sum_xx-sum_x²/n), intercept=sum_y/n-gradient*sum_x/n; результат slope в 0x638, intercept в 0x640; EMA по 0x648 с alpha 0x54de2a8 (LinRegEMAAlpha).
func (ls *linreg_servo) regress() float64 {
	n := float64(ls.count)
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < ls.count; i++ {
		x := ls.timestamps[i]
		y := ls.data[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-15 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// linreg_reset сбрасывает servo
func (ls *linreg_servo) linreg_reset() {
	ls.count = 0
	ls.index = 0
	ls.refTime = 0
	ls.refOffset = 0
}

// linreg_leap обрабатывает leap second
func (ls *linreg_servo) linreg_leap() {
	ls.linreg_reset()
}

// linreg_sync_interval устанавливает интервал
func (ls *linreg_servo) linreg_sync_interval(interval float64) {
	ls.syncInterval = interval
}

// linreg_rate_ratio возвращает rate ratio
func (ls *linreg_servo) linreg_rate_ratio() float64 {
	return ls.freqRatio
}

// update_reference обновляет референс
func (ls *linreg_servo) update_reference() {
	if ls.count > 0 {
		ls.refTime = ls.timestamps[(ls.index-1+ls.windowSize)%ls.windowSize]
		ls.refOffset = ls.data[(ls.index-1+ls.windowSize)%ls.windowSize]
	}
}

// update_size изменяет размер окна
func (ls *linreg_servo) update_size(newSize int) {
	if newSize <= 0 || newSize == ls.windowSize {
		return
	}
	newData := make([]float64, newSize)
	newTs := make([]float64, newSize)
	copy(newData, ls.data)
	copy(newTs, ls.timestamps)
	ls.data = newData
	ls.timestamps = newTs
	ls.windowSize = newSize
	if ls.count > newSize {
		ls.count = newSize
	}
	if ls.index >= newSize {
		ls.index = 0
	}
}

// move_reference перемещает референс
func (ls *linreg_servo) move_reference(delta float64) {
	ls.refOffset += delta
}

// ResetServo сбрасывает состояние
func (lr *LinReg) ResetServo() {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.count = 0
	lr.index = 0
	lr.lastSlope = 0
}

// UpdatePPS по дизассемблеру (__LinReg_.UpdatePPS): аргумент bl (scale byte); pow(10, scale); 54de498/result → 54de498/val → 0x6e8(inner).
// Внутренняя структура: 0x6e8 = pow(10, scale). Здесь задаём syncInterval = pow(10, scale) для внутреннего servo.
func (lr *LinReg) UpdatePPS(scale byte) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.syncInterval = math.Pow(10, float64(scale))
}

// SetTSType по дизассемблеру (__LinReg_.SetTSType): ret — пустой метод.
func (lr *LinReg) SetTSType() {}

// UpdateClockFreq по дизассемблеру (__LinReg_.UpdateClockFreq@@Base): запись по смещению 0x6e0 (внутренняя структура).
// Минимальная реконструкция: пустой метод (hostclocks вызывает для commitFrequency).
func (lr *LinReg) UpdateClockFreq(freq float64) { _ = freq }

// UpdateScaleFromStore по дизассемблеру updateAlgoCoefficients: обновление коэффициентов в store для типа linreg (1).
func (lr *LinReg) UpdateScaleFromStore(scale byte) {
	store := GetCoefficientStore()
	c := store.GetCoefficientsForTypeInt(1, scale)
	if c != nil {
		store.ChangeSteeringCoefficientsInt(1, c)
	}
}

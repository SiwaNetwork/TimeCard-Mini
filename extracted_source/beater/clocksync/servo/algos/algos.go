// Package algos — алгоритмы servo (PID, PI, LinReg).
// Реконструировано по дизассемблеру бинарника timebeat-2.2.20 (code_analysis/disassembly/AlgoPID_*.txt).
// Константы и таблицы извлечены из .rodata/.noptrdata.
package algos

import (
	"math"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos/tuner"
)

// Константы из бинарника
const (
	DefaultWindowSize = 64 // LinReg
	MaxDCoeffs        = 3  // PID D-массив
)

// DefaultAlgoCoefficients — дефолтные коэффициенты (0x7c1a040)
var DefaultAlgoCoefficients = struct {
	Kp float64
	Ki float64
	Kd float64
}{
	Kp: 0.5,
	Ki: 0.5946035575013605,
	Kd: 0.7071067811865475, // 1/√2
}

// dComponentLookup — D-массив для PID (из бинарника 0x7c18b30 .noptrdata: три float64)
var dComponentLookup = [MaxDCoeffs]float64{1.0, 1.0, 1.0}

// logScaleD — множитель для log при индексе D (бинарник 0x54de3a8 .rodata = 1/ln(10))
const logScaleD = 0.4342944819032518

// AlgoPID — PID алгоритм (CalculateNewFrequency, adjustDComponent)
type AlgoPID struct {
	mu              sync.Mutex
	kp, ki, kd      float64
	integral        float64
	lastError       float64
	lastTime        time.Time
	dCoeffs         [MaxDCoeffs]float64
	movingMinimum   *MovingMinimum
	bestFitFiltered *BestFitFiltered
	maxAdjustment   float64
	minAdjustment   float64
	debug           bool
}

// NewAlgoPID создаёт PID с дефолтными коэффициентами
func NewAlgoPID() *AlgoPID {
	return &AlgoPID{
		kp:            DefaultAlgoCoefficients.Kp,
		ki:            DefaultAlgoCoefficients.Ki,
		kd:            DefaultAlgoCoefficients.Kd,
		dCoeffs:       dComponentLookup,
		maxAdjustment: 100e-6,
		minAdjustment: -100e-6,
	}
}

// NewAlgoPIDWithTuner создаёт PID с тюнером
func NewAlgoPIDWithTuner(t *tuner.OnlinePIDTuner) *AlgoPID {
	p := NewAlgoPID()
	// TODO: интеграция с tuner
	return p
}

// SetCoeffs задаёт Kp, Ki, Kd (для конфига).
func (p *AlgoPID) SetCoeffs(kp, ki, kd float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kp > 0 {
		p.kp = kp
	}
	if ki >= 0 {
		p.ki = ki
	}
	if kd >= 0 {
		p.kd = kd
	}
}

// CalculateNewFrequency — основной метод расчёта коррекции
func (p *AlgoPID) CalculateNewFrequency(offsetNs float64, dt time.Duration) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	dtSec := dt.Seconds()
	if dtSec <= 0 {
		return 0
	}

	// P
	pTerm := p.kp * offsetNs

	// I
	p.integral += offsetNs * dtSec
	iTerm := p.ki * p.integral

	// D с адаптивным коэффициентом (adjustDComponent)
	derivative := (offsetNs - p.lastError) / dtSec
	dTerm := p.adjustDComponent(offsetNs, derivative)

	p.lastError = offsetNs
	p.lastTime = time.Now()

	freq := pTerm + iTerm + dTerm
	return p.enforceAdjustmentLimit(freq)
}

// adjustDComponent по дизассемблеру (0x457fc80): receiver+0x28=ptr, log(abs(value)), logScaleD*log, индекс 0..2, dComponentLookup[idx]*derivative; в бинарнике также вычитание из raw и idiv. Здесь — упрощённо: D = dCoeffs[idx]*derivative.
func (p *AlgoPID) adjustDComponent(offset, derivative float64) float64 {
	absOff := math.Abs(offset)
	if absOff < 1 {
		absOff = 1
	}
	logVal := math.Log(absOff)
	idx := int(logScaleD * logVal)
	if idx < 0 {
		idx = 0
	}
	if idx >= MaxDCoeffs {
		idx = MaxDCoeffs - 1
	}
	return p.dCoeffs[idx] * derivative
}

// enforceAdjustmentLimit по дизассемблеру (0x457fdc0): receiver+0x58/0x68 = limits; limit=min(58,68) или 0x20(ptr); clamp(adjustment) в [-limit,limit]; return (clamped, wasClamped). Здесь — clamp freq в [minAdjustment,maxAdjustment].
func (p *AlgoPID) enforceAdjustmentLimit(freq float64) float64 {
	if freq > p.maxAdjustment {
		return p.maxAdjustment
	}
	if freq < p.minAdjustment {
		return p.minAdjustment
	}
	return freq
}

// CalculateFrequency — совместимость
func (p *AlgoPID) CalculateFrequency(offsetNs float64, dt time.Duration) float64 {
	return p.CalculateNewFrequency(offsetNs, dt)
}

// GetIntegral возвращает интегральную компоненту
func (p *AlgoPID) GetIntegral() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.integral
}

// SetMovingMinimum устанавливает сглаживание
func (p *AlgoPID) SetMovingMinimum(mm *MovingMinimum) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.movingMinimum = mm
}

// UpdatePPS по дизассемблеру: hostclocks вызывает с scale (byte); реализует algoPPSUpdater.
func (p *AlgoPID) UpdatePPS(scale byte) {
	_ = scale
}

// ResetServo сбрасывает состояние
func (p *AlgoPID) ResetServo() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.integral = 0
	p.lastError = 0
}

// UpdateClockFreq по дизассемблеру (__AlgoPID_.UpdateClockFreq@@Base): hostclocks вызывает после SetHoldoverFrequency для commitFrequency.
func (p *AlgoPID) UpdateClockFreq(freq float64) { _ = freq }

// UpdateScaleFromStore по дизассемблеру updateAlgoCoefficients: обновление коэффициентов в store для типа pid (0).
func (p *AlgoPID) UpdateScaleFromStore(scale byte) {
	store := GetCoefficientStore()
	c := store.GetCoefficientsForTypeInt(0, scale)
	if c != nil {
		store.ChangeSteeringCoefficientsInt(0, c)
	}
}

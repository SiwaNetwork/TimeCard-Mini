package servo

import (
	"math"
	"time"
)

// Algorithm — интерфейс алгоритма синхронизации часов (PID, PI, LinReg)
type Algorithm interface {
	Update(errorNs float64, dt time.Duration) (freqAdjustment float64)
	Reset()
}

// PID — PID регулятор (на основе анализа shiwatime: AlgoPID 0x41c8680).
// Опционально: DCoeffs[3] — массив D-коэффициентов по индексу от log(abs(offset)),
// адрес 0x770a430 в бинарнике.
type PID struct {
	Kp, Ki, Kd       float64
	Integral         float64
	LastError        float64
	LastTime         time.Time
	MaxIntegral      float64
	MaxAdjustment    float64
	MinAdjustment    float64
	DCoeffs          [3]float64 // D-массив из анализа; если все 0 — используется Kd
}

// NewPID создаёт PID с разумными коэффициентами по умолчанию
func NewPID(kp, ki, kd float64) *PID {
	if kp == 0 && ki == 0 && kd == 0 {
		kp, ki, kd = 0.1, 0.01, 0.001
	}
	return &PID{
		Kp:             kp,
		Ki:             ki,
		Kd:             kd,
		MaxIntegral:    1e9,
		MaxAdjustment:  100e-6,
		MinAdjustment:  -100e-6,
	}
}

// Update возвращает коррекцию частоты (относительная доля, например 1e-6 = 1 ppm)
func (p *PID) Update(errorNs float64, dt time.Duration) float64 {
	dtSec := dt.Seconds()
	if dtSec <= 0 {
		return 0
	}
	p.Integral += errorNs * dtSec
	if p.Integral > p.MaxIntegral {
		p.Integral = p.MaxIntegral
	} else if p.Integral < -p.MaxIntegral {
		p.Integral = -p.MaxIntegral
	}
	derivative := (errorNs - p.LastError) / dtSec
	p.LastError = errorNs
	p.LastTime = time.Now()

	var dTerm float64
	if p.DCoeffs[0] != 0 || p.DCoeffs[1] != 0 || p.DCoeffs[2] != 0 {
		// D-массив из анализа: индекс от log(abs(value)), value = offset
		absOff := math.Abs(errorNs)
		if absOff < 1 {
			absOff = 1
		}
		logVal := math.Log(absOff)
		idx := int(logVal / 5) // масштаб ~1-20 для типичных offset 1e6-1e9
		if idx < 0 {
			idx = 0
		} else if idx > 2 {
			idx = 2
		}
		dTerm = p.DCoeffs[idx] * derivative
	} else {
		dTerm = p.Kd * derivative
	}

	out := p.Kp*errorNs + p.Ki*p.Integral + dTerm
	if out > p.MaxAdjustment {
		out = p.MaxAdjustment
	} else if out < p.MinAdjustment {
		out = p.MinAdjustment
	}
	return out
}

// Reset сбрасывает интеграл и последнюю ошибку
func (p *PID) Reset() {
	p.Integral = 0
	p.LastError = 0
}

// PI — PI регулятор (без D), аналог Pi 0x41c8310 (shiwatime).
// Поддерживает два режима:
// - стандартный: I += Ki*offset*dt (UseShiwatimeFormula=false)
// - shiwatime-style: I += (IntegralTarget - I) * offset_diff / time_diff (UseShiwatimeFormula=true)
type PI struct {
	Kp, Ki           float64
	Integral         float64
	LastError        float64
	LastTime         time.Time
	MaxIntegral      float64
	MaxAdjustment    float64
	UseShiwatimeFormula bool // true = формула из бинарника: I += (target - I) * offset_diff / time_diff
	IntegralTarget   float64 // 1e9 по анализу (0x3b9aca00), константа для shiwatime-style
}

// NewPI создаёт PI регулятор (стандартный режим)
func NewPI(kp, ki float64) *PI {
	if kp == 0 && ki == 0 {
		kp, ki = 0.1, 0.01
	}
	return &PI{
		Kp:               kp,
		Ki:               ki,
		MaxIntegral:      1e9,
		MaxAdjustment:    100e-6,
		IntegralTarget:   1e9, // из анализа: 0x3b9aca00
	}
}

// NewPIShiwatime создаёт PI в режиме shiwatime: I += (1e9 - I) * offset_diff / time_diff
func NewPIShiwatime(kp float64) *PI {
	if kp == 0 {
		kp = 0.1
	}
	return &PI{
		Kp:                 kp,
		Ki:                 0,
		MaxIntegral:        1e9,
		MaxAdjustment:      100e-6,
		UseShiwatimeFormula: true,
		IntegralTarget:     1e9, // 0x3b9aca00 из pi_sample
	}
}

// Update возвращает коррекцию частоты
func (pi *PI) Update(errorNs float64, dt time.Duration) float64 {
	dtSec := dt.Seconds()
	if dtSec <= 0 {
		return 0
	}
	if pi.UseShiwatimeFormula {
		// Формула из pi_sample (0x41c8da0): I += (target - I) * offset_diff / time_diff
		// Константа 0x3b9aca00 = 1e9. Масштаб: delta_I = (target-I)*offsetDiff/timeDiffNs, ограничиваем рост
		offsetDiff := errorNs - pi.LastError
		timeDiffNs := dt.Nanoseconds()
		if timeDiffNs > 0 && !pi.LastTime.IsZero() {
			ratio := offsetDiff / float64(timeDiffNs)
			// Ограничиваем ratio для устойчивости (|ratio| < 1)
			if ratio > 1 {
				ratio = 1
			} else if ratio < -1 {
				ratio = -1
			}
			delta := (pi.IntegralTarget - pi.Integral) * ratio * 1e-9 * dtSec
			pi.Integral += delta
			if pi.Integral > pi.MaxIntegral {
				pi.Integral = pi.MaxIntegral
			} else if pi.Integral < -pi.MaxIntegral {
				pi.Integral = -pi.MaxIntegral
			}
		} else if pi.LastTime.IsZero() {
			pi.Integral = errorNs * dtSec * 0.01 // начальная установка
		}
	} else {
		pi.Integral += errorNs * dtSec
		if pi.Integral > pi.MaxIntegral {
			pi.Integral = pi.MaxIntegral
		} else if pi.Integral < -pi.MaxIntegral {
			pi.Integral = -pi.MaxIntegral
		}
	}
	pi.LastError = errorNs
	pi.LastTime = time.Now()
	var out float64
	if pi.UseShiwatimeFormula {
		out = pi.Kp*errorNs + pi.Integral*1e-9 // integral в ns, переводим в ppm-подобную величину
	} else {
		out = pi.Kp*errorNs + pi.Ki*pi.Integral
	}
	if out > pi.MaxAdjustment {
		out = pi.MaxAdjustment
	} else if out < -pi.MaxAdjustment {
		out = -pi.MaxAdjustment
	}
	return out
}

// Reset сбрасывает интеграл
func (pi *PI) Reset() {
	pi.Integral = 0
	pi.LastError = 0
	pi.LastTime = time.Time{}
}

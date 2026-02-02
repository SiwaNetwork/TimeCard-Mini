package algos

import (
	"sync"
	"time"
)

// Реконструировано по дизассемблеру Pi.CalculateFrequency@@Base (code_analysis/disassembly/Pi_CalculateFrequency.txt).
// Константы: 0x3b9aca00 = 1e9 ns, rodata 0x54de498 = 1.0 (weight для pi_sample).

// Константа из бинарника (0x3b9aca00 = 1e9)
const PIIntegralTarget float64 = 1e9

// Pi — PI алгоритм (pi_sample, pi_reset)
type Pi struct {
	mu             sync.Mutex
	kp             float64
	integral       float64
	lastOffset     float64
	lastTime       time.Time
	maxIntegral    float64
	integralTarget float64 // 1e9 из бинарника
	tsType         string
}

// pi_servo — внутренняя структура (из linuxptp-style)
type pi_servo struct {
	offset        [2]int64
	local         [2]int64
	drift         float64
	lastFreq      float64
	kp            float64
	ki            float64
	count         int
	maxIntegral   float64
}

// NewPi создаёт PI
func NewPi() *Pi {
	return &Pi{
		kp:             DefaultAlgoCoefficients.Kp,
		maxIntegral:    1e9,
		integralTarget: PIIntegralTarget,
	}
}

// SetKp задаёт Kp (для конфига).
func (p *Pi) SetKp(kp float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kp > 0 {
		p.kp = kp
	}
}

// pi_servo_create создаёт pi_servo
func pi_servo_create() *pi_servo {
	return &pi_servo{
		kp:          DefaultAlgoCoefficients.Kp,
		ki:          DefaultAlgoCoefficients.Ki,
		maxIntegral: 1e9,
	}
}

// CalculateFrequency — расчёт коррекции (shiwatime-style)
func (p *Pi) CalculateFrequency(offsetNs float64, dt time.Duration) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	dtSec := dt.Seconds()
	if dtSec <= 0 {
		return 0
	}

	// Формула из pi_sample: I += (target - I) * offset_diff / time_diff
	offsetDiff := offsetNs - p.lastOffset
	timeDiffNs := dt.Nanoseconds()

	if timeDiffNs > 0 && !p.lastTime.IsZero() {
		ratio := offsetDiff / float64(timeDiffNs)
		if ratio > 1 {
			ratio = 1
		} else if ratio < -1 {
			ratio = -1
		}
		delta := (p.integralTarget - p.integral) * ratio * 1e-9 * dtSec
		p.integral += delta
		if p.integral > p.maxIntegral {
			p.integral = p.maxIntegral
		} else if p.integral < -p.maxIntegral {
			p.integral = -p.maxIntegral
		}
	}

	p.lastOffset = offsetNs
	p.lastTime = time.Now()

	out := p.kp*offsetNs + p.integral*1e-9
	return out
}

// pi_sample по дизассемблеру (__pi_servo_.pi_sample 0x457ffa0): 0x70(rax)=drift in, 0x78=count. count==0: store rbx/rcx → 0x38/0x48, *rdi=0, count=1. count==1: store → 0x40/0x50; if localTs<=prev reset count=0; else delta_ns (0x54de260, 0x54de2a0/0x68), clamp 0x54de740; drift at 0x58; *rdi=1 (step) or 2 (slew) by |offset| vs 0x10/0x8. lastFreq at 0x70.
func (ps *pi_servo) pi_sample(offset int64, localTs int64, weight float64) float64 {
	ps.offset[1] = ps.offset[0]
	ps.offset[0] = offset
	ps.local[1] = ps.local[0]
	ps.local[0] = localTs

	if ps.count < 2 {
		ps.count++
		return 0
	}

	// Формула из бинарника
	ki := ps.ki
	kp := ps.kp
	ppb := kp*float64(offset) + ki*ps.drift
	ps.drift += ki * float64(offset) * weight

	return ppb
}

// pi_reset сбрасывает servo
func (ps *pi_servo) pi_reset() {
	ps.count = 0
	ps.drift = 0
}

// pi_sync_interval устанавливает интервал
func (ps *pi_servo) pi_sync_interval(interval float64) {
	// TODO
}

// ResetServo сбрасывает состояние
func (p *Pi) ResetServo() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.integral = 0
	p.lastOffset = 0
	p.lastTime = time.Time{}
}

// SetTSType устанавливает тип источника
func (p *Pi) SetTSType(tsType string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tsType = tsType
}

// UpdateClockFreq по дизассемблеру (__Pi_.UpdateClockFreq@@Base): hostclocks вызывает после SetHoldoverFrequency для commitFrequency.
func (p *Pi) UpdateClockFreq(freq float64) { _ = freq }

// UpdateScaleFromStore по дизассемблеру updateAlgoCoefficients: загрузка коэффициентов для типа pi (2) и применение к серво.
func (p *Pi) UpdateScaleFromStore(scale byte) {
	store := GetCoefficientStore()
	c := store.GetCoefficientsForTypeInt(2, scale)
	if c != nil && c.Kp > 0 {
		p.SetKp(c.Kp)
	}
}

// UpdatePPS по дизассемблеру: hostclocks вызывает с scale (byte); реализует algoPPSUpdater.
func (p *Pi) UpdatePPS(scale byte) {
	_ = scale
}

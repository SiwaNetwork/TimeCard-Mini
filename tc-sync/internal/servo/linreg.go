package servo

import "time"

// LinRegWindow — размер окна линейной регрессии (по анализу shiwatime: 64)
const LinRegWindow = 64

// LinReg — servo на основе линейной регрессии по окну смещений (аналог LinReg 0x41c6c00).
// x = время в секундах (накопленное), y = offset_ns; slope = ns/s → freq = slope/1e9.
type LinReg struct {
	xs      [LinRegWindow]float64 // время в секундах от начала
	ys      [LinRegWindow]float64 // смещение в наносекундах
	n       int
	idx     int
	filled  bool
	timeSec float64 // накопленное время для следующего сэмпла
}

// NewLinReg создаёт LinReg servo.
func NewLinReg() *LinReg {
	return &LinReg{}
}

// Update добавляет сэмпл (errorNs, dt) в окно и возвращает коррекцию частоты (относительная доля).
// После заполнения окна считает slope регрессии offset vs time (ns/s) и возвращает slope/1e9.
func (l *LinReg) Update(errorNs float64, dt time.Duration) float64 {
	dtSec := dt.Seconds()
	if dtSec <= 0 {
		dtSec = 1.0
	}
	l.xs[l.idx] = l.timeSec
	l.ys[l.idx] = errorNs
	l.timeSec += dtSec
	l.idx++
	if l.idx >= LinRegWindow {
		l.idx = 0
		l.filled = true
	}
	if l.n < LinRegWindow {
		l.n++
	}
	if !l.filled || l.n < 4 {
		return 0
	}
	n := float64(l.n)
	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < l.n; i++ {
		sumX += l.xs[i]
		sumY += l.ys[i]
		sumXY += l.xs[i] * l.ys[i]
		sumX2 += l.xs[i] * l.xs[i]
	}
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	slope := (n*sumXY - sumX*sumY) / denom // ns/s
	freq := slope / 1e9
	if freq > 100e-6 {
		freq = 100e-6
	} else if freq < -100e-6 {
		freq = -100e-6
	}
	return freq
}

// Reset сбрасывает окно и накопленное время
func (l *LinReg) Reset() {
	l.n = 0
	l.idx = 0
	l.filled = false
	l.timeSec = 0
}

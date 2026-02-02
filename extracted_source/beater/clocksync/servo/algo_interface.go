// Интерфейс алгоритма servo и адаптеры к algos (PID, PI, LinReg).
package servo

import (
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/algos"
)

// Algo — общий интерфейс алгоритма коррекции (offset → частота).
type Algo interface {
	Calculate(offsetNs float64, dt time.Duration) float64
}

type algoPID struct{ *algos.AlgoPID }
type algoPI struct{ *algos.Pi }
type algoLinReg struct{ *algos.LinReg }

func (a *algoPID) Calculate(offsetNs float64, dt time.Duration) float64 {
	return a.CalculateNewFrequency(offsetNs, dt)
}
func (a *algoPI) Calculate(offsetNs float64, dt time.Duration) float64 {
	return a.CalculateFrequency(offsetNs, dt)
}
func (a *algoLinReg) Calculate(offsetNs float64, dt time.Duration) float64 {
	return a.CalculateFrequency(offsetNs, dt)
}

// ServoAlgoConfig — параметры алгоритма (из конфига, без импорта config).
type ServoAlgoConfig struct {
	Algorithm string
	Kp        float64
	Ki        float64
	Kd        float64
}

// NewAlgo создаёт Algo по имени алгоритма и коэффициентам.
func NewAlgo(algorithm string, kp, ki, kd float64) Algo {
	if kp == 0 {
		kp = 0.5
	}
	if ki == 0 {
		ki = algos.DefaultAlgoCoefficients.Ki
	}
	if kd == 0 {
		kd = algos.DefaultAlgoCoefficients.Kd
	}
	switch algorithm {
	case "pi", "pi_shiwatime":
		p := algos.NewPi()
		p.SetKp(kp)
		return &algoPI{p}
	case "linreg":
		return &algoLinReg{algos.NewLinReg()}
	default:
		pid := algos.NewAlgoPID()
		pid.SetCoeffs(kp, ki, kd)
		return &algoPID{pid}
	}
}

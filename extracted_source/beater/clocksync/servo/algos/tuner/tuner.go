// Package tuner — автоматическая подстройка PID
package tuner

import (
	"math"
	"sync"
)

// OnlinePIDTuner — онлайн-тюнер PID
type OnlinePIDTuner struct {
	mu       sync.Mutex
	kp, ki, kd float64
	interval   float64
	kalman     *KalmanFilter
}

// NewOnlinePIDTuner создаёт тюнер
func NewOnlinePIDTuner(kp, ki, kd float64) *OnlinePIDTuner {
	return &OnlinePIDTuner{
		kp:     kp,
		ki:     ki,
		kd:     kd,
		kalman: NewKalmanFilter(0.1, 1.0),
	}
}

// Update обновляет тюнер
func (t *OnlinePIDTuner) Update(error, dt float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// TODO: реализовать адаптивную подстройку
}

// Predict возвращает предсказание
func (t *OnlinePIDTuner) Predict() (kp, ki, kd float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.kp, t.ki, t.kd
}

// SetInterval устанавливает интервал
func (t *OnlinePIDTuner) SetInterval(interval float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.interval = interval
}

// KalmanFilter — фильтр Калмана
type KalmanFilter struct {
	mu         sync.Mutex
	q          float64 // process noise
	r          float64 // measurement noise
	x          float64 // estimate
	p          float64 // error covariance
	k          float64 // kalman gain
	initialized bool
}

// NewKalmanFilter создаёт фильтр
func NewKalmanFilter(processNoise, measurementNoise float64) *KalmanFilter {
	return &KalmanFilter{
		q: processNoise,
		r: measurementNoise,
		p: 1.0,
	}
}

// Update обновляет фильтр
func (kf *KalmanFilter) Update(measurement float64) float64 {
	kf.mu.Lock()
	defer kf.mu.Unlock()

	if !kf.initialized {
		kf.x = measurement
		kf.initialized = true
		return kf.x
	}

	// Predict
	kf.p += kf.q

	// Update
	kf.k = kf.p / (kf.p + kf.r)
	kf.x += kf.k * (measurement - kf.x)
	kf.p *= (1 - kf.k)

	return kf.x
}

// clamp ограничивает значение
func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

package algos

import (
	"math"
	"sync"
)

// StdDev — стандартное отклонение (Welford's algorithm)
type StdDev struct {
	mu    sync.Mutex
	n     int
	mean  float64
	m2    float64
	data  []float64
}

// NewStdDev создаёт StdDev
func NewStdDev() *StdDev {
	return &StdDev{}
}

// AddValue добавляет значение
func (s *StdDev) AddValue(x float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	delta := x - s.mean
	s.mean += delta / float64(s.n)
	delta2 := x - s.mean
	s.m2 += delta * delta2
	s.data = append(s.data, x)
}

// GetMean возвращает среднее
func (s *StdDev) GetMean() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mean
}

// Mean алиас
func (s *StdDev) Mean() float64 { return s.GetMean() }

// GetVariance возвращает дисперсию
func (s *StdDev) GetVariance() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n < 2 {
		return 0
	}
	return s.m2 / float64(s.n-1)
}

// Variance алиас
func (s *StdDev) Variance() float64 { return s.GetVariance() }

// GetStdDev возвращает стандартное отклонение
func (s *StdDev) GetStdDev() float64 {
	return math.Sqrt(s.GetVariance())
}

// GetN возвращает количество
func (s *StdDev) GetN() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// GetData возвращает данные
func (s *StdDev) GetData() []float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float64{}, s.data...)
}

// GetResults возвращает результаты
func (s *StdDev) GetResults() (mean, stddev float64, n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v float64
	if s.n >= 2 {
		v = s.m2 / float64(s.n-1)
	}
	return s.mean, math.Sqrt(v), s.n
}

// Weight возвращает вес
func (s *StdDev) Weight() float64 {
	if s.n == 0 {
		return 0
	}
	return 1 / s.GetStdDev()
}

// MovingMinimum — скользящий минимум
type MovingMinimum struct {
	mu         sync.Mutex
	windowSize int
	data       []float64
	index      int
	count      int
}

// NewMovingMinimum создаёт MovingMinimum
func NewMovingMinimum(windowSize int) *MovingMinimum {
	return &MovingMinimum{
		windowSize: windowSize,
		data:       make([]float64, windowSize),
	}
}

// Sample добавляет значение и возвращает минимум
func (mm *MovingMinimum) Sample(value float64) float64 {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.data[mm.index] = value
	mm.index = (mm.index + 1) % mm.windowSize
	if mm.count < mm.windowSize {
		mm.count++
	}
	min := mm.data[0]
	for i := 1; i < mm.count; i++ {
		if mm.data[i] < min {
			min = mm.data[i]
		}
	}
	return min
}

// Reset сбрасывает
func (mm *MovingMinimum) Reset() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.count = 0
	mm.index = 0
}

// MovingMedian — скользящая медиана.
// Sample по дизассемблеру (__MovingMedian_.Sample.txt): флаг debug 0x48; буфер 0x30/0x38, индекс 0x10;
// отсортированные индексы 0x18/0x20; медиана хранится в 0x50 (GetMovingMedian возвращает 0x50(rax)).
type MovingMedian struct {
	mu          sync.Mutex
	windowSize  int
	data        []float64
	sorted      []float64
	index       int
	count       int
	medianValue float64 // 0x50 по дизассемблеру GetMovingMedian
}

// NewMovingMedian создаёт MovingMedian
func NewMovingMedian(windowSize int) *MovingMedian {
	return &MovingMedian{
		windowSize: windowSize,
		data:       make([]float64, windowSize),
		sorted:     make([]float64, windowSize),
	}
}

// Sample по дизассемблеру: добавляет значение (int64, например offset в ns) и возвращает медиану (int64).
func (mm *MovingMedian) Sample(value int64) int64 {
	return int64(mm.Add(float64(value)))
}

// Add добавляет значение
func (mm *MovingMedian) Add(value float64) float64 {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.data[mm.index] = value
	mm.index = (mm.index + 1) % mm.windowSize
	if mm.count < mm.windowSize {
		mm.count++
	}
	// Копируем и сортируем
	copy(mm.sorted[:mm.count], mm.data[:mm.count])
	for i := 1; i < mm.count; i++ {
		for j := i; j > 0 && mm.sorted[j] < mm.sorted[j-1]; j-- {
			mm.sorted[j], mm.sorted[j-1] = mm.sorted[j-1], mm.sorted[j]
		}
	}
	var med float64
	if mm.count%2 == 0 {
		med = (mm.sorted[mm.count/2-1] + mm.sorted[mm.count/2]) / 2
	} else {
		med = mm.sorted[mm.count/2]
	}
	mm.medianValue = med
	return med
}

// GetMovingMedian по дизассемблеру (__MovingMedian_.GetMovingMedian): mov 0x50(%rax),%rax — возврат поля medianValue.
func (mm *MovingMedian) GetMovingMedian() float64 {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.medianValue
}

// Reset по дизассемблеру (__MovingMedian_.Reset): movq $0, 0(rax); movq $0, 0x10(rax) — обнуление полей count и index.
func (mm *MovingMedian) Reset() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.count = 0
	mm.index = 0
}

// printDebug для отладки
func (mm *MovingMedian) printDebug() {}

// RMS — среднеквадратичное
type RMS struct {
	mu   sync.Mutex
	sum  float64
	n    int
}

// Add добавляет значение
func (r *RMS) Add(value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sum += value * value
	r.n++
}

// Get возвращает RMS
func (r *RMS) Get() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.n == 0 {
		return 0
	}
	return math.Sqrt(r.sum / float64(r.n))
}

// Reset сбрасывает
func (r *RMS) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sum = 0
	r.n = 0
}

// SteadyState — определение устойчивого состояния
type SteadyState struct {
	mu          sync.Mutex
	threshold   float64
	windowSize  int
	values      []float64
	derivatives []float64
	index       int
	count       int
	armed       bool
	steady      bool
}

// NewSteadyState создаёт SteadyState
func NewSteadyState(threshold float64, windowSize int) *SteadyState {
	return &SteadyState{
		threshold:   threshold,
		windowSize:  windowSize,
		values:      make([]float64, windowSize),
		derivatives: make([]float64, windowSize),
	}
}

// Update обновляет состояние
func (ss *SteadyState) Update(value float64) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.addValue(value)
	ss.updateStateIfRequired()
}

func (ss *SteadyState) addValue(value float64) {
	oldValue := ss.values[ss.index]
	ss.values[ss.index] = value
	if ss.count > 0 {
		ss.addDerivative(value - oldValue)
	}
	ss.index = (ss.index + 1) % ss.windowSize
	if ss.count < ss.windowSize {
		ss.count++
	}
}

func (ss *SteadyState) addDerivative(d float64) {
	ss.derivatives[ss.index] = d
}

func (ss *SteadyState) updateStateIfRequired() {
	if ss.count < ss.windowSize {
		return
	}
	var sumAbs float64
	for i := 0; i < ss.count; i++ {
		sumAbs += math.Abs(ss.derivatives[i])
	}
	avg := sumAbs / float64(ss.count)
	ss.steady = avg < ss.threshold
}

// IsArmed проверяет готовность
func (ss *SteadyState) IsArmed() bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.armed
}

// IsSteady проверяет устойчивость
func (ss *SteadyState) IsSteady() bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.steady
}

// BestFitFiltered — фильтр линейной аппроксимации.
// GetLeastSquaresGradientFiltered по дизассемблеру (BestFitFiltered_GetLeastSquaresGradientFiltered.txt):
// slice пар (time, offset), sum_x/sum_y/sum_xy/sum_xx; gradient = (n*sum_xy - sum_x*sum_y)/(n*sum_xx - sum_x²);
// clamp по .rodata 54de9b8 (max), 54dec00 (min); return gradient * 54de850 (масштаб в ppb).
const (
	bestFitGradientMaxRo = 1e9   // 54de9b8 — верхняя граница gradient
	bestFitGradientMinRo = -1e9  // 54dec00 — нижняя граница
	bestFitGradientScale = 1e9   // 54de850 — множитель результата (ppb)
)

type BestFitFiltered struct {
	mu         sync.Mutex
	windowSize int
	data       []float64
	timestamps []float64
	index      int
	count      int
	// По дизассемблеру: 0x28=absMean (GetAbsMean/GetMean), 0x38=GetClosest, 0x50/0x58/0x68/0x70 — ResetFilter.
	absMean    float64 // 0x28 — GetAbsMean/GetMean
	closestVal int64   // 0x38 — GetClosest возвращает это поле
	reset50    int64   // 0x50 — ResetFilter: 0x7fffffffffffffff
	reset58    int64   // 0x58 — -1
	reset68    int64   // 0x68 — 0x8000000000000000
	reset70    int64   // 0x70 — -1
}

// NewBestFitFiltered создаёт фильтр
func NewBestFitFiltered(windowSize int) *BestFitFiltered {
	return &BestFitFiltered{
		windowSize: windowSize,
		data:       make([]float64, windowSize),
		timestamps: make([]float64, windowSize),
	}
}

// Add добавляет точку (алиас AddValue по дизассемблеру __BestFitFiltered_.AddValue).
func (bf *BestFitFiltered) Add(value, timestamp float64) float64 {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	bf.data[bf.index] = value
	bf.timestamps[bf.index] = timestamp
	bf.index = (bf.index + 1) % bf.windowSize
	if bf.count < bf.windowSize {
		bf.count++
	}
	// Обновляем absMean для GetAbsMean (среднее по data)
	if bf.count > 0 {
		var sum float64
		for i := 0; i < bf.count; i++ {
			sum += bf.data[i]
		}
		bf.absMean = sum / float64(bf.count)
	}
	return bf.getFitted(timestamp)
}

// AddValue по дизассемблеру (__BestFitFiltered_.AddValue): то же, что Add.
func (bf *BestFitFiltered) AddValue(value, timestamp float64) float64 {
	return bf.Add(value, timestamp)
}

// GetLeastSquaresGradientFiltered возвращает градиент МНК по окну, ограниченный ±1e9 и умноженный на 1e9 (ppb).
// По дизассемблеру: gradient = (n*sum_xy - sum_x*sum_y)/(n*sum_xx - sum_x²); clamp [54dec00, 54de9b8]; return gradient*54de850.
func (bf *BestFitFiltered) GetLeastSquaresGradientFiltered() float64 {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	n := float64(bf.count)
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < bf.count; i++ {
		x := bf.timestamps[i]
		y := bf.data[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-15 {
		return 0
	}
	gradient := (n*sumXY - sumX*sumY) / denom
	if gradient > bestFitGradientMaxRo || gradient < bestFitGradientMinRo {
		return 0
	}
	return gradient * bestFitGradientScale
}

func (bf *BestFitFiltered) getFitted(t float64) float64 {
	if bf.count < 2 {
		return bf.data[(bf.index-1+bf.windowSize)%bf.windowSize]
	}
	// Линейная аппроксимация
	n := float64(bf.count)
	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < bf.count; i++ {
		x := bf.timestamps[i]
		y := bf.data[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-15 {
		return sumY / n
	}
	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n
	return slope*t + intercept
}

// GetAbsMean по дизассемблеру (__BestFitFiltered_.GetAbsMean@@Base): movsd 0x28(%rax), btr 0x3f (знак) — возврат |absMean|.
func (bf *BestFitFiltered) GetAbsMean() float64 {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return math.Abs(bf.absMean)
}

// DetermineExtremes по дизассемблеру (__BestFitFiltered_.DetermineExtremes 0x457c1a0): 0x50=MaxInt64, 0x68=MinInt64, 0x58/0x70=-1; цикл по count — min/max по data[i], индексы в 0x58/0x70.
func (bf *BestFitFiltered) DetermineExtremes() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	bf.reset58 = -1
	bf.reset70 = -1
	for i := 0; i < bf.count; i++ {
		v := bf.data[i]
		if bf.reset58 < 0 || v < bf.data[bf.reset58] {
			bf.reset58 = int64(i)
		}
		if bf.reset70 < 0 || v > bf.data[bf.reset70] {
			bf.reset70 = int64(i)
		}
	}
}

// TransposeXValues по дизассемблеру (__BestFitFiltered_.TransposeXValues@@Base): нормализация X (timestamps); makeslice пар (x_norm, y), цикл — x_norm = timestamp - min или остаток от деления.
// Реконструкция: нормализуем timestamps относительно первого (x = t - t0).
func (bf *BestFitFiltered) TransposeXValues() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	if bf.count == 0 {
		return
	}
	t0 := bf.timestamps[0]
	for i := 0; i < bf.count; i++ {
		bf.timestamps[i] = bf.timestamps[i] - t0
	}
}

// RemoveExtremes по дизассемблеру (__BestFitFiltered_.RemoveExtremes@@Base): копирование в новый slice всех элементов кроме индексов 0x58 и 0x70 (min/max).
func (bf *BestFitFiltered) RemoveExtremes() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	if bf.count == 0 || (bf.reset58 < 0 && bf.reset70 < 0) {
		return
	}
	dst := 0
	for i := 0; i < bf.count; i++ {
		if i == int(bf.reset58) || i == int(bf.reset70) {
			continue
		}
		bf.data[dst] = bf.data[i]
		bf.timestamps[dst] = bf.timestamps[i]
		dst++
	}
	bf.count = dst
	bf.index = dst % bf.windowSize
	if bf.index < 0 {
		bf.index += bf.windowSize
	}
}

// ResetFilter по дизассемблеру (__BestFitFiltered_.ResetFilter@@Base): 0x50=0x7fffffffffffffff, 0x58=-1, 0x68=0x8000000000000000, 0x70=-1.
func (bf *BestFitFiltered) ResetFilter() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	const maxInt64 = 1<<63 - 1
	const minInt64 = -1 << 63
	bf.reset50 = maxInt64
	bf.reset58 = -1
	bf.reset68 = minInt64
	bf.reset70 = -1
}

// GetMean по дизассемблеру (__BestFitFiltered_.GetMean@@Base): в бинарнике вызов GetLeastSquaresGradientFiltered, затем movsd 0x28 — возврат absMean.
func (bf *BestFitFiltered) GetMean() float64 {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.absMean
}

// GetClosest по дизассемблеру (__BestFitFiltered_.GetClosest@@Base): mov 0x38(%rax),%rax — возврат поля closestVal.
func (bf *BestFitFiltered) GetClosest() int64 {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.closestVal
}

// CircularBuffer — кольцевой буфер
type CircularBuffer struct {
	data  []float64
	size  int
	index int
	count int
}

// NewCircularBuffer создаёт буфер
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]float64, size),
		size: size,
	}
}

// Add добавляет значение
func (cb *CircularBuffer) Add(value float64) {
	cb.data[cb.index] = value
	cb.index = (cb.index + 1) % cb.size
	if cb.count < cb.size {
		cb.count++
	}
}

// Get возвращает значение по индексу (0 = самое старое)
func (cb *CircularBuffer) Get(i int) float64 {
	if i >= cb.count {
		return 0
	}
	idx := (cb.index - cb.count + i + cb.size) % cb.size
	return cb.data[idx]
}

// Count возвращает количество
func (cb *CircularBuffer) Count() int { return cb.count }

// Min по дизассемблеру (__CircularBuffer_.Min): 0x18/0x20=data, 0x30/0x38=отсортированные индексы минимума; возврат data[sorted[0]].
// Здесь — линейный поиск минимума по буферу (полная реконструкция — поддержка sorted slice как в бинарнике).
func (cb *CircularBuffer) Min() float64 {
	if cb.count == 0 {
		return 0
	}
	min := cb.data[0]
	for i := 1; i < cb.count; i++ {
		idx := (cb.index - cb.count + i + cb.size) % cb.size
		if cb.data[idx] < min {
			min = cb.data[idx]
		}
	}
	return min
}

// Max по дизассемблеру (__CircularBuffer_.Max): 0x48/0x50=отсортированные индексы максимума; возврат data[sorted[0]].
func (cb *CircularBuffer) Max() float64 {
	if cb.count == 0 {
		return 0
	}
	max := cb.data[0]
	for i := 1; i < cb.count; i++ {
		idx := (cb.index - cb.count + i + cb.size) % cb.size
		if cb.data[idx] > max {
			max = cb.data[idx]
		}
	}
	return max
}

package servo

import "sync"

// ClockQuality — качество часов (PTP clock quality). По дизассемблеру: setDefaultValues (0x459fc00), UpdateClockQuality (0x459f640), Subscribe/Unsubscribe (0x459f820/0x459fa20).
type ClockQuality struct {
	mu            sync.RWMutex
	clockClass    uint8
	clockAccuracy uint8
	timeSource    uint8
	sourceIPAddr  string
	subscribers   []chan struct{}
}

// NewClockQuality создаёт ClockQuality
func NewClockQuality() *ClockQuality {
	cq := &ClockQuality{}
	cq.setDefaultValues()
	return cq
}

// setDefaultValues по дизассемблеру (0x459fc00): Lock(0x10); загрузка 5 байт из appConfig+0x2c8..0x2e8 в cq+0..5 (clockClass, clockAccuracy, timeSource и др.); Unlock.
func (cq *ClockQuality) setDefaultValues() {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.clockClass = 248     // default
	cq.clockAccuracy = 0xFE // unknown
	cq.timeSource = 0xA0    // internal oscillator
}

// GetClockClass возвращает clock class
func (cq *ClockQuality) GetClockClass() uint8 {
	cq.mu.RLock()
	defer cq.mu.RUnlock()
	return cq.clockClass
}

// GetClockAccuracy возвращает clock accuracy
func (cq *ClockQuality) GetClockAccuracy() uint8 {
	cq.mu.RLock()
	defer cq.mu.RUnlock()
	return cq.clockAccuracy
}

// GetTimeSource возвращает time source
func (cq *ClockQuality) GetTimeSource() uint8 {
	cq.mu.RLock()
	defer cq.mu.RUnlock()
	return cq.timeSource
}

// GetSourceIPAddr возвращает IP источника
func (cq *ClockQuality) GetSourceIPAddr() string {
	cq.mu.RLock()
	defer cq.mu.RUnlock()
	return cq.sourceIPAddr
}

// setClockAccuracy устанавливает clock accuracy
func (cq *ClockQuality) setClockAccuracy(accuracy uint8) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.clockAccuracy = accuracy
}

// setTimeSource устанавливает time source
func (cq *ClockQuality) setTimeSource(source uint8) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.timeSource = source
}

// UpdateClockQuality по дизассемблеру (0x459f640): при arg==nil или appConfig.0x2c0==0 → setDefaultValues; иначе getNMEAExtTsValueForLinkedDevice, setTimeSource, setClockAccuracy; итерация по subscribers (0x20), selectnbsend; при неудаче Logger.Warn.
func (cq *ClockQuality) UpdateClockQuality(class, accuracy, source uint8, ip string) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.clockClass = class
	cq.clockAccuracy = accuracy
	cq.timeSource = source
	cq.sourceIPAddr = ip
	// Notify subscribers
	for _, ch := range cq.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Subscribe подписывается на обновления
func (cq *ClockQuality) Subscribe() <-chan struct{} {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	ch := make(chan struct{}, 1)
	cq.subscribers = append(cq.subscribers, ch)
	return ch
}

// Unsubscribe отписывается от обновлений
func (cq *ClockQuality) Unsubscribe(ch <-chan struct{}) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	for i, sub := range cq.subscribers {
		if sub == ch {
			cq.subscribers = append(cq.subscribers[:i], cq.subscribers[i+1:]...)
			break
		}
	}
}

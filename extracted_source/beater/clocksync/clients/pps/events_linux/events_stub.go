//go:build !linux

package events_linux

import "time"

// FetchPPSNsec — заглушка для не-Linux: симуляция (возврат 0, false).
func FetchPPSNsec(ppsIndex int) (int64, bool) {
	_ = ppsIndex
	return 0, false
}

// RunPPSPollLoop — заглушка для не-Linux: тикер раз в секунду, callback с time.Now().UnixNano().
func RunPPSPollLoop(ppsIndex int, interval time.Duration, onPPS func(nsec int64)) {
	_ = ppsIndex
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		onPPS(time.Now().UnixNano())
	}
}

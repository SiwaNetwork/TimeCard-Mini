// Запуск clocksync с конфигом. По дизассемблеру: RunWithConfig устанавливает appConfig и вызывает c.Run(ctx).
// Controller.Run (runWithStore) выполняет GenerateTimeSourcesFromConfig, SetConfig, EnablePPSIfRequired,
// старт контроллеров по IsClockProtocolEnabled, затем servo.Run(ctx).
package clocksync

import (
	"context"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/config"
)

// RunWithConfig запускает clocksync с конфигом. По дизассемблеру: SetAppConfig, затем c.Run(ctx).
// Вся логика (GenerateTimeSourcesFromConfig, PTP/NTP/PPS/NMEA/PHC/oscillator, servo) выполняется в runWithStore.
func (c *Controller) RunWithConfig(ctx context.Context, cfg *config.Config) error {
	if cfg == nil || cfg.ClockSync == nil {
		return c.Run(ctx)
	}
	config.SetAppConfig(cfg)
	return c.Run(ctx)
}

// ParseInterval разбирает интервал из конфига (1s, 500ms).
func ParseInterval(s string) time.Duration {
	return config.ParseInterval(s)
}

// Package clocksync предоставляет запуск цикла синхронизации времени для встраивания в Beat.
package clocksync

import (
	"context"
	"time"

	"github.com/shiwa/timecard-mini/tc-sync/internal/clockadj"
	"github.com/shiwa/timecard-mini/tc-sync/internal/clockselect"
	"github.com/shiwa/timecard-mini/tc-sync/internal/config"
	"github.com/shiwa/timecard-mini/tc-sync/internal/logger"
	"github.com/shiwa/timecard-mini/tc-sync/internal/ptp4l"
	"github.com/shiwa/timecard-mini/tc-sync/internal/servo"
	"github.com/shiwa/timecard-mini/tc-sync/internal/source"
	pkgconfig "github.com/shiwa/timecard-mini/tc-sync/pkg/config"
)

// RunDaemon запускает цикл синхронизации (выбор источника + servo) до отмены ctx.
// cfg должен содержать clock_sync. Используется из Beat (libbeat).
func RunDaemon(ctx context.Context, cfg *pkgconfig.Config, quiet bool) error {
	if cfg == nil || cfg.ClockSync == nil {
		return nil
	}
	logger.Quiet = quiet
	cs := cfg.ClockSync
	// Преобразуем в internal config для source factory
	internalCfg := toInternalConfig(cfg)

	// Запуск ptp4l внутри tc-sync для источников ptp с start_ptp4l: true
	if internalCfg.ClockSync != nil {
		stopPtp4l := ptp4l.Start(internalCfg.ClockSync.Ptp4lJobs(), quiet)
		go func() {
			<-ctx.Done()
			stopPtp4l()
		}()
	}

	var primary, secondary []source.TimeSource
	for _, c := range cs.PrimaryClocks {
		if c.Disable || c.MonitorOnly {
			continue
		}
		s, err := source.NewFromClockSource(toInternalClockSource(c))
		if err != nil {
			logger.Info("primary %s: %v", c.Protocol, err)
			continue
		}
		primary = append(primary, s)
	}
	for _, c := range cs.SecondaryClocks {
		if c.Disable || c.MonitorOnly {
			continue
		}
		s, err := source.NewFromClockSource(toInternalClockSource(c))
		if err != nil {
			logger.Info("secondary %s: %v", c.Protocol, err)
			continue
		}
		secondary = append(secondary, s)
	}
	defer func() {
		for _, s := range primary {
			_ = s.Close()
		}
		for _, s := range secondary {
			_ = s.Close()
		}
	}()

	if len(primary) == 0 && len(secondary) == 0 {
		return nil
	}

	election := clockselect.NewElection(primary, secondary)
	interval := parseInterval(cfg.Servo.Interval)
	var algo servo.Algorithm
	switch cfg.Servo.Algorithm {
	case "pi":
		algo = servo.NewPI(cfg.Servo.Kp, cfg.Servo.Ki)
	case "pi_shiwatime":
		algo = servo.NewPIShiwatime(cfg.Servo.Kp)
	case "linreg":
		algo = servo.NewLinReg()
	default:
		algo = servo.NewPID(cfg.Servo.Kp, cfg.Servo.Ki, cfg.Servo.Kd)
	}

	logger.Info("clocksync: primary=%d secondary=%d interval=%v adjust_clock=%v",
		len(primary), len(secondary), interval, cs.AdjustClock)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lastRun := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		active := election.Select()
		if active == nil {
			algo.Reset()
			continue
		}
		refTime, ok := election.GetTimeFromActive()
		if !ok {
			continue
		}
		localNow := time.Now().UTC()
		offsetNs := refTime.Sub(localNow).Nanoseconds()
		dt := time.Since(lastRun)
		lastRun = time.Now()

		freqAdj := algo.Update(float64(offsetNs), dt)
		if cs.AdjustClock {
			stepThresholdNs := ParseStepLimit(cs.StepLimit)
			if offsetNs > stepThresholdNs || offsetNs < -stepThresholdNs {
				_ = clockadj.Step(refTime)
				algo.Reset()
			} else {
				if offsetNs != 0 {
					_ = clockadj.Slew(offsetNs)
				}
				if freqAdj != 0 {
					ppm := freqAdj * 1e6
					_ = clockadj.SetFrequency(ppm)
				}
			}
		}
	}
}

// ToPkgConfig преобразует internal config в pkg config (для вызова RunDaemon из cmd/tc-sync).
func ToPkgConfig(c *config.Config) *pkgconfig.Config {
	if c == nil {
		return nil
	}
	out := &pkgconfig.Config{
		Device:    pkgconfig.DeviceConfig(c.Device),
		Timepulse: pkgconfig.TimepulseConfig(c.Timepulse),
		Servo:     pkgconfig.ServoConfig(c.Servo),
	}
	if c.ClockSync != nil {
		out.ClockSync = &pkgconfig.ClockSyncConfig{
			AdjustClock:     c.ClockSync.AdjustClock,
			StepLimit:       c.ClockSync.StepLimit,
			PrimaryClocks:   make([]pkgconfig.ClockSource, len(c.ClockSync.PrimaryClocks)),
			SecondaryClocks: make([]pkgconfig.ClockSource, len(c.ClockSync.SecondaryClocks)),
		}
		for i := range c.ClockSync.PrimaryClocks {
			out.ClockSync.PrimaryClocks[i] = fromInternalClockSource(c.ClockSync.PrimaryClocks[i])
		}
		for i := range c.ClockSync.SecondaryClocks {
			out.ClockSync.SecondaryClocks[i] = fromInternalClockSource(c.ClockSync.SecondaryClocks[i])
		}
	}
	return out
}

func fromInternalClockSource(c config.ClockSource) pkgconfig.ClockSource {
	return pkgconfig.ClockSource{
		Protocol:          c.Protocol,
		Disable:           c.Disable,
		MonitorOnly:      c.MonitorOnly,
		Device:            c.Device,
		Baud:              c.Baud,
		IP:                c.IP,
		PollInterval:     c.PollInterval,
		Domain:            c.Domain,
		Interface:         c.Interface,
		UnicastMasterTable: c.UnicastMasterTable,
		StartPtp4l:        c.StartPtp4l,
		Ptp4lPath:         c.Ptp4lPath,
		Ptp4lArgs:         c.Ptp4lArgs,
		Pin:               c.Pin,
		Index:             c.Index,
		LinkedDevice:      c.LinkedDevice,
		CableDelay:        c.CableDelay,
		Offset:            c.Offset,
	}
}

func toInternalConfig(c *pkgconfig.Config) *config.Config {
	if c == nil {
		return nil
	}
	out := &config.Config{
		Device:    config.DeviceConfig(c.Device),
		Timepulse: config.TimepulseConfig(c.Timepulse),
		Servo:     config.ServoConfig(c.Servo),
	}
	if c.ClockSync != nil {
		out.ClockSync = &config.ClockSyncConfig{
			AdjustClock:     c.ClockSync.AdjustClock,
			PrimaryClocks:   make([]config.ClockSource, len(c.ClockSync.PrimaryClocks)),
			SecondaryClocks: make([]config.ClockSource, len(c.ClockSync.SecondaryClocks)),
		}
		for i := range c.ClockSync.PrimaryClocks {
			out.ClockSync.PrimaryClocks[i] = toInternalClockSource(c.ClockSync.PrimaryClocks[i])
		}
		for i := range c.ClockSync.SecondaryClocks {
			out.ClockSync.SecondaryClocks[i] = toInternalClockSource(c.ClockSync.SecondaryClocks[i])
		}
	}
	return out
}

func toInternalClockSource(c pkgconfig.ClockSource) config.ClockSource {
	return config.ClockSource{
		Protocol:          c.Protocol,
		Disable:           c.Disable,
		MonitorOnly:       c.MonitorOnly,
		Device:            c.Device,
		Baud:              c.Baud,
		IP:                c.IP,
		PollInterval:      c.PollInterval,
		Domain:            c.Domain,
		Interface:         c.Interface,
		UnicastMasterTable: c.UnicastMasterTable,
		StartPtp4l:        c.StartPtp4l,
		Ptp4lPath:         c.Ptp4lPath,
		Ptp4lArgs:         c.Ptp4lArgs,
		Pin:               c.Pin,
		Index:             c.Index,
		LinkedDevice:      c.LinkedDevice,
		CableDelay:        c.CableDelay,
		Offset:            c.Offset,
	}
}

func parseInterval(s string) time.Duration {
	if s == "" {
		return time.Second
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Second
	}
	return d
}

// ParseStepLimit парсит step_limit из конфига (например "500ms", "15m") в наносекунды.
// Порог выше которого делается step, иначе slew. Пустая строка или ошибка — 500 ms.
func ParseStepLimit(s string) int64 {
	const defaultStepNs = 500_000_000 // 500 ms
	if s == "" {
		return defaultStepNs
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultStepNs
	}
	return d.Nanoseconds()
}

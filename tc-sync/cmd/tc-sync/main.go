// tc-sync — аналог Timebeat: платформа синхронизации времени (GNSS, NTP, PTP, PPS, servo).
//
// Реализовано на основе анализа бинарника shiwatime (Timebeat-based, code_analysis/):
//   - Конфиг в стиле Timebeat: clock_sync, primary_clocks, secondary_clocks
//   - Источники: GNSS (UBX/Timecard Mini), NTP, PPS, PTP (ptp4l+PHC)
//   - Выбор источника (primary → secondary), servo (PID/PI)
//   - configure — настройка time pulse на UBX
//
// Использование:
//
//	tc-sync -configure              — настроить time pulse и выйти
//	tc-sync -run -config tc-sync.yml — запуск daemon (выбор источника + servo)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shiwa/timecard-mini/tc-sync/internal/config"
	"github.com/shiwa/timecard-mini/tc-sync/internal/logger"
	"github.com/shiwa/timecard-mini/tc-sync/internal/ubx"
	"github.com/shiwa/timecard-mini/tc-sync/pkg/clocksync"
)

func main() {
	configure := flag.Bool("configure", false, "настроить time pulse на UBX устройстве и выйти")
	run := flag.Bool("run", false, "запуск daemon: выбор источника времени + servo (аналог Timebeat)")
	configPath := flag.String("config", "", "путь к YAML конфигу (по умолчанию tc-sync.yml)")
	port := flag.String("port", "", "последовательный порт (переопределяет config)")
	baud := flag.Int("baud", 0, "скорость порта (переопределяет config)")
	pulseMs := flag.Float64("pulse-width-ms", 0, "длительность импульса в мс (переопределяет config)")
	quiet := flag.Bool("quiet", false, "меньше вывода")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil && *configPath != "" {
		log.Fatalf("config: %v", err)
	}
	if cfg == nil {
		cfg = config.Default()
	}

	if *port != "" {
		cfg.Device.Port = *port
	}
	if *baud != 0 {
		cfg.Device.Baud = *baud
	}
	if *pulseMs > 0 {
		cfg.Timepulse.PulseWidthMs = *pulseMs
	}

	if *configure {
		runConfigure(cfg, *quiet)
		return
	}

	if *run {
		logger.Quiet = *quiet
		runDaemonWithShutdown(cfg, *quiet)
		return
	}

	// По умолчанию: только configure
	runConfigure(cfg, *quiet)
	if !*quiet {
		fmt.Println("tc-sync: для daemon (аналог Timebeat) используйте -run с конфигом (clock_sync).")
	}
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		path = "tc-sync.yml"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return config.Load(path)
}

func runConfigure(cfg *config.Config, quiet bool) {
	port, err := ubx.Open(cfg.Device.Port, cfg.Device.Baud)
	if err != nil {
		log.Fatalf("открытие порта %s: %v", cfg.Device.Port, err)
	}
	defer port.Close()

	tp := ubx.TP5Config{
		TPIdx:              cfg.Timepulse.TPIdx,
		AntCableDelayNs:    cfg.Timepulse.AntCableDelayNs,
		FreqPeriod:         1000000,
		FreqPeriodLock:     1000000,
		PulseLenRatioNs:    uint32(cfg.Timepulse.PulseWidthMs * 1e6),
		PulseLenRatioLock:  uint32(cfg.Timepulse.PulseWidthMs * 1e6),
		Active:             true,
		LockGnssFreq:       true,
		LockedOtherSet:     true,
		IsLength:           true,
		AlignToTow:         cfg.Timepulse.AlignToTow,
	}

	if err := port.ConfigureTimePulse(tp); err != nil {
		log.Fatalf("настройка time pulse: %v", err)
	}
	if !quiet {
		fmt.Printf("Time pulse настроен: %s, %d baud, импульс %.2f мс\n",
			cfg.Device.Port, cfg.Device.Baud, cfg.Timepulse.PulseWidthMs)
	}
}

// runDaemonWithShutdown запускает цикл синхронизации через clocksync.RunDaemon с контекстом;
// по SIGINT/SIGTERM контекст отменяется, ptp4l и источники корректно останавливаются.
func runDaemonWithShutdown(cfg *config.Config, quiet bool) {
	if cfg.ClockSync == nil {
		log.Fatal("для -run нужен конфиг с clock_sync (primary_clocks / secondary_clocks)")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("получен сигнал %v, завершение...", sig)
		cancel()
	}()

	pkgCfg := clocksync.ToPkgConfig(cfg)
	if err := clocksync.RunDaemon(ctx, pkgCfg, quiet); err != nil && err != context.Canceled {
		logger.Error("%v", err)
	}
}

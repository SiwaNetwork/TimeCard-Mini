// Реконструировано по аналогии (не по дизассемблеру). Точка входа: main → clocksync → servo.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync"
	"github.com/shiwa/timecard-mini/extracted-source/config"
)

func main() {
	configPath := flag.String("config", "timebeat.yml", "path to YAML config (timebeat.clock_sync)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctrl := clocksync.GetController()

	cfg, err := config.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Без конфига — только servo loop (без NTP и др.)
			if err := ctrl.Run(ctx); err != nil && ctx.Err() == nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	if err := ctrl.RunWithConfig(ctx, cfg); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

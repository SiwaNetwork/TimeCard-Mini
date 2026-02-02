// Package beater реализует интерфейс Beater для Timebeat (libbeat v7).
package beater

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/shiwa/timecard-mini/tc-sync/pkg/clocksync"
	pkgconfig "github.com/shiwa/timecard-mini/tc-sync/pkg/config"
)

// Timebeat реализует beat.Beater.
type Timebeat struct {
	done   chan struct{}
	config *pkgconfig.Config
	client beat.Client
}

// New создаёт Beater из конфигурации Beat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	sub, err := cfg.Child("timebeat", -1)
	if err != nil || sub == nil {
		return nil, fmt.Errorf("конфиг timebeat не найден: %v", err)
	}
	config := defaultConfig()
	if err := sub.Unpack(&config); err != nil {
		return nil, fmt.Errorf("ошибка разбора конфига timebeat: %w", err)
	}
	bt := &Timebeat{
		done:   make(chan struct{}),
		config: &config,
	}
	return bt, nil
}

// Run запускает цикл синхронизации времени (tc-sync) до Stop().
func (bt *Timebeat) Run(b *beat.Beat) error {
	logp.Info("timebeat запущен (clock_sync на базе tc-sync)")
	bt.client, _ = b.Publisher.Connect()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-bt.done
		cancel()
	}()

	err := clocksync.RunDaemon(ctx, bt.config, true)
	if err != nil && err != context.Canceled {
		logp.Warnf("clocksync завершён: %v", err)
	}
	return nil
}

// Stop останавливает Run.
func (bt *Timebeat) Stop() {
	if bt.client != nil {
		bt.client.Close()
	}
	close(bt.done)
}

func defaultConfig() pkgconfig.Config {
	return pkgconfig.Config{
		Servo: pkgconfig.ServoConfig{
			Algorithm: "pid",
			Kp:        0.1,
			Ki:        0.01,
			Kd:        0.001,
			Interval:  "1s",
		},
	}
}

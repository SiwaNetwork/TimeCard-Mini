// Package client — PHC клиент. Реконструировано по дизассемблеру:
// NewClient(config), Start() — go runReadloop (по дизассемблеру Start: newproc(func1), func1 вызывает readloop).
// runReadloop (0x45eeb00): NewTicker(1s), цикл: RegisterObservation, select ticker/ctx, GetPreciseTime, DeterminePHCOffset.
package client

import (
	"time"

	phclib "github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/phc"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
)

// Client по дизассемблеру: конфиг PHC-источника, readloop для опроса offset.
type Client struct {
	config *sources.TimeSourceConfig
}

// NewClient по дизассемблеру (0x45ee860): создаёт Client из config.
func NewClient(config *sources.TimeSourceConfig) *Client {
	if config == nil {
		return nil
	}
	return &Client{config: config}
}

// Start по дизассемблеру (0x45eea20): go func1 — запускает readloop в горутине.
func (c *Client) Start() {
	if c == nil {
		return
	}
	go c.runReadloop()
}

// runReadloop по дизассемблеру (0x45eeb00): NewTicker(1s=0x3b9aca00), цикл:
// select ticker; GetPreciseTime; DeterminePHCOffset; RegisterObservation(offsets, sourceID, offset); category=0 для "system".
func (c *Client) runReadloop() {
	if c == nil || c.config == nil {
		return
	}
	offsets := servo.GetController().GetOffsets()
	if offsets == nil {
		return
	}
	inst := phclib.GetInstance()
	if inst == nil {
		return
	}
	device := inst.GetDeviceWithName(c.config.Name)
	sourceID := "phc:" + c.config.Name
	category := 1
	if c.config.Name == "system" {
		category = 0 // по дизассемблеру: "system" → 0x78=0
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		var offsetNs int64
		if device != nil {
			offsetNs = device.DeterminePHCOffset()
		}
		_ = adjusttime.GetPreciseTime()
		if category != 0 {
			offsets.RegisterObservationWithCategory(sourceID, offsetNs, category)
		} else {
			offsets.RegisterObservation(sourceID, offsetNs)
		}
	}
}

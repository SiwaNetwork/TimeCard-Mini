// Package clocksync — верхний контроллер синхронизации времени (timebeat).
// Реконструировано по дизассемблеру Controller.Run (0x4c2c500): GetStore, GenerateTimeSourcesFromConfig,
// servo.GetController, IsClockProtocolEnabled → PTP/NTP/PPS/NMEA/PHC/oscillator, device variants, servo.Run.
package clocksync

import (
	"context"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
)

// Controller — контроллер clocksync, запускает и держит servo.
// По дизассемблеру: receiver+0x00=ctx, +0x08=servo.Controller, +0x18=debug (0x18).
type Controller struct {
	mu      sync.Mutex
	servo   *servo.Controller
	running bool
	once    sync.Once
	debug   bool // +0x18: при true — Logger.Info после каждого контроллера
}

var controllerInstance *Controller
var controllerOnce sync.Once

// GetController возвращает singleton контроллер clocksync.
// По дизассемблеру GetController@@Base (0x4c2c4a0): sync.Once(once), doSlow(func1), return controller.
func GetController() *Controller {
	controllerOnce.Do(func() {
		controllerInstance = NewController()
	})
	return controllerInstance
}

// NewController создаёт контроллер и инициализирует servo.
func NewController() *Controller {
	return &Controller{
		servo: servo.GetController(),
	}
}

// Run запускает clocksync: GetController().Run(ctx).
func Run(ctx context.Context) error {
	c := GetController()
	return c.Run(ctx)
}

// Controller.Run запускает контроллер. По дизассемблеру (0x4c2c500):
// GetStore → GenerateTimeSourcesFromConfig → servo.GetController → по IsClockProtocolEnabled(1..7)
// и IsDeviceVariantEnabled(2..5) стартуют PTP, NTP, PPS, NMEA, PHC, oscillator, vendor controllers.
// Затем external_devices, taas, ptpsquared, SSH/HTTP, syslog, servo.Run(ctx).
// Реализация в run_impl.go (вызов runWithStore).
func (c *Controller) Run(ctx context.Context) error {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()
	runWithStore(c, ctx)
	return c.servo.Run(ctx)
}

// GetServo возвращает внутренний servo-контроллер.
func (c *Controller) GetServo() *servo.Controller {
	return c.servo
}


// run_impl.go — реализация runWithStore по дизассемблеру Controller.Run (0x4c2c500).
// GetStore → GenerateTimeSourcesFromConfig → servo.GetController → IsClockProtocolEnabled(1..7)
// → PTP, NTP, PPS, NMEA, PHC, oscillator; IsDeviceVariantEnabled(2..5) → ocp_timecard, timebeat_timecard_mini и др.
package clocksync

import (
	"context"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/nmea"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/oscillator"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/phc"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/pps"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ptp"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ptpsquared"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/ocp_timecard"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/open_timecard_mini_v2_pt"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/open_timecard_v1"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat_timecard_mini"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/external_devices"
	phclib "github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/phc"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/taas"
	"github.com/shiwa/timecard-mini/extracted-source/config"
	"github.com/shiwa/timecard-mini/extracted-source/interactive/daemon"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// runWithStore выполняет фазу запуска контроллеров по дизассемблеру Controller.Run.
// Вызывается из Controller.Run перед servo.Run(ctx).
func runWithStore(c *Controller, ctx context.Context) {
	store := sources.GetStore()
	if store == nil {
		return
	}
	cfg := config.GetAppConfig()
	if cfg != nil && cfg.ClockSync != nil {
		store.GenerateTimeSourcesFromConfig(cfg.ClockSync)
		c.servo.SetConfigWithIntervalStr(
			cfg.ClockSync.AdjustClock,
			config.ParseStepLimit(cfg.ClockSync.StepLimit),
			config.ParseInterval(cfg.Servo.Interval),
			cfg.Servo.Interval,
			cfg.Servo.Algorithm,
			cfg.Servo.Kp, cfg.Servo.Ki, cfg.Servo.Kd,
		)
		c.servo.SetStepAndExitEnabled(cfg.Servo.StepAndExit)
		if len(cfg.ClockSync.PPSDevices) > 0 {
			if pcc := phclib.GetPHCController(); pcc != nil {
				pcc.SetPPSConfig(cfg.ClockSync.PPSDevices)
				pcc.EnablePPSIfRequired()
			}
		}
	}
	offsets := c.servo.GetOffsets()

	// По дизассемблеру: mapaccess2_fast64(store+0x68, key) — IsClockProtocolEnabled
	// key 1 → PTP, 2 → NTP, 3 → PPS, 4 → NMEA, 6 → PHC, 7 → oscillator
	if store.IsClockProtocolEnabled(1) {
		ptpNewControllerAndStart(store, offsets)
	}
	if store.IsClockProtocolEnabled(2) {
		ntpCtrl := ntp.NewController(offsets)
		ntpCtrl.Start()
	}
	if store.IsClockProtocolEnabled(3) {
		ppsCtrl := pps.NewController(offsets)
		ppsCtrl.Start()
	}
	if store.IsClockProtocolEnabled(4) {
		nmeaCtrl := nmea.NewController()
		nmeaCtrl.Start()
	}
	if store.IsClockProtocolEnabled(6) {
		phcNewControllerAndLoadConfig()
	}
	if store.IsClockProtocolEnabled(7) {
		oscillator.NewController().Start()
	}
	// Device variants (store+0x70): key 2→ocp_timecard, 3→timebeat_timecard_mini, 4→open_timecard_v1, 5→open_timecard_mini_v2_pt
	startDeviceVariantControllers(store)

	// По дизассемблеру Controller.Run: appConfig+0x80 → external_devices; +0x4b1 → taas; +0x3c1 → ptpsquared
	if cfg != nil && len(cfg.ExternalDevices) > 0 {
		extCtrl := external_devices.NewController()
		if extCtrl != nil {
			extCtrl.Start()
		}
	}
	if cfg != nil && cfg.TaasEnabled {
		taasCtrl := taas.NewController()
		if taasCtrl != nil {
			taasCtrl.Start()
		}
	}
	if cfg != nil && cfg.PTPsquared != nil && cfg.PTPsquared.Enabled {
		ptpsqCtrl := ptpsquared.NewController()
		if ptpsqCtrl != nil {
			ptpsqCtrl.Start()
		}
	}

	// appConfig+0x318 → SSH; +0x370 → HTTP; +0x398 → syslog (по дизассемблеру: newproc GetSSHServerInstance/GetHTTPServerInstance, StartSyslogClient)
	if cfg != nil && cfg.SSHEnabled {
		if srv := daemon.GetSSHServerInstance(); srv != nil {
			go srv.Run()
		}
	}
	if cfg != nil && cfg.HTTPEnabled {
		if srv := daemon.GetHTTPServerInstance(); srv != nil {
			go srv.Run()
		}
	}
	if cfg != nil && cfg.SyslogEnabled {
		go logging.StartSyslogClient()
	}
}

// ptpNewControllerAndStart — по дизассемблеру: ptp.NewController(store, offsets), ptp.(*Controller).Start.
func ptpNewControllerAndStart(store *sources.TimeSourceStore, offsets *servo.Offsets) {
	ctrl := ptp.NewController(store, offsets)
	if ctrl != nil {
		ctrl.Start()
	}
}

// phcNewControllerAndLoadConfig — по дизассемблеру: clients/phc.NewController, loadConfig (Range ConfigureTimeSource).
func phcNewControllerAndLoadConfig() {
	phc.NewController().Start()
}

// startDeviceVariantControllers — по дизассемблеру store+0x70 mapaccess2: key 2→ocp_timecard, 3→timebeat_timecard_mini, 4→open_timecard_v1, 5→open_timecard_mini_v2_pt.
func startDeviceVariantControllers(store *sources.TimeSourceStore) {
	if store == nil {
		return
	}
	if store.IsDeviceVariantEnabled(2) {
		ocp_timecard.NewController()
		ocp_timecard.LoadConfig()
	}
	if store.IsDeviceVariantEnabled(3) {
		timebeat_timecard_mini.NewController()
		timebeat_timecard_mini.LoadConfig()
	}
	if store.IsDeviceVariantEnabled(4) {
		open_timecard_v1.NewController()
		open_timecard_v1.LoadConfig()
	}
	if store.IsDeviceVariantEnabled(5) {
		open_timecard_mini_v2_pt.NewController()
		open_timecard_mini_v2_pt.LoadConfig()
	}
}

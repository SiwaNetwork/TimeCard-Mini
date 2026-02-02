package config

import (
	"fmt"
	"os"

	"github.com/shiwa/timecard-mini/tc-sync/internal/ptp4l"
	"gopkg.in/yaml.v3"
)

// Config — конфигурация tc-sync (аналог Timebeat/shiwatime)
// Поддерживается два формата: простой (device/timepulse) и полный (clock_sync как в Timebeat)
type Config struct {
	// Простой формат — для configure timepulse
	Device   DeviceConfig   `yaml:"device"`
	Timepulse TimepulseConfig `yaml:"timepulse"`
	Servo    ServoConfig   `yaml:"servo"`

	// Формат Timebeat: clock_sync с primary/secondary clocks
	ClockSync *ClockSyncConfig `yaml:"clock_sync"`
}

// ClockSyncConfig — как в Timebeat: primary_clocks, secondary_clocks, adjust_clock
type ClockSyncConfig struct {
	AdjustClock     bool          `yaml:"adjust_clock"`
	StepLimit       string        `yaml:"step_limit"` // порог step vs slew, например "500ms", "15m"; пусто = 500ms
	PrimaryClocks   []ClockSource `yaml:"primary_clocks"`
	SecondaryClocks []ClockSource `yaml:"secondary_clocks"`
}

// ClockSource — один источник времени (protocol: gnss, ntp, pps, ptp)
type ClockSource struct {
	Protocol string `yaml:"protocol"` // gnss, timebeat_opentimecard_mini, ntp, pps, ptp
	Disable  bool   `yaml:"disable"`
	MonitorOnly bool `yaml:"monitor_only"`

	// GNSS / Timecard Mini (UBX)
	Device string `yaml:"device"`
	Baud   int    `yaml:"baud"`
	// NTP
	IP         string `yaml:"ip"`
	PollInterval string `yaml:"pollinterval"`
	// PTP
	Domain     int    `yaml:"domain"`
	Interface  string `yaml:"interface"`
	UnicastMasterTable []string `yaml:"unicast_master_table"`
	// Запуск ptp4l внутри tc-sync (linuxptp)
	StartPtp4l bool     `yaml:"start_ptp4l"`
	Ptp4lPath  string   `yaml:"ptp4l_path"`
	Ptp4lArgs  []string `yaml:"ptp4l_args"`
	// PPS
	Pin        int    `yaml:"pin"`
	Index      int    `yaml:"index"`
	LinkedDevice string `yaml:"linked_device"`
	CableDelay int    `yaml:"cable_delay"`
	// NMEA (RMC): статическое смещение в наносекундах
	Offset int64 `yaml:"offset"`
}

// DeviceConfig — последовательный порт UBX/GNSS
type DeviceConfig struct {
	Port string `yaml:"port"`
	Baud int    `yaml:"baud"`
}

// TimepulseConfig — параметры PPS/time pulse (CFG-TP5)
type TimepulseConfig struct {
	PulseWidthMs    float64 `yaml:"pulse_width_ms"`
	TPIdx          uint8   `yaml:"tp_idx"`
	AntCableDelayNs int16  `yaml:"ant_cable_delay_ns"`
	AlignToTow     bool    `yaml:"align_to_tow"`
}

// ServoConfig — алгоритм синхронизации (PID/PI/LinReg).
// Дефолты Kp/Ki/Kd — в стиле shiwatime; точные значения из бинарника см. code_analysis/FOUND_COEFFICIENTS.md.
type ServoConfig struct {
	Algorithm string  `yaml:"algorithm"`
	Kp        float64 `yaml:"kp"`
	Ki        float64 `yaml:"ki"`
	Kd        float64 `yaml:"kd"`
	Interval  string  `yaml:"interval"`
}

// Default возвращает конфиг по умолчанию
func Default() *Config {
	return &Config{
		Device: DeviceConfig{
			Port: "/dev/ttyS0",
			Baud: 9600,
		},
		Timepulse: TimepulseConfig{
			PulseWidthMs: 5,
			TPIdx:        0,
			AlignToTow:   true,
		},
		Servo: ServoConfig{
			Algorithm: "pid",
			Kp:        0.1,
			Ki:        0.01,
			Kd:        0.001,
			Interval:  "1s",
		},
	}
}

// Load читает конфиг из YAML
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&c)
	return &c, nil
}

// Ptp4lJobs возвращает список заданий ptp4l для источников ptp с start_ptp4l: true.
func (c *ClockSyncConfig) Ptp4lJobs() []ptp4l.Job {
	if c == nil {
		return nil
	}
	var jobs []ptp4l.Job
	for _, s := range c.PrimaryClocks {
		if s.Protocol == "ptp" && s.StartPtp4l && s.Interface != "" {
			jobs = append(jobs, ptp4l.Job{Interface: s.Interface, Domain: s.Domain, Path: s.Ptp4lPath, Args: s.Ptp4lArgs})
		}
	}
	for _, s := range c.SecondaryClocks {
		if s.Protocol == "ptp" && s.StartPtp4l && s.Interface != "" {
			jobs = append(jobs, ptp4l.Job{Interface: s.Interface, Domain: s.Domain, Path: s.Ptp4lPath, Args: s.Ptp4lArgs})
		}
	}
	return jobs
}

func applyDefaults(c *Config) {
	d := Default()
	if c.Device.Port == "" {
		c.Device.Port = d.Device.Port
	}
	if c.Device.Baud == 0 {
		c.Device.Baud = d.Device.Baud
	}
	if c.Timepulse.PulseWidthMs == 0 {
		c.Timepulse.PulseWidthMs = d.Timepulse.PulseWidthMs
	}
	if c.Servo.Algorithm == "" {
		c.Servo.Algorithm = d.Servo.Algorithm
	}
	if c.Servo.Kp == 0 && c.Servo.Ki == 0 && c.Servo.Kd == 0 {
		c.Servo.Kp, c.Servo.Ki, c.Servo.Kd = d.Servo.Kp, d.Servo.Ki, d.Servo.Kd
	}
	if c.Servo.Interval == "" {
		c.Servo.Interval = d.Servo.Interval
	}
	// Timebeat-style: если задан clock_sync, подставить device из первого gnss
	if c.ClockSync != nil {
		for i := range c.ClockSync.PrimaryClocks {
			s := &c.ClockSync.PrimaryClocks[i]
			if (s.Protocol == "gnss" || s.Protocol == "timebeat_opentimecard_mini" || s.Protocol == "nmea") && s.Device == "" {
				s.Device = c.Device.Port
				if s.Baud == 0 {
					s.Baud = c.Device.Baud
				}
			}
		}
		for i := range c.ClockSync.SecondaryClocks {
			s := &c.ClockSync.SecondaryClocks[i]
			if (s.Protocol == "gnss" || s.Protocol == "timebeat_opentimecard_mini" || s.Protocol == "nmea") && s.Device == "" {
				s.Device = c.Device.Port
				if s.Baud == 0 {
					s.Baud = c.Device.Baud
				}
			}
		}
	}
}

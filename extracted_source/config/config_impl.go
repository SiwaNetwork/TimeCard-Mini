// Package config — загрузка конфигурации YAML (реализация по аналогии, не из бинарника).
// appConfig по дизассемблеру: глобальная ссылка на конфиг (0xe0 и др. — SlewClockPossiblyAsync, RunRTCSetLoop, ClockQuality, EnablePPSIfRequired).
package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	appConfigMu sync.RWMutex
	appConfigVar *Config
)

// GetAppConfig возвращает текущий глобальный конфиг (appConfig по дизассемблеру). Для чтения флагов 0xe0 и др.
func GetAppConfig() *Config {
	appConfigMu.RLock()
	defer appConfigMu.RUnlock()
	return appConfigVar
}

// SetAppConfig по дизассемблеру (0x4403300): lock (0x7e7ac88); appConfig = arg (0x7e44be0), startupConfig = arg (0x7e45100) — rep movsq 0xa2 qwords; defer unlock.
func SetAppConfig(cfg *Config) {
	appConfigMu.Lock()
	defer appConfigMu.Unlock()
	appConfigVar = cfg
}

// Config — корневая конфигурация (clock_sync и servo). appConfig+0x330/0x338 — host key path для SSH (configureServerKeys).
// appConfig+0x80 — external_devices; +0x4b1 — taas; +0x3c1 — ptpsquared; +0x318 — ssh; +0x370 — http; +0x398 — syslog.
type Config struct {
	ClockSync       *ClockSyncConfig `yaml:"clock_sync"`
	Servo           ServoConfig      `yaml:"servo"`
	SSHHostKey      string           `yaml:"ssh_host_key"`       // путь к хостовому ключу SSH (appConfig+0x330)
	SSHListenAddr   string           `yaml:"ssh_listen_addr"`    // appConfig+0x320: host для SSH (пусто = "0.0.0.0")
	SSHPort         uint16           `yaml:"ssh_port"`           // appConfig+0x31a: порт (0 = 22)
	ExternalDevices []string       `yaml:"external_devices"` // appConfig+0x80: если len>0 — external_devices.NewController().Start()
	TaasEnabled     bool           `yaml:"taas_enabled"`     // appConfig+0x4b1
	TaasClients     []TaasClientCfg `yaml:"taas_clients"`     // appConfig+0x4d0: список TaaS клиентов для loadConfig
	PTPsquared      *PTPsquaredCfg `yaml:"ptpsquared"`       // appConfig+0x3c1
	SSHEnabled      bool           `yaml:"ssh_enabled"`      // appConfig+0x318
	HTTPEnabled     bool           `yaml:"http_enabled"`     // appConfig+0x370
	HTTPListenAddr  string         `yaml:"http_listen_addr"` // appConfig+0x370/0x378 host
	HTTPPort        uint16         `yaml:"http_port"`        // appConfig+0x37a port (0 = 8080)
	SyslogEnabled   bool           `yaml:"syslog_enabled"`   // appConfig+0x398
}

// PTPsquaredCfg — конфиг ptpsquared (appConfig+0x3c1: включён если != nil).
type PTPsquaredCfg struct {
	Enabled bool `yaml:"enabled"`
}

// TaasClientCfg — конфиг одного TaaS клиента (appConfig+0x4d0: name, identifier, iface, vlan и др.).
type TaasClientCfg struct {
	Name        string `yaml:"name"`
	Identifier  string `yaml:"identifier"`
	Iface       string `yaml:"iface"`
	Vlan        int    `yaml:"vlan"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

// TimebeatRoot — обёртка для YAML с корнем timebeat (как в shiwatime_ru.yml).
type TimebeatRoot struct {
	Timebeat struct {
		ClockSync       *ClockSyncConfig `yaml:"clock_sync"`
		Servo           *ServoConfig     `yaml:"servo"`
		SSHHostKey      string           `yaml:"ssh_host_key"`
		SSHListenAddr   string           `yaml:"ssh_listen_addr"`
		SSHPort         uint16           `yaml:"ssh_port"`
		ExternalDevices []string         `yaml:"external_devices"`
		TaasEnabled     bool             `yaml:"taas_enabled"`
		TaasClients     []TaasClientCfg   `yaml:"taas_clients"`
		PTPsquared      *PTPsquaredCfg   `yaml:"ptpsquared"`
		SSHEnabled      bool             `yaml:"ssh_enabled"`
		HTTPEnabled     bool             `yaml:"http_enabled"`
		HTTPListenAddr  string           `yaml:"http_listen_addr"`
		HTTPPort        uint16           `yaml:"http_port"`
		SyslogEnabled   bool             `yaml:"syslog_enabled"`
	} `yaml:"timebeat"`
}

// ClockSyncConfig — primary/secondary clocks, adjust_clock, step_limit, PPS (appConfig+0x238/0x240).
type ClockSyncConfig struct {
	AdjustClock     bool          `yaml:"adjust_clock"`
	StepLimit       string        `yaml:"step_limit"`
	PrimaryClocks   []ClockSource `yaml:"primary_clocks"`
	SecondaryClocks []ClockSource `yaml:"secondary_clocks"`
	// PPSDevices — список записей "ifName:idx:channel" или "channel:ifName:channelNum" для EnablePPSIfRequired (по дизассемблеру appConfig+0x238/0x240).
	PPSDevices []string `yaml:"pps_devices"`
}

// ClockSource — один источник времени (protocol: ntp, ptp, gnss, pps).
type ClockSource struct {
	Protocol    string `yaml:"protocol"`
	Disable     bool   `yaml:"disable"`
	MonitorOnly bool   `yaml:"monitor_only"`
	// Profile — имя профиля для adjustForProfile (hybrid, G.8265.1, G.8275.1, G.8275.2, enterprise-draft, IEC_IEEE_61850_9_3). По дизассемблеру 0x4414900.
	Profile string `yaml:"profile"`
	// NTP
	IP           string `yaml:"ip"`
	PollInterval string `yaml:"pollinterval"`
	// PTP
	Domain    int    `yaml:"domain"`
	Interface string `yaml:"interface"`
	// GNSS / serial
	Device string `yaml:"device"`
	Baud   int    `yaml:"baud"`
	// PPS
	Pin          int    `yaml:"pin"`
	Index        int    `yaml:"index"`
	LinkedDevice string `yaml:"linked_device"`
	CableDelay   int    `yaml:"cable_delay"`
	Offset       int64  `yaml:"offset"`
}

// ServoConfig — алгоритм и интервал.
type ServoConfig struct {
	Algorithm   string  `yaml:"algorithm"`
	Interval    string  `yaml:"interval"`
	StepAndExit bool    `yaml:"step_and_exit"` // по дизассемблеру: Run.func2 EnableStepAndExitDieTimeout
	Kp          float64 `yaml:"kp"`
	Ki          float64 `yaml:"ki"`
	Kd          float64 `yaml:"kd"`
}

// Load читает конфиг из path. Поддерживает корень timebeat (как в shiwatime_ru.yml).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}

	var root TimebeatRoot
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("config yaml: %w", err)
	}

	cfg := &Config{
		ClockSync:       root.Timebeat.ClockSync,
		SSHHostKey:      root.Timebeat.SSHHostKey,
		SSHListenAddr:   root.Timebeat.SSHListenAddr,
		SSHPort:         root.Timebeat.SSHPort,
		ExternalDevices: root.Timebeat.ExternalDevices,
		TaasEnabled:     root.Timebeat.TaasEnabled,
		TaasClients:     root.Timebeat.TaasClients,
		PTPsquared:      root.Timebeat.PTPsquared,
		SSHEnabled:      root.Timebeat.SSHEnabled,
		HTTPEnabled:     root.Timebeat.HTTPEnabled,
		HTTPListenAddr:  root.Timebeat.HTTPListenAddr,
		HTTPPort:        root.Timebeat.HTTPPort,
		SyslogEnabled:   root.Timebeat.SyslogEnabled,
	}
	if root.Timebeat.Servo != nil {
		cfg.Servo = *root.Timebeat.Servo
	}
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.ClockSync == nil {
		cfg.ClockSync = &ClockSyncConfig{AdjustClock: true}
		return
	}
	if cfg.Servo.Interval == "" {
		cfg.Servo.Interval = "1s"
	}
	if cfg.Servo.Algorithm == "" {
		cfg.Servo.Algorithm = "pid"
	}
}

// ParseInterval разбирает строку интервала (1s, 500ms, 10s).
func ParseInterval(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	if d <= 0 {
		d = time.Second
	}
	return d
}

// ParseStepLimit разбирает порог step (500ms, 15m). Возвращает наносекунды. Пусто = 500ms.
func ParseStepLimit(s string) int64 {
	if s == "" {
		return 500 * int64(time.Millisecond)
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 500 * int64(time.Millisecond)
	}
	return d.Nanoseconds()
}

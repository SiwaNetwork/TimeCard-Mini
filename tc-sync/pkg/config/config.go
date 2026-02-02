// Package config предоставляет конфигурацию clock_sync для использования из Beat и других модулей.
// Формат совместим с Timebeat/shiwatime (shiwatime_ru.yml); неизвестные ключи игнорируются.
package config

// Config — конфигурация clock sync (аналог Timebeat/shiwatime).
type Config struct {
	// Опционально: лицензия и peerids (как в shiwatime)
	LicenseKeyfile string `yaml:"license.keyfile" config:"license.keyfile"`
	ConfigPeerids  string `yaml:"config.peerids" config:"config.peerids"`
	Device         DeviceConfig `yaml:"device" config:"device"`
	Timepulse      TimepulseConfig `yaml:"timepulse" config:"timepulse"`
	Servo          ServoConfig `yaml:"servo" config:"servo"`
	ClockSync      *ClockSyncConfig `yaml:"clock_sync" config:"clock_sync"`
}

// ClockSyncConfig — primary_clocks, secondary_clocks, adjust_clock (как в Timebeat).
type ClockSyncConfig struct {
	AdjustClock     bool   `yaml:"adjust_clock" config:"adjust_clock"`
	StepLimit       string `yaml:"step_limit" config:"step_limit"` // например "15m" — лимит шага времени
	PrimaryClocks   []ClockSource `yaml:"primary_clocks" config:"primary_clocks"`
	SecondaryClocks []ClockSource `yaml:"secondary_clocks" config:"secondary_clocks"`
}

// ClockSource — один источник времени (поля как в shiwatime_ru.yml; неиспользуемые игнорируются).
type ClockSource struct {
	Protocol     string   `yaml:"protocol" config:"protocol"`
	Disable      bool     `yaml:"disable" config:"disable"`
	MonitorOnly  bool     `yaml:"monitor_only" config:"monitor_only"`
	Device       string   `yaml:"device" config:"device"`
	Baud         int      `yaml:"baud" config:"baud"`
	IP           string   `yaml:"ip" config:"ip"`
	PollInterval string   `yaml:"pollinterval" config:"pollinterval"`
	Domain       int      `yaml:"domain" config:"domain"`
	Interface    string   `yaml:"interface" config:"interface"`
	UnicastMasterTable []string `yaml:"unicast_master_table" config:"unicast_master_table"`
	StartPtp4l   bool     `yaml:"start_ptp4l" config:"start_ptp4l"`
	Ptp4lPath    string   `yaml:"ptp4l_path" config:"ptp4l_path"`
	Ptp4lArgs    []string `yaml:"ptp4l_args" config:"ptp4l_args"`
	Pin          int      `yaml:"pin" config:"pin"`
	Index        int      `yaml:"index" config:"index"`
	LinkedDevice string   `yaml:"linked_device" config:"linked_device"`
	CableDelay   int      `yaml:"cable_delay" config:"cable_delay"`
	// Доп. поля из shiwatime (принимаем, не используем пока)
	Offset       int64    `yaml:"offset" config:"offset"`
	Atomic       bool     `yaml:"atomic" config:"atomic"`
	EdgeMode     string   `yaml:"edge_mode" config:"edge_mode"`
	CardConfig   []string `yaml:"card_config" config:"card_config"`
	ServeUnicast bool     `yaml:"serve_unicast" config:"serve_unicast"`
	ServeMulticast bool   `yaml:"serve_multicast" config:"serve_multicast"`
	ServerOnly   bool     `yaml:"server_only" config:"server_only"`
	AnnounceInterval int  `yaml:"announce_interval" config:"announce_interval"`
	SyncInterval int     `yaml:"sync_interval" config:"sync_interval"`
	DelayRequestInterval int `yaml:"delayrequest_interval" config:"delayrequest_interval"`
	DelayStrategy string `yaml:"delay_strategy" config:"delay_strategy"`
	Priority1    int     `yaml:"priority1" config:"priority1"`
	Priority2    int     `yaml:"priority2" config:"priority2"`
	MaxUnicastSubscribers int `yaml:"max_unicast_subscribers" config:"max_unicast_subscribers"`
	UseLayer2    bool    `yaml:"use_layer2" config:"use_layer2"`
	Profile      string  `yaml:"profile" config:"profile"`
	OcpDevice    int     `yaml:"ocp_device" config:"ocp_device"`
	OscillatorType string `yaml:"oscillator_type" config:"oscillator_type"`
}

// DeviceConfig — порт и скорость.
type DeviceConfig struct {
	Port string `yaml:"port" config:"port"`
	Baud int    `yaml:"baud" config:"baud"`
}

// TimepulseConfig — параметры CFG-TP5.
type TimepulseConfig struct {
	PulseWidthMs     float64 `yaml:"pulse_width_ms" config:"pulse_width_ms"`
	TPIdx            uint8   `yaml:"tp_idx" config:"tp_idx"`
	AntCableDelayNs  int16   `yaml:"ant_cable_delay_ns" config:"ant_cable_delay_ns"`
	AlignToTow       bool    `yaml:"align_to_tow" config:"align_to_tow"`
}

// ServoConfig — PID/PI параметры.
type ServoConfig struct {
	Algorithm string  `yaml:"algorithm" config:"algorithm"`
	Kp        float64 `yaml:"kp" config:"kp"`
	Ki        float64 `yaml:"ki" config:"ki"`
	Kd        float64 `yaml:"kd" config:"kd"`
	Interval  string  `yaml:"interval" config:"interval"`
}

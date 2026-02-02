package utility

import "time"

// Автоматически извлечено из timebeat-2.2.20

func Abs() {
	// TODO: реконструировать
}

// ParseTimeString по дизассемблеру (ParseTimeString@@Base 0x4404c20): разбор строки длительности (например "16s", "500ms"); при ошибке возврат defaultVal.
// Вызывается из RunWakeupServoLoop, configureAndStartClient, getStepAndExit, setRTCIntervalOrExit и др.
func ParseTimeString(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	if d < 0 {
		return defaultVal
	}
	return d
}

func TrimLeadingZeroAndTrailingNulls() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}


//go:build linux

package clockadj

import (
	"time"

	"golang.org/x/sys/unix"
)

// Slew применяет плавную коррекцию времени через adjtimex (offset в наносекундах).
// На Linux требует CAP_SYS_TIME или root.
func Slew(offsetNs int64) error {
	// adjtimex: offset в микросекундах (старый API) или используем clock_adjtime с ADJ_SETOFFSET
	// На Linux 2.6.26+ используем ADJ_OFFSET (микросекунды)
	offsetUs := offsetNs / 1000
	if offsetUs == 0 {
		return nil
	}
	buf := &unix.Timex{
		Modes: unix.ADJ_OFFSET,
		Offset: offsetUs,
	}
	_, err := unix.Adjtimex(buf)
	return err
}

// SetFrequency устанавливает коррекцию частоты (ppm — части на миллион).
// Положительное значение ускоряет часы. Требует CAP_SYS_TIME или root.
func SetFrequency(ppm float64) error {
	// freq в adjtimex: единицы = 65536 * 1e-6 Hz, т.е. ppm * 65536 / 1e6 не подходит.
	// В Linux kernel: offset и freq. freq = (ppm / 1e6) * 65536 в старых единицах?
	// По документации: "Freq" in timex is in "scaled ppm" — (ppm * 65536) для значения в ppm.
	scaledPpm := int64(ppm * 65536)
	if scaledPpm == 0 {
		return nil
	}
	buf := &unix.Timex{
		Modes: unix.ADJ_FREQUENCY,
		Freq:  scaledPpm,
	}
	_, err := unix.Adjtimex(buf)
	return err
}

// Step устанавливает системное время (скачок). Требует CAP_SYS_TIME или root.
func Step(t time.Time) error {
	ts := unix.NsecToTimespec(t.UnixNano())
	return unix.ClockSettime(unix.CLOCK_REALTIME, &ts)
}

// GetFrequency возвращает текущую коррекцию частоты из ядра (ppm).
// Читает adjtimex без изменения; Freq в timex — scaled ppm (freq/65536 = ppm).
func GetFrequency() (ppm float64, err error) {
	buf := &unix.Timex{}
	_, err = unix.Adjtimex(buf)
	if err != nil {
		return 0, err
	}
	ppm = float64(buf.Freq) / 65536
	return ppm, nil
}

// GranularityNs выполняет простое измерение гранулярности часов (разрешение clock_gettime).
// Делает несколько вызовов clock_gettime и возвращает минимальный ненулевой интервал в наносекундах.
func GranularityNs() int64 {
	const rounds = 20
	var minDt int64 = 1e9
	for i := 0; i < rounds; i++ {
		var t1, t2 unix.Timespec
		_ = unix.ClockGettime(unix.CLOCK_REALTIME, &t1)
		_ = unix.ClockGettime(unix.CLOCK_REALTIME, &t2)
		dt := (t2.Sec-t1.Sec)*1e9 + int64(t2.Nsec-t1.Nsec)
		if dt > 0 && dt < minDt {
			minDt = dt
		}
	}
	if minDt == 1e9 {
		return 0
	}
	return minDt
}

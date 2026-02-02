package common

import "time"

// NTP epoch: 1900-01-01 00:00:00 UTC (по дизассемблеру receiveMessage: TimeToNtpTime@@Base 0x45b69c0).
const ntpEpochUnix = int64(-2208988800)

// TimeToNtpTime по дизассемблеру (TimeToNtpTime@@Base): конвертация time в NTP 64-bit (32 bit sec since 1900 + 32 bit frac).
// Возврат (sec<<32 | frac) или пары (sec, frac) для записи в пакет big-endian.
func TimeToNtpTime(t time.Time) (sec uint32, frac uint32) {
	unix := t.Unix()
	nsec := t.Nanosecond()
	secNTP := uint32(unix - ntpEpochUnix)
	fracNTP := uint32(uint64(nsec) * 0x100000000 / 1e9)
	return secNTP, fracNTP
}

// TimeToNtpTimeShort — укороченная форма: возвращает NTP-время для текущего момента. Отдельного дампа в index нет; реконструкция как обёртка над TimeToNtpTime(time.Now()).
func TimeToNtpTimeShort() (sec uint32, frac uint32) {
	return TimeToNtpTime(time.Now())
}

func init() {}


package client

import (
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp/common"
)

// Реконструировано по дизассемблеру бинарника timebeat-2.2.20 (getTime 0x45b5400, getTime.func1 0x45b6000).

// Msg — NTP message (первые байты). По дизассемблеру (*msg).getMode: movzbl (%rax), and $0x7.
type Msg []byte

// GetMode по дизассемблеру (__msg_.getMode.txt): первый байт & 7.
func (m *Msg) GetMode() uint8 {
	if m == nil || len(*m) == 0 {
		return 0
	}
	return (*m)[0] & 7
}

// Offset по дизассемблеру (client.offset@@Base): sub1 = t2.Sub(t1), sub2 = t3.Sub(t4); return (sub1+sub2)>>1 с округлением (sum + sum>>63)>>1.
func Offset(t1, t2, t3, t4 time.Time) int64 {
	sub1 := t2.Sub(t1).Nanoseconds()
	sub2 := t3.Sub(t4).Nanoseconds()
	sum := sub1 + sub2
	return (sum + sum>>63) >> 1
}

// GetTime по дизассемблеру (client.getTime 0x45b5400): version default 4; ResolveUDPAddr(host:port), DialUDP; defer getTime.func1 (net.(*conn).Close); SetDeadline(GetPreciseTime+timeout); req mode=3 version=4, originate=ToNtpTime(t1); Write, Read; validate mode 4, originate match, RTT>=0; return (GetTimeResponse, nil). Реализация в пакете ntp.GetTime.
func GetTime(host, port string, version int, timeout time.Duration) (ntp.GetTimeResponse, error) {
	return ntp.GetTime(host, port, version, timeout)
}

// getTime.func1 (0x45b6000): defer-функция — вызывает net.(*conn).Close для UDP-соединения.

func Duration() {
	// TODO: реконструировать
}

func NewClient() {
	// TODO: реконструировать
}

func QueryWithOptions() {
	// TODO: реконструировать
}

func RunNTPClient() {
	// TODO: реконструировать
}

// Time возвращает оценку времени сервера по одному NTP round-trip (делегирует ntp.Query).
func Time(host string) (time.Time, error) {
	return ntp.Query(host)
}

func Validate() {
	// TODO: реконструировать
}

func func1() {
	// TODO: реконструировать
}

func getLeap() {
	// TODO: реконструировать
}

// getMode — см. (*Msg).GetMode (экспортирован как GetMode).

// getTime — см. GetTime (экспортированная функция).

func getVersion() {
	// TODO: реконструировать
}

func init() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}

func kissCode() {
	// TODO: реконструировать
}

func minError() {
	// TODO: реконструировать
}

func ntpEpoch() {
	// TODO: реконструировать
}

// offset — см. Offset (экспортированная функция).

func parseTime() {
	// TODO: реконструировать
}

func rootDistance() {
	// TODO: реконструировать
}

func rtt() {
	// TODO: реконструировать
}

func setLeap() {
	// TODO: реконструировать
}

func setMode() {
	// TODO: реконструировать
}

func setVersion() {
	// TODO: реконструировать
}

func toInterval() {
	// TODO: реконструировать
}

// ToNtpTime по дизассемблеру (client.toNtpTime@@Base 0x45b4f00): time − ntpEpoch → 64-bit NTP (sec в старших 32 бит, frac в младших). Делегирует common.TimeToNtpTime.
func ToNtpTime(t time.Time) (sec uint32, frac uint32) {
	return common.TimeToNtpTime(t)
}


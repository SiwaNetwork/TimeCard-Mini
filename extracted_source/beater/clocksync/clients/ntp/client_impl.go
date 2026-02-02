// Package ntp — NTP клиент. Реконструировано по дизассемблеру: NewController.func1, Start→loadConfig, loadConfig→GetStore+Range(ConfigureTimeSource), ConfigureTimeSource(key=="ntp")→configureAndStartClient/Server, client.offset→(sub1+sub2)>>1.
package ntp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp/common"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp/server"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/beater/utility"
)

// Controller по дизассемблеру: 0x00=offsets, 0x08=logger (NewController.func1).
type Controller struct {
	mu      sync.Mutex
	offsets *servo.Offsets
	logger  *logging.Logger
}

var (
	ntpOnce      sync.Once
	ntpController *Controller
)

// NewController по дизассемблеру (NewController@@Base): once.Do(func1); func1 — NewLogger, newobject(Controller), 0=offsets 0x8(rdx), 8=logger, controller=package var.
func NewController(offsets *servo.Offsets) *Controller {
	ntpOnce.Do(func() {
		ntpController = &Controller{
			offsets: offsets,
			logger:  logging.NewLogger("ntp-controller"),
		}
	})
	return ntpController
}

// GetController возвращает синглтон контроллера (по бинарнику — package var после NewController).
func GetController() *Controller {
	return ntpController
}

// NTPTimeSourceConfig — конфиг из sources (key "ntp"); по ConfigureTimeSource: 0x299=isClient, 0x288=isServer.
type NTPTimeSourceConfig struct {
	Host         string
	PollInterval string
	IsClient     bool
	IsServer     bool
	// Category для GetSourceCandidates: 1=primary, 2=secondary (servo.CategoryPrimary/CategorySecondary).
	Category int
}

// ConfigureTimeSource по дизассемблеру (0x45bc820): type assert *Controller; key len==3, cmpw $0x746e \"nt\", cmpb $0x70 'p' → \"ntp\"; byte 0x299=isClient → configureAndStartClient; byte 0x288=isServer → configureAndStartServer.
// Поддержка key \"ntp\"+*NTPTimeSourceConfig и value *sources.TimeSourceConfig (Type==\"ntp\") из store.Range.
func (c *Controller) ConfigureTimeSource(key, value interface{}) {
	// Прямой ключ "ntp" (len 3) из конфига
	keyStr, ok := key.(string)
	if ok && len(keyStr) == 3 && keyStr == "ntp" {
		if cfg, ok := value.(*NTPTimeSourceConfig); ok && cfg != nil {
			if cfg.IsClient {
				c.configureAndStartClient(cfg)
			}
			if cfg.IsServer {
				c.configureAndStartServer(cfg)
			}
		}
		return
	}
	// Из store.Range: value=*sources.TimeSourceConfig, Type=="ntp"
	if cfg, ok := value.(*sources.TimeSourceConfig); ok && cfg != nil && cfg.Type == "ntp" {
		ntpCfg := &NTPTimeSourceConfig{
			Host:     cfg.Name,
			Category: 1,
			IsClient: true,
		}
		if cfg.Category != 0 {
			ntpCfg.Category = int(cfg.Category)
		} else if cfg.Index != 0 {
			ntpCfg.Category = int(cfg.Index)
		}
		c.configureAndStartClient(ntpCfg)
	}
}

// configureAndStartClient по дизассемблеру (0x45bc9e0): pollInterval nil → default string len 2 (0x2); ParseTimeString; при ошибке Logger.Error; иначе newobject(Client), go func1 (runPoller).
func (c *Controller) configureAndStartClient(cfg *NTPTimeSourceConfig) {
	if cfg.Host == "" {
		return
	}
	defaultPoll := 2 * time.Second // по дампу: movq $0x2, 0x340 — default string length 2 ("2")
	interval := utility.ParseTimeString(cfg.PollInterval, defaultPoll)
	c.mu.Lock()
	off := c.offsets
	c.mu.Unlock()
	if off == nil {
		return
	}
	category := cfg.Category
	if category == 0 {
		category = 1 // default primary
	}
	go c.runPoller(cfg.Host, interval, off, category)
}

// configureAndStartServer по дизассемблеру (__Controller_.configureAndStartServer 0x45bcce0): NewServer(offsets); при err — Logger.Error; иначе go func1 (запуск сервера).
func (c *Controller) configureAndStartServer(cfg *NTPTimeSourceConfig) {
	_ = cfg
	srv, err := server.NewServer(c.offsets)
	if err != nil {
		c.logger.Error("failed to start NTP server")
		return
	}
	go srv.Serve()
}

// loadConfig по дизассемблеру (0x45bc7a0): GetStore(); store+8 → sync.Map; Range(ConfigureTimeSource-fm, controller).
func (c *Controller) loadConfig() {
	store := sources.GetStore()
	if store == nil {
		return
	}
	m := store.GetSources()
	if m == nil {
		return
	}
	m.Range(func(key, value interface{}) bool {
		c.ConfigureTimeSource(key, value)
		return true
	})
}

// Start по дизассемблеру (Start@@Base): вызов loadConfig.
func (c *Controller) Start() {
	c.loadConfig()
}

// runPoller периодически опрашивает NTP и регистрирует offset в Offsets.
// category: 1=primary, 2=secondary для GetSourceCandidates.
func (c *Controller) runPoller(host string, interval time.Duration, offsets *servo.Offsets, category int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	sourceID := fmt.Sprintf("ntp:%s", host)
	for range ticker.C {
		offsetNs, err := QueryOffset(host)
		if err != nil {
			continue
		}
		if category != 0 {
			offsets.RegisterObservationWithCategory(sourceID, offsetNs, category)
		} else {
			offsets.RegisterObservation(sourceID, offsetNs)
		}
	}
}

// QueryOffset возвращает смещение в наносекундах. По дизассемблеру client.offset: sub1=t2.Sub(t1), sub2=t3.Sub(t4); (sub1+sub2)>>1 с округлением (shr 0x3f, lea, sar 1).
func QueryOffset(host string) (int64, error) {
	t1, t2, t3, t4, err := queryRoundTrip(host)
	if err != nil {
		return 0, err
	}
	sub1 := t2.Sub(t1).Nanoseconds()
	sub2 := t3.Sub(t4).Nanoseconds()
	sum := sub1 + sub2
	// По дизассемблеру client.offset: add, shr 0x3f (sign), lea (sum+sign), sar 1 → (sum + (sum>>63)) >> 1
	return (sum + sum>>63) >> 1, nil
}

// По дизассемблеру getTime (0x45b5400): ResolveUDPAddr(host:123), DialUDP; SetDeadline(GetPreciseTime()+timeout); req: mode=3, version=4 (0x23), originate=TimeToNtpTime(t1) в байты 40-47; Write, Read; t4=GetPreciseTime(); проверка resp.Mode&7==4; originate в ответе (24-31) == запрос (40-47); RTT>=0; t2=Receive(32-39), t3=Transmit(40-47).
var (
	errNTPMode       = errors.New("ntp: response mode is not server (4)")
	errNTPOriginate  = errors.New("ntp: originate timestamp mismatch")
	errNTPNegativeRTT = errors.New("ntp: negative RTT")
)

// GetTimeResponse — результат getTime (0x45b5400): t1=local send, t2=server recv, t3=server xmit, t4=local recv; по дизассемблеру возврат в 0x90(rsp), error в 0x50(rsp).
type GetTimeResponse struct {
	T1, T2, T3, T4 time.Time
}

// GetTime по дизассемблеру (client.getTime 0x45b5400): version default 4 (0x138==0 → 4); ResolveUDPAddr(host:port), DialUDP; defer getTime.func1 (Close); SetDeadline(GetPreciseTime+timeout); req mode=3 version=4; Write/Read; validate mode 4, originate match, RTT>=0; return (response, nil).
// host — адрес NTP сервера; port — пустая строка = ":123"; timeout 0 = 5s (0x12a05f200 в бинарнике).
func GetTime(host, port string, version int, timeout time.Duration) (GetTimeResponse, error) {
	if version == 0 {
		version = 4
	}
	if port == "" {
		port = "123"
	}
	addrStr := net.JoinHostPort(host, port)
	t1, t2, t3, t4, err := queryRoundTripWithAddr(addrStr, timeout)
	if err != nil {
		return GetTimeResponse{}, err
	}
	return GetTimeResponse{T1: t1, T2: t2, T3: t3, T4: t4}, nil
}

// queryRoundTripWithAddr выполняет обмен по адресу addr (host:port). timeout 0 = 5s.
func queryRoundTripWithAddr(addr string, timeout time.Duration) (t1, t2, t3, t4 time.Time, err error) {
	addrUDP, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	conn, err := net.DialUDP("udp", nil, addrUDP)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	defer conn.Close()
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	t1 = adjusttime.GetPreciseTime()
	conn.SetDeadline(t1.Add(timeout))
	req := make([]byte, 48)
	req[0] = 0x23 // mode=3, version=4
	sec, frac := common.TimeToNtpTime(t1)
	binary.BigEndian.PutUint32(req[40:44], sec)
	binary.BigEndian.PutUint32(req[44:48], frac)
	if _, err := conn.Write(req); err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	resp := make([]byte, 48)
	n, err := conn.Read(resp)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	if n < 48 {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, errors.New("ntp: short response")
	}
	t4 = adjusttime.GetPreciseTime()
	if resp[0]&0x07 != 4 {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, errNTPMode
	}
	if binary.BigEndian.Uint32(resp[24:28]) != sec || binary.BigEndian.Uint32(resp[28:32]) != frac {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, errNTPOriginate
	}
	if t4.Sub(t1) < 0 {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, errNTPNegativeRTT
	}
	t2 = ntpBytesToTime(resp[32:40])
	t3 = ntpBytesToTime(resp[40:48])
	return t1, t2, t3, t4, nil
}

// queryRoundTrip выполняет один NTP обмен и возвращает t1=local send, t2=server recv, t3=server xmit, t4=local recv. По дизассемблеру getTime: GetPreciseTime для t1/t4, запрос с originate, валидация mode 4 и совпадение originate.
func queryRoundTrip(host string) (t1, t2, t3, t4 time.Time, err error) {
	return queryRoundTripWithAddr(host+":123", 0)
}

func ntpBytesToTime(b []byte) time.Time {
	if len(b) < 8 {
		return time.Time{}
	}
	sec := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	frac := uint32(b[4])<<24 | uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7])
	ntpEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	return ntpEpoch.Add(time.Duration(sec)*time.Second + time.Duration(frac)*time.Second/0x100000000).UTC()
}

// Query возвращает оценку времени сервера по одному round-trip (для совместимости API).
func Query(host string) (time.Time, error) {
	t1, t2, t3, t4, err := queryRoundTrip(host)
	if err != nil {
		return time.Time{}, err
	}
	sub1 := t2.Sub(t1).Nanoseconds()
	sub2 := t3.Sub(t4).Nanoseconds()
	offsetNs := (sub1 + sub2) / 2
	return t4.Add(time.Duration(offsetNs)), nil
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	if d < time.Second {
		d = time.Second
	}
	return d
}

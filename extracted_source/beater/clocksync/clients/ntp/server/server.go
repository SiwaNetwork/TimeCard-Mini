package server

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/ntp/common"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo/adjusttime"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/servo"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// ClockQuality по дизассемблеру UpdateTimeSource (0x45bbc60): +0x8 reference timestamp (runReferenceUpdateLoop: xchg); +0x10/+0x18 RootDisp/RootDelay; Protocol, QualityType.
type ClockQuality struct {
	RefSec   uint32 // NTP reference timestamp (sec) — runReferenceUpdateLoop 0x45bbdf8 xchg 0x8(rdx)
	RefFrac  uint32
	RefID    uint32
	RootDisp uint32
	RootDelay uint32
	RootDispHigh uint32
	Protocol  string
	QualityType byte
}

// Server по дизассемблеру NewServer (0x45bc0a0): 0=logger, 0x8=packetConn (RunServer: ListenPacket), 0x10=ch, 0x18=offsets, 0x20=clockQualityPtr.
type Server struct {
	logger          *logging.Logger
	packetConn      net.PacketConn
	ch              chan struct{}
	offsets         *servo.Offsets
	clockQualityPtr atomic.Pointer[ClockQuality]
}

// NewServer по дизассемблеру (NewServer@@Base 0x45bc0a0): NewLogger("ntp-server"), makechan(10), newobject(Server); 0=logger, 0x18=offsets, 0x10=ch; return (*Server, nil).
func NewServer(offsets *servo.Offsets) (*Server, error) {
	return &Server{
		logger:  logging.NewLogger("ntp-server"),
		offsets: offsets,
		ch:      make(chan struct{}, 10),
	}, nil
}

// receiveMessage по дизассемблеру (0x45bb7c0): проверка len/mode/version; заполнение resp из s.clockQualityPtr (0x20): byte1=cq+4, byte3=cq+0x18, 0x44=cq+0x14 RootDelay, 0x48=cq+0x10 RootDisp, 0x4c=RefSec(bswap), 0x50=8 bytes cq+8; originate=req[40:48]; receive/transmit=GetPreciseTime; binary.Write.
func (s *Server) receiveMessage(req []byte) []byte {
	if len(req) < 48 {
		return nil
	}
	mode := req[0] & 0x07
	version := (req[0] >> 3) & 0x07
	if mode != 3 || version != 4 {
		return nil
	}
	resp := make([]byte, 48)
	resp[0] = 0x24 // li=0, vn=4, mode=4
	resp[1] = 0x01 // stratum (default)
	resp[2] = 0x00 // poll
	resp[3] = 0xec // precision -6 (2^-6 sec)
	// bytes 4-11: root delay, root dispersion (RFC 5905)
	// bytes 12-15: reference identifier; 16-23: reference timestamp
	if cq := s.GetClockQuality(); cq != nil {
		resp[1] = byte(cq.QualityType) // по дампу: cq+4 → 0x41
		if cq.RootDispHigh != 0 {
			resp[2] = byte(cq.RootDispHigh) // по дампу: cq+0x18 → 0x43 (poll)
		}
		binary.BigEndian.PutUint32(resp[4:8], cq.RootDelay)
		binary.BigEndian.PutUint32(resp[8:12], cq.RootDisp)
		binary.BigEndian.PutUint32(resp[12:16], cq.RefID)
		binary.BigEndian.PutUint32(resp[16:20], cq.RefSec)
		binary.BigEndian.PutUint32(resp[20:24], cq.RefFrac)
	}
	// originate timestamp = copy from request (bytes 40-47 → resp 24-31)
	copy(resp[24:32], req[40:48])
	// receive timestamp = now (bytes 32-39)
	recvSec, recvFrac := common.TimeToNtpTime(adjusttime.GetPreciseTime())
	binary.BigEndian.PutUint32(resp[32:36], recvSec)
	binary.BigEndian.PutUint32(resp[36:40], recvFrac)
	// transmit timestamp = now (bytes 40-47)
	xmitSec, xmitFrac := common.TimeToNtpTime(adjusttime.GetPreciseTime())
	binary.BigEndian.PutUint32(resp[40:44], xmitSec)
	binary.BigEndian.PutUint32(resp[44:48], xmitFrac)
	return resp
}

// runMessageReceiver по дизассемблеру (0x45bbfc0): Sleep(15s); в бинарнике PTPSocket.RunSocket; цикл select → receiveMessage. У нас: Sleep(15s), цикл ReadFrom/receiveMessage/WriteTo по s.packetConn.
func (s *Server) runMessageReceiver() {
	time.Sleep(15 * time.Second)
	pc := s.packetConn
	if pc == nil {
		return
	}
	buf := make([]byte, 48)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		if n < 48 {
			continue
		}
		resp := s.receiveMessage(buf[:n])
		if resp != nil {
			_, _ = pc.WriteTo(resp, addr)
		}
	}
}

// Serve запускает NTP-сервер; по дизассемблеру: UDP listen, ReadFrom, receiveMessage, WriteTo. Без начальной задержки 15s (её даёт runMessageReceiver при запуске через RunServer).
func (s *Server) Serve() {
	pc, err := net.ListenPacket("udp4", ":123")
	if err != nil {
		s.logger.Error("NTP server listen: " + err.Error())
		return
	}
	defer pc.Close()
	buf := make([]byte, 48)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}
		if n < 48 {
			continue
		}
		resp := s.receiveMessage(buf[:n])
		if resp != nil {
			_, _ = pc.WriteTo(resp, addr)
		}
	}
}

// RunServer по дизассемблеру (0x45bc3c0): newproc(func1), newproc(func2), newproc(func3) — три горутины: runClockQualityReceiverLoop, runReferenceUpdateLoop, runMessageReceiver. У нас: NewServer(offsets), ListenPacket, s.packetConn=pc, затем go по трём циклам.
func RunServer(offsets *servo.Offsets) error {
	if offsets == nil {
		return nil
	}
	s, err := NewServer(offsets)
	if err != nil {
		return err
	}
	pc, err := net.ListenPacket("udp4", ":123")
	if err != nil {
		return err
	}
	s.packetConn = pc
	go s.runClockQualityReceiverLoop()
	go s.runReferenceUpdateLoop()
	go s.runMessageReceiver()
	return nil
}

// UpdateTimeSource по дизассемблеру (0x45bbc60): newobject; server+0x20 → текущий *ClockQuality; копирование +0x8/+0x18/+0x10/+0x14 из текущего в obj; quality (bl): 0x10→"PPS"+type=1, 0x20→"GPS"+type=1, 0x40/0x50+sourceName→первые 3 байта+type=2, иначе type=0x10; atomic.SwapPointer(server+0x20, obj).
func (s *Server) UpdateTimeSource(quality byte, sourceName *string) {
	q := &ClockQuality{}
	if cur := s.clockQualityPtr.Load(); cur != nil {
		q.RefSec = cur.RefSec
		q.RefFrac = cur.RefFrac
		q.RefID = cur.RefID
		q.RootDisp = cur.RootDisp
		q.RootDelay = cur.RootDelay
		q.RootDispHigh = cur.RootDispHigh
	}
	switch {
	case quality == 0x10:
		q.Protocol = "PPS"
		q.QualityType = 1
	case quality == 0x20:
		q.Protocol = "GPS"
		q.QualityType = 1
	case (quality == 0x40 || quality == 0x50) && sourceName != nil && len(*sourceName) >= 3:
		q.Protocol = (*sourceName)[:3] // в бинарнике movzwl+movzbl — первые 3 байта
		q.QualityType = 2
	default:
		q.QualityType = 0x10
	}
	s.clockQualityPtr.Store(q)
}

// GetClockQuality возвращает текущий ClockQuality (по дизассемблеру чтение s+0x20).
func (s *Server) GetClockQuality() *ClockQuality {
	return s.clockQualityPtr.Load()
}

// runClockQualityReceiverLoop по дизассемблеру (0x45bbb60): servo.GetController().GetClockQuality().Subscribe(); select на канал; при приходе — GetTimeSource(), GetSourceIPAddr(), UpdateTimeSource(quality, &addr).
func (s *Server) runClockQualityReceiverLoop() {
	ctrl := servo.GetController()
	if ctrl == nil {
		return
	}
	cq := ctrl.GetClockQuality()
	if cq == nil {
		return
	}
	ch := cq.Subscribe()
	for range ch {
		quality := cq.GetTimeSource()
		addr := cq.GetSourceIPAddr()
		s.UpdateTimeSource(byte(quality), &addr)
	}
}

// runReferenceUpdateLoop по дизассемблеру (0x45bbda0): NewTicker(0x12a05f200)=5s; GetController(hostclocks), GetTimeOfLastMasterClockAdjustment, TimeToNtpTime; xchg 0x8(clockQuality) — обновление reference timestamp; GetClockWithURI("system"), обновление RootDisp/RootDelay из offset.
func (s *Server) runReferenceUpdateLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		hcc := hostclocks.GetController()
		if hcc == nil {
			continue
		}
		t := hcc.GetTimeOfLastMasterClockAdjustment()
		sec, frac := common.TimeToNtpTime(t)
		q := s.clockQualityPtr.Load()
		if q == nil {
			q = &ClockQuality{}
		}
		next := &ClockQuality{
			RefSec:         sec,
			RefFrac:        frac,
			RefID:          q.RefID,
			RootDisp:       q.RootDisp,
			RootDelay:      q.RootDelay,
			RootDispHigh:   q.RootDispHigh,
			Protocol:       q.Protocol,
			QualityType:    q.QualityType,
		}
		s.clockQualityPtr.Store(next)
	}
}


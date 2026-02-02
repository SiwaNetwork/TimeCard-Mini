package generic_gnss_device

import (
	"bufio"
	"bytes"
	"io"
	"errors"

	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/helper/ubx"
)

const readloopBufSize = 0x400

// GNSSDeviceConfig — конфиг для makeSerialDevice (Port, Baud по дизассемблеру 0x45affe0).
type GNSSDeviceConfig struct {
	Port string
	Baud int
}

// makeSerialDevice по дизассемблеру (0x45affe0): открывает serial по d.Config (Port, Baud).
// Без зависимости go.bug.st/serial возвращает nil; при nil NewDevice отдаёт stubGNSSDevice.
func (d *GenericGNSSDevice) makeSerialDevice() interface{} {
	if d.Config == nil {
		return nil
	}
	if c, ok := d.Config.(*GNSSDeviceConfig); ok && c.Port != "" && c.Baud > 0 {
		// TODO: при добавлении go.bug.st/serial — serial.Open(c.Port, &serial.Mode{BaudRate: c.Baud})
		return nil
	}
	return nil
}

// Start реализует DeviceInterface для *GenericGNSSDevice; запускает runReadloop в горутине.
func (d *GenericGNSSDevice) Start() {
	if d.Reader != nil {
		go d.runReadloop()
	}
}

// GetObservationChan реализует GNSSChannels для *GenericGNSSDevice.
func (d *GenericGNSSDevice) GetObservationChan() chan interface{} { return d.ChObs }

// GetTaiChan реализует GNSSChannels для *GenericGNSSDevice.
func (d *GenericGNSSDevice) GetTaiChan() chan TAIEvent { return d.ChTAI }

// GetGSVChan реализует GNSSChannelsWithGSV для *GenericGNSSDevice.
func (d *GenericGNSSDevice) GetGSVChan() chan string { return d.ChGSV }

// containsNMEA по дизассемблеру: включён ли приём NMEA (конфиг/флаг). Заглушка: false.
func (d *GenericGNSSDevice) containsNMEA() bool {
	return false
}

// runReadloop по дизассемблеру (0x45af3e0): чтение из d.Reader, буфер, поиск UBX_HEADER,
// ubx.IsEntireUBXMessageReceived → ubx.ToUBXMessage → ChUBX; при NMEA — ToNMEAMessage, Parse, recordGSV/RMC.
func (d *GenericGNSSDevice) runReadloop() {
	if d.Reader == nil {
		return
	}
	br, ok := d.Reader.(*bufio.Reader)
	if !ok {
		return
	}
	readBuf := make([]byte, readloopBufSize)
	var buf []byte
	for {
		n, err := br.Read(readBuf)
		if n > 0 {
			buf = append(buf, readBuf[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			// log and continue or return
			break
		}
		if n == 0 {
			continue
		}
		// Обработка буфера: поиск UBX, затем при необходимости NMEA
		for len(buf) >= 2 {
			idx := bytes.Index(buf, ubx.UBX_HEADER)
			if idx < 0 {
				// Нет заголовка UBX — оставляем последний байт (может дописать следующий Read)
				if len(buf) > 1 {
					buf = buf[len(buf)-1:]
				}
				break
			}
			if idx > 0 {
				buf = buf[idx:]
				continue
			}
			// idx == 0: проверяем полный UBX пакет
			if len(buf) < 8 {
				break
			}
			if !ubx.IsEntireUBXMessageReceived(buf, 0, len(buf)) {
				break
			}
			msg, consumed := ubx.ToUBXMessage(buf, 0, len(buf))
			if consumed > 0 {
				if d.ChUBX != nil && msg != nil {
					select {
					case d.ChUBX <- msg:
					default:
					}
				}
				buf = buf[consumed:]
				continue
			}
			buf = buf[1:]
		}
		// Ограничиваем размер буфера
		if len(buf) > 65536 {
			buf = buf[len(buf)-8192:]
		}
	}
}

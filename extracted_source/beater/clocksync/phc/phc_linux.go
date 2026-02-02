//go:build linux

package phc

import (
	"syscall"
	"unsafe"
)

// PTP_SYS_OFFSET — ioctl для получения сэмплов PHC vs system clock (linux/ptp_clock.h).
const PTP_SYS_OFFSET = 0x40103d01

// PTP_PEROUT_REQUEST — ioctl для периодического выхода (linux/ptp_clock.h).
const PTP_PEROUT_REQUEST_Linux uintptr = 0x40103d02

func init() {
	getPHCToSysClockSamplesBasicImpl = getPHCToSysClockSamplesBasicLinux
	PTP_ENABLE_PPS = 0x40043d01
	PTP_PIN_SETFUNC = 0x40103d06
	PTP_PEROUT_REQUEST = PTP_PEROUT_REQUEST_Linux
	Ioctl = ioctlLinux
}

// ioctlLinux по дизассемблеру (Ioctl@@Base 0x4589220): syscall SYS_IOCTL(fd, request, ptr).
func ioctlLinux(fd int, request uintptr, ptr unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, uintptr(ptr))
	if errno != 0 {
		return errno
	}
	return nil
}

func getPHCToSysClockSamplesBasicLinux(d *PHCDevice, nSamples int) *ptpSysOffsetData {
	if d == nil || d.FD <= 0 {
		return nil
	}
	if nSamples <= 0 || nSamples > maxPTPOffsetSamples {
		nSamples = maxPTPOffsetSamples
	}
	// По дизассемблеру: newobject(type), (struct).n_samples = arg2, Ioctl(phc.0x60, PTP_SYS_OFFSET, &struct).
	req := &ptpSysOffsetData{N_samples: uint32(nSamples)}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(d.FD), uintptr(PTP_SYS_OFFSET), uintptr(unsafe.Pointer(req)))
	if errno != 0 {
		return nil
	}
	return req
}

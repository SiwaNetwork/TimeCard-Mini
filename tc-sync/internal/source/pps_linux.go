//go:build linux

package source

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux PPS API (include/uapi/linux/pps.h):
// PPS_FETCH = _IOWR('p', 0xa4, struct pps_fdata)
// pps_fdata { pps_kinfo info; pps_ktime timeout; }
// pps_kinfo: assert_sequence, clear_sequence, assert_tu (sec int64, nsec int32, flags uint32), clear_tu, current_mode
// assert_tu at offset 8 in pps_kinfo
const (
	ppsIoctlFetch = 0xc00470a4 // _IOWR('p', 0xa4, 64)
	ppsFdataSize  = 64
	ppsAssertSec  = 8  // offset of assert_tu.sec in pps_kinfo
	ppsAssertNsec = 16 // offset of assert_tu.nsec
)

func init() {
	getPPSSubSecond = fetchPPSSubSecond
}

// fetchPPSSubSecond читает последний PPS assert с /dev/pps{index}, возвращает подсекунду (nsec).
func fetchPPSSubSecond(ppsIndex int) (nsec int32, ok bool) {
	path := filepath.Join("/dev", fmt.Sprintf("pps%d", ppsIndex))
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	buf := make([]byte, ppsFdataSize)
	// We pass it via ptr; syscall ioctl takes (fd, request, argp).
	// In Go we need IoctlGetPointer. Let me use unix.IoctlGetPtr - but we need to pass pointer to buf.
	// Simpler: use os.File and syscall. IoctlGetInt is for int. We need IoctlGetPtr for struct.
	// So: open with os.OpenFile, then use syscall.Syscall(ioctl, fd, PPS_FETCH, &buf[0]).
	// IoctlGetPtr doesn't exist - we need IoctlGetInt or custom. Let me check unix package.
	// Actually we need to pass a pointer to kernel and kernel fills it. So we use IoctlSetPointer or raw Syscall.
	// unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&buf[0]))). Then parse buf.
	// So we need to use unsafe.Pointer. Let me use syscall.Syscall with unsafe.Pointer(&buf[0]).
	// I'll use a different approach: read from /sys/class/pps/pps0/... - but that doesn't give timestamp.
	// So we need ioctl. In Go: import "unsafe"; syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), uintptr(ppsIoctlFetch), uintptr(unsafe.Pointer(&buf[0]))). Then r, _, errno := ...; if errno != 0 return false; parse buf[ppsAssertSec:ppsAssertSec+8] for sec, buf[ppsAssertNsec:ppsAssertNsec+4] for nsec.
	// Let me add the import and fix.
	if ioctlPPSFetch(int(f.Fd()), buf) != nil {
		return 0, false
	}
	nsec = int32(binary.LittleEndian.Uint32(buf[ppsAssertNsec : ppsAssertNsec+4]))
	return nsec, true
}

// ioctlPPSFetch выполняет PPS_FETCH и заполняет buf (kernel заполняет info).
func ioctlPPSFetch(fd int, buf []byte) error {
	if len(buf) < ppsFdataSize {
		return fmt.Errorf("buffer too small")
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(ppsIoctlFetch), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

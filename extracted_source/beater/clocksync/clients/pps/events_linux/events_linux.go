//go:build linux

package events_linux

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// PPS_FETCH = _IOWR('p', 0xa4, struct pps_fdata) — по дампу phc.PPS_FETCH 0x7e7aa34.
const (
	ppsIoctlFetch = 0xc00470a4
	ppsFdataSize  = 64
	ppsAssertNsec = 16 // assert_tu.nsec в pps_kinfo
)

// FetchPPSNsec по дампу phc.(*KernelPPSSource).TimePPSFetch (0x45898e0): ioctl PPS_FETCH на /dev/pps{index}, возврат nsec.
func FetchPPSNsec(ppsIndex int) (int64, bool) {
	path := filepath.Join("/dev", fmt.Sprintf("pps%d", ppsIndex))
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	buf := make([]byte, ppsFdataSize)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(f.Fd()), uintptr(ppsIoctlFetch), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return 0, false
	}
	nsec := int64(int32(binary.LittleEndian.Uint32(buf[ppsAssertNsec : ppsAssertNsec+4])))
	return nsec, true
}

// RunPPSPollLoop по дампу: цикл опроса PPS, при каждом успешном Fetch — callback(nsec).
func RunPPSPollLoop(ppsIndex int, interval time.Duration, onPPS func(nsec int64)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var lastNsec int64
	for range ticker.C {
		nsec, ok := FetchPPSNsec(ppsIndex)
		if ok && nsec != lastNsec {
			lastNsec = nsec
			onPPS(nsec)
		}
	}
}


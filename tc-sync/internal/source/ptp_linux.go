//go:build linux

package source

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Linux PTP PHC: FD_TO_CLOCKID(fd) = (fd << 3) | PTP_CLK_MAGIC (include/uapi/linux/ptp_clock.h)
const ptpClkMagic = 35 // '#' = 0x23

func init() {
	getTimeFromPHC = readPHCTime
}

// readPHCTime открывает PHC-устройство (например /dev/ptp0), получает clockid,
// читает время через clock_gettime и закрывает fd. ptp4l должен уже синхронизировать PHC.
func readPHCTime(phcDevice string) (time.Time, bool) {
	f, err := os.OpenFile(phcDevice, os.O_RDONLY, 0)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()
	fd := int(f.Fd())
	clockid := int32(fd)<<3 | ptpClkMagic
	var ts unix.Timespec
	if err := unix.ClockGettime(clockid, &ts); err != nil {
		return time.Time{}, false
	}
	return time.Unix(ts.Sec, int64(ts.Nsec)), true
}

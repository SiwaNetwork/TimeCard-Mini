package source

import (
	"fmt"
	"time"

	"github.com/shiwa/timecard-mini/tc-sync/internal/config"
)

// NewFromClockSource создаёт TimeSource из конфига (аналог Timebeat: primary_clocks / secondary_clocks)
func NewFromClockSource(c config.ClockSource) (TimeSource, error) {
	if c.Disable {
		return nil, fmt.Errorf("source disabled")
	}
	proto := c.Protocol
	if proto == "timebeat_opentimecard_mini" {
		proto = "gnss"
	}
	switch proto {
	case "gnss":
		dev := c.Device
		if dev == "" {
			dev = "/dev/ttyS0"
		}
		baud := c.Baud
		if baud == 0 {
			baud = 9600
		}
		return NewGNSS(dev, baud)
	case "nmea":
		dev := c.Device
		if dev == "" {
			dev = "/dev/ttyS0"
		}
		baud := c.Baud
		if baud == 0 {
			baud = 9600
		}
		return NewNMEA(dev, baud, c.Offset)
	case "ntp":
		host := c.IP
		if host == "" {
			return nil, fmt.Errorf("ntp: ip required")
		}
		interval := parseDuration(c.PollInterval, 4*time.Second)
		return NewNTP(host, interval), nil
	case "pps":
		iface := c.Interface
		if iface == "" {
			iface = "eth0"
		}
		return NewPPS(iface, c.Pin, c.LinkedDevice, c.CableDelay, 0, c.Index)
	case "ptp":
		iface := c.Interface
		if iface == "" {
			iface = "eth0"
		}
		phcDevice := c.Device // /dev/ptp0 и т.д.; пусто → NewPTP подставит /dev/ptp0
		return NewPTP(c.Domain, iface, c.UnicastMasterTable, phcDevice), nil
	default:
		return nil, fmt.Errorf("unknown protocol: %s", c.Protocol)
	}
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

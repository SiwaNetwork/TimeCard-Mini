// Timebeat — Beat на базе Elastic Beats v7 (libbeat) для синхронизации времени.
// Использует tc-sync: GNSS (UBX/Timecard Mini), NTP, PTP, PPS, servo (PID/PI).
package main

import (
	"os"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/shiwa/timecard-mini/timebeat/beater"
)

func main() {
	rootCmd := cmd.GenRootCmdWithSettings(beater.New, instance.Settings{
		Name: "timebeat",
	})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

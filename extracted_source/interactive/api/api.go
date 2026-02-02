package api

import (
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/clients/vendors/timebeat/open_timecard_v1"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/hostclocks"
	"github.com/shiwa/timecard-mini/extracted-source/beater/clocksync/sources"
	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// GetAllClockOffsets по дизассемблеру (0x4bec480): вызов hostclocks.GetAllClockOffsets(); возврат среза офсетов всех часов.
func GetAllClockOffsets() []interface{} {
	return hostclocks.GetAllClockOffsets()
}

// GetConfigurationTimeSources по дизассемблеру (0x4bec3e0): sources.GetStore().GetSourcesForCLI(); возврат (keys, values) для CLI.
func GetConfigurationTimeSources() (keys []interface{}, values []interface{}) {
	store := sources.GetStore()
	if store == nil {
		return nil, nil
	}
	return store.GetSourcesForCLI()
}

// GetHTTPTimeSourcesStatus по дизассемблеру (outputTimeSourcesStatus 0x4c26d60): возвращает срез данных о статусе источников времени для HTTP — clock_offsets из hostclocks, time_sources из sources и servo offsets из logging (SetLogEntriesLoop).
func GetHTTPTimeSourcesStatus() []interface{} {
	payload := make(map[string]interface{})
	if offsets := hostclocks.GetAllClockOffsets(); len(offsets) > 0 {
		payload["clock_offsets"] = offsets
	}
	store := sources.GetStore()
	if store != nil {
		keys, values := store.GetSourcesForCLI()
		if len(keys) > 0 || len(values) > 0 {
			payload["time_sources_keys"] = keys
			payload["time_sources_values"] = values
		}
	}
	if entries := logging.GetHTTPTimeSourcesStatus(); len(entries) > 0 {
		payload["servo_time_sources"] = entries
	}
	if len(payload) == 0 {
		return []interface{}{}
	}
	return []interface{}{payload}
}

func GetDispatchStatistics() {
	// TODO: реконструировать
}

func GetPHCDevicePin() {
	// TODO: реконструировать
}

func GetPHCDevices() {
	// TODO: реконструировать
}

func GetPTPClientEos() {
	// TODO: реконструировать
}

func GetPTPClientSubscriptions() {
	// TODO: реконструировать
}

func GetPTPClients() {
	// TODO: реконструировать
}

func GetPTPSockets() {
	// TODO: реконструировать
}

func GetPTPSquaredCapacityStats() {
	// TODO: реконструировать
}

func GetPTPSquaredClientPeers() {
	// TODO: реконструировать
}

func GetPTPSquaredNetworkStats() {
	// TODO: реконструировать
}

func GetPTPSquaredReservationStats() {
	// TODO: реконструировать
}

func GetPTPSquaredSeatStats() {
	// TODO: реконструировать
}

func GetPTPSquaredServerPeers() {
	// TODO: реконструировать
}

func GetServoSources() {
	// TODO: реконструировать
}

func GetTaasClients() {
	// TODO: реконструировать
}

func SetDisableDispatchStatistics() {
	// TODO: реконструировать
}

func SetEnableDispatchStatistics() {
	// TODO: реконструировать
}

func SetMonitorOnly() {
	// TODO: реконструировать
}

func SetOcpTapTimecardIrigBMode() {
	// TODO: реконструировать
}

func SetOcpTapTimecardSMA() {
	// TODO: реконструировать
}

func SetOcpTapTimecardTsWindowAdjust() {
	// TODO: реконструировать
}

func SetOcpTapTimecardUtcTaiOffset() {
	// TODO: реконструировать
}

func SetOpentimecardMiniPtFirmwareFlash() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenDpllFodFreq() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenDpllResetFrequency() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenExecuteDpllBandwidth() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenExecuteDpllPullIn() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenOutputDiv() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenOutputDutyCycleHigh() {
	// TODO: реконструировать
}

func SetOpentimecardSbcClockgenReset() {
	// TODO: реконструировать
}

func SetOpentimecardSbcFirmwareFlash() {
	// TODO: реконструировать
}

func SetPHCDeviceExtts() {
	// TODO: реконструировать
}

func SetPHCDeviceFrequency() {
	// TODO: реконструировать
}

func SetPHCDevicePPS() {
	// TODO: реконструировать
}

func SetPTPClientSimulator() {
	// TODO: реконструировать
}

func SetPTPPeerAnnounceUpdate() {
	// TODO: реконструировать
}

func SetPTPPeerInfoAssociationsAdd() {
	// TODO: реконструировать
}

func SetPTPPeerInfoAssociationsDelete() {
	// TODO: реконструировать
}

func SetPTPPeerInfoData() {
	// TODO: реконструировать
}

func SetPTPPeerInfoJSONAssociationsAdd() {
	// TODO: реконструировать
}

func SetPTPPeerInfoJSONAssociationsDelete() {
	// TODO: реконструировать
}

func SetPTPPeerInfoJSONData() {
	// TODO: реконструировать
}

func SetTaasClientAdd() {
	// TODO: реконструировать
}

func SetTaasClientDelete() {
	// TODO: реконструировать
}

func ShowOcpTapTimecard() {
	// TODO: реконструировать
}

func ShowOcpTapTimecards() {
	// TODO: реконструировать
}

// ShowOpentimecardSbcClockgenConfigStatus по дампу: open_timecard_v1.ShowClockgenConfigStatus().
func ShowOpentimecardSbcClockgenConfigStatus() string {
	return open_timecard_v1.ShowClockgenConfigStatus()
}

// ShowOpentimecardSbcClockgenDpllStatus по дампу: open_timecard_v1.ShowDpllStatus(phaseStr). phaseStr — индекс DPLL, напр. "3".
func ShowOpentimecardSbcClockgenDpllStatus(phaseStr string) string {
	return open_timecard_v1.ShowDpllStatus(phaseStr)
}

// ShowOpentimecardSbcClockgenInputStatus по дампу: open_timecard_v1.ShowInputStatus(inputStr).
func ShowOpentimecardSbcClockgenInputStatus(inputStr string) string {
	return open_timecard_v1.ShowInputStatus(inputStr)
}

// ShowOpentimecardSbcClockgenVersion по дампу (0x4bedcd2): open_timecard_v1.ShowClockgenVersion().
func ShowOpentimecardSbcClockgenVersion() string {
	return open_timecard_v1.ShowClockgenVersion()
}

func ShowOpentimecardSbcEthVlanDefaultTag() {
	// TODO: реконструировать
}

func ShowOpentimecardSbcGnssSats() {
	// TODO: реконструировать
}

func ShowOpentimecardSbcGnssTmode() {
	// TODO: реконструировать
}

func ShowOpentimecardSbcPowerSensor() {
	// TODO: реконструировать
}

func ShowPTPClientSimulator() {
	// TODO: реконструировать
}

func ShowPTPForeignClocks() {
	// TODO: реконструировать
}

func ShowPTPPeerInfoAssociations() {
	// TODO: реконструировать
}

func ShowPTPPeerInfoData() {
	// TODO: реконструировать
}

func ShowPTPPeerInfoJSONAssociations() {
	// TODO: реконструировать
}

func ShowPTPPeerInfoJSONData() {
	// TODO: реконструировать
}

func ShowPTPServerTimeslots() {
	// TODO: реконструировать
}

func inittask() {
	// TODO: реконструировать
}


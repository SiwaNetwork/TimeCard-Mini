# План реконструкции бинарника timebeat до полной реализации

## Цель

Разбирать бинарник `timebeat-extracted/usr/share/timebeat/bin/timebeat` по дизассемблеру и восстанавливать логику в `extracted_source` так, чтобы поведение совпадало с оригиналом.

## Порядок (фазы)

### Фаза 1 — Ядро clocksync (hostclocks, PHC, adjusttime)

1. **hostclocks**
   - [x] lockPHCFunctions / unlockPHCFunctions (условие RPI+eth0+0x108)
   - [x] AddCurrentPHCOffset (полная логика по дизассемблеру)
   - [x] StepClock (GetClockUsingGetTimeSyscall, time.Add, InterferenceMonitor, StepClockUsingSetTimeSyscall)
   - [x] SlewClockPossiblyAsync (реализация: GetClockFrequency, isFrequencyUnchanged, triggerInterference, SlewClock, commitFrequency)
   - [x] GetController
   - [x] InterferenceMonitor: triggerInterference, isFrequencyUnchanged, commitFrequency

2. **adjusttime**
   - [x] IsRPIComputeModule (stub)
   - [x] GetClockFrequency(clockID)
   - [x] SetFrequency(clockID, ppm)
   - [x] SetOffset(clockID, offsetNs)
   - [x] StepClockUsingSetTimeSyscall(clockID, t) по дизассемблеру (syscall 227)
   - [x] PerformGranularityMeasurement (уже был)

3. **phc**
   - [x] (*PHCDevice).GetDeviceName() (string, int)
   - [x] (*PHCDevice).GetClockID() int
   - [x] (*PHCDevice).DeterminePHCOffset() int64 — полная логика по дизассемблеру (switch 0xc0, sort.Slice, median)
   - [x] PHCDevice: NumSamples (0xf0), StrategyType (0xc0), FallbackOffset (0xc8)
   - [x] DeterminePTPOffsetBasic (полная логика: GetPHCToSysClockSamplesBasic, триплеты (t1+t2)/2−t3, slice[i]=offset+round, return slice[0])
   - [x] GetPHCToSysClockSamplesBasic (phc_linux.go: Ioctl(fd, PTP_SYS_OFFSET, &req))
   - [x] DeterminePTPOffsetExtended / EFX / Precise (fallback на Basic; полная реализация — по дизассемблеру при наличии)

4. **servo/filters**
   - [x] NoneGaussianFilter.IsFiltered (по дизассемблеру __NoneGaussianFilter_.IsFiltered.txt: config.0x28, lock 0x60, упрощённая логика)
   - [x] NoneGaussianFilter.IsFilteredCache

### Фаза 1 (доп.) — hostclocks TODO

- [x] StepSlaveClocksIfNecessary: master.name=="system" skip, getClockWithURILocked("system"), порог 500e6, StepClock(clock), DetermineSlaveClockOffsets, цикл по slaves
- [x] logWouldHaveSteppedMessage (0x58 check, triggerWouldHaveSteppedMessageTimer stub)
- [x] HostClock.SetFrequency: adjusttime.SetFrequency (ADJ_FREQUENCY)
- [x] LogForSlaveClocks: Lock, цикл slaveClocks, LogRawAndEMAData(hc)
- [x] LogRawAndEMAData, TriggerWouldHaveSteppedMessageTimer (заглушки по дизассемблеру)

### Фаза 2 — Servo (controller, offsets, algos)

- [x] Controller: getSourcesSnapshot, doWeHaveSourcesForType, addClockNamesToSourcesSnapshot; doPeriodicTasks вызывает getSourcesSnapshot и addClockNamesToSourcesSnapshot
- [x] Offsets: GetSourcesSnapshotForCategory(category); observationChan (0x28), RunProcessObservationsLoop (select → ProcessObservation); RegisterObservation: selectnbsend(sourceID); TimeSource.clockName (0x80/0x88)
- [x] AlgoPID: комментарии по дизассемблеру enforceAdjustmentLimit (0x457fdc0), adjustDComponent (0x457fc80)

### Фаза 3 — Клиенты и источники

- [x] NTP (частично): RunWithConfig подключает NTP по primary/secondary, ConfigureTimeSource("ntp", config), Start()
- [x] PPS: Controller, NewController(offsets), Start(), loadConfig()→Range(ConfigureTimeSource); подключение в RunWithConfig; ConfigureTimeSource по key "pps" или value.Type "pps"
- [x] NMEA: Controller, Start() в RunWithConfig; ConfigureTimeSource по key "nmea" или value.Type "nmea"/"gnss", makeNewClient, client.Start()
- [x] Oscillator: минимальный Controller, NewController(), Start(), loadConfig() (заглушка)
- [x] Sources: TimeSourceStore — GetNewIndex, GenerateTimeSourcesFromConfig, generateClockSourceForType, adjustForProfile, Is/Enable ClockProtocol, Is/Enable PTPPeerDelayMulticast, GetNextAvailablePTPAutoDomain, GetPTPDomainMap

### Фаза 4 — Конфиг, логирование, daemon

- [x] config.appConfig (0xe0): GetAppConfig(), SetAppConfig(cfg); вызов SetAppConfig(cfg) в RunWithConfig
- [x] logging: GetErrorLogger() (глобальный errorLogger), AlgoLogEntry.Log() (algo), HostClockLogEntry.Log() (hostclock) — заглушки
- [x] interactive/daemon (при необходимости): http_server.OutputTimeSourcesStatus, OutputFormattedJSON, ssh_server.ConfigureServerKeys, loadSSHKey, generateNewSSHKey реализованы; Run/CreateCommands — заглушки до дизассемблера.

## Извлечение дизассемблера

Скрипт (запускать в WSL или среде с `objdump` и `nm -D`):

```bash
BIN="timebeat-extracted/usr/share/timebeat/bin/timebeat"
OUT="code_analysis/disassembly"
python3 code_analysis/extract_disassembly_for_functions.py "$BIN" -o "$OUT" \
  -f "adjusttime.IsRPIComputeModule" \
  -f "adjusttime.GetClockFrequency" \
  -f "adjusttime.SetOffset" \
  -f "adjusttime.StepClockUsingSetTimeSyscall" \
  -f "adjusttime.PerformGranularityMeasurement" \
  -f "phc.(*PHCDevice).GetDeviceName" \
  -f "filters.(*NoneGaussianFilter).IsFiltered" \
  -f "InterferenceMonitor.triggerInterference" \
  -f "InterferenceMonitor.isFrequencyUnchanged" \
  -f "InterferenceMonitor.commitFrequency"
```

После появления файлов в `code_analysis/disassembly/` — разбирать их и переносить логику в Go.

## Уже реконструировано

См. таблицу в `RECONSTRUCTION_FROM_BINARY.md`.

## Следующие шаги (после текущего коммита)

1. ~~Реализовать AddCurrentPHCOffset по дизассемблеру~~ — сделано: комментарии 0x4593120, флаг 0xf1/0x48, *hc.0=offset.
2. ~~Дописать комментарий StepClock~~ — сделано: полный комментарий 0x4592980, при успехе Logger.Info, LogSteppedMessage, NewKalmanFilter→hc.0x108; добавлены logSteppedMessage, kalmanFilter, вызов filters.NewKalmanFilter.
3. ~~Запустить скрипт извлечения для adjusttime/phc/InterferenceMonitor~~ — выполнено: извлечено 143 символа; по дизассемблеру добавлены комментарии adjusttime (GetClockFrequency 0x4418840, SetOffset 0x4418e40), InterferenceMonitor (getState 0x4596e80, setState 0x4596d40, runTimer 0x4597360), заглушки RunTimer, logInterferenceStateChange, getStateAsString.
4. **Фаза 3/4 — дизассемблер клиентов и config/logging (янв 2025):**
   - **NMEA:** loadConfig 0x45b4220 (GetStore, store+8→Range(ConfigureTimeSource-fm)); ConfigureTimeSource 0x45b42a0 (key len 4 "nmea", baud в STANDARD_BAUD_RATES, nativeOpen, makeIDHashFromConfig "%s%d", NewClient, Swap, Start); makeIDHashFromConfig 0x45b48a0 (Sprintf("%s%d", string, int)) — реализовано для *TimeSourceConfig как Name+Index.
   - **NTP:** loadConfig 0x45bc7a0 (GetStore, store+8→Range); ConfigureTimeSource 0x45bc820 (key len 3 "ntp", isClient→configureAndStartClient, isServer→configureAndStartServer); добавлена поддержка value *sources.TimeSourceConfig (Type=="ntp") из store.Range.
   - **PPS:** loadConfig 0x45bece0; ConfigureTimeSource 0x45bed60 (key len 3 "pps", phc.GetInstance/IsDeviceRegistered, makeIDHashFromConfig "%s%d", NewClient, Swap, Start) — комментарии по дизассемблеру.
   - **config:** SetAppConfig 0x4403300 (lock, appConfig/startupConfig = arg, rep movsq 0xa2 qwords, defer unlock).
   - **logging:** Logger.Error 0x4407060 — комментарий; GetErrorLogger (errorLogger).
   - **Oscillator:** loadConfig 0x45edfc0 (GetStore, store+8→Range); ConfigureTimeSource 0x45ee040 (key len 10 "oscillator", phc.GetDeviceWithName, makeIDHashFromConfig "%s%d", makeNewClient, Swap, Start); client.NewClient(config) *Client, Client.Start() — заглушки; вызов oscillator.NewController().Start() в RunWithConfig.
   - **ntp/controller:** комментарий, что реализация в client_impl.go (пакет ntp).
5. **Продолжение дизассемблирования (следующая партия):**
   - **NMEA client:** runGNSSRunloop 0x45b3840 (selectgo по 4 каналам; case 0 → NMEAGSVLogEntry.Log; case 1 → decorateObservation, Offsets.RegisterObservation; case 2 → hostclocks.NotifyTAIOffset); decorateObservation 0x45b3ac0 (заполнение полей out из obs, Sprintf, 0x78=4/6).
   - **sources:** GetStore 0x4416fe0 (once.Do(func1), return store 0x7e2c268); GetSources 0x4417160 (return store+8 = &Sources).
   - Для извлечения следующих символов: `.\code_analysis\run_disassembly_wsl.ps1` с бинарником timebeat; в -Patterns добавлены: "sources/store", "runGNSSRunloop", "ProcessEvent", "SubmitOffset", "receiveMessage", "UpdateTimeSource", "addSource", "CreateSource".
   - Дополнительные паттерны (янв 2025): "interactive", "daemon", "logging", "GetSecondarySourcesOffset", "GetClockWithURI", "ubx/conf", "ShouldLog", "uriRegister", "parseSourceForCLI", "GetSourcesForCLI".
6. **Перенос логики из дампов в Go (по имеющимся дизассемблерам):**
   - **PPS client** (0x45bd7e0 ProcessEvent, 0x45bdca0 SubmitOffset): комментарии по дизассемблеру в clients/pps/client/client.go — Timer.Reset(10s), GetController/GetClockWithURI/PPSRegistered, разбор event (nsec, 1e9), getSecondaryOffset, counter 0x308, Logger.Warn, RegisterObservation.
   - **NTP server** (0x45bb7c0 receiveMessage, 0x45bbc60 UpdateTimeSource): комментарии в server.go — binary.Read, mode 3/version 4, SwapPointer(clockQuality).
   - **sources** (0x4417b80 addSource, 0x44159c0 CreateSource): комментарии AddSource (key [20]uint8, value *TimeSourceConfig, Map.Swap), CreateSource (адрес в комментарии).
7. **Продолжение по дампам (янв 2025):**
   - **NTP client:** GetTime (0x45b5400) — реализовано: GetTimeResponse, GetTime(host, port, version, timeout), queryRoundTripWithAddr; client.GetTime/Time(host) делегируют ntp.GetTime/Query. getTime.func1 (0x45b6000) — defer Close.
   - **LinReg:** комментарий regress (0x457d240) — реализовано в algos/linreg.go: 0x620=count, 0x608=refTime, 0x638/0x640=slope/intercept, 0x648=EMA, alpha 0x54de2a8 (LinRegEMAAlpha).
   - **logging:** TimeSourceLogEntry (0x44063c0 Log, 0x44063a0 ValueSet) — реализовано: поля Source, TimeSource, Message, Flags (0x68), ValueSet(byte); Log() — uriRegister/ShouldLog(Source), makemap, mapassign "source"/"timesource"/"message", logMessageStub; LogStdout() заглушка.
   - **sources:** ProtocolName/TypeName — реализовано: protocolNames (9 записей), typeNames (3 записи), ProtocolName(i int) string, TypeName(i int) string по дампу 7997f20/798bd40.
   - **Pi:** комментарий pi_sample (0x457ffa0) — реализовано в algos/pi.go: 0x70/0x78, 0x38/0x48, 0x40/0x50, 0x58 drift, 0x68 interval, step/slew по |offset|.
   - **PPS client:** getSecondaryOffset (0x45be0a0) — реализовано: GetController().GetSecondarySourcesOffset(), GetClockWithURI(client+0x8/0x10), offset−ref в clients/pps/client/client.go.

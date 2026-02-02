# Строгая реконструкция из бинарника

## Текущее состояние

| Источник кода | Описание |
|---------------|----------|
| **Из бинарника** | Извлечены только **имена** пакетов и функций (через `strings` + парсинг путей `github.com/lasselj/timebeat/...`). На их основе созданы **заглушки** вида `func Name() { // TODO: реконструировать }`. |
| **По аналогии** | Вся **логика** (NTP-клиент, формулы servo PID/PI, config YAML, step/slew, adjusttime) написана **вручную** по типичным алгоритмам и документации, **не** из дизассемблера бинарника. Поведение может отличаться от оригинала. |

Итог: текущая реализация **не гарантирует идентичность** оригиналу. Для идентичности нужна **строгая реконструкция из бинарника**.

---

## Что нужно для идентичной реализации

1. **Дизассемблирование**  
   Для каждой целевой функции из бинарника:
   - по символам (`objdump -t`, `nm -D`) получить адрес входа и границы;
   - выгрузить машинный код: `objdump -d --start-address=X --stop-address=Y BINARY`.

2. **Анализ кода**  
   По дизассемблеру восстановить:
   - константы (immediate, ссылки на `.rodata`/`.data`);
   - ветвления и циклы;
   - вызовы других функций;
   - формулы (арифметика, порядок операций).

3. **Реконструкция в Go**  
   Писать Go-код **строго по** этому анализу: те же константы, та же логика, те же вызовы — без подстановки «как обычно делают» из головы.

---

## Инструменты и скрипты

- **Извлечение имён и заглушек**  
  `code_analysis/extract_full_source.py` — создаёт дерево пакетов и заглушки по `strings` (без дизассемблера).

- **Извлечение дизассемблера по функциям**  
  `code_analysis/extract_disassembly_for_functions.py` — по списку имён функций находит адреса в бинарнике и сохраняет дизассемблер в файлы. Дальше реконструкция делается **вручную по этим дампам**.

- **Локальный бинарник**  
  После распаковки deb:  
  `timebeat-extracted/usr/share/timebeat/bin/timebeat`  
  Или: `timebeat-extracted/usr/bin/timebeat`

Запуск скрипта дизассемблера:

**Вариант 1 — WSL (рекомендуется на Windows):** в среде есть `objdump` и `nm`. Бинарник stripped — размеры функций берутся по следующему символу.

```powershell
# Из корня репозитория (полный набор паттернов — займёт несколько минут):
.\code_analysis\run_disassembly_wsl.ps1

# Только отдельные функции (быстро):
wsl -e bash -c "cd /mnt/c/Users/SHIWA/GitHub/TimeCard-Mini && python3 code_analysis/extract_disassembly_for_functions.py timebeat-extracted/usr/share/timebeat/bin/timebeat -o code_analysis/disassembly --functions 'servo.(*Controller).Run' --functions 'servo.(*Controller).GetUTCTimeFromMasterClock' --max-size 4096"
```

**Вариант 2 — Linux/WSL напрямую:**

```bash
cd /path/to/TimeCard-Mini
python3 code_analysis/extract_disassembly_for_functions.py \
  timebeat-extracted/usr/share/timebeat/bin/timebeat \
  --output code_analysis/disassembly \
  --functions "servo.(*Controller)" \
  --functions "clients/ntp"
```

Индекс дизассемблера: `code_analysis/disassembly/index.txt`. Далее по файлам в `code_analysis/disassembly/` вручную восстанавливать Go-код, идентичный оригиналу.

**Рекомендуемые паттерны для пакетного дизассемблирования** (по одному вызову, т.к. полный набор ~2000 символов занимает много времени):

- `servo.(*Controller).Run` — основной цикл и таймеры Controller
- `servo.(*Controller).GetUTCTimeFromMasterClock` — уже реконструирован по дампу
- `servo.(*Controller).GetOffsets` — доступ к Offsets
- `servo.(*Controller).GetConstructedUTCTimeFromMasterClock` — для полной реконструкции GetUTCTimeFromMasterClock
- `hostclocks.(*HostClockController)` — методы контроллера часов
- `phc.(*PHCDevice)` — методы PHC (частично уже есть)
- `clients/ntp` — NTP-клиент
- `servo/algos` — алгоритмы (PID, Pi, LinReg и т.д.)
- `servo/adjusttime` — adjtimex, step, slew
- `servo/filters` — NoneGaussianFilter и др.
- `servo/offsets` — Offsets, TimeSource

---

## Рекомендуемый порядок строгой реконструкции

1. Выбрать критичные функции (servo Calculate, NTP Query, парсинг NMEA, adjtimex/step).
2. Запустить `extract_disassembly_for_functions.py` для этих имён.
3. По дизассемблеру выписать константы, условия, вызовы.
4. Заменить в `extracted_source` текущую «по аналогии» реализацию на код, повторяющий только то, что видно в бинарнике.
5. Помечать такие файлы/функции комментарием: `// Реконструировано по дизассемблеру бинарника timebeat-2.2.20`.

Так можно добиться **строгой реализации из бинарника**, идентичной оригиналу, а не «похожей» по аналогии.

---

## Индекс дизассемблера (code_analysis/disassembly/index.txt)

По индексу восстанавливаются компоненты в порядке: **filters** (IsFiltered, isOutlier, IsFilteredCache) → **algos** (LinReg, MovingMedian, BestFitFiltered) → **hostclocks** → **phc** → **clients/ntp**.

- **servo/filters**: NewFilterConfig, NewKalmanFilter, (*KalmanFilter).AddValue, NewNoneGausianFilter, IsFilteredCache, IsFiltered, isOutlier.
- **servo/algos**: LinReg (CalculateFrequency, UpdatePPS, SetTSType, ResetServo), MovingMedian (Sample, GetMovingMedian, Reset), BestFitFiltered (AddValue, GetLeastSquaresGradientFiltered, TransposeXValues, RemoveExtremes), CoefficientStore, AlgoPID, Pi, CircularBuffer, tuner.
- **hostclocks**: HostClock (StepClock, SlewClock, AddCurrentPHCOffset, GetPHCDeviceName, GetStaticPHCOffset), HostClockController (AddSlaveHostClock, updateRelevantSlaveClocks), InterferenceMonitor, lockPHCFunctions, DetermineSlaveClockOffsets.
- **phc**: PHCDevice (DeterminePHCOffset, GetDeviceName, GetDeviceNames), PHCController (AddPHCDevice, GetDeviceWithName), DeterminePTPOffsetBasic/Extended.
- **clients/ntp**: Controller (configureAndStartClient, configureAndStartServer), client (offset, getMode), NewController.

---

## Уже реконструировано по дизассемблеру

| Компонент | Файл | Источник |
|-----------|------|----------|
| **AlgoPID** | `extracted_source/beater/clocksync/servo/algos/algos.go` | `code_analysis/disassembly/AlgoPID_CalculateNewFrequency.txt`, `AlgoPID_adjustDComponent.txt`, `AlgoPID_enforceAdjustmentLimit.txt`. Константы: `dComponentLookup` из .noptrdata 0x7c18b30 (1.0, 1.0, 1.0), `logScaleD` из .rodata 0x54de3a8 (1/ln(10)). |
| **NTP offset** | `extracted_source/beater/clocksync/clients/ntp/client_impl.go` | `code_analysis/disassembly/ntp_offset.txt`: формула offset = (sub1+sub2)/2, round-trip t1..t4, QueryOffset и queryRoundTrip. |
| **Pi** | `extracted_source/beater/clocksync/servo/algos/pi.go` | `code_analysis/disassembly/Pi_CalculateFrequency.txt`, `pi_servo_pi_sample.txt`: pi_sample по count 0/1/2, константы 54de260, 54de2a0, 54de740, 54de9c8 (±1e9). PIIntegralTarget = 1e9 (0x3b9aca00). |
| **LinReg** | `extracted_source/beater/clocksync/servo/algos/linreg.go` | `code_analysis/disassembly/linreg_regress.txt`: окно 64 (0x40), регрессия gradient=(sum_xy-sum_x*sum_y/n)/(sum_xx-sum_x²/n), EMA с .rodata 0x54de2a8. Константа `LinRegEMAAlpha` = 0.02. |
| **BestFitFiltered** | `extracted_source/beater/clocksync/servo/algos/helpers.go` | `code_analysis/disassembly/BestFitFiltered_GetLeastSquaresGradientFiltered.txt`: МНК gradient, clamp по 54de9b8/54dec00, return gradient*54de850 (ppb). **GetAbsMean**: movsd 0x28, btr 0x3f — возврат \|absMean\|. **TransposeXValues**, **RemoveExtremes**: заглушки (makeslice, цикл по data). **ResetFilter**: 0x50=maxInt64, 0x58/0x70=-1, 0x68=minInt64. **GetMean**: возврат absMean (0x28). **GetClosest**: возврат 0x38 (closestVal). **AddValue**: алиас Add. Поля: absMean (0x28), closestVal (0x38), reset50/58/68/70. |
| **Controller** | `extracted_source/beater/clocksync/servo/controller.go` | `code_analysis/disassembly/Controller_RunPeriodicAdjustSlaveClocks.txt`: NewTicker(1e9)=1s, Sleep(50ms+rand(0..150ms)), select; StepSlaveClocksIfNecessary, SlewSlaveClocks (hostclocks). |
| **Offsets** | `extracted_source/beater/clocksync/servo/offsets.go` | `code_analysis/disassembly/Offsets_RegisterObservation.txt`, `Offsets_ProcessObservation.txt`, `TimeSource_updateTimeSourceUnfilteredOffset.txt`: RegisterObservation = selectnbsend; ProcessObservation = Lock, getTimeSource, updateUnfilteredOffset/Internals, IsFiltered/EMA, selectnbsend/Warn, RMS→ts.70; updateTimeSourceUnfilteredOffset: +0x30=offset, +0x50=1, +0xc0=0. |
| **adjusttime** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `adjusttime_SetFrequency.txt`: freq * rodata 0x54de698, mode 0x4002, syscall.Adjtimex. SetFrequency: Freq = int64(ppm * 65536). **GetClockFrequency**: duffzero timex, Adjtimex(ptr); при clockID≠0 — Syscall(0x131, clockID, &tx, 0); возврат timex.Freq/65536 (ppm). **SetOffset**: mode 0x2100; offsetNs&lt;0 → time_sec=-1, time_nsec=offsetNs+1e9; иначе time_sec=0, time_nsec=offsetNs; при clockID≠0 — clock_adjtime(305). **PerformGranularityMeasurement(clockID)**: GetClockUsingGetTimeSyscall(clockID) дважды, return t2.Sub(t1). **IsRPIComputeModule**: sync.Once(onceDetectBCM54210PE), doSlow(func1); func1: os.Stat/ReadFile `/proc/device-tree/model`, strings.Index "Compute Module 4"/"Compute Module 5" → detectedBCM54210PE=1, computeModuleType=4/5; возврат detectedBCM54210PE. |
| **maintainEMA** | `extracted_source/beater/clocksync/servo/offsets.go` | `__Offsets_.maintainEMA.txt`: 0x38=EMA, 0x50=value, EMA.AddValue; результат в 0x60, флаг 0x68 or 0x8. |
| **maintainRMS** | `extracted_source/beater/clocksync/servo/offsets.go` | `__TimeSource_.maintainRMS.txt`: sum_sq += value² (0x58), count++ (0x60). |
| **modifyObservationIfRequired** | `extracted_source/beater/clocksync/servo/offsets.go` | `__TimeSource_.modifyObservationIfRequired.txt`: если ts+0xf0 != 0, то obs+0x99 = 0. |
| **StepSlaveClocksIfNecessary** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | GetMasterHostClock (до lock); Lock; если master.name=="system" — пропуск блока step; иначе getClockWithURILocked("system"), \|offset\|>500e6 → StepClock(clock), DetermineSlaveClockOffsets; цикл по slaveClocks: name!="system" и \|offset\|>500e6 → StepClock(slave). GetClockWithURI вынесен в getClockWithURILocked для вызова под lock. |
| **SlewSlaveClocks** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `SlewSlaveClocks.txt`: Lock(0x38), цикл по slice 0x20 len 0x28 — SlewClock. |
| **MovingMedian.Sample** | `extracted_source/beater/clocksync/servo/algos/helpers.go` | `__MovingMedian_.Sample.txt`: буфер 0x30/0x38, индекс 0x10, sorted 0x18/0x20, медиана (n/2). |
| **HostClock.StepClock** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `__HostClock_.StepClock.txt`: 0xf0, 0x40=PHCDevice, GetDeviceName, GetClockUsingGetTimeSyscall, time.Add, InterferenceMonitor. |
| **HostClock.SlewClock** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `__HostClock_.SlewClock.txt`: 0x91, 0x40, 0xc0==2, "system", GetPreciseTime, порог 0x989680, SlewClockPossiblyAsync. |
| **GetClockWithURI** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `GetClockWithURI.txt`: цикл по slice 0x8 len 0x10, GetDeviceName, DoesDeviceHaveName(uri). |
| **DetermineSlaveClockOffsets** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `DetermineSlaveClockOffsets.txt`: Lock(0x38), цикл по 0x20/0x28 — AddCurrentPHCOffset(clock). |
| **updateTimeSourceInternals** | `extracted_source/beater/clocksync/servo/offsets.go` | `updateTimeSourceInternals.txt`: GetController, GetUTCTimeFromMasterClock; копирование полей source→ts; RMS 0x58/0x60. |
| **EMA.AddValue** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.AddValue.txt`: окно 20 (0x14), alpha .rodata 54decb8, буфер 0x10 или формула new=old+alpha*(value-old). |
| **EMA.GetValue** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.GetValue.txt`: mov (%rax),%rax — возврат первого поля (value). **GetValue() int64**. |
| **EMA.Reset** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.reset.txt`: movq $0, 0x8(%rax) — обнуление поля count (0x8). **Reset()**. |
| **EMA.getSMA** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.getSMA.txt`: копия buf (0x10), цикл 0..0x14 sum+=buf[i]; sum/20 (0xcccc...cd, sar 4). **getSMA() int64**. |
| **EMA.EnableDebug** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.EnableDebug.txt`: NewLogger(name); 0xb8=logger; 0xb0=1. **EnableDebug(loggerName string)**. EMA: debugEnabled (0xb0), logger (0xb8). |
| **EMA.logDebug** | `extracted_source/beater/clocksync/servo/statistics/statistics.go` | `__EMA_.logDebug.txt`: Logger.Debug(count), Debug(value), (value-diff)*alpha, Sprintf float64, alpha. **logDebug()**. |
| **AddCurrentPHCOffset** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `AddCurrentPHCOffset.txt`: controller 0x100, 0x40=PHCDevice, "system" branch, lockPHCFunctions, DeterminePHCOffset, 0xf1 IsFiltered, *clock.0=offset. |
| **InterferenceMonitor** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | getState 0x4596e80: Lock(0x28), return 0x10(rcx). setState 0x4596d40: mov bl→0x10(rcx). runTimer 0x4597360: NewTimer(im.0x8), chanrecv1, movb $1,(im+0), setState(0), getState(), StateNames[state], logInterferenceStateChange. Добавлены RunTimer(), logInterferenceStateChange(), getStateAsString() (заглушки). New(enable,name): 0x10=0/1. |
| **GetClockUsingGetTimeSyscall** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `GetClockUsingGetTimeSyscall.txt`: syscall 0xe4 (228 = clock_gettime), timespec 0x38/0x40, time.Unix(sec, nsec). |
| **GetPreciseTime** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `GetPreciseTime.txt`: time.Now(), маска 0x3fffffff, возврат (low30, high, 0). |
| **lockPHCFunctions** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `lockPHCFunctions.txt`: аргумент *HostClock. IsRPIComputeModule(); 0x40=PHCDevice; GetDeviceName=="eth0" (4 bytes, 0x30687465); 0x108(phc)==0; lock 0x110(hc). |
| **unlockPHCFunctions** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `unlockPHCFunctions.txt`: те же проверки RPI+eth0+0x108; lock xadd -1, 0x110(hc); при 0 — unlockSlow. |
| **InterferenceMonitor.setState** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `setState.txt`: аргумент byte (bl). Lock(0x28), defer unlock, 0x10(im)=state; вызовы SetManualOverride — setState(0) или setState(3). |
| **SlewClockPossiblyAsync** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `SlewClockPossiblyAsync.txt`: полный путь: 0xf0→AddCurrentPHCOffset; getState→branch state 3/config; lockPHCFunctions, GetClockFrequency, GetClockUsingGetTimeSyscall, isFrequencyUnchanged/triggerInterference, unlock; 0x91→AddCurrentPHCOffset; 0x48→return; time.Add; switch type 1/2 (system/PHC); lock, GetClockFrequency, unlock, commitFrequency. |
| **PHCDevice.DeterminePHCOffset** | `extracted_source/beater/clocksync/phc/phc.go` | `__PHCDevice_.DeterminePHCOffset.txt`: 0x64==0→return; makeslice(0xf0); switch 0xc0: 1→Basic single, 2→Basic+0xc8, 3→Extended, 4→EFX, 5→Precise; sort.Slice; median = slice[len/2+1]. |
| **AddCurrentPHCOffset (реализация)** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По __HostClock_.AddCurrentPHCOffset 0x4593120: controller.master.0x100=masterOffset; name=="system" (len 6)→lock(master), DeterminePHCOffset, unlock, offset=-phcOffset-masterOffset; иначе lock(hc), DeterminePHCOffset, unlock, offset=phcOffset+hc.0x100; 0xf1: filter=hc.0x18, IsFiltered(offset)→при true flag=1; hc.0x48=flag, hc.offset=offset. Флаг: 0 при не отфильтрованном, 1 при IsFiltered. |
| **StepClockUsingSetTimeSyscall** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `adjusttime_StepClockUsingSetTimeSyscall.txt`: syscall 0xe3 (227 = clock_settime)(clockID, &ts); конвертация time в timespec (0x3b9aca00, 0x3fffffff). |
| **StepRTCClock** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `StepRTCClock@@Base 0x4419260`: openat(AT_FDCWD, /dev/rtc0, O_RDWR\|O_CLOEXEC); GetPreciseTime(); rtc_time (sec, min, hour, mday, mon-1, year-1900); ioctl RTC_SET_TIME; close. |
| **GetSystemClockMaxFrequency** | `extracted_source/beater/clocksync/servo/adjusttime/adjusttime.go` | `GetSystemClockMaxFrequency@@Base 0x4418a40`: duffzero timex; Adjtimex(ptr); при err — return 0x7a120 (500000 ppm); иначе return timex.Freq/65536. |
| **EnablePPSIfRequired (PHC)** | `extracted_source/beater/clocksync/phc/phc.go` | Полная логика по `__PHCController_.EnablePPSIfRequired 0x4584f00`: appConfig+0x238 (slice строк "ifName:idx:channel"); Split(":"); len==3 → GetDeviceWithName, EnablePPSOutOnChannel/EnablePPSOut. Упрощённо: обход c.devices, EnablePPS(). |
| **StepClock (реализация)** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По __HostClock_.StepClock 0x4592980: 0xf0→return; GetDeviceName, Logger.Warn; GetClockUsingGetTimeSyscall, time.Add(offset); getState→logWouldHaveSteppedMessage; порог 500 ms→logWouldHaveSteppedMessage; lockPHCFunctions, SetFrequency(0x70), GetClockFrequency, commitFrequency, StepClockUsingSetTimeSyscall, unlock; при err→Logger.Error; при успехе: Logger.Info, LogSteppedMessage (0x4593e00), NewKalmanFilter(GetDeviceName)→hc.0x108. Добавлены logSteppedMessage(), kalmanFilter (0x108), вызов filters.NewKalmanFilter. |
| **logWouldHaveSteppedMessage** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По дизассемблеру LogWouldHaveSteppedMessage@@Base: если 0x58(hc)!=0 return; иначе triggerWouldHaveSteppedMessageTimer(); поле suppressWouldHaveSteppedLog (0x58). |
| **TriggerWouldHaveSteppedMessageTimer** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | Заглушка по дизассемблеру (closure в LogWouldHaveSteppedMessage). |
| **LogForSlaveClocks** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По дизассемблеру: Lock(0x38), defer unlock; цикл по 0x20/0x28 (slaveClocks); для каждого hc — LogRawAndEMAData(hc). |
| **LogRawAndEMAData** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | Заглушка по дизассемблеру (0x48 читается); логирование raw/EMA до подключения логгера. |
| **HostClock.SetFrequency** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По дизассемблеру adjtimex ADJ_FREQUENCY: hc.frequency/frequencyPpm = ppm; при clockID≠0 вызов adjusttime.SetFrequency(clockID, ppm). |
| **SlewClockPossiblyAsync (реализация)** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По комментарию: enabled→AddCurrentPHCOffset; getState→branch; lockPHCFunctions, GetClockFrequency, GetClockUsingGetTimeSyscall, isFrequencyUnchanged/triggerInterference, unlock; isMaster vs addCurrentPHCOffsetLocked; flag48→return; SlewClock(offsetNs); lock, GetClockFrequency, commitFrequency, unlock. |
| **InterferenceMonitor** (triggerInterference, isFrequencyUnchanged, commitFrequency) | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | triggerInterference→stateByte=1; isFrequencyUnchanged→сравнение с savedFrequencyPpm (0x38/0x40); commitFrequency→запись freq в savedFrequencyPpm; добавлены поля savedFrequencyPpm, hasSavedFrequency. |
| **phc.(*PHCDevice)** | `extracted_source/beater/clocksync/phc/phc.go` | **GetDeviceName** / **GetDeviceNameLocked** (см. выше). **DeterminePHCOffset**: по __PHCDevice_.DeterminePHCOffset.txt — 0x64==0→return 0; makeslice(*0xf0); switch 0xc0: 1→Basic return; 2→Basic, если ret>=5e6 return 0 иначе 0xc8; 3→Extended return; 4/5→EFX/Precise, sort.Slice(descending), median=slice[len/2]. **PHCDevice**: FD (0x60), NumSamples (0xf0), StrategyType (0xc0), FallbackOffset (0xc8). **GetPHCToSysClockSamplesBasic**: phc_linux.go (SYS_IOCTL, PTP_SYS_OFFSET=0x40103d01). **DeterminePTPOffsetBasic**: полная реализация. **DeterminePTPOffsetExtended/EFX/Precise**: fallback на Basic (полная реализация — по дизассемблеру при наличии). **SetPPS(scale byte)** по дизассемблеру (__PHCDevice_.SetPPS@@Base): Ioctl(phc.FD, PTP_ENABLE_PPS, &scale). **EnablePPS()**: вызов SetPPS(1). **SetPinFunction(pin, funcIndex, channel int32)**: ptpPinFuncStruct по 0x68/0x6c/0x70; Ioctl(phc.FD, PTP_PIN_SETFUNC, &struct). **Ioctl**, **PTP_ENABLE_PPS**, **PTP_PIN_SETFUNC** — в phc.go (var); реализация Ioctl в phc_linux (ioctlLinux). |
| **filters.NoneGaussianFilter** | `extracted_source/beater/clocksync/servo/filters/filters.go` | **IsFiltered(offset int64) bool** по __NoneGaussianFilter_.IsFiltered.txt: config.0x28==0→false; lock 0x60; (offset в [-10,10] и counter<5→true, иначе counter=0, false). **IsFilteredCache** по __NoneGaussianFilter_.IsFilteredCache.txt: config.0x28; count>=window→false; lock; band [medianVal-diff, medianIdx+diff], diff=(medianIdx-medianVal)*config.mult; in band→false; (offset+10)<=20→false; counter>=5→false; иначе true. **isOutlier(offset)** по __NoneGaussianFilter_.isOutlier.txt: band [predicted-diff, predicted+diff]; вне band и \|offset\|>10→true. Реализует hostclocks.OffsetFilterInterface. |
| **HostClock.GetTimeNow** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `__HostClock_.GetTimeNow.txt`: lockPHCFunctions; 0x40(hc)=phcDevice, 0x64(phc)=clockID; GetClockUsingGetTimeSyscall(clockID); unlockPHCFunctions; return time. |
| **HostClock.addOffset** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | `__HostClock_.addOffset.txt`: 0x48(rax)=cl (flag); mov (%rax),%rax → *receiver.0; mov %rbx,(%rax) — запись value. Реконструкция: flag48=flag, offset=value. |
| **HostClock методы** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | **GetClockID**: возврат phc.0x64. **GetClockName**: GetDeviceName(phc.0x40). **GetRawData**: возврат *hc.0 (заглушка). **IsMaster**: возврат 0x91. **SetMaster**: запись в 0x91. **LogRawAndEMAData**: lockPHCFunctions; PerformGranularityMeasurement; unlockPHCFunctions; if !isMaster: clientStore=nil; maintainEMA(hc,clientStore); GetDeviceName(phc) для логирования. **maintainExponentiallyWeightedMovingAveragesClientStore**: заглушка (EMA.AddValue). **SetHoldoverFrequency** (0x45943c0): lockPHCFunctions; defer unlock; Logger.Warn; hc.bestFitFiltered.GetLeastSquaresGradientFiltered()→ppm; SetFrequency(clockID, ppm); GetClockFrequency; algorithm.UpdateClockFreq(freq); commitFrequency(im). HostClock: bestFitFiltered *algos.BestFitFiltered (0x30), algorithm interface{}. **StepFromMasterClock** (0x4592880): phc==nil или name=="system"→return false; lockPHCFunctions; DeterminePHCOffset; unlock; если (phcOffset+500e6)>1e9: flag48=0, offset=phcOffset, StepClock; return true; иначе return false. |
| **HostClockController** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | **AddSlaveHostClock**: Lock; append hc to slaveClocks; updateRelevantSlaveClocks; Unlock; if masterClock!=nil: AddCurrentPHCOffset(hc), DetermineSlaveClockOffsets. **updateRelevantSlaveClocks**: по дизассемблеру AddSlaveHostClock.txt (0x45952e0): цикл по slaveClocks; для каждого slave (!isMaster) включаем в relevantSlaveClocks если name=="system" или controller.includeAllRelevantSlaves!=0; запись в 0x20/0x28/0x30 (relevantSlaveClocks). **CreateSystemClock**: по дизассемблеру createSystemClock@@Base (0x45988e0): объект с 0x8=-1, 0x64=0 и [1]string "system"; NewHostClock(config); 0x60(clock)=0; AddSlaveHostClock(clock). **NewHostClock(phc, enabled)**: ветки по GetDeviceName — **algoTypeByDeviceName** (system→pi, eth0/rho→pid, alpha/beta→pi, gamma/sigma→linreg); HostClock.algoType. **systemClockDevice**: фиктивное PHC-устройство GetDeviceName()=("system", 6), GetClockID()=-1, DeterminePHCOffset()=0. **maintainExponentiallyWeightedMovingAveragesClientStore**: по дизассемблеру — clientStore в 0x8.0x8; два вызова EMA.AddValue (value из 0x0 и 0x8.0x10). **GetPHCDeviceName**, **GetStaticPHCOffset** — для servo.GetUTCTimeFromMasterClock. **updateAlgoCoefficients** (0x4595020): без параметров; master=controller.masterClock; если name(master)=="system" — для каждого slave UpdateScaleFromStore(0); иначе master.UpdateScaleFromStore(1) и для каждого slave UpdateScaleFromStore(1). **algoTypeToInt**, **updateStoreForClock**, **UpdateScaleFromStore** на HostClock. **algoScaleUpdater**, **algoClockFreqUpdater**, **algoPPSUpdater** интерфейсы. **ElectMasterClock** (0x4594c00): Lock(0x38); current=masterClock; если current==nil → promoteToMaster(newMaster); иначе если current!=newMaster → demoteCurrentMaster(), promoteToMaster(newMaster); updateRelevantSlaveClocks(); updateAlgoCoefficients(); Unlock(); UpdatePPS(ppsScale). **demoteCurrentMaster**, **promoteToMaster**: Logger.Info; для system — setStateByte(0); иначе SetMaster(false); promote: isMaster=true, masterClock=newMaster. **UpdatePPS(scale)** на Controller: 0x68=scale; цикл по slaveClocks: isMaster → clock.UpdatePPS(scale), иначе UpdatePPS(0). **HostClock.UpdatePPS(scale)**: вызов algorithm.UpdatePPS(scale) (algoPPSUpdater). **ppsScale** (0x68) в HostClockController. |
| **Controller.GetUTCTimeFromMasterClock** | `extracted_source/beater/clocksync/servo/controller.go` | По дизассемблеру GetUTCTimeFromMasterClock@@Base (0x45a1ac0): IsRPIComputeModule→branch; при !IsRPI или (RPI и master PHC name=="eth0") — master.GetTimeNow().Add(-master.staticPHCOffset); иначе GetConstructedUTCTimeFromMasterClock. |
| **nmea** | `extracted_source/beater/clocksync/clients/nmea/` | **NewController**: sync.Once, func1 создаёт Logger, sync.Map, controller. **Start**: loadConfig. **loadConfig**: sources.GetStore().Sources.Range(ConfigureTimeSource). **ConfigureTimeSource**: key=="nmea" (len 4, 0x61656d6e) → makeNewClient, clients.Store. **makeNewClient**: client.NewClient(config). **Client**: device, logger; **Start**: device.Start(), go runGNSSRunloop. **sources.GetStore**: *TimeSourceStore; **GetSources**, **AddSource** по дизассемблеру. **logging.Logger**: NewLogger(name), Error(msg). |
| **clients/ntp** | `extracted_source/beater/clocksync/clients/ntp/client_impl.go`, `client/client.go` | **NewController**: sync.Once(func1); func1 — NewLogger("ntp-controller"), new(Controller), 0=offsets 0x8(rdx), 8=logger, package var. **Start**: loadConfig. **loadConfig**: GetStore().GetSources().Range(ConfigureTimeSource). **ConfigureTimeSource(key, value)**: key=="ntp" (len 3); NTPTimeSourceConfig IsClient→configureAndStartClient, IsServer→configureAndStartServer. **configureAndStartClient**: ParseTimeString(pollInterval), go runPoller. **client.offset**: (sub1+sub2)>>1, округление (sum+sum>>63)>>1. **msg.GetMode**: первый байт & 7. |
| **ntp/server.runMessageReceiver** | `extracted_source/beater/clocksync/clients/ntp/server/server.go` | По дизассемблеру (__NTPServer_.runMessageReceiver 0x45bbfc0): time.Sleep(15s); в бинарнике RunSocket (PTPSocket), select: case 0 → receiveMessage(s, buf). У нас: runMessageReceiver(pc) — Sleep(15s), затем цикл ReadFrom/receiveMessage/WriteTo. |
| **ntp/server.UpdateTimeSource** | `extracted_source/beater/clocksync/clients/ntp/server/server.go` | По дизассемблеру (__NTPServer_.UpdateTimeSource 0x45bbc60): newobject; копирование полей из s.0x20 в объект; switch quality byte: 0x10→"PPS" QualityType=1, 0x20→"GPS" QualityType=1, 0x40/0x50 и sourceName≥3 байт→имя QualityType=2, иначе 0x10; atomic.SwapPointer(s+0x20). Добавлены ClockQuality, GetClockQuality. |
| **ntp client queryRoundTrip/getTime** | `extracted_source/beater/clocksync/clients/ntp/client_impl.go` | По дизассемблеру getTime (0x45b5400): GetPreciseTime для t1/t4; SetDeadline(t1+5s); запрос mode=3, version=4 (0x23), originate=TimeToNtpTime(t1) в байты 40-47; Write, Read; валидация resp[0]&7==4, originate в ответе (24-31)==запрос (40-47), RTT≥0; t2=Receive(32-39), t3=Transmit(40-47). Ошибки errNTPMode, errNTPOriginate, errNTPNegativeRTT. |
| **ntp client.ToNtpTime** | `extracted_source/beater/clocksync/clients/ntp/client/client.go` | По дизассемблеру (client.toNtpTime 0x45b4f00): делегирует common.TimeToNtpTime(t). |
| **phc.PHCDevice.DoesDeviceHaveName** | `extracted_source/beater/clocksync/phc/phc.go` | **DoesDeviceHaveName(name)**: lock 0x100; цикл по DeviceNames (0x30/0x38); сравнение с name; при совпадении return true. **PHCController.GetDeviceWithName(name)**: цикл по devices; device.DoesDeviceHaveName(name); при совпадении return device. |
| **Offsets/TimeSource** | `extracted_source/beater/clocksync/servo/offsets.go` | **ProcessObservation**: Lock, get ts, ts.active=true, updateTimeSourceInternals(ts), updateTimeSourceFilteredOffset/For, maintainEMA(ts). **updateTimeSourceInternals**: GetController().GetUTCTimeFromMasterClock(); ts.active=true; ts.rmsAccumulator += offset²; ts.rmsCount++. **maintainEMA(ts)**: Offsets.ema.AddValue(ts.filteredOffset); flag68\|=8. **Offsets**: добавлены ema *statistics.EMA, flag68. **sources.TimeSourceStore**: GetSources() возвращает &s.Sources; **AddSource(key, value)** по дизассемблеру __TimeSourceStore_.addSource: Sources.**Swap**(key, value) (не Store). |
| **sources.addSource** | `extracted_source/beater/clocksync/sources/sources.go` | По дизассемблеру 0x4417b80: store+8 = sync.Map; convT(key), convT(value); **sync.(*Map).Swap(key, value)**. AddSource реализован как s.Sources.Swap(key, value). |
| **GetClockWithURI / getClockWithURILocked** | `extracted_source/beater/clocksync/hostclocks/hostclocks.go` | По дизассемблеру GetClockWithURI: цикл по slice 0x8/0x10; для каждого clock 0x40=PHCDevice; GetDeviceName(phc); memequal(name, uri) → return clock; **DoesDeviceHaveName(phc, uri)** → return clock. getClockWithURILocked обновлён: сравнение по GetDeviceName и DoesDeviceHaveName(uri). |
| **filters.NewFilterConfig** | `extracted_source/beater/clocksync/servo/filters/filters.go` | По дизассемблеру NewFilterConfig@@Base: (filterType, enabled); switch 1/2/4/5; newobject; +0 type, +8 divisor, +0x10 mult, +0x18 idx, +0x20 extra, +0x28 enabled. Константы FilterTypeNoneGaussian1/2/4/5; тип 1: divisor=100, mult=0.02, idx=89, extra=10; тип 2: divisor=20, idx=17, extra=2; тип 4/5: divisor=100/200, mult=0.05. |
| **filters.NewNoneGausianFilter** | `extracted_source/beater/clocksync/servo/filters/filters.go` | По дизассемблеру NewNoneGausianFilter@@Base: NewFilterConfig(filterType); makeslice(int64, config.divisor) x2; concatstring2(prefix, \"-filter\"); NewLogger; newobject; 0x8/0x10/0x18 slice1, 0x20/0x28/0x30 slice2, 0x38=logger, 0x40=config. Добавлены slice2, logger в NoneGaussianFilter; noneGaussianConfig расширен (filterType, divisor, mult, idx, extra, enabled). |
| **filters.KalmanFilter** | `extracted_source/beater/clocksync/servo/filters/filters.go` | **KalmanFilter**: 0x10 state (Update), 0x20 storeFunc. **AddValue(value int64)**: cvtsi2sd→float64; вызов state.Update(measurement). **NewKalmanFilter(config, _)**: заглушка (бинарник 2464 байт — матрицы 2x2, буферы). |
| **algos.CoefficientStore** | `extracted_source/beater/clocksync/servo/algos/coefficients.go` | **SteeringCoefficients**: Kp, Ki, Kd float64; Field18=8, Field20=0x1efe920. **CoefficientStore**: +0 logger, +8 coeffType1, +0x10 coeffType0; map для совместимости. **newCoefficientStore**: NewLogger(\"coefficient-store\"), getCoefficientsInt(1,...)→coeffType1, Logger.Info, getCoefficientsInt(0,...)→coeffType0. **getCoefficientsInt(algoType, kp, ki, kd, scale)**: при scale==0 Critical; switch 0/1/2/3, new(SteeringCoefficients), перезапись и Info при kp/ki/kd/scale≠1. **GetCoefficientsForTypeInt(algoType, scale)**: type 0→coeffType0, 1→coeffType1, 2/3→getCoefficientsInt + масштаб math.Pow(10, scale). **GetCoefficientStore**: sync.Once, func1=newCoefficientStore, return instanceCoefficientStore. |
| **sources.GetSourcesForCLI** | `extracted_source/beater/clocksync/sources/sources.go` | По дизассемблеру GetSourcesForCLI@@Base: Lock(0x98); обнуление слайсов 0xa8/0xb0; Range(Sources, parseSourceForCLI-fm); Unlock. GetSourcesForCLI вызывает в Range **parseSourceForCLI(key, value)** для каждой пары. **parseSourceForCLI** по дизассемблеру (0x44124e0): коллбэк для Range; проверка типа value (K0rE3Tf5), convT64, fmt.Sprintf, growslice; append в слайсы 0xa0 и 0xa8/0xb0. Минимальная реконструкция: append key/value в cliKeys/cliValues. |
| **sources.CreateSource** | `extracted_source/beater/clocksync/sources/sources.go` | По дизассемблеру CreateSource@@Base (0x44159c0): NewLogger("sources"), strings.Join, strconv.FormatInt(index), конкатенация полей, **crypto/sha1.Sum**, сравнение с ProtocolName (switch по sourceType). **TimeSourceConfig** (Type, Name, Index, ID=hex(SHA1)); возврат cfg для nmea/ntp/ptp/gnss. |
| **algos.ChangeSteeringCoefficients** | `extracted_source/beater/clocksync/servo/algos/coefficients.go` | По дизассемблеру ChangeSteeringCoefficients@@Base (0x457cae0): (receiver, algoType int, newCoeffs *SteeringCoefficients). algoType==0: Logger.Info(0x35), target=store+0x10 (coeffType0); algoType==1: Logger.Info(0x33), target=store+0x8 (coeffType1); иначе return. Копирование в target: если Kp/Ki/Kd/Field18/Field20 != 0 — записать. **ChangeSteeringCoefficientsInt(algoType int, c *SteeringCoefficients)** реализован. |
| **algos.GetCoefficients (rodata)** | `extracted_source/beater/clocksync/servo/algos/coefficients.go` | Раскладка по типам: **getCoefficientsDefaults[4]** — type 0: 54de498 Kp, 54de218 Ki, 54de660 Kd; type 1 — те же; type 2: 54de410 Kp, 54de368 Ki, 54de498 Kd; type 3: 54de2e0 Kp, 54de260 Ki, 54de498 Kd. Field18=8, Field20=0x1efe920. getCoefficientsInt использует массив для заполнения SteeringCoefficients. |
| **logging.Logger** | `extracted_source/beater/logging/logging.go` | **Critical(msg, _ int)** и **Info(msg, _ int, _ ...interface{})** — заглушки для вызовов из CoefficientStore. **Debug(msg, _ int, _ ...interface{})** — для statistics.(*EMA).logDebug. |
| **algos.CircularBuffer** | `extracted_source/beater/clocksync/servo/algos/helpers.go` | **Min/Max** по дизассемблеру (__CircularBuffer_.Min/Max): в бинарнике 0x18/0x20=data, 0x30/0x38 и 0x48/0x50 — отсортированные индексы; возврат data[sorted[0]]. Здесь — линейный поиск min/max по буферу. LinReg/MovingMedian/BestFitFiltered/tuner — уже есть в linreg.go, helpers.go, tuner/tuner.go; доп. методы по дампу — при необходимости. |
| **LinReg.UpdateClockFreq, UpdateScaleFromStore** | `extracted_source/beater/clocksync/servo/algos/linreg.go` | **UpdateClockFreq(freq)**: по дизассемблеру __LinReg_.UpdateClockFreq (запись по 0x6e0); заглушка. **UpdateScaleFromStore(scale)**: GetCoefficientStore().GetCoefficientsForTypeInt(1, scale); ChangeSteeringCoefficientsInt(1, c). |
| **Pi.UpdateClockFreq, UpdateScaleFromStore** | `extracted_source/beater/clocksync/servo/algos/pi.go` | **UpdateClockFreq(freq)**: заглушка. **UpdateScaleFromStore(scale)**: GetCoefficientsForTypeInt(2, scale); SetKp(c.Kp). |
| **AlgoPID.UpdateClockFreq, UpdateScaleFromStore** | `extracted_source/beater/clocksync/servo/algos/algos.go` | **UpdateClockFreq(freq)**: заглушка. **UpdateScaleFromStore(scale)**: GetCoefficientsForTypeInt(0, scale); ChangeSteeringCoefficientsInt(0, c). |

### Удалены фиктивные подпакеты

Подпакеты `clockdata`, `controller`, `cli`, `setup`, `clock`, `hostcompliance`, `interference_monitor`, `phc_and_tai_offsets` в оригинальном бинарнике `timebeat` **не существуют** как отдельные пакеты. Весь код находится в основном пакете `hostclocks`. Эти директории и соответствующие файлы-заглушки были удалены.

Все функции hostclocks в бинарнике имеют путь `github.com/lasselj/timebeat/beater/clocksync/hostclocks.*`, без подпакетов.

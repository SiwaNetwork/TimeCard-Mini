# Разрыв между анализом и реализацией (tc-sync / timebeat)

Документ сопоставляет **что разобрано в code_analysis** и **что реализовано в tc-sync/timebeat**. Используйте его для планирования доработок.

---

## 1. UBX протокол

| Анализ (code_analysis) | Реализация (tc-sync) | Статус |
|------------------------|----------------------|--------|
| **UBXTP5Message** — 62 смещения (algorithm_details.txt), program_structure.go | **internal/ubx/tp5.go** — только 32 байта payload (FreqPeriod, PulseLenRatio, UserConfigDelay, Flags) | ⚠️ Частично |
| ToBytes(), SetPulseWidth(), GetPulseWidth() | Marshal(), TP5Config, BuildCFGTP5 | ✅ Есть |
| send1PPSOnTimepulsePin, detectUbloxUnit | Нет | ❌ Нет |
| Остальные 56 полей структуры (offset 40–2796) | Не реализованы | ❌ Нет |

**Где смотреть:** `program_structure.go`, `algorithm_details.txt`, `IMPLEMENTATION_GUIDE.md`.

**Что добавить (по приоритету):**
- При необходимости — расширить TP5 до полной структуры по 62 смещениям (если нужны доп. поля u-blox).
- send1PPSOnTimepulsePin / detectUbloxUnit — при необходимости интеграции с 1-PPS или автоопределением устройства.

---

## 2. Servo алгоритмы

| Анализ (code_analysis) | Реализация (tc-sync) | Статус |
|------------------------|----------------------|--------|
| **AlgoPID** (0x41c8680), PID с Kp/Ki/Kd, D-массив, BestFitFiltered, enforceAdjustmentLimit | **internal/servo/servo.go** — PID с Kp/Ki/Kd, integral/derivative, MaxIntegral/MaxAdjustment | ⚠️ Упрощённо |
| **Pi** (0x41c8310), двухэтапная инициализация, константа 51712 | **internal/servo/servo.go** — PI с Kp/Ki, integral clamping | ⚠️ Упрощённо |
| **LinReg** (0x41c6c00), окно 64, регрессия по интервалам 2–6, gonum | **internal/servo/linreg.go** — LinReg с окном 64, slope = ns/s, freq = slope/1e9; алгоритм `linreg` в конфиге | ✅ Реализовано |
| **GetClockUsingGetTimeSyscall** | clockadj не вызывается явно; время через time.Now() в цикле | ✅ По сути есть (time + adjtimex) |
| **StepClockUsingSetTimeSyscall** | **internal/clockadj** — Step() через clock_settime | ✅ Есть |
| **SetFrequency** / **SetOffset** | **internal/clockadj** — SetFrequency(ppm), Slew(offsetNs) через adjtimex | ✅ Есть |
| **PerformGranularityMeasurement** | **internal/clockadj** — GranularityNs() (простое измерение разрешения clock_gettime) | ✅ Реализовано |
| **GetClockFrequency** | **internal/clockadj** — GetFrequency() (чтение freq из adjtimex) | ✅ Реализовано |
| **CoefficientStore**, DefaultAlgoCoefficients, D-массив (0x770a430) | Коэффициенты задаются в конфиге (kp, ki, kd) | ⚠️ Без извлечённых констант |

**Где смотреть:** `program_structure.go`, `FOUND_COEFFICIENTS.md`, `servo_functions_detailed_analysis.txt`, `MASTER_ANALYSIS_REPORT.md`.

**Что добавить (по приоритету):**
1. ~~**LinReg**~~ — реализовано в `internal/servo/linreg.go`, алгоритм `linreg` в конфиге.
2. ~~**PerformGranularityMeasurement**~~ — реализовано как `clockadj.GranularityNs()`.
3. ~~**GetClockFrequency**~~ — реализовано как `clockadj.GetFrequency()`.
4. Подстройка PID/PI под извлечённые коэффициенты и константы (51712, D-массив) из анализа.

---

## 3. HostClock / Controller (выбор источника и цикл)

| Анализ (code_analysis) | Реализация (tc-sync) | Статус |
|------------------------|----------------------|--------|
| **HostClock** (GetTimeNow, StepClock, SlewClockPossiblyAsync) | Нет отдельной сущности; время и коррекция в pkg/clocksync + clockadj | ✅ Логика есть |
| **ServoController** (RunPeriodicAdjustSlaveClocks, ChangeMasterClock, HoldMasterClockElection) | **internal/clockselect** — Election, Select(), GetTimeFromActive(); цикл в **pkg/clocksync** | ✅ Есть |
| Выбор master среди нескольких источников | primary → secondary, первый доступный | ✅ Есть |

Отдельный тип HostClock в tc-sync не вводился — роль «master» выполняет выбранный TimeSource, коррекция — через clockadj. Для полного соответствия программе можно ввести HostClock и вызывать из него clockadj.

---

## 4. Источники времени (clients/vendors)

| Анализ (code_analysis) | Реализация (tc-sync) | Статус |
|------------------------|----------------------|--------|
| **UBX / generic_gnss_device** | **internal/source/gnss.go** — чтение UBX-NAV-PVT с порта, парсинг UTC (year/month/day/hour/min/sec/nano), GetTime() возвращает время приёмника при validTime | ✅ Реализовано (**internal/ubx/navpvt.go**) |
| **NMEA** | **internal/source/nmea.go** — чтение NMEA RMC (GPRMC/GNRMC) с serial, парсинг UTC время/дата, опционально offset (нс) | ✅ Реализовано |
| **NTP** | **internal/source/ntp.go** — простой NTP-клиент (time по IP) | ✅ Есть |
| **PTP** | **internal/source/ptp.go** + **ptp_linux.go** — чтение времени из PHC через clock_gettime (FD_TO_CLOCKID); ptp4l синхронизирует PHC отдельно | ✅ Реализовано (ptp4l+PHC) |
| **PPS** | **internal/source/pps.go** — время секунды с linked_device (GNSS), начало секунды + cable_delay; на Linux опционально подсекунда с /dev/pps{N} (pps_linux.go) | ✅ Реализовано |
| **PHC** | Чтение PHC встроено в источник **ptp** (device=/dev/ptp0 и т.д.) | ✅ Реализовано |

**Что добавить (по приоритету):**
1. **GNSS реальное время** — парсинг UBX-NAV-PVT или NMEA RMC с приёмника вместо time.Now() в gnss.go.
2. ~~**NMEA**~~ — сделано: **internal/source/nmea.go**, протокол `nmea`, device/baud/offset в конфиге.
3. **PPS** — интеграция с Linux PPS API (pin, linked_device).
4. ~~**PTP/PHC**~~ — сделано: **internal/source/ptp_linux.go** — открытие /dev/ptp{N}, FD_TO_CLOCKID, clock_gettime; конфиг: protocol ptp, device=/dev/ptp0, ptp4l запускается отдельно.

---

## 5. Конфигурация и Beats

| Анализ (code_analysis) | Реализация | Статус |
|------------------------|------------|--------|
| YAML в стиле Timebeat (clock_sync, primary_clocks, secondary_clocks) | **tc-sync** internal/config + **pkg/config**; **timebeat** распаковывает секцию timebeat | ✅ Есть |
| Конфиг совместим с shiwatime_ru.yml (поля источников, step_limit и т.д.) | pkg/config расширен под shiwatime; лишние ключи игнорируются | ✅ Есть |
| Elastic Beats v7 (libbeat) | **timebeat** — main, beater, timebeat.yml, _meta | ✅ Есть |

---

## 6. Что из анализа пока не использовано (актуально на 2025-01)

- **62 смещения UBXTP5Message** — в tc-sync используются первые 32 байта payload; остальные 56 полей — по необходимости.
- **Извлечённые коэффициенты и константы** (DefaultAlgoCoefficients, D-массив, 51712 для PI) — в servo задаются через конфиг (kp/ki/kd); точные константы не подставлены.
- **send1PPSOnTimepulsePin**, **detectUbloxUnit** — не реализованы; по необходимости.
- ~~LinReg~~ — реализовано. ~~NMEA, PPS, PTP, PHC~~ — реализованы как полноценные источники. ~~PerformGranularityMeasurement, GetClockFrequency~~ — реализованы в clockadj.

---

## 7. Рекомендуемый порядок доработок

1. ~~**Реальное время с GNSS**~~ — сделано: UBX-NAV-PVT в **internal/ubx/navpvt.go**, gnss.GetTime() читает с приёмника.
2. ~~**PPS**~~ — сделано: linked_device (GNSS) для секунды + cable_delay; на Linux опционально /dev/pps{N} для подсекунды (pps_linux.go).
3. **Коэффициенты servo** — при желании приблизиться к оригиналу: подставить константы из FOUND_COEFFICIENTS.md и coeffs_*.txt.
4. ~~**LinReg**~~ — сделано: **internal/servo/linreg.go**, алгоритм `linreg` в конфиге servo.
5. **PTP-клиент** — при необходимости синхронизации по сети (библиотека или свой минимальный клиент).
6. **Расширение UBX TP5** — только если понадобятся дополнительные поля из 62 смещений.

---

## 8. Ссылки на файлы анализа

| Цель | Файл |
|------|------|
| Структура программы, UBX, Servo, HostClock | `program_structure.go` |
| Смещения UBXTP5Message (62) | `algorithm_details.txt` |
| Servo: PID/PI/LinReg, коэффициенты, структуры | `FOUND_COEFFICIENTS.md`, `EXTRACTED_COEFFICIENTS.md`, `coeffs_*.txt` |
| Список функций по модулям | `check_completeness.py`, `completeness_check.txt` |
| Детали servo-функций | `servo_functions_detailed_analysis.txt`, `analyze_found_servo_functions.py` |
| Общий отчёт | `MASTER_ANALYSIS_REPORT.md`, `IMPLEMENTATION_GUIDE.md` |

Итог: **базовый цикл (выбор источника + PID/PI/LinReg + clockadj) и конфиг в стиле Timebeat реализованы**. Реализовано: **реальное время с GNSS (UBX-NAV-PVT)**, **LinReg servo**, **GetFrequency** и **GranularityNs** в clockadj, **PPS** (linked_device + cable_delay, на Linux опционально /dev/pps), **NMEA** (RMC с serial, offset). Не реализованы или упрощены: полная структура UBX TP5 (62 поля), **PTP** (заглушка; рекомендуется linuxptp+PHC или библиотека), PHC и использование извлечённых коэффициентов (51712, D-массив).

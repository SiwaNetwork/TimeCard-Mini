# Статус извлечённого кода: что есть и можно ли собрать бинарник

## Источник реализации: из бинарника или по аналогии

| Тип | Что сделано |
|-----|-------------|
| **Из бинарника** | Извлечены только **имена** пакетов и функций (`strings` + пути). На их основе созданы **заглушки** (`// TODO: реконструировать`). Логика из машинного кода **не извлекалась**. |
| **По аналогии** | Вся **реализованная логика** (NTP Query, servo PID/PI, config YAML, step/slew, adjusttime) написана **вручную** по типичным алгоритмам и документации. Она **не гарантирует идентичность** оригинальному бинарнику. |

Для **строгой реализации, идентичной оригиналу**, нужно реконструировать код **по дизассемблеру** бинарника. См. **`code_analysis/RECONSTRUCTION_FROM_BINARY.md`** и скрипт **`code_analysis/extract_disassembly_for_functions.py`** (извлечение дизассемблера по списку функций).

---

## Краткий ответ (обновлено после реконструкции заглушек)

| Вопрос | Ответ |
|--------|--------|
| **Всё ли извлечено?** | Имена ~2875 функций извлечены; логика реконструирована для **servo** (PID, PI, LinReg, Controller, Offsets, ClockQuality), **adjusttime**, **hostclocks**, **UBX**. Остальные пакеты — валидные заглушки (`// TODO` или `Extracted_Go()`). |
| **Можно ли собрать бинарник из `extracted_source/`?** | **Да.** Добавлены `go.mod`, исправлены невалидные идентификаторы (`func 1()` → `Extracted_Init_1`, `func go()` → `main` в main / `Extracted_Go` в остальных). Сборка: `cd extracted_source && go build -o timebeat-reconstructed.exe ./main`. |
| **Что делает собранный бинарник?** | Запускает **servo.Controller**: `GetController().Run(ctx)` — цикл синхронизации (таймеры, RunWakeupServoLoop). Полная интеграция NTP/PTP/GNSS — в заглушках, можно дописывать по мере необходимости. |

---

## Что реально извлечено

### 1. Полноценная реконструкция (логика есть)

- **`beater/clocksync/servo/algos/`** — PID, PI (shiwatime-style), LinReg, StdDev, MovingMinimum, BestFitFiltered, CoefficientStore, tuner (Kalman, OnlinePIDTuner).
- **`beater/clocksync/servo/`** — controller.go, offsets.go, clock_quality.go (каркас контроллера и источников).
- **`beater/clocksync/servo/adjusttime/`** — adjtimex, slew/step (Linux); заглушка для не-Linux.
- **`beater/clocksync/hostclocks/`** — HostClock, HostClockController (каркас).
- **`beater/clocksync/clients/vendors/helper/ubx/`** — разбор UBX (CFG-TP5, NAV-PVT).
- **`config/`** — Load(path), ClockSyncConfig, ClockSource, ServoConfig (YAML, формат timebeat.clock_sync).
- **`beater/clocksync/clients/ntp/`** — Query(host), Controller, ConfigureTimeSource(ip, pollInterval); поллер пишет offset в servo.Offsets.
- **`beater/clocksync/run_with_config.go`** — RunWithConfig(ctx, cfg): поднимает NTP по primary_clocks/secondary_clocks, затем servo.Run(ctx).
- **main** — флаг `-config timebeat.yml`; при наличии конфига вызывает RunWithConfig, иначе Run (только servo).

### 2. Реконструкция заглушек (реализовано полностью)

- **Скрипт `code_analysis/reconstruct_stubs.py`** — проходит по всем `.go` в `extracted_source` и исправляет невалидные идентификаторы:
  - `func 1()` … `func 9()` → `func Extracted_Init_1()` … `Extracted_Init_9()`;
  - `func go()` → `func main()` в `package main`, в остальных пакетах → `func Extracted_Go()`;
  - `func type()` → `func Extracted_Type()`.
  - Поддержка `--dry-run`. После перегенерации заглушек: `python3 code_analysis/reconstruct_stubs.py` (или через WSL).
- Исправлено **304 файла**. Удалён дублирующий **`servo.go`** (логика в `controller.go`, `offsets.go`, `clock_quality.go`).
- **`go.mod`** — модуль `github.com/shiwa/timecard-mini/extracted-source`.
- **Точка входа** — **`main/main.go`**: `clocksync.GetController().Run(ctx)` по Ctrl+C (цепочка main → clocksync → servo).
- **Пакет clocksync** — реализован: `Controller` держит `servo.Controller`, `GetController()`, `NewController()`, `Run(ctx)` делегируют в servo.
- **servo.RunPeriodicAdjustSlaveClocks** — реализован: медиана offset по кандидатам → `algos.AlgoPID.CalculateNewFrequency` → `adjusttime.SetFrequency` (на Linux — adjtimex; на не-Linux — заглушка в `adjusttime_stub.go`).

---

## Что собирать: tc-sync и timebeat

| Проект | Путь | Назначение |
|--------|------|------------|
| **tc-sync** | `tc-sync/` | Библиотека синхронизации времени (servo PID/PI/LinReg, NTP, GNSS, PPS, ptp4l). Есть `go.mod`, собирается в библиотеку. |
| **timebeat** | `timebeat/` | Elastic Beat (v7), использует tc-sync. Есть `go.mod`, `main.go` — собирается в бинарник Beat. |

### Сборка своего бинарника

**Вариант 1: бинарник timebeat (Beat с Elastic)**

```bash
cd timebeat
go build -o timebeat .
```

**Вариант 2: свой демон на базе tc-sync**

```bash
cd tc-sync
go build ./cmd/...   # если есть cmd
# или используйте tc-sync как библиотеку в своём main
```

Используемые алгоритмы (PID, pi_shiwatime, LinReg) и коэффициенты (Kp=0.5, Ki≈0.5946, Kd=1/√2) в **tc-sync** уже приведены в соответствие с тем, что извлечено из бинарника timebeat.

---

## Итог

- **Из `extracted_source/` бинарник теперь собирается**: `cd extracted_source && go build -o timebeat-reconstructed.exe ./main`. Заглушки приведены к валидному Go, добавлены `go.mod` и рабочий `main`.
- Дальнейшая реконструкция: подставлять логику в пакеты с `// TODO` и `Extracted_Go()` по мере анализа бинарника (дизассемблер, строки, вызовы).
- **tc-sync** и **timebeat** по-прежнему дают полный рабочий стек (NTP, PTP, GNSS, Elastic); `extracted_source` — реконструкция из бинарника и основа для доработки под себя.

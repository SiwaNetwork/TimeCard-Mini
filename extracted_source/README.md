# Извлечённые исходники timebeat

Реконструкция исходного кода из бинарника **timebeat-2.2.20-amd64**.

**Важно:** из бинарника извлечены только **имена** пакетов и функций; реализованная логика (NTP, servo, config) написана **по аналогии** с типичным поведением и **не гарантирует идентичность** оригиналу. Для **строгой реализации из бинарника** см. **`code_analysis/RECONSTRUCTION_FROM_BINARY.md`** и скрипт **`extract_disassembly_for_functions.py`**.

## Структура

```
extracted_source/
├── beater/
│   └── clocksync/
│       ├── servo/
│       │   ├── algos/           # PID, PI, LinReg, StdDev, etc.
│       │   │   ├── algos.go     # AlgoPID
│       │   │   ├── pi.go        # Pi, pi_servo
│       │   │   ├── linreg.go    # LinReg, linreg_servo
│       │   │   ├── helpers.go   # StdDev, MovingMinimum, etc.
│       │   │   ├── coefficients.go # CoefficientStore
│       │   │   └── tuner/       # OnlinePIDTuner, KalmanFilter
│       │   ├── adjusttime/      # adjtimex, clock_settime
│       │   ├── controller.go    # Controller
│       │   ├── offsets.go       # Offsets, TimeSource
│       │   └── clock_quality.go # ClockQuality
│       ├── hostclocks/          # HostClock, HostClockController
│       └── clients/
│           └── vendors/
│               └── helper/
│                   └── ubx/     # UBX протокол
└── README.md
```

## Извлечённые алгоритмы

### PID (AlgoPID)
- `CalculateNewFrequency(offsetNs, dt)` — основной расчёт
- `adjustDComponent(offset, derivative)` — адаптивный D по log(abs(offset))
- `enforceAdjustmentLimit(freq)` — ограничение коррекции

### PI (Pi, pi_servo)
- `CalculateFrequency(offsetNs, dt)` — shiwatime-style формула
- `pi_sample(offset, localTs, weight)` — linuxptp-style
- Константа IntegralTarget = 1e9 (0x3b9aca00)

### LinReg (LinReg, linreg_servo)
- Окно 64 точки
- `regress()` — линейная регрессия
- `linreg_sample`, `linreg_leap`, `linreg_reset`

## Коэффициенты (DefaultAlgoCoefficients)

Адрес 0x7c1a040:
- Kp = 0.5
- Ki = 0.5946 (≈ 2^(-0.75))
- Kd = 0.7071 (= 1/√2)

## Сборка бинарника

```bash
cd extracted_source
go build -o timebeat-reconstructed.exe ./main
```

Точка входа: **main** → **clocksync.GetController().Run(ctx)** → **servo.Controller.Run** (таймеры + RunWakeupServoLoop). В цикле: `RunPeriodicAdjustSlaveClocks` берёт медиану offset по источникам, считает коррекцию через **algos.AlgoPID**, применяет **adjusttime.SetFrequency** (на Linux — adjtimex). Остальные пакеты — заглушки; после реконструкции идентификаторов (`python3 code_analysis/reconstruct_stubs.py`) сборка проходит.

## Конфиг и источники времени

- **config.Load(path)** — загрузка YAML (формат `timebeat.clock_sync`, как в shiwatime_ru.yml).
- **NTP**: в `primary_clocks` / `secondary_clocks` укажите `protocol: ntp`, `ip: pool.ntp.org`, `pollinterval: 16s`. Поллеры пишут offset в **servo.Offsets**, servo применяет PID и **adjusttime.SetFrequency**.
- Запуск: `timebeat-reconstructed.exe -config timebeat.yml` (без конфига — только servo loop без источников).

## Дальнейшая реконструкция заглушек

- Заглушки с `// TODO: реконструировать` и `Extracted_Go()` можно заполнять по дизассемблированию бинарника (`objdump -d`, Ghidra) или по аналогии с `tc-sync`.
- Критичные пути: `beater/clocksync/controller`, `beater/clocksync/clients/ntp`, `clients/ptp`, `clients/nmea` — подключать к `servo.Controller` и источникам времени.

## Использование как библиотеки

Эти файлы — реконструкция логики. Для полного работающего кода можно также использовать `tc-sync`:

```go
import "github.com/shiwa/timecard-mini/tc-sync/internal/servo"

algo := servo.NewPID(0.5, 0.5946, 0.7071)
freq := algo.Update(offsetNs, dt)
```

## Источник

- Бинарник: `timebeat-extracted/usr/share/timebeat/bin/timebeat`
- Формат: ELF64 x86-64, Go
- Пакеты: `github.com/lasselj/timebeat/beater/clocksync/...`

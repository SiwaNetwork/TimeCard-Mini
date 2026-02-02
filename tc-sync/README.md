# tc-sync — аналог Timebeat

**Платформа синхронизации времени** (как [Timebeat](https://www.timebeat.app/)): несколько источников времени (GNSS, NTP, PTP, PPS), выбор primary/secondary, servo-коррекция системных часов.

Разработано на основе анализа бинарника shiwatime (Timebeat-based) в [code_analysis/](../code_analysis/).

**Про Timebeat:** продукт [Timebeat](https://www.timebeat.app/) (синхронизация времени, PTP, GNSS) действительно сделан на базе **Elastic Beats** (в бинарнике shiwatime видны пути вида `github.com/lasselj/timebeat/beater/clocksync/...`). Репозиторий [davdjl/timebeat](https://github.com/davdjl/timebeat) — это другой проект (Beat для мониторинга времени жизни файлов), не продукт Timebeat для синхронизации времени. tc-sync — самостоятельная реализация логики синхронизации (без фреймворка Beats).

## Возможности

| Компонент | Timebeat | tc-sync |
|-----------|----------|---------|
| Конфиг | YAML, clock_sync, primary/secondary_clocks | ✅ Такой же формат |
| GNSS (UBX / Timecard Mini) | Да | ✅ UBX, CFG-TP5, serial |
| NTP клиент | Да | ✅ Простой NTP client |
| PTP клиент | Да | ✅ ptp4l+PHC (чтение /dev/ptpN, start_ptp4l в конфиге) |
| PPS | Да | ✅ linked_device + cable_delay; на Linux опционально /dev/pps{N} |
| Выбор источника | Primary → Secondary | ✅ Election |
| Servo | PID, PI, LinReg | ✅ PID, PI, pi_shiwatime, LinReg |
| Коррекция часов | adjtimex, step/slew | ✅ Linux: adjtimex + clock_settime (нужен root/CAP_SYS_TIME) |
| Time pulse (CFG-TP5) | CLI / конфиг | ✅ configure |
| Elastic/Beats, CLI, HTTP | Да | ❌ Нет |

## Требования: Go

Установка/обновление Go на Windows (winget):

```powershell
winget install GoLang.Go --accept-package-agreements --accept-source-agreements
```

После установки **перезапустите терминал** или Cursor, чтобы обновился PATH. Либо в текущей сессии:

```powershell
. .\scripts\ensure-go.ps1   # добавить Go в PATH и проверить go version
```

## Сборка

**Целевая ОС — Linux** (сервер, Raspberry Pi / Grand Mini и т.п.). С Windows собираем под Linux кросс-компиляцией.

Сборка под Linux (из папки `tc-sync` или из корня репо):

```powershell
# Сначала добавить Go в PATH (если ещё не добавлен)
. .\scripts\ensure-go.ps1

# Linux amd64 (по умолчанию)
.\scripts\build-linux.ps1

# Linux arm64 (Raspberry Pi 3/4, Grand Mini)
.\scripts\build-linux.ps1 -Arch arm64

# Linux arm (32-bit, например RPi 2)
.\scripts\build-linux.ps1 -Arch arm
```

Или вручную:

```bash
cd tc-sync
go mod tidy
GOOS=linux GOARCH=amd64 go build -o tc-sync-linux-amd64 ./cmd/tc-sync
GOOS=linux GOARCH=arm64 go build -o tc-sync-linux-arm64 ./cmd/tc-sync
```

Бинарник `tc-sync-linux-<arch>` копируйте на целевой Linux и запускайте там (для PTP/PHC, adjtimex и т.д. нужен Linux).

## Использование

### 1. Настройка time pulse (UBX)

Как в Timebeat: настроить длительность импульса PPS на модуле u-blox:

```bash
# По конфигу (device/timepulse в tc-sync.yml)
./tc-sync -configure

# Без конфига
./tc-sync -configure -port /dev/ttyS0 -baud 9600 -pulse-width-ms 5
./tc-sync -configure -port /dev/ttyS0 -baud 115200 -pulse-width-ms 5 -quiet
```

### 2. Daemon (аналог Timebeat)

Выбор источника времени (primary → secondary) и цикл servo:

```bash
./tc-sync -run -config tc-sync.yml
```

В конфиге должен быть блок **clock_sync** (как в Timebeat):

```yaml
clock_sync:
  adjust_clock: true
  primary_clocks:
    - protocol: timebeat_opentimecard_mini  # или gnss
      device: /dev/ttyS0
      baud: 115200
  secondary_clocks:
    - protocol: ntp
      ip: pool.ntp.org
      pollinterval: 4s
```

Поддерживаемые протоколы в **primary_clocks** / **secondary_clocks**:

- **gnss** или **timebeat_opentimecard_mini** — UBX/Timecard Mini (device, baud)
- **ntp** — NTP клиент (ip, pollinterval)
- **pps** — секунда с linked_device (GNSS), cable_delay; на Linux опционально подсекунда с /dev/pps{N}
- **ptp** — чтение времени из PHC (/dev/ptpN), синхронизированного ptp4l (linuxptp); device=/dev/ptp0, domain, interface

## Конфиг (формат Timebeat)

- **device** / **timepulse** — для `-configure` (порт, скорость, длительность импульса).
- **clock_sync** — для `-run`:
  - **adjust_clock** — разрешить коррекцию часов (пока только лог).
  - **primary_clocks** — список источников (первый доступный используется).
  - **secondary_clocks** — резерв при недоступности primary.

Пример полного конфига: [tc-sync.example.yml](tc-sync.example.yml).

## Структура проекта

```
tc-sync/
├── cmd/tc-sync/main.go     # configure, run (daemon)
├── internal/
│   ├── ubx/                # UBX, CFG-TP5, serial
│   ├── source/             # GNSS, NTP, PPS, PTP (источники времени)
│   ├── clockselect/        # выбор primary/secondary
│   ├── servo/              # PID, PI
│   └── config/             # YAML (формат Timebeat)
├── go.mod
├── tc-sync.example.yml
└── README.md
```

## Дальнейшее развитие

- ~~Реальная коррекция часов на Linux~~ — сделано: **adjtimex** (slew, SetFrequency), **clock_settime** (step при offset > 500 ms). Запуск с `adjust_clock: true` и правами root или CAP_SYS_TIME.
- Полная реализация **PPS** (Linux PPS API) и **PTP** клиента (IEEE 1588).
- Опционально: PTP Grandmaster, NTP server, экспорт метрик, HTTP/CLI как в Timebeat.

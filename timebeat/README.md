# Timebeat

Beat на базе **Elastic Beats v7** (libbeat) для синхронизации времени — аналог коммерческого Timebeat/shiwatime. Использует библиотеку [tc-sync](../tc-sync) для выбора источника времени (GNSS/UBX, NTP, PTP, PPS) и коррекции системных часов через servo (PID/PI).

**Формат конфига** совместим с Timebeat/shiwatime: секция `timebeat` как в [shiwatime_ru.yml](../shiwatime_ru.yml) — те же ключи `clock_sync`, `primary_clocks`, `secondary_clocks`, опционально `license.keyfile`, `config.peerids`, расширенные поля источников (PTP, PPS, NTP, timebeat_opentimecard_mini). Лишние ключи игнорируются; поддерживается подмножество функций.

## Требования

- Go 1.21+
- Модуль [tc-sync](../tc-sync) (подключён через `replace` в `go.mod`)
- На Linux для коррекции часов — права root или `CAP_SYS_TIME`

## Сборка

```bash
cd timebeat
go mod tidy
go build -o timebeat .
```

Либо из корня репозитория:

```bash
cd TimeCard-Mini/timebeat
go build -o timebeat .
```

## Конфигурация

Файл `timebeat.yml` или `timebeat.reference.yml` (или путь через `-c`). Можно взять секцию `timebeat` из конфига shiwatime (`shiwatime_ru.yml`) — наш timebeat примет тот же формат.

Минимальный пример:

```yaml
timebeat:
  device:
    port: /dev/ttyS0
    baud: 9600
  timepulse:
    pulse_width_ms: 5
    tp_idx: 0
    ant_cable_delay_ns: 0
    align_to_tow: true
  servo:
    algorithm: pid
    kp: 0.1
    ki: 0.01
    kd: 0.001
    interval: 1s
  clock_sync:
    adjust_clock: true
    primary_clocks:
      - protocol: timebeat_opentimecard_mini
        device: /dev/ttyS0
        baud: 115200
        disable: false
        monitor_only: false
    secondary_clocks:
      - protocol: ntp
        ip: pool.ntp.org
        pollinterval: 4s
        disable: false
```

Секция `timebeat` соответствует структуре конфига tc-sync (device, timepulse, servo, clock_sync). Остальные опции libbeat (output, logging и т.д.) задаются как у любого Beat.

## Запуск

```bash
./timebeat -c timebeat.yml
```

Остановка: Ctrl+C или SIGTERM (корректно завершает цикл синхронизации).

## Сравнение с Timebeat и tc-sync

| | Timebeat (коммерческий) | timebeat (этот Beat) | tc-sync (standalone) |
|---|---|---|---|
| Основа | Elastic Beats v7 | Elastic Beats v7 | Отдельный бинарник |
| Конфиг | timebeat.yml (libbeat) | timebeat.yml (libbeat) | tc-sync.yml |
| Выход в ES/Logstash | Да | Да (через pipeline) | Нет |
| Синхронизация времени | tc-sync-логика | tc-sync (pkg/clocksync) | Встроенная |

Сборка на базе Elastic Beats v7 даёт единый формат конфигурации, логирования и вывода событий, как у оригинального Timebeat.

# Подробное руководство по конфигурации Shiwa Time

## ⚠️ Важное примечание

На устройстве может быть установлено программное обеспечение как **Timebeat**, так и **Shiwa Network** (Shiwa Time).

**Данное руководство описывает конфигурацию только Shiwa Time.**

---

## Содержание

1. [Введение](#введение)
2. [Структура конфигурационного файла](#структура-конфигурационного-файла)
3. [Базовые настройки](#базовые-настройки)
4. [Синхронизация времени (Clock Sync)](#синхронизация-времени-clock-sync)
5. [Первичные источники времени (Primary Clocks)](#первичные-источники-времени-primary-clocks)
6. [Вторичные источники времени (Secondary Clocks)](#вторичные-источники-времени-secondary-clocks)
7. [Расширенные настройки](#расширенные-настройки)
8. [Вывод данных (Output)](#вывод-данных-output)
9. [Логирование](#логирование)
10. [Примеры конфигурации](#примеры-конфигурации)

---

## Введение

Файл конфигурации **shiwatime.yml** расположен в `/etc/shiwatime/shiwatime.yml` и управляет всеми аспектами работы Shiwa Time, включая:
- Источники времени (PTP, NTP, PPS, GNSS)
- Алгоритмы синхронизации
- Вывод метрик
- CLI интерфейс
- Логирование

**⚠️ Важно:** После изменения конфигурации всегда перезапускайте сервис:
```bash
sudo systemctl restart shiwatime
```

---

## Структура конфигурационного файла

Файл разделен на основные секции:

```yaml
timebeat:                    # Основная секция настроек Shiwa Time
  clock_sync:                # Настройки синхронизации часов
    primary_clocks:          # Первичные источники (главные)
    secondary_clocks:        # Вторичные источники (резервные)
    advanced:                # Расширенные настройки
  
name:                        # Имя устройства
output.elasticsearch:        # Вывод в Elasticsearch
output.file:                 # Вывод в файл
logging:                     # Настройки логирования
```

---

## Базовые настройки

### Лицензия и идентификация

```yaml
timebeat:
  # Путь к файлу лицензии для премиум функций
  license.keyfile: '/etc/shiwatime/timebeat.lic'
  
  # Путь к файлу идентификации пиров
  config.peerids: '/etc/shiwatime/peerids.json'
```

**Описание:**
- `license.keyfile` - лицензионный ключ для разблокировки дополнительных функций
- `config.peerids` - файл для сохранения идентификаторов подключенных устройств

### Имя устройства

```yaml
name: TESTER
```

**Описание:**
- Уникальное имя устройства для идентификации в системе мониторинга
- Отображается в логах и Elasticsearch
- **Рекомендация:** используйте понятные имена (например, "Moscow-DC1-TimeServer")

---

## Синхронизация времени (Clock Sync)

### Основные параметры

```yaml
clock_sync:
  # Включить/выключить синхронизацию системных часов
  adjust_clock: true
  
  # Лимит шага времени (необязательно)
  # step_limit: 15m
```

**Описание:**

#### `adjust_clock`
- **true** - активная синхронизация (изменяет системное время)
- **false** - только мониторинг (не изменяет время)
- **Применение:** установите `false` для тестирования без изменения времени

#### `step_limit`
- Максимальное отклонение, при котором разрешен "прыжок" времени
- Форматы: `s` (секунды), `m` (минуты), `h` (часы), `d` (дни)
- **Пример:** `step_limit: 15m` - разрешить прыжок до 15 минут
- **По умолчанию:** без ограничений (закомментировано)

---

## Первичные источники времени (Primary Clocks)

Первичные источники - это главные источники времени, которые используются в первую очередь.

### 1. PTP (Precision Time Protocol)

```yaml
primary_clocks:
  - protocol: ptp
    domain: 0
    serve_unicast: true
    serve_multicast: true
    server_only: true
    announce_interval: 1
    sync_interval: 0
    delayrequest_interval: 0
    disable: false
    interface: eth0
```

**Параметры PTP:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `protocol` | string | Протокол: `ptp` |
| `domain` | int | PTP домен (0-127). Разные домены изолированы друг от друга |
| `serve_unicast` | bool | Раздавать время через unicast (точка-точка) |
| `serve_multicast` | bool | Раздавать время через multicast (широковещательно) |
| `server_only` | bool | Только сервер (grandmaster режим) |
| `max_unicast_subscribers` | int | Максимум unicast клиентов (0 = без ограничений) |
| `announce_interval` | int | Интервал announce сообщений (log₂ секунд) |
| `sync_interval` | int | Интервал sync сообщений (log₂ секунд) |
| `delayrequest_interval` | int | Интервал delay request (log₂ секунд) |
| `disable` | bool | Отключить этот источник (true/false) |
| `interface` | string | Сетевой интерфейс (eth0, ens1, etc.) |

**Интервалы (log₂ формат):**
- `-3` = 125 мс (2⁻³ секунды)
- `-2` = 250 мс
- `-1` = 500 мс
- `0` = 1 секунда
- `1` = 2 секунды
- `2` = 4 секунды

**Дополнительные параметры PTP:**

```yaml
priority1: 128                    # Приоритет 1 (меньше = выше)
priority2: 128                    # Приоритет 2
profile: 'enterprise-draft'       # Профиль PTP
asymmetry_compensation: 0         # Компенсация асимметрии (нс)
unicast_master_table: ['1.2.3.4'] # Список unicast мастеров
delay_strategy: e2e               # Стратегия: e2e или p2p
hybrid_e2e: false                 # Unicast delay requests
use_layer2: false                 # PTP через Ethernet (не UDP)
monitor_only: false               # Только мониторинг
```

**Профили PTP:**
- `enterprise-draft` - корпоративный (рекомендуется)
- `G.8275.1` - Telecom профиль (полный on-path support)
- `G.8275.2` - Telecom профиль (частичный on-path support)
- `G.8265.1` - Frequency профиль

### 2. NTP (Network Time Protocol)

```yaml
primary_clocks:
  - protocol: ntp
    ip: 'pool.ntp.org'
    pollinterval: 4s
    monitor_only: false
```

**Параметры NTP:**

| Параметр | Описание |
|----------|----------|
| `protocol` | `ntp` |
| `ip` | IP адрес или доменное имя NTP сервера |
| `pollinterval` | Интервал опроса (500ms, 1s, 2s, 4s, 8s) |
| `monitor_only` | Только мониторинг без синхронизации |

**Рекомендации:**
- Для публичных NTP серверов: `pollinterval: 4s`
- Для локальных NTP серверов: можно использовать `1s` или `2s`
- Популярные NTP серверы: `pool.ntp.org`, `time.google.com`, `time.cloudflare.com`

### 3. PPS (Pulse Per Second)

```yaml
primary_clocks:
  - protocol: pps
    interface: eth0
    pin: 0
    index: 0
    cable_delay: 0
    edge_mode: "rising"
    monitor_only: false
    atomic: false
    linked_device: '/dev/ttyS0'
```

**Параметры PPS:**

| Параметр | Описание |
|----------|----------|
| `protocol` | `pps` |
| `interface` | Сетевой интерфейс с PPS |
| `pin` | Номер контакта (SDP - Software Defined Pin) |
| `index` | Индекс/канал PPS |
| `cable_delay` | Задержка кабеля в наносекундах (~5 нс/метр) |
| `edge_mode` | Триггер: `rising` (нарастающий), `falling` (падающий), `both` (оба) |
| `monitor_only` | Только мониторинг |
| `atomic` | Атомные часы? |
| `linked_device` | Связанное NMEA/Timecard устройство |

**Расчет задержки кабеля:**
- Скорость распространения сигнала: ~200,000 км/с (66% скорости света)
- Приблизительная задержка: **5 наносекунд на метр**
- Пример: кабель 10 метров = `cable_delay: 50`

**Режимы edge_mode:**
- `rising` - синхронизация по нарастающему фронту (0→1)
- `falling` - по падающему фронту (1→0)
- `both` - по обоим фронтам (для старых карт)

---

## Вторичные источники времени (Secondary Clocks)

Вторичные источники активируются, когда все первичные источники недоступны.

### 1. Timecard Mini (GNSS)

```yaml
secondary_clocks:
  - protocol: timebeat_opentimecard_mini
    device: '/dev/ttyS0'
    baud: 115200
    card_config: ['gnss1:signal:gps+glonass']
    offset: 225000000
    atomic: false
    monitor_only: false
```

**Параметры Timecard Mini:**

| Параметр | Описание |
|----------|----------|
| `protocol` | `timebeat_opentimecard_mini` |
| `device` | Последовательный порт устройства |
| `baud` | Скорость передачи (обычно 115200) |
| `card_config` | Конфигурация GNSS сигналов |
| `offset` | Статическое смещение (наносекунды) |
| `atomic` | Атомные часы |
| `monitor_only` | Только мониторинг |

**Конфигурация GNSS сигналов:**
- `gps` - GPS (США)
- `glonass` - ГЛОНАСС (Россия)
- `galileo` - Galileo (ЕС)
- `beidou` - BeiDou (Китай)
- `sbas` - SBAS (дополнительная система)
- `qzss` - QZSS (Япония)

**Примеры:**
```yaml
# GPS + ГЛОНАСС (рекомендуется для России)
card_config: ['gnss1:signal:gps+glonass']

# GPS + Galileo
card_config: ['gnss1:signal:gps+galileo']

# Все системы
card_config: ['gnss1:signal:gps+glonass+galileo+beidou']
```

### 2. NMEA (Generic GNSS)

```yaml
secondary_clocks:
  - protocol: nmea
    device: '/dev/ttyS0'
    baud: 115200
    offset: 0
    monitor_only: false
```

**Описание:**
- Универсальный протокол для GNSS приемников
- Формат 8N1 (8 бит данных, без паритета, 1 стоп-бит)
- Используется для совместимости со стандартными GNSS модулями

### 3. OCP Timecard

```yaml
secondary_clocks:
  - protocol: ocp_timecard
    ocp_device: 0
    oscillator_type: 'timebeat-rb-ql'
    card_config:
      - 'sma1:out:mac'
      - 'gnss1:signal:gps+galileo+sbas'
      - 'mac:type:timebeat-rb-ql'
    offset: 0
    atomic: false
    monitor_only: false
```

**Описание:**
- Профессиональная Timecard от OCP (Open Compute Project)
- Поддержка высокоточных осцилляторов (Rubidium, OCXO)
- Конфигурация SMA выходов и GNSS

### 4. PHC (PTP Hardware Clock)

```yaml
secondary_clocks:
  - protocol: phc
    device: '/dev/ptp_hyperv'
    offset: 0
    monitor_only: false
```

**Описание:**
- Аппаратные часы PTP в сетевых картах
- Полезно для виртуализации (Azure, VMware)
- Устройства: `/dev/ptp0`, `/dev/ptp1`, `/dev/ptp_hyperv`

---

## Расширенные настройки

### Алгоритмы синхронизации (Steering)

```yaml
advanced:
  steering:
    algo: sigma
    algo_logging: false
    outlier_filter_enabled: true
    outlier_filter_type: strict
    servo_offset_arrival_driven: true
```

**Алгоритмы:**
- **sigma** - рекомендуется для стабильных сетей с hardware timestamping
- **rho** - для нестабильных сетей
- **alpha, beta, gamma** - специализированные алгоритмы

**Фильтры выбросов:**
- `strict` - строгая фильтрация (рекомендуется)
- `moderate` - умеренная
- `relaxed` - мягкая

### Настройки Clock Quality (PTP Grandmaster)

```yaml
ptp_tuning:
  clock_quality:
    auto: false
    class: 6
    accuracy: 0x20
    variance: 0x4E20
    timesource: 0x20
```

**Clock Class:**
| Class | Описание |
|-------|----------|
| 6 | Primary Reference (GNSS синхронизирован) |
| 7 | Primary Reference Holdover (режим удержания) |
| 13 | Application Specific |
| 52 | Degraded mode |
| 248 | Default (не синхронизирован) |

**Accuracy (точность):**
- `0x20` = 25 наносекунд (для GNSS)
- `0x21` = 100 наносекунд
- `0x22` = 250 наносекунд
- `0xFE` = неизвестна

**Time Source:**
- `0x10` = ATOMIC_CLOCK (атомные часы)
- `0x20` = GPS (GNSS синхронизация)
- `0xA0` = INTERNAL_OSCILLATOR

### Linux специфичные настройки

```yaml
linux_specific:
  hardware_timestamping: true
  external_software_timestamping: true
  sync_nic_slaves: true
  tai_offset: auto
```

**Параметры:**
- `hardware_timestamping` - аппаратные метки времени (SOF_TIMESTAMPING)
- `sync_nic_slaves` - синхронизировать PHC других сетевых карт
- `tai_offset` - смещение TAI (auto/ptp/nmea/37s)

### CLI интерфейс

```yaml
cli:
  enable: true
  bind_port: 65129
  bind_host: 127.0.0.1
  server_key: "/etc/shiwatime/cli_id_rsa"
  username: "admin"
  password: "password"
```

**Описание:**
- SSH интерфейс для управления
- **Подключение:** `ssh -p 65129 admin@127.0.0.1`
- **⚠️ Важно:** смените пароль по умолчанию!

**Смена пароля:**
```yaml
username: "admin"
password: "ваш_надежный_пароль"
```

---

## Вывод данных (Output)

### Elasticsearch

```yaml
output.elasticsearch:
  hosts: ['90.156.231.174:9200']
  # protocol: 'https'
  # username: 'elastic'
  # password: 'changeme'
```

**Описание:**
- Отправка метрик в Elasticsearch для визуализации в Kibana
- Поддержка HTTP и HTTPS
- Аутентификация: API key или username/password

⚠️ **Важно:** Выгрузка данных работает только при наличии подключения к интернету. Для отключения отправки данных на сервер закомментируйте или удалите секцию `output.elasticsearch`.

**Для Timebeat Cloud:**
```yaml
output.elasticsearch:
  hosts: ['elastic.customer.timebeat.app:9200']
  protocol: 'https'
  username: 'elastic'
  password: 'your_password'
```

### Вывод в файл

```yaml
output.file:
  enabled: false
  path: "/tmp/shiwatime"
  filename: timebeat
  rotate_every_kb: 10000
  number_of_files: 7
```

**Описание:**
- Сохранение метрик в локальные файлы JSON
- Ротация по размеру
- Полезно для отладки без Elasticsearch

---

## Логирование

```yaml
logging.to_files: true
logging.files:
  path: /var/log/shiwatime
  name: timebeat
  rotateeverybytes: 10485760  # 10 МБ
  keepfiles: 7
  permissions: 0600
```

**Параметры:**
- `path` - директория для логов
- `rotateeverybytes` - размер файла для ротации (в байтах)
- `keepfiles` - количество хранимых файлов
- `permissions` - права доступа (octal)

**Просмотр логов:**
```bash
# Последние логи
sudo journalctl -u shiwatime -n 50

# В реальном времени
sudo journalctl -u shiwatime -f

# Файлы логов
sudo tail -f /var/log/shiwatime/timebeat
```

---

## Примеры конфигурации

### Пример 1: PTP Grandmaster с GNSS

```yaml
timebeat:
  clock_sync:
    adjust_clock: true
    
    primary_clocks:
      # PTP сервер (раздача времени)
      - protocol: ptp
        domain: 0
        serve_unicast: true
        serve_multicast: true
        server_only: true
        interface: eth0
        disable: false
      
      # PPS от GNSS
      - protocol: pps
        interface: eth0
        pin: 0
        index: 0
        edge_mode: "rising"
        linked_device: '/dev/ttyS0'
    
    secondary_clocks:
      # Timecard Mini
      - protocol: timebeat_opentimecard_mini
        device: '/dev/ttyS0'
        baud: 115200
        card_config: ['gnss1:signal:gps+glonass']
    
    advanced:
      ptp_tuning:
        clock_quality:
          auto: false
          class: 6          # GNSS синхронизирован
          accuracy: 0x20    # 25 ns
          timesource: 0x20  # GPS

name: PTP-GRANDMASTER-1

output.elasticsearch:
  hosts: ['monitoring.example.com:9200']
```

### Пример 2: PTP Client (получение времени)

```yaml
timebeat:
  clock_sync:
    adjust_clock: true
    
    primary_clocks:
      # PTP клиент
      - protocol: ptp
        domain: 0
        interface: eth0
        monitor_only: false
        disable: false
    
    secondary_clocks:
      # NTP резерв
      - protocol: ntp
        ip: 'pool.ntp.org'
        pollinterval: 4s

name: PTP-CLIENT-1
```

### Пример 3: Мониторинг без синхронизации

```yaml
timebeat:
  clock_sync:
    adjust_clock: false  # Только мониторинг!
    
    primary_clocks:
      - protocol: ptp
        domain: 0
        interface: eth0
        monitor_only: true

name: MONITOR-1

output.file:
  enabled: true
  path: "/var/log/shiwatime/monitoring"
```

---

## Проверка конфигурации

**Тестирование конфига:**
```bash
sudo /usr/share/shiwatime/bin/shiwatime test config -c /etc/shiwatime/shiwatime.yml
```

**Проверка синтаксиса YAML:**
```bash
# Установка yamllint
sudo apt install yamllint

# Проверка
yamllint /etc/shiwatime/shiwatime.yml
```

**Применение изменений:**
```bash
# Перезапуск сервиса
sudo systemctl restart shiwatime

# Проверка статуса
sudo systemctl status shiwatime

# Просмотр логов
sudo journalctl -u shiwatime -f
```

---

## Резервное копирование конфигурации

**Создать бэкап:**
```bash
sudo cp /etc/shiwatime/shiwatime.yml /etc/shiwatime/shiwatime.yml.backup-$(date +%Y%m%d)
```

**Восстановить из бэкапа:**
```bash
sudo cp /etc/shiwatime/shiwatime.yml.backup-20251002 /etc/shiwatime/shiwatime.yml
sudo systemctl restart shiwatime
```

---

## Полезные команды

```bash
# Проверка PTP портов
sudo ss -ulnp | grep -E "319|320"

# Мониторинг PTP трафика
sudo tcpdump -i eth0 -nn port 319 or port 320 -c 10

# Информация о PHC
ethtool -T eth0

# Список PTP устройств
ls -l /dev/ptp*

# Время PHC
sudo phc_ctl /dev/ptp0 get
```

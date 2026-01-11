# Настройка NTP сервера в shiwatime

## Введение

Shiwatime может работать как NTP сервер, раздавая время в локальную сеть. Это полезно, когда нужно синхронизировать другие устройства в сети через NTP протокол.

## Конфигурация NTP сервера

### Шаг 1: Откройте конфигурационный файл

```bash
sudo nano /etc/shiwatime/shiwatime.yml
```

### Шаг 2: Добавьте NTP сервер в секцию `secondary_clocks`

Найдите секцию `secondary_clocks` и раскомментируйте или добавьте конфигурацию NTP сервера:

```yaml
timebeat:
  clock_sync:
    secondary_clocks:
      # NTP сервер - раздача времени в локальную сеть
      - protocol:      ntp
        serve_unicast: true       # Раздавать время через unicast (точка-точка)
        interface:     eth0       # Сетевой интерфейс (замените на ваш интерфейс)
        server_only:   true       # Только сервер (не получать время от других NTP серверов)
        monitor_only:  false      # Раздача времени, а не только мониторинг
```

**Примечание:** Если вы хотите использовать NTP как источник времени (получать время от внешних серверов), то используйте конфигурацию без `server_only: true` и с параметром `ip`.

### Полный пример конфигурации

Пример конфигурации с PTP Grandmaster, PPS и NTP сервером:

```yaml
timebeat:
  clock_sync:
    adjust_clock: true
    
    primary_clocks:
      # PTP Grandmaster
      - protocol:                 ptp
        domain:                   0
        serve_unicast:           true
        serve_multicast:         true
        server_only:             true
        interface:               eth0
        disable:                 false
      
      # PPS от GNSS модуля
      - protocol:          pps
        interface:         eth0
        pin:               0
        index:             0
        cable_delay:       0
        edge_mode:         "rising"
        monitor_only:      false
        atomic:            false
        linked_device:     '/dev/ttyS0'
    
    secondary_clocks:
      # GNSS модуль для majortime
      - protocol:     timebeat_opentimecard_mini
        device:       '/dev/ttyS0'
        baud:         9600
        card_config:  ['gnss1:signal:gps+glonass']
        offset:       225000000
        atomic:       false
        monitor_only: false
      
      # NTP сервер - раздача времени в локальную сеть
      - protocol:      ntp
        serve_unicast: true
        interface:     eth0
        server_only:   true
        monitor_only:  false
```

## Параметры NTP сервера

| Параметр | Описание | Обязательный | Значение по умолчанию |
|----------|----------|--------------|----------------------|
| `protocol` | Протокол | Да | `ntp` |
| `serve_unicast` | Раздавать время через unicast | Да (для сервера) | `false` |
| `interface` | Сетевой интерфейс | Да (для сервера) | - |
| `server_only` | Только сервер (не получать время) | Нет | `false` |
| `monitor_only` | Только мониторинг (не раздавать) | Нет | `false` |

**Важно:**
- `serve_unicast: true` - включает раздачу времени через NTP
- `server_only: true` - устройство работает только как сервер, не запрашивает время от других NTP серверов
- `interface: eth0` - интерфейс, через который будет раздаваться время (замените на ваш интерфейс)

## Определение сетевого интерфейса

Чтобы узнать имя вашего сетевого интерфейса:

```bash
ip addr show
# Или
ifconfig
```

Обычно это `eth0` для Ethernet или `ens1`, `enp2s0` и т.д. в зависимости от системы.

## Применение конфигурации

После изменения конфигурации:

1. **Проверьте синтаксис YAML:**
```bash
sudo /usr/share/shiwatime/bin/shiwatime test config -c /etc/shiwatime/shiwatime.yml
```

2. **Перезапустите сервис:**
```bash
sudo systemctl restart shiwatime
```

3. **Проверьте статус:**
```bash
sudo systemctl status shiwatime
```

## Проверка работы NTP сервера

### Проверка открытых портов

NTP использует UDP порт 123:

```bash
sudo ss -ulnp | grep 123
# Или
sudo netstat -ulnp | grep 123
```

Вы должны увидеть что-то вроде:
```
UNCONN 0  0  0.0.0.0:123  0.0.0.0:*  users:(("shiwatime",pid=1234,fd=10))
```

### Тестирование с другого компьютера

С другого компьютера в локальной сети:

```bash
# Узнайте IP адрес вашего shiwatime устройства
# Например: 192.168.16.238

# Проверка доступности NTP сервера
ntpdate -q 192.168.16.238

# Или используя ntpq
ntpq -p 192.168.16.238
```

### Настройка клиентов для использования вашего NTP сервера

На клиентских устройствах (Linux):

1. **Отредактируйте `/etc/ntp.conf` или `/etc/systemd/timesyncd.conf`:**

Для systemd-timesyncd:
```ini
[Time]
NTP=192.168.16.238
FallbackNTP=pool.ntp.org
```

Или для ntpd:
```
server 192.168.16.238 iburst
```

2. **Перезапустите службу синхронизации времени:**
```bash
sudo systemctl restart systemd-timesyncd
# Или
sudo systemctl restart ntpd
```

3. **Проверьте статус синхронизации:**
```bash
timedatectl status
# Или
ntpq -p
```

## Настройка Windows клиентов

На компьютерах с Windows:

1. Откройте **Панель управления** → **Дата и время**
2. Перейдите на вкладку **Время по Интернету**
3. Нажмите **Изменить параметры**
4. Введите IP адрес вашего NTP сервера (например, `192.168.16.238`)
5. Нажмите **Обновить сейчас**

Или через командную строку (от имени администратора):
```cmd
w32tm /config /manualpeerlist:"192.168.16.238" /syncfromflags:manual /reliable:YES /update
net stop w32time
net start w32time
w32tm /resync
```

## Диагностика проблем

### NTP сервер не отвечает

1. **Проверьте, что сервис запущен:**
```bash
sudo systemctl status shiwatime
```

2. **Проверьте логи:**
```bash
sudo journalctl -u shiwatime -n 50 | grep -i ntp
```

3. **Проверьте файрвол:**
```bash
# Если используется ufw
sudo ufw status
sudo ufw allow 123/udp

# Если используется firewalld
sudo firewall-cmd --list-ports
sudo firewall-cmd --add-service=ntp --permanent
sudo firewall-cmd --reload
```

4. **Проверьте, что порт прослушивается:**
```bash
sudo ss -ulnp | grep 123
```

### Клиенты не могут синхронизироваться

1. **Проверьте сетевую связность:**
```bash
# С клиента
ping 192.168.16.238
```

2. **Проверьте, что NTP сервер доступен:**
```bash
# С клиента
ntpdate -q 192.168.16.238
```

3. **Проверьте настройки файрвола на клиенте**

## Альтернатива: Использование PTP вместо NTP

Если ваша цель - раздача точного времени в локальную сеть, рассмотрите использование **PTP (Precision Time Protocol)** вместо NTP:

- **PTP** обеспечивает точность до **наносекунд** (NTP - миллисекунды)
- **PTP** уже настроен в вашей конфигурации как Grandmaster
- Подходит для сетевого оборудования с поддержкой PTP

См. раздел "Настройка PTP Grandmaster" в документации для подробностей.

---

**Дата создания:** 2025-01-XX  
**Примечание:** Убедитесь, что ваша версия shiwatime поддерживает работу в режиме NTP сервера. Если возникают проблемы, проверьте документацию или обратитесь к поддержке.


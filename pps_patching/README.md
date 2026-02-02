# PPS Patching

Эта папка содержит все файлы, связанные с патчингом и настройкой PPS (Pulse Per Second) сигналов.

## Структура

### Python скрипты

- **`set_pps_5ms.py`** - Установка длительности PPS импульса на 5 мс
- **`set_pps_100ms.py`** - Установка длительности PPS импульса на 100 мс
- **`check_pps_patch.py`** - Проверка примененного патча PPS

### Документация

- **`PPS_PULSE_DURATION_GUIDE.md`** - Руководство по настройке длительности PPS импульсов

## Использование

### Установка длительности PPS импульса

```bash
# Установить 5 мс
python3 set_pps_5ms.py

# Установить 100 мс
python3 set_pps_100ms.py

# Проверить примененный патч
python3 check_pps_patch.py
```

### На устройстве

```bash
# Скопировать скрипты на устройство
scp pps_patching/*.py shiwa@grandmini.local:~/

# Запустить на устройстве
ssh shiwa@grandmini.local
python3 set_pps_5ms.py
```

## Дополнительная информация

Подробную информацию о настройке PPS импульсов можно найти в файле `PPS_PULSE_DURATION_GUIDE.md`.

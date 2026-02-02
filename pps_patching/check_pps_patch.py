#!/usr/bin/env python3
"""
Скрипт для проверки применения патча настройки длительности импульса PPS

Использование:
    sudo python3 check_pps_patch.py

Проверяет:
1. Наличие значения 5000000 (5 мс) в бинарнике shiwatime
2. Отсутствие значения 100000000 (100 мс) в бинарнике
3. Статус резервной копии
"""

import sys
import os
from struct import pack

# Путь к бинарнику
BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"
BACKUP_PATH = "/usr/share/shiwatime/bin/shiwatime.backup"

# Значения для проверки
OLD_VALUE_NS = 100000000  # 0x05F5E100 (100 мс)
NEW_VALUE_NS = 5000000    # 0x004C4B40 (5 мс)

def find_value_in_binary(filepath, value, description):
    """Ищет значение в бинарном файле и возвращает список смещений"""
    if not os.path.exists(filepath):
        return None
    
    with open(filepath, 'rb') as f:
        data = f.read()
    
    offsets = []
    
    # Little-endian (наиболее вероятно для x86/ARM)
    value_le = pack('<I', value)  # 4 байта, little-endian
    offset = 0
    while True:
        pos = data.find(value_le, offset)
        if pos == -1:
            break
        offsets.append(('little-endian', pos))
        offset = pos + 1
    
    # Big-endian (на случай, если используется)
    value_be = pack('>I', value)  # 4 байта, big-endian
    offset = 0
    while True:
        pos = data.find(value_be, offset)
        if pos == -1:
            break
        offsets.append(('big-endian', pos))
        offset = pos + 1
    
    # Также ищем как 8-байтовое значение (на случай, если используется int64)
    value_le64 = pack('<Q', value)  # 8 байт, little-endian
    offset = 0
    while True:
        pos = data.find(value_le64, offset)
        if pos == -1:
            break
        offsets.append(('int64 little-endian', pos))
        offset = pos + 1
    
    return offsets

def check_patch_status():
    """Проверяет статус патча"""
    print("=" * 80)
    print("Проверка применения патча настройки длительности импульса PPS")
    print("=" * 80)
    print()
    
    # Проверяем, что файл существует
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        print()
        print("Возможные причины:")
        print("  - shiwatime не установлен")
        print("  - Файл находится в другом месте")
        return 1
    
    file_size = os.path.getsize(BINARY_PATH)
    print(f"✓ Файл найден: {BINARY_PATH}")
    print(f"  Размер: {file_size:,} байт ({file_size / 1024 / 1024:.2f} МБ)")
    print()
    
    # Проверяем резервную копию
    backup_exists = os.path.exists(BACKUP_PATH)
    if backup_exists:
        backup_size = os.path.getsize(BACKUP_PATH)
        print(f"✓ Резервная копия найдена: {BACKUP_PATH}")
        print(f"  Размер: {backup_size:,} байт")
        if backup_size != file_size:
            print(f"  ⚠ Размеры отличаются: {abs(backup_size - file_size)} байт")
    else:
        print(f"⚠ Резервная копия не найдена: {BACKUP_PATH}")
    print()
    
    # Ищем значение 5 мс (NEW_VALUE_NS) - ожидаемое после патча
    print(f"Поиск значения {NEW_VALUE_NS:,} (5 мс) в бинарнике...")
    new_value_offsets = find_value_in_binary(BINARY_PATH, NEW_VALUE_NS, "5 мс")
    
    if new_value_offsets:
        print(f"  ✓ Найдено {len(new_value_offsets)} вхождений:")
        for fmt, offset in new_value_offsets[:5]:  # Показываем первые 5
            print(f"    - {fmt} по смещению 0x{offset:X} ({offset})")
        if len(new_value_offsets) > 5:
            print(f"    ... и еще {len(new_value_offsets) - 5} вхождений")
    else:
        print(f"  ✗ Значение {NEW_VALUE_NS:,} (5 мс) не найдено")
    print()
    
    # Ищем значение 100 мс (OLD_VALUE_NS) - не должно быть после патча
    print(f"Поиск значения {OLD_VALUE_NS:,} (100 мс) в бинарнике...")
    old_value_offsets = find_value_in_binary(BINARY_PATH, OLD_VALUE_NS, "100 мс")
    
    if old_value_offsets:
        print(f"  ⚠ Найдено {len(old_value_offsets)} вхождений:")
        for fmt, offset in old_value_offsets[:5]:  # Показываем первые 5
            print(f"    - {fmt} по смещению 0x{offset:X} ({offset})")
        if len(old_value_offsets) > 5:
            print(f"    ... и еще {len(old_value_offsets) - 5} вхождений")
        print()
        print("  ⚠ ВНИМАНИЕ: Значение 100 мс все еще присутствует в бинарнике!")
        print("     Это может означать:")
        print("     - Патч не был применен")
        print("     - Значение используется в другом контексте")
    else:
        print(f"  ✓ Значение {OLD_VALUE_NS:,} (100 мс) не найдено")
    print()
    
    # Итоговый вывод
    print("=" * 80)
    print("РЕЗУЛЬТАТ ПРОВЕРКИ:")
    print("=" * 80)
    
    patch_applied = False
    if new_value_offsets and not old_value_offsets:
        print("✓ Патч ПРИМЕНЕН")
        print()
        print("  - Значение 5 мс найдено в бинарнике")
        print("  - Значение 100 мс отсутствует")
        print()
        print("Рекомендации:")
        print("  1. Убедитесь, что shiwatime перезапущен: sudo systemctl restart shiwatime")
        print("  2. Проверьте логи: sudo journalctl -u shiwatime -f")
        print("  3. Проверьте через мониторинг UBX команд:")
        print("     sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600")
        patch_applied = True
    elif new_value_offsets and old_value_offsets:
        print("⚠ Патч ЧАСТИЧНО ПРИМЕНЕН")
        print()
        print("  - Значение 5 мс найдено в бинарнике")
        print("  - Значение 100 мс также присутствует")
        print()
        print("Возможные причины:")
        print("  - Значение 100 мс используется в другом контексте")
        print("  - Не все вхождения были заменены")
        print()
        print("Рекомендации:")
        print("  1. Проверьте мониторинг UBX команд для подтверждения:")
        print("     sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600")
        print("  2. Если импульс все еще 100 мс, патч может быть неполным")
    elif not new_value_offsets and old_value_offsets:
        print("✗ Патч НЕ ПРИМЕНЕН")
        print()
        print("  - Значение 5 мс не найдено")
        print("  - Значение 100 мс присутствует")
        print()
        print("Рекомендации:")
        print("  1. Примените патч: sudo python3 patch_shiwatime_pulse.py")
        print("  2. Перезапустите shiwatime: sudo systemctl restart shiwatime")
    else:
        print("? Статус неопределен")
        print()
        print("  - Значение 5 мс не найдено")
        print("  - Значение 100 мс также не найдено")
        print()
        print("Возможные причины:")
        print("  - Значения хранятся в другом формате")
        print("  - Значения вычисляются динамически")
        print("  - Бинарник имеет другую структуру")
        print()
        print("Рекомендации:")
        print("  1. Проверьте мониторинг UBX команд:")
        print("     sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600")
        print("  2. Проверьте логи shiwatime:")
        print("     sudo journalctl -u shiwatime -f")
    
    print("=" * 80)
    print()
    
    return 0 if patch_applied else 1

if __name__ == "__main__":
    sys.exit(check_patch_status())

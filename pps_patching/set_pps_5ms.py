#!/usr/bin/env python3
"""
Скрипт для установки длительности импульса PPS на 5 мс

Использование:
    sudo python3 set_pps_5ms.py

Заменяет значение 100 мс (100000000 нс) на 5 мс (5000000 нс) в бинарнике shiwatime
"""

import sys
import os
import shutil
from struct import pack

# Путь к бинарнику
BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"
BACKUP_PATH = "/usr/share/shiwatime/bin/shiwatime.backup"

# Значения для поиска и замены
OLD_VALUE_NS = 100000000  # 0x05F5E100 (100 мс)
NEW_VALUE_NS = 5000000    # 0x004C4B40 (5 мс)

def find_and_replace_in_binary(filepath, old_value, new_value):
    """Ищет и заменяет значение в бинарном файле"""
    
    # Читаем файл
    with open(filepath, 'rb') as f:
        data = bytearray(f.read())
    
    original_size = len(data)
    replacements = 0
    
    # Ищем значение в разных форматах (little-endian и big-endian)
    # Little-endian (наиболее вероятно для x86/ARM)
    old_le = pack('<I', old_value)  # 4 байта, little-endian
    new_le = pack('<I', new_value)
    
    # Big-endian (на случай, если используется)
    old_be = pack('>I', old_value)  # 4 байта, big-endian
    new_be = pack('>I', new_value)
    
    # Поиск и замена (little-endian)
    offset = 0
    while True:
        pos = data.find(old_le, offset)
        if pos == -1:
            break
        print(f"  Найдено значение {old_value:,} (little-endian) по смещению 0x{pos:X}")
        data[pos:pos+4] = new_le
        replacements += 1
        offset = pos + 1
    
    # Поиск и замена (big-endian)
    offset = 0
    while True:
        pos = data.find(old_be, offset)
        if pos == -1:
            break
        print(f"  Найдено значение {old_value:,} (big-endian) по смещению 0x{pos:X}")
        data[pos:pos+4] = new_be
        replacements += 1
        offset = pos + 1
    
    # Также ищем как 8-байтовое значение (на случай, если используется int64)
    old_le64 = pack('<Q', old_value)  # 8 байт, little-endian
    new_le64 = pack('<Q', new_value)
    
    offset = 0
    while True:
        pos = data.find(old_le64, offset)
        if pos == -1:
            break
        print(f"  Найдено значение {old_value:,} (int64, little-endian) по смещению 0x{pos:X}")
        data[pos:pos+8] = new_le64
        replacements += 1
        offset = pos + 1
    
    if replacements == 0:
        print(f"⚠ Значение {old_value:,} не найдено в бинарнике")
        # Проверяем, может быть уже установлено новое значение
        new_le_check = pack('<I', new_value)
        if data.find(new_le_check) != -1:
            print(f"✓ Значение {new_value:,} (5 мс) уже присутствует в бинарнике")
            print("  Изменения не требуются")
            return True
        return False
    
    # Записываем измененный файл
    with open(filepath, 'wb') as f:
        f.write(data)
    
    print(f"✓ Заменено {replacements} вхождений")
    print(f"✓ Размер файла: {original_size:,} байт (не изменился)")
    return True

def main():
    print("=" * 80)
    print("Установка длительности импульса PPS на 5 мс")
    print("=" * 80)
    print()
    
    # Проверяем, что файл существует
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    # Проверяем права доступа
    if os.geteuid() != 0:
        print("✗ Ошибка: требуется права root (sudo)")
        return 1
    
    # Создаем резервную копию
    if not os.path.exists(BACKUP_PATH):
        print(f"Создание резервной копии: {BACKUP_PATH}")
        shutil.copy2(BINARY_PATH, BACKUP_PATH)
        print("✓ Резервная копия создана")
    else:
        print(f"⚠ Резервная копия уже существует: {BACKUP_PATH}")
        print("  (Используется существующая резервная копия)")
    print()
    
    print(f"Поиск значения {OLD_VALUE_NS:,} (100 мс) в бинарнике...")
    print()
    
    # Выполняем поиск и замену
    if find_and_replace_in_binary(BINARY_PATH, OLD_VALUE_NS, NEW_VALUE_NS):
        print()
        print("=" * 80)
        print("✓ Установка длительности импульса на 5 мс выполнена успешно!")
        print("=" * 80)
        print()
        print("Следующие шаги:")
        print("1. Перезапустите shiwatime: sudo systemctl restart shiwatime")
        print("2. Проверьте логи: sudo journalctl -u shiwatime -f")
        print("3. Проверьте применение патча:")
        print("   sudo python3 check_pps_patch.py")
        print("4. Проверьте через мониторинг UBX команд:")
        print("   sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600")
        print()
        return 0
    else:
        print()
        print("=" * 80)
        print("⚠ Изменения не были внесены. Возможные причины:")
        print("  - Значение 100 мс не найдено в бинарнике")
        print("  - Значение 5 мс уже установлено")
        print("  - Значение закодировано в другом формате")
        print("=" * 80)
        return 1

if __name__ == "__main__":
    sys.exit(main())

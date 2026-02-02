#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Извлечение коэффициентов алгоритмов из бинарника (версия 2)
Использует readelf для определения секций и прямое чтение файла для извлечения данных
"""

import subprocess
import struct
import sys
import os

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

def run_command(cmd, shell=False):
    """Выполняет команду и возвращает результат"""
    try:
        if shell:
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True, check=False)
        else:
            result = subprocess.run(cmd, capture_output=True, text=True, check=False)
        return result.stdout if result.returncode == 0 else ""
    except Exception as e:
        return f"Ошибка: {str(e)}"

def find_section_for_address(addr):
    """Находит секцию, содержащую указанный адрес"""
    print(f"Поиск секции для адреса 0x{addr:x}...")
    
    # Получаем информацию о секциях
    result = run_command(["readelf", "-S", BINARY_PATH])
    
    sections = []
    for line in result.split('\n'):
        # Ищем строки с секциями (формат: [NN] name TYPE ADDR OFFSET SIZE ...)
        # Нужно искать и PROGBITS и NOBITS секции
        if '[' in line and ']' in line:
            parts = line.split()
            # Проверяем, что это секция с данными (PROGBITS или NOBITS)
            # Также ищем .noptrdata, .data, .rodata и другие секции
            if len(parts) >= 7:
                try:
                    section_name = parts[1].strip('[]')
                    # Пропускаем заголовок таблицы
                    if section_name == 'Nr' or section_name == 'Name':
                        continue
                    section_addr = int(parts[4], 16)
                    section_offset = int(parts[5], 16)
                    section_size = int(parts[6], 16)
                    
                    sections.append({
                        'name': section_name,
                        'addr': section_addr,
                        'size': section_size,
                        'offset': section_offset
                    })
                except (ValueError, IndexError) as e:
                    continue
    
    # Ищем секцию, содержащую адрес
    for section in sections:
        if section['size'] > 0 and section['addr'] <= addr < section['addr'] + section['size']:
            file_offset = section['offset'] + (addr - section['addr'])
            print(f"✅ Найдено в секции {section['name']}:")
            print(f"   Адрес секции: 0x{section['addr']:x}")
            print(f"   Размер секции: 0x{section['size']:x}")
            print(f"   Offset секции: 0x{section['offset']:x}")
            print(f"   Смещение в файле: 0x{file_offset:x}")
            return section, file_offset
    
    print("⚠ Адрес не найден ни в одной секции")
    print(f"Всего найдено секций: {len(sections)}")
    print("Проверяем все секции с данными:")
    for section in sections:
        if section['size'] > 0:
            end_addr = section['addr'] + section['size']
            print(f"  {section['name']:20s}: 0x{section['addr']:016x} - 0x{end_addr:016x} (size: 0x{section['size']:x}, offset: 0x{section['offset']:x})")
            # Проверяем, находится ли адрес в диапазоне этой секции
            if section['addr'] <= addr < end_addr:
                print(f"    ✅ Адрес 0x{addr:x} находится в этой секции!")
                file_offset = section['offset'] + (addr - section['addr'])
                print(f"    ✅ Вычисленное смещение в файле: 0x{file_offset:x}")
                return section, file_offset
            # Проверяем, близок ли адрес к этой секции
            elif abs(addr - section['addr']) < 0x1000000:  # В пределах 16MB
                print(f"    ⚠ Адрес 0x{addr:x} близок к этой секции (разница: 0x{abs(addr - section['addr']):x})")
                if section['addr'] < addr:
                    print(f"    💡 Адрес больше начала секции, возможно в диапазоне...")
                    # Проверяем еще раз с учетом возможной ошибки округления
                    if addr < section['addr'] + section['size'] + 0x1000:  # Небольшой запас
                        file_offset = section['offset'] + (addr - section['addr'])
                        print(f"    ✅ Вычисленное смещение в файле: 0x{file_offset:x}")
                        return section, file_offset
    return None, None

def extract_d_coefficients():
    """Извлекает массив D-коэффициентов"""
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ МАССИВА D-КОЭФФИЦИЕНТОВ (0x770a430)")
    print("=" * 80)
    print()
    
    addr = 0x770a430
    size = 3 * 8  # 3 элемента float64 (8 байт каждый)
    
    section, file_offset = find_section_for_address(addr)
    
    # Альтернативный метод через objdump, если парсинг секций не сработал
    if not file_offset:
        print("Попытка извлечения через objdump...")
        result = run_command(["objdump", "-s", "--start-address", f"0x{addr:x}", "--stop-address", f"0x{addr+size:x}", BINARY_PATH])
        
        if result and f"{addr:x}" in result.lower():
            print("✅ Данные найдены через objdump!")
            
            # Парсим hex данные из вывода objdump
            lines = result.split('\n')
            hex_data = []
            for line in lines:
                # Ищем строки с адресом (формат: 770a430 00000000 0000e03f ...)
                if f"{addr:x}" in line.lower() and len(line.split()) > 1:
                    parts = line.split()
                    # Пропускаем адрес, берем hex данные
                    for part in parts[1:]:
                        # Проверяем, что это hex (8 символов)
                        if len(part) == 8:
                            try:
                                # Проверяем, что это hex
                                int(part, 16)
                                hex_data.append(part)
                            except ValueError:
                                continue
            
            if len(hex_data) >= 6:  # Нужно 6 групп по 4 байта = 24 байта
                # Собираем байты (little-endian: младшие байты сначала)
                data_bytes = bytearray()
                for i in range(0, len(hex_data), 2):
                    # Каждые 2 группы по 4 байта = 8 байт (float64)
                    if i+1 < len(hex_data):
                        # Little-endian: первая группа - младшие 4 байта
                        bytes1 = bytes.fromhex(hex_data[i])
                        bytes2 = bytes.fromhex(hex_data[i+1])
                        data_bytes.extend(bytes1)
                        data_bytes.extend(bytes2)
                
                if len(data_bytes) >= size:
                    print(f"\n✅ Данные успешно извлечены ({len(data_bytes)} байт):")
                    print(f"Hex dump:")
                    for i in range(0, min(len(data_bytes), size), 16):
                        hex_str = ' '.join(f'{b:02x}' for b in data_bytes[i:i+16])
                        ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in data_bytes[i:i+16])
                        print(f"  {addr+i:08x}: {hex_str:<48} {ascii_str}")
                    
                    # Парсим float64 значения (little-endian для ARM)
                    print("\n📊 Интерпретация как float64 (little-endian):")
                    for i in range(3):
                        value = struct.unpack('<d', data_bytes[i*8:(i+1)*8])[0]
                        print(f"  D[{i}] = {value:.15e} = {value}")
                    
                    print("\n✅ РЕЗУЛЬТАТ:")
                    print(f"  Все три D-коэффициента равны: {struct.unpack('<d', data_bytes[0:8])[0]}")
                    return
            else:
                print(f"⚠ Недостаточно данных: найдено {len(hex_data)} групп, нужно 6")
    
    if file_offset:
        print(f"Извлечение {size} байт из файла...")
        print(f"Смещение в файле: 0x{file_offset:x}")
        print()
        
        # Прямое чтение из файла
        try:
            with open(BINARY_PATH, 'rb') as f:
                f.seek(file_offset)
                data = f.read(size)
                
                if len(data) == size:
                    print("✅ Данные успешно извлечены:")
                    print(f"Hex dump ({len(data)} байт):")
                    for i in range(0, len(data), 16):
                        hex_str = ' '.join(f'{b:02x}' for b in data[i:i+16])
                        ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in data[i:i+16])
                        print(f"  {file_offset+i:08x}: {hex_str:<48} {ascii_str}")
                    
                    # Парсим float64 значения (little-endian для ARM)
                    print("\n📊 Интерпретация как float64 (little-endian):")
                    for i in range(3):
                        value = struct.unpack('<d', data[i*8:(i+1)*8])[0]
                        print(f"  D[{i}] = {value:.15e} ({value})")
                    
                    # Также пробуем big-endian на случай
                    print("\n📊 Интерпретация как float64 (big-endian):")
                    for i in range(3):
                        value = struct.unpack('>d', data[i*8:(i+1)*8])[0]
                        print(f"  D[{i}] = {value:.15e} ({value})")
                else:
                    print(f"⚠ Прочитано только {len(data)} байт из {size}")
        except Exception as e:
            print(f"❌ Ошибка при чтении файла: {e}")
    else:
        print("⚠ Не удалось определить смещение в файле")
    print()

def extract_default_coefficients():
    """Извлекает DefaultAlgoCoefficients"""
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ DefaultAlgoCoefficients (0x770b7e0)")
    print("=" * 80)
    print()
    
    addr = 0x770b7e0
    
    section, file_offset = find_section_for_address(addr)
    
    # Альтернативный метод через objdump
    if not file_offset:
        print("Попытка извлечения через objdump...")
        result = run_command(["objdump", "-s", "--start-address", f"0x{addr:x}", "--stop-address", f"0x{addr+256:x}", BINARY_PATH])
        
        if result and f"{addr:x}" in result.lower():
            print("✅ Данные найдены через objdump!")
            # Показываем первые строки
            lines = result.split('\n')
            for line in lines[:20]:
                if line.strip():
                    print(line)
            print("\n💡 Для полного анализа используйте gdb или извлеките данные вручную")
            return
    
    if file_offset:
        print(f"Смещение в файле: 0x{file_offset:x}")
        print("Размер структуры неизвестен, извлекаем первые 256 байт...")
        print()
        
        size = 256
        try:
            with open(BINARY_PATH, 'rb') as f:
                f.seek(file_offset)
                data = f.read(size)
                
                if data:
                    print("✅ Данные успешно извлечены:")
                    print(f"Hex dump (первые {len(data)} байт):")
                    for i in range(0, min(len(data), 128), 16):  # Первые 128 байт
                        hex_str = ' '.join(f'{b:02x}' for b in data[i:i+16])
                        ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in data[i:i+16])
                        print(f"  {file_offset+i:08x}: {hex_str:<48} {ascii_str}")
                    
                    # Пытаемся интерпретировать как структуру коэффициентов
                    # Предполагаем, что это может быть массив структур с Kp, Ki, Kd
                    print("\n📊 Попытка интерпретации как коэффициенты (float64):")
                    if len(data) >= 24:  # Хотя бы 3 float64
                        for i in range(0, min(len(data), 24), 8):
                            try:
                                value_le = struct.unpack('<d', data[i:i+8])[0]
                                value_be = struct.unpack('>d', data[i:i+8])[0]
                                print(f"  offset {i:02d}: LE={value_le:.15e}, BE={value_be:.15e}")
                            except:
                                pass
        except Exception as e:
            print(f"❌ Ошибка при чтении файла: {e}")
    else:
        print("⚠ Не удалось определить смещение в файле")
    print()

def analyze_coefficient_structure():
    """Анализирует структуру коэффициентов из ассемблера"""
    print("=" * 80)
    print("АНАЛИЗ СТРУКТУРЫ КОЭФФИЦИЕНТОВ ИЗ АССЕМБЛЕРА")
    print("=" * 80)
    print()
    
    print("Из CalculateNewFrequency (0x41c87c0):")
    print("  [x0, #40] -> указатель на структуру коэффициентов")
    print("  [x2, #0]  -> первый коэффициент (Kp?)")
    print("  [x2, #8]  -> второй коэффициент (Ki?)")
    print("  [x2, #16] -> третий коэффициент (Kd?)")
    print()
    
    print("Из adjustDComponent (0x41c8bc0):")
    print("  Массив D-коэффициентов: 0x770a430")
    print("  Размер: 3 элемента (float64)")
    print("  Использование: выбор по индексу от log(abs(value))")
    print()
    
    print("Из enforceAdjustmentLimit (0x41c8cc0):")
    print("  [x0, #96]  -> максимальный лимит коррекции")
    print("  [x0, #112] -> минимальный лимит коррекции")
    print("  [x0, #40] -> [x3, #32] -> дефолтный лимит")
    print()

def try_alternative_extraction():
    """Пробует альтернативные способы извлечения"""
    print("=" * 80)
    print("АЛЬТЕРНАТИВНЫЕ СПОСОБЫ ИЗВЛЕЧЕНИЯ")
    print("=" * 80)
    print()
    
    print("1. Поиск через strings (может найти строковые представления):")
    result = run_command(f"strings {BINARY_PATH} | grep -E '^[0-9]+\\.[0-9]+$' | head -20", shell=True)
    if result:
        print(result)
    else:
        print("  Не найдено")
    print()
    
    print("2. Поиск констант в .rodata секции:")
    result = run_command(["objdump", "-s", "-j", ".rodata", BINARY_PATH])
    # Ищем адреса в выводе
    lines = result.split('\n')
    found = False
    for i, line in enumerate(lines):
        if '770a430' in line.lower() or '770b7e0' in line.lower():
            print(f"✅ Найдено в строке {i+1}:")
            print(line)
            for j in range(min(5, len(lines) - i - 1)):
                print(lines[i + j + 1])
            found = True
            break
    if not found:
        print("  Адреса не найдены в .rodata")
    print()
    
    print("3. Поиск всех секций, содержащих данные:")
    result = run_command(["readelf", "-S", BINARY_PATH])
    for line in result.split('\n'):
        if any(name in line for name in ['.data', '.rodata', '.bss', '.data.rel.ro']):
            print(f"  {line}")
    print()

def main():
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ КОЭФФИЦИЕНТОВ АЛГОРИТМОВ (версия 2)")
    print("=" * 80)
    print()
    
    # Проверяем существование файла
    if not os.path.exists(BINARY_PATH):
        print(f"❌ Файл не найден: {BINARY_PATH}")
        print("Убедитесь, что скрипт запущен на устройстве с установленным shiwatime")
        return 1
    
    # 1. Анализ структуры
    analyze_coefficient_structure()
    
    # 2. Извлечение D-коэффициентов
    extract_d_coefficients()
    
    # 3. Извлечение DefaultAlgoCoefficients
    extract_default_coefficients()
    
    # 4. Альтернативные способы
    try_alternative_extraction()
    
    print("=" * 80)
    print("РЕКОМЕНДАЦИИ")
    print("=" * 80)
    print()
    print("Если данные не найдены:")
    print("1. Адреса могут быть виртуальными (VMA) - нужно найти смещение в файле")
    print("2. Данные могут быть в .rodata (read-only data)")
    print("3. Использовать gdb для чтения из памяти во время выполнения:")
    print("   gdb /usr/share/shiwatime/bin/shiwatime")
    print("   (gdb) x/3g 0x770a430  # для D-коэффициентов")
    print("   (gdb) x/32g 0x770b7e0  # для DefaultAlgoCoefficients")
    print("4. Проверить конфигурационные файлы (YAML)")
    print("5. Проанализировать логи работы программы")
    print()
    print("Альтернативный способ через objdump:")
    print("  objdump -s --start-address=0x770a430 --stop-address=0x770a448 /usr/share/shiwatime/bin/shiwatime")
    print()
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

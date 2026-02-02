#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Извлечение коэффициентов алгоритмов из бинарника
"""

import subprocess
import struct
import re
import sys

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

def extract_default_coefficients():
    """Извлекает DefaultAlgoCoefficients по адресу 0x770b7e0"""
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ DefaultAlgoCoefficients")
    print("=" * 80)
    print()
    
    # Адрес данных
    data_addr = 0x770b7e0
    
    # Попробуем найти через strings или hexdump
    print(f"Поиск данных по адресу 0x{data_addr:x}...")
    
    # Через objdump -s
    result = run_command(["objdump", "-s", "-j", ".data", BINARY_PATH])
    
    # Ищем адрес в выводе
    lines = result.split('\n')
    found = False
    for i, line in enumerate(lines):
        if f"{data_addr:08x}" in line.lower():
            print(f"✅ Найдено в строке {i+1}:")
            print(line)
            # Показываем следующие строки
            for j in range(min(5, len(lines) - i - 1)):
                print(lines[i + j + 1])
            found = True
            break
    
    if not found:
        print("⚠ Адрес не найден в .data секции")
        print("Попробуем через hexdump...")
        
        # Через hexdump (требует расчета смещения в файле)
        # Нужно найти смещение секции .data
        result = run_command(["readelf", "-S", BINARY_PATH])
        data_section = None
        for line in result.split('\n'):
            if '.data' in line:
                parts = line.split()
                if len(parts) >= 5:
                    try:
                        offset = int(parts[5], 16)
                        addr = int(parts[4], 16)
                        data_section = {'offset': offset, 'addr': addr}
                        print(f"✅ Секция .data: offset=0x{offset:x}, addr=0x{addr:x}")
                        break
                    except:
                        pass
        
        if data_section:
            file_offset = data_section['offset'] + (data_addr - data_section['addr'])
            print(f"Смещение в файле: 0x{file_offset:x}")
    
    print()

def extract_d_coefficients_array():
    """Извлекает массив D-коэффициентов по адресу 0x770a430"""
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ МАССИВА D-КОЭФФИЦИЕНТОВ")
    print("=" * 80)
    print()
    
    # Адрес массива
    array_addr = 0x770a430
    
    print(f"Поиск массива по адресу 0x{array_addr:x}...")
    print("Размер массива: 3 элемента (float64, 24 байта)")
    print()
    
    # Аналогично DefaultAlgoCoefficients
    result = run_command(["objdump", "-s", "-j", ".data", BINARY_PATH])
    
    lines = result.split('\n')
    found = False
    for i, line in enumerate(lines):
        if f"{array_addr:08x}" in line.lower():
            print(f"✅ Найдено в строке {i+1}:")
            print(line)
            for j in range(min(3, len(lines) - i - 1)):
                print(lines[i + j + 1])
            found = True
            break
    
    if not found:
        print("⚠ Адрес не найден")
    
    print()

def analyze_get_coefficients():
    """Анализирует функцию GetCoefficients"""
    print("=" * 80)
    print("АНАЛИЗ GetCoefficients")
    print("=" * 80)
    print()
    
    func_addr = "41c6520"
    
    print(f"Дизассемблирование функции 0x{func_addr}...")
    result = run_command(["objdump", "-d", BINARY_PATH])
    
    # Ищем функцию
    lines = result.split('\n')
    in_function = False
    func_lines = []
    
    for line in lines:
        if func_addr in line and "GetCoefficients" in line:
            in_function = True
            func_lines.append(line)
        elif in_function:
            if line.strip() and not line.strip().startswith('...'):
                func_lines.append(line)
            else:
                break
    
    if func_lines:
        print("✅ Функция найдена:")
        for line in func_lines[:50]:  # Первые 50 строк
            print(line)
    else:
        print("⚠ Функция не найдена")
    
    print()

def find_coefficient_constants():
    """Ищет константы коэффициентов в ассемблере"""
    print("=" * 80)
    print("ПОИСК КОНСТАНТ КОЭФФИЦИЕНТОВ")
    print("=" * 80)
    print()
    
    # Ищем загрузки констант с плавающей точкой
    print("Поиск констант с плавающей точкой...")
    
    result = run_command(["objdump", "-d", BINARY_PATH])
    
    # Ищем паттерны загрузки констант
    patterns = [
        r"ldr\s+d\d+,\s+\[x\d+,\s+#(\d+)\]",  # Загрузка из памяти
        r"fmov\s+d\d+,\s+#([0-9.]+)",  # Прямая константа
    ]
    
    found_constants = []
    
    for line in result.split('\n'):
        # Ищем в функциях алгоритмов
        if any(func in line for func in ["CalculateNewFrequency", "pi_sample", "regress"]):
            # Следующие строки могут содержать константы
            pass
        
        # Ищем загрузки из памяти по известным адресам
        if "504b000" in line or "770a000" in line:
            found_constants.append(line)
    
    if found_constants:
        print(f"✅ Найдено {len(found_constants)} потенциальных констант:")
        for const in found_constants[:20]:
            print(const)
    else:
        print("⚠ Константы не найдены автоматически")
    
    print()

def extract_from_calculate_new_frequency():
    """Извлекает коэффициенты из CalculateNewFrequency"""
    print("=" * 80)
    print("АНАЛИЗ CalculateNewFrequency ДЛЯ КОЭФФИЦИЕНТОВ")
    print("=" * 80)
    print()
    
    func_addr = "41c87c0"
    
    print("Известные смещения в структуре:")
    print("  [x0, #40] -> [x2, #0]  - Kp (предположительно)")
    print("  [x0, #40] -> [x2, #8]  - Ki (предположительно)")
    print("  [x0, #40] -> [x2, #16] - Kd (предположительно)")
    print("  [x0, #48] -> [x3, #40] - значение для D")
    print()
    
    print("Массив D-коэффициентов:")
    print("  Адрес: 0x770a430")
    print("  Размер: 3 элемента (float64)")
    print("  Использование: выбор по индексу от log(abs(value))")
    print()
    
    print("Константы из памяти (504b000):")
    print("  offset 664  - константа для D компоненты")
    print("  offset 1904 - минимальное значение")
    print("  offset 2360 - максимальное значение")
    print()

def main():
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ КОЭФФИЦИЕНТОВ АЛГОРИТМОВ")
    print("=" * 80)
    print()
    
    # 1. DefaultAlgoCoefficients
    extract_default_coefficients()
    
    # 2. Массив D-коэффициентов
    extract_d_coefficients_array()
    
    # 3. GetCoefficients
    analyze_get_coefficients()
    
    # 4. Константы
    find_coefficient_constants()
    
    # 5. Анализ CalculateNewFrequency
    extract_from_calculate_new_frequency()
    
    print("=" * 80)
    print("РЕКОМЕНДАЦИИ")
    print("=" * 80)
    print()
    print("Для извлечения значений коэффициентов:")
    print("1. Используйте hexdump для чтения данных по адресам")
    print("2. Проанализируйте GetCoefficients для понимания структуры")
    print("3. Проверьте конфигурационные файлы на наличие коэффициентов")
    print()
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

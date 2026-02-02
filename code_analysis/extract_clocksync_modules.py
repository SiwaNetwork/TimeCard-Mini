#!/usr/bin/env python3
"""
Скрипт для извлечения и анализа модулей clocksync из бинарника shiwatime
- Поиск функций из clocksync модулей
- Извлечение строк и структур данных
- Анализ вызовов функций
- Понимание логики работы

Использование:
    sudo python3 extract_clocksync_modules.py
"""

import os
import sys
import subprocess
import re
from struct import pack, unpack

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

# Модули для анализа
TARGET_MODULES = [
    "clocksync",
    "generic_gnss_device",
    "ubx",
    "ptp",
    "ntp",
    "servo"
]

def run_command(cmd, shell=False):
    """Выполняет команду и возвращает результат"""
    try:
        result = subprocess.run(
            cmd, 
            shell=shell, 
            capture_output=True, 
            check=True
        )
        return result.stdout.decode('utf-8', errors='replace')
    except subprocess.CalledProcessError as e:
        try:
            error_msg = e.stderr.decode('utf-8', errors='replace') if isinstance(e.stderr, bytes) else e.stderr
        except:
            error_msg = str(e.stderr)
        return f"Ошибка: {error_msg}"
    except FileNotFoundError:
        return f"Команда не найдена: {cmd[0] if isinstance(cmd, list) else cmd}"
    except Exception as e:
        return f"Неожиданная ошибка: {str(e)}"

def extract_module_functions(module_name):
    """Извлекает все функции из указанного модуля"""
    print("=" * 80)
    print(f"МОДУЛЬ: {module_name}")
    print("=" * 80)
    
    # Поиск функций через objdump
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        functions = []
        
        # Паттерны для поиска
        patterns = [
            f"clocksync.*{module_name}",
            f"{module_name}.*clocksync",
            f".*{module_name}.*",
        ]
        
        for line in lines:
            line_lower = line.lower()
            for pattern in patterns:
                if re.search(pattern, line_lower, re.IGNORECASE):
                    functions.append(line)
                    break
        
        if functions:
            print(f"\nНайдено {len(functions)} функций:")
            for func in functions[:100]:  # Первые 100
                print(f"  {func}")
            if len(functions) > 100:
                print(f"\n... (показано 100 из {len(functions)})")
        else:
            print("Функции не найдены через objdump")
    
    print()

def extract_module_strings(module_name):
    """Извлекает строки, связанные с модулем"""
    print("=" * 80)
    print(f"СТРОКИ МОДУЛЯ: {module_name}")
    print("=" * 80)
    
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        found = []
        
        for line in lines:
            line_lower = line.lower()
            # Ищем строки, содержащие название модуля
            if module_name.lower() in line_lower:
                found.append(line)
        
        if found:
            print(f"\nНайдено {len(found)} строк:")
            for line in sorted(set(found))[:50]:
                print(f"  {line}")
        else:
            print("Строки не найдены")
    
    print()

def analyze_function_calls(module_name):
    """Анализирует вызовы функций модуля"""
    print("=" * 80)
    print(f"ВЫЗОВЫ ФУНКЦИЙ МОДУЛЯ: {module_name}")
    print("=" * 80)
    
    # Получаем дизассемблирование
    result = run_command(["objdump", "-d", "-C", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        # Ищем функции модуля и их вызовы
        current_function = None
        calls = []
        
        for line in lines:
            # Определяем начало функции
            if '>:' in line and '<' in line:
                func_match = re.search(r'<([^>]+)>:', line)
                if func_match:
                    func_name = func_match.group(1)
                    if module_name.lower() in func_name.lower():
                        current_function = func_name
            
            # Ищем вызовы функций
            if current_function and ('call' in line.lower() or ' bl ' in line.lower()):
                call_match = re.search(r'<([^>]+)>', line)
                if call_match:
                    called_func = call_match.group(1)
                    calls.append((current_function, called_func))
        
        if calls:
            print(f"\nНайдено {len(calls)} вызовов:")
            # Группируем по вызывающим функциям
            call_dict = {}
            for caller, callee in calls[:50]:
                if caller not in call_dict:
                    call_dict[caller] = []
                if callee not in call_dict[caller]:
                    call_dict[caller].append(callee)
            
            for caller, callees in list(call_dict.items())[:10]:
                print(f"\n  {caller} ->")
                for callee in callees[:5]:
                    print(f"    - {callee}")
        else:
            print("Вызовы не найдены")
    
    print()

def extract_ubx_structures():
    """Извлекает информацию о UBX структурах"""
    print("=" * 80)
    print("UBX СТРУКТУРЫ И КОНСТАНТЫ")
    print("=" * 80)
    
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        ubx_keywords = [
            "UBX", "ubx",
            "TP5", "CFG-TP5", "CFG_TP5",
            "timepulse", "time pulse",
            "pulse", "pps",
            "ublox", "gnss"
        ]
        
        found = []
        for line in lines:
            line_lower = line.lower()
            for keyword in ubx_keywords:
                if keyword in line_lower:
                    found.append(line)
                    break
        
        if found:
            print(f"\nНайдено {len(found)} релевантных строк:")
            for line in sorted(set(found))[:100]:
                print(f"  {line}")
    
    # Поиск UBX констант в бинарнике
    if os.path.exists(BINARY_PATH):
        with open(BINARY_PATH, 'rb') as f:
            data = f.read()
        
        # UBX sync bytes
        ubx_sync = b'\xb5\x62'
        positions = []
        offset = 0
        while True:
            pos = data.find(ubx_sync, offset)
            if pos == -1:
                break
            positions.append(pos)
            offset = pos + 1
        
        if positions:
            print(f"\n\nНайдено {len(positions)} UBX sync паттернов (0xB562)")
            print("Первые 10 позиций:")
            for pos in positions[:10]:
                # Показываем контекст (32 байта после sync)
                if pos + 34 < len(data):
                    context = data[pos:pos+34]
                    hex_context = ' '.join(f'{b:02x}' for b in context)
                    print(f"  Смещение 0x{pos:X}: {hex_context}")
    
    print()

def analyze_gnss_device_functions():
    """Анализирует функции работы с GNSS устройствами"""
    print("=" * 80)
    print("GNSS DEVICE ФУНКЦИИ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        gnss_keywords = [
            "gnss", "gps", "ublox",
            "detect", "send", "receive",
            "timepulse", "pps",
            "nmea", "ubx"
        ]
        
        functions = []
        for line in lines:
            line_lower = line.lower()
            if any(keyword in line_lower for keyword in gnss_keywords):
                # Проверяем, что это функция из нужного модуля
                if "generic_gnss_device" in line_lower or "gnss" in line_lower:
                    functions.append(line)
        
        if functions:
            print(f"\nНайдено {len(functions)} функций:")
            for func in functions[:50]:
                print(f"  {func}")
        else:
            print("Функции не найдены")
    
    print()

def extract_servo_algorithms():
    """Пытается извлечь информацию о servo алгоритмах"""
    print("=" * 80)
    print("SERVO АЛГОРИТМЫ")
    print("=" * 80)
    
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        servo_keywords = [
            "servo", "dpll", "phase",
            "lock", "sync", "offset",
            "frequency", "adjust",
            "coefficient", "algorithm"
        ]
        
        found = []
        for line in lines:
            line_lower = line.lower()
            if any(keyword in line_lower for keyword in servo_keywords):
                if "servo" in line_lower or "dpll" in line_lower:
                    found.append(line)
        
        if found:
            print(f"\nНайдено {len(found)} релевантных строк:")
            for line in sorted(set(found))[:50]:
                print(f"  {line}")
        else:
            print("Строки не найдены")
    
    print()

def analyze_ptp_ntp_clients():
    """Анализирует PTP и NTP клиенты"""
    print("=" * 80)
    print("PTP/NTP КЛИЕНТЫ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        ptp_ntp_functions = []
        for line in lines:
            line_lower = line.lower()
            if ("ptp" in line_lower or "ntp" in line_lower) and "client" in line_lower:
                ptp_ntp_functions.append(line)
        
        if ptp_ntp_functions:
            print(f"\nНайдено {len(ptp_ntp_functions)} функций:")
            for func in ptp_ntp_functions[:50]:
                print(f"  {func}")
        else:
            print("Функции не найдены")
    
    print()

def create_function_map():
    """Создает карту функций всех модулей"""
    print("=" * 80)
    print("КАРТА ФУНКЦИЙ ВСЕХ МОДУЛЕЙ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        module_map = {module: [] for module in TARGET_MODULES}
        
        for line in lines:
            for module in TARGET_MODULES:
                if module.lower() in line.lower() and "clocksync" in line.lower():
                    module_map[module].append(line)
        
        for module, functions in module_map.items():
            if functions:
                print(f"\n{module.upper()}: {len(functions)} функций")
                for func in functions[:10]:
                    # Извлекаем имя функции
                    match = re.search(r'<([^>]+)>', func)
                    if match:
                        print(f"  - {match.group(1)}")
                if len(functions) > 10:
                    print(f"  ... (еще {len(functions) - 10})")
    
    print()

def main():
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ И АНАЛИЗ МОДУЛЕЙ CLOCKSYNC")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # Общая карта функций
    create_function_map()
    
    # Анализ каждого модуля
    for module in TARGET_MODULES:
        extract_module_functions(module)
        extract_module_strings(module)
        analyze_function_calls(module)
    
    # Специальный анализ
    extract_ubx_structures()
    analyze_gnss_device_functions()
    extract_servo_algorithms()
    analyze_ptp_ntp_clients()
    
    print("=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    print()
    print("Для сохранения результатов:")
    print("  sudo python3 extract_clocksync_modules.py > clocksync_modules_analysis.txt 2>&1")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

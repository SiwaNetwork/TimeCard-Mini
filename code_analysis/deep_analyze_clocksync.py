#!/usr/bin/env python3
"""
Глубокий анализ модулей clocksync - попытка понять логику работы
- Извлечение структур данных
- Анализ параметров функций
- Понимание алгоритмов через дизассемблирование
- Построение графа зависимостей

Использование:
    sudo python3 deep_analyze_clocksync.py
"""

import os
import sys
import subprocess
import re
from collections import defaultdict
from struct import pack, unpack

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

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
    except Exception as e:
        return f"Ошибка: {str(e)}"

def extract_function_signatures():
    """Извлекает сигнатуры функций из модулей clocksync"""
    print("=" * 80)
    print("СИГНАТУРЫ ФУНКЦИЙ CLOCKSYNC МОДУЛЕЙ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        clocksync_functions = defaultdict(list)
        
        for line in lines:
            if "clocksync" in line.lower():
                # Извлекаем имя функции
                match = re.search(r'<([^>]+)>', line)
                if match:
                    func_name = match.group(1)
                    # Определяем модуль
                    for module in ["ubx", "gnss", "ptp", "ntp", "servo", "generic_gnss"]:
                        if module in func_name.lower():
                            clocksync_functions[module].append((func_name, line))
                            break
        
        for module, functions in clocksync_functions.items():
            print(f"\n{module.upper()}: {len(functions)} функций")
            for func_name, line in functions[:20]:
                print(f"  {func_name}")
    
    print()

def analyze_ubx_message_handling():
    """Анализирует обработку UBX сообщений"""
    print("=" * 80)
    print("АНАЛИЗ ОБРАБОТКИ UBX СООБЩЕНИЙ")
    print("=" * 80)
    
    # Поиск функций, связанных с UBX
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        ubx_functions = []
        for line in lines:
            line_lower = line.lower()
            if "ubx" in line_lower and ("send" in line_lower or "receive" in line_lower or "parse" in line_lower):
                match = re.search(r'<([^>]+)>', line)
                if match:
                    ubx_functions.append(match.group(1))
        
        if ubx_functions:
            print(f"\nНайдено {len(ubx_functions)} функций обработки UBX:")
            for func in ubx_functions[:30]:
                print(f"  - {func}")
    
    # Поиск строк, связанных с UBX
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        ubx_strings = []
        for line in lines:
            if "ubx" in line.lower() and any(keyword in line.lower() for keyword in ["message", "packet", "header", "payload"]):
                ubx_strings.append(line)
        
        if ubx_strings:
            print(f"\n\nНайдено {len(ubx_strings)} строк, связанных с UBX:")
            for line in sorted(set(ubx_strings))[:30]:
                print(f"  {line}")
    
    print()

def analyze_timepulse_configuration():
    """Анализирует конфигурацию timepulse"""
    print("=" * 80)
    print("АНАЛИЗ КОНФИГУРАЦИИ TIMEPULSE")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        tp_functions = []
        for line in lines:
            line_lower = line.lower()
            if "timepulse" in line_lower or "tp5" in line_lower:
                match = re.search(r'<([^>]+)>', line)
                if match:
                    tp_functions.append(match.group(1))
        
        if tp_functions:
            print(f"\nНайдено {len(tp_functions)} функций timepulse:")
            for func in tp_functions[:30]:
                print(f"  - {func}")
    
    # Поиск значений pulse width
    if os.path.exists(BINARY_PATH):
        with open(BINARY_PATH, 'rb') as f:
            data = f.read()
        
        # Ищем значения 100000000 и 5000000
        values = {
            100000000: "100 мс",
            5000000: "5 мс",
            1000000: "1 мс",
            10000000: "10 мс"
        }
        
        print("\n\nЗначения pulse width в бинарнике:")
        for value, desc in values.items():
            le_bytes = pack('<I', value)
            count = data.count(le_bytes)
            if count > 0:
                print(f"  {value} нс ({desc}): {count} вхождений")
    
    print()

def analyze_servo_algorithms():
    """Анализирует servo алгоритмы"""
    print("=" * 80)
    print("АНАЛИЗ SERVO АЛГОРИТМОВ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        servo_functions = []
        for line in lines:
            line_lower = line.lower()
            if "servo" in line_lower:
                match = re.search(r'<([^>]+)>', line)
                if match:
                    func_name = match.group(1)
                    servo_functions.append(func_name)
        
        if servo_functions:
            print(f"\nНайдено {len(servo_functions)} servo функций:")
            for func in servo_functions[:30]:
                print(f"  - {func}")
        
        # Группировка по типам
        print("\n\nГруппировка по функциональности:")
        groups = {
            "update": [f for f in servo_functions if "update" in f.lower()],
            "calculate": [f for f in servo_functions if "calc" in f.lower() or "compute" in f.lower()],
            "adjust": [f for f in servo_functions if "adjust" in f.lower()],
            "sync": [f for f in servo_functions if "sync" in f.lower()],
        }
        
        for group_name, funcs in groups.items():
            if funcs:
                print(f"\n  {group_name.upper()}: {len(funcs)} функций")
                for func in funcs[:10]:
                    print(f"    - {func}")
    
    print()

def analyze_ptp_implementation():
    """Анализирует реализацию PTP"""
    print("=" * 80)
    print("АНАЛИЗ PTP РЕАЛИЗАЦИИ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        ptp_functions = []
        for line in lines:
            line_lower = line.lower()
            if "ptp" in line_lower and "client" in line_lower:
                match = re.search(r'<([^>]+)>', line)
                if match:
                    ptp_functions.append(match.group(1))
        
        if ptp_functions:
            print(f"\nНайдено {len(ptp_functions)} PTP функций:")
            for func in ptp_functions[:30]:
                print(f"  - {func}")
        
        # Поиск PTP сообщений
        result = run_command(["strings", BINARY_PATH])
        if result and not result.startswith("Ошибка"):
            lines = result.split('\n')
            
            ptp_strings = []
            for line in lines:
                if "ptp" in line.lower() and any(keyword in line.lower() for keyword in ["sync", "delay", "follow", "announce"]):
                    ptp_strings.append(line)
            
            if ptp_strings:
                print(f"\n\nНайдено {len(ptp_strings)} строк, связанных с PTP:")
                for line in sorted(set(ptp_strings))[:20]:
                    print(f"  {line}")
    
    print()

def create_dependency_graph():
    """Создает граф зависимостей между модулями"""
    print("=" * 80)
    print("ГРАФ ЗАВИСИМОСТЕЙ МОДУЛЕЙ")
    print("=" * 80)
    
    result = run_command(["objdump", "-d", "-C", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        # Ищем вызовы между модулями
        dependencies = defaultdict(set)
        current_module = None
        
        for line in lines:
            # Определяем текущий модуль
            if '>:' in line and '<' in line:
                func_match = re.search(r'<([^>]+)>:', line)
                if func_match:
                    func_name = func_match.group(1)
                    for module in ["ubx", "gnss", "ptp", "ntp", "servo"]:
                        if module in func_name.lower():
                            current_module = module
                            break
            
            # Ищем вызовы других модулей
            if current_module:
                call_match = re.search(r'<([^>]+)>', line)
                if call_match and ('call' in line.lower() or ' bl ' in line.lower()):
                    called_func = call_match.group(1)
                    for module in ["ubx", "gnss", "ptp", "ntp", "servo"]:
                        if module in called_func.lower() and module != current_module:
                            dependencies[current_module].add(module)
        
        print("\nЗависимости между модулями:")
        for module, deps in dependencies.items():
            if deps:
                print(f"\n  {module.upper()} ->")
                for dep in sorted(deps):
                    print(f"    - {dep}")
    
    print()

def extract_configuration_constants():
    """Извлекает константы конфигурации"""
    print("=" * 80)
    print("КОНСТАНТЫ КОНФИГУРАЦИИ")
    print("=" * 80)
    
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        config_strings = []
        for line in lines:
            if any(keyword in line.lower() for keyword in ["config", "default", "timeout", "interval", "delay"]):
                if "clocksync" in line.lower() or any(module in line.lower() for module in ["ubx", "ptp", "ntp", "servo"]):
                    config_strings.append(line)
        
        if config_strings:
            print(f"\nНайдено {len(config_strings)} констант конфигурации:")
            for line in sorted(set(config_strings))[:50]:
                print(f"  {line}")
    
    print()

def main():
    print("=" * 80)
    print("ГЛУБОКИЙ АНАЛИЗ МОДУЛЕЙ CLOCKSYNC")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # Анализ модулей
    extract_function_signatures()
    analyze_ubx_message_handling()
    analyze_timepulse_configuration()
    analyze_servo_algorithms()
    analyze_ptp_implementation()
    create_dependency_graph()
    extract_configuration_constants()
    
    print("=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    print()
    print("ВАЖНО:")
    print("  - Полный исходный код извлечь НЕВОЗМОЖНО")
    print("  - Можно понять структуру и логику работы")
    print("  - Можно воссоздать функциональность на основе анализа")
    print()
    print("Для сохранения результатов:")
    print("  sudo python3 deep_analyze_clocksync.py > clocksync_deep_analysis.txt 2>&1")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

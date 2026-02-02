#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Поиск servo функций через граф вызовов и nm
"""

import subprocess
import re
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
        return result.stdout if result.returncode == 0 else f"Ошибка: {result.stderr}"
    except Exception as e:
        return f"Ошибка: {str(e)}"

def find_servo_functions():
    """Находит все servo функции"""
    print("=" * 80)
    print("ПОИСК SERVO ФУНКЦИЙ")
    print("=" * 80)
    print()
    
    # Известные имена из графа вызовов
    known_functions = [
        "GetClockUsingGetTimeSyscall",
        "StepClockUsingSetTimeSyscall",
        "PerformGranularityMeasurement"
    ]
    
    # Полные имена из графа вызовов
    full_names = [
        "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockUsingGetTimeSyscall@@Base",
        "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepClockUsingSetTimeSyscall@@Base",
        "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.PerformGranularityMeasurement@@Base"
    ]
    
    print("🔍 Поиск через nm -D:")
    nm_result = run_command(["nm", "-D", BINARY_PATH])
    found_functions = {}
    
    if nm_result and not nm_result.startswith("Ошибка"):
        lines = nm_result.split('\n')
        for line in lines:
            line_lower = line.lower()
            if 'servo' in line_lower and 'adjusttime' in line_lower:
                # Проверяем каждую известную функцию
                for i, short_name in enumerate(known_functions):
                    if short_name.lower() in line_lower:
                        match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                        full_name_match = re.search(r'<([^>]+)>', line)
                        if match and full_name_match:
                            addr = match.group(1)
                            full_name = full_name_match.group(1)
                            found_functions[short_name] = {
                                'full_name': full_name,
                                'addr': addr,
                                'line': line.strip()
                            }
                            print(f"  ✓ {short_name}:")
                            print(f"    Адрес: 0x{addr}")
                            print(f"    Полное имя: {full_name}")
                            print()
    
    # Если не нашли через nm, пробуем objdump -T
    if len(found_functions) < len(known_functions):
        print("\n🔍 Поиск через objdump -T:")
        objdump_result = run_command(["objdump", "-T", BINARY_PATH])
        if objdump_result and not objdump_result.startswith("Ошибка"):
            lines = objdump_result.split('\n')
            for line in lines:
                line_lower = line.lower()
                if 'servo' in line_lower and 'adjusttime' in line_lower:
                    for short_name in known_functions:
                        if short_name not in found_functions and short_name.lower() in line_lower:
                            match = re.search(r'([0-9a-f]+)\s+.*?\s+<([^>]+)>', line)
                            if match:
                                addr = match.group(1)
                                full_name = match.group(2)
                                found_functions[short_name] = {
                                    'full_name': full_name,
                                    'addr': addr,
                                    'line': line.strip()
                                }
                                print(f"  ✓ {short_name}:")
                                print(f"    Адрес: 0x{addr}")
                                print(f"    Полное имя: {full_name}")
                                print()
    
    # Выводим итоги
    print("\n" + "=" * 80)
    print("ИТОГИ ПОИСКА")
    print("=" * 80)
    print(f"\nНайдено функций: {len(found_functions)} из {len(known_functions)}")
    
    if found_functions:
        print("\n✅ Найденные функции:")
        for name, info in found_functions.items():
            print(f"  {name}:")
            print(f"    Адрес: 0x{info['addr']}")
            print(f"    Полное имя: {info['full_name']}")
    else:
        print("\n⚠ Функции не найдены")
        print("\nПопробуйте вручную:")
        print(f"  nm -D {BINARY_PATH} | grep -i 'servo.*adjusttime'")
        print(f"  objdump -T {BINARY_PATH} | grep -i 'servo.*adjusttime'")
    
    return found_functions

def main():
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    functions = find_servo_functions()
    
    if functions:
        print("\n" + "=" * 80)
        print("РЕКОМЕНДАЦИИ")
        print("=" * 80)
        print("\nИспользуйте найденные адреса для анализа:")
        for name, info in functions.items():
            print(f"\n  {name}:")
            print(f"    objdump -d -C --start-address 0x{info['addr']} --stop-address 0x{int(info['addr'], 16) + 0x2000:x} {BINARY_PATH}")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

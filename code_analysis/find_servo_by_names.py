#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Поиск servo функций по известным именам из графа вызовов
"""

import subprocess
import re
import sys

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

# Известные имена из графа вызовов
KNOWN_NAMES = [
    "GetClockUsingGetTimeSyscall@@Base",
    "StepClockUsingSetTimeSyscall@@Base",
    "PerformGranularityMeasurement@@Base",
    "GetTimeNow@@Base",
    "StepClock@@Base",
    "SlewClockPossiblyAsync@@Base",
    "GetAllClockOffsets@@Base"
]

def find_functions():
    """Поиск функций по известным именам"""
    print("=" * 80)
    print("ПОИСК SERVO ФУНКЦИЙ ПО ИЗВЕСТНЫМ ИМЕНАМ")
    print("=" * 80)
    print()
    
    found = {}
    
    # Поиск через nm
    print("🔍 Поиск через nm -D:")
    result = subprocess.run(["nm", "-D", BINARY_PATH], capture_output=True, text=True)
    
    if result.returncode == 0:
        for line in result.stdout.split('\n'):
            for name in KNOWN_NAMES:
                if name in line:
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    full_match = re.search(r'<([^>]+)>', line)
                    if match and full_match:
                        addr = match.group(1)
                        full_name = full_match.group(1)
                        if name not in found:
                            found[name] = {
                                'addr': addr,
                                'full_name': full_name,
                                'line': line.strip()
                            }
                            print(f"  ✓ {name}:")
                            print(f"    Адрес: 0x{addr}")
                            print(f"    Полное имя: {full_name}")
                            print()
    
    # Поиск через objdump -T
    if len(found) < len(KNOWN_NAMES):
        print("\n🔍 Поиск через objdump -T:")
        result = subprocess.run(["objdump", "-T", BINARY_PATH], capture_output=True, text=True)
        
        if result.returncode == 0:
            for line in result.stdout.split('\n'):
                for name in KNOWN_NAMES:
                    if name not in found and name in line:
                        match = re.search(r'([0-9a-f]+)\s+.*?\s+<([^>]+)>', line)
                        if match:
                            addr = match.group(1)
                            full_name = match.group(2)
                            found[name] = {
                                'addr': addr,
                                'full_name': full_name,
                                'line': line.strip()
                            }
                            print(f"  ✓ {name}:")
                            print(f"    Адрес: 0x{addr}")
                            print(f"    Полное имя: {full_name}")
                            print()
    
    # Поиск по частичным совпадениям
    if len(found) < len(KNOWN_NAMES):
        print("\n🔍 Поиск по частичным совпадениям:")
        result = subprocess.run(["nm", "-D", BINARY_PATH], capture_output=True, text=True)
        
        if result.returncode == 0:
            for line in result.stdout.split('\n'):
                line_lower = line.lower()
                # Ищем по ключевым словам
                if 'adjusttime' in line_lower or ('servo' in line_lower and 'clock' in line_lower):
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    full_match = re.search(r'<([^>]+)>', line)
                    if match and full_match:
                        addr = match.group(1)
                        full_name = full_match.group(1)
                        # Проверяем, не нашли ли мы уже эту функцию
                        for known_name in KNOWN_NAMES:
                            if known_name.lower().replace('@@base', '') in full_name.lower():
                                if known_name not in found:
                                    found[known_name] = {
                                        'addr': addr,
                                        'full_name': full_name,
                                        'line': line.strip()
                                    }
                                    print(f"  ✓ {known_name}:")
                                    print(f"    Адрес: 0x{addr}")
                                    print(f"    Полное имя: {full_name}")
                                    print()
    
    return found

def main():
    found = find_functions()
    
    print("\n" + "=" * 80)
    print("ИТОГИ")
    print("=" * 80)
    print(f"\nНайдено: {len(found)} из {len(KNOWN_NAMES)} функций\n")
    
    if found:
        print("Найденные функции:")
        for name, info in found.items():
            print(f"  {name}: 0x{info['addr']}")
        
        # Сохраняем результаты
        with open("servo_functions_found.txt", "w") as f:
            f.write("=" * 80 + "\n")
            f.write("НАЙДЕННЫЕ SERVO ФУНКЦИИ\n")
            f.write("=" * 80 + "\n\n")
            for name, info in found.items():
                f.write(f"{name}:\n")
                f.write(f"  Адрес: 0x{info['addr']}\n")
                f.write(f"  Полное имя: {info['full_name']}\n")
                f.write(f"  Строка: {info['line']}\n\n")
        
        print("\n✅ Результаты сохранены в servo_functions_found.txt")
    else:
        print("⚠ Функции не найдены")
        print("\nПопробуйте вручную:")
        print(f"  nm -D {BINARY_PATH} | grep -i adjusttime")
        print(f"  nm -D {BINARY_PATH} | grep -i 'servo.*clock'")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

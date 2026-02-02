#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Проверка полноты анализа функционала программы
"""

import subprocess
import re
import sys
from collections import defaultdict

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
        return ""

def find_all_clocksync_functions():
    """Находит все функции clocksync модулей"""
    print("=" * 80)
    print("ПОИСК ВСЕХ CLOCKSYNC ФУНКЦИЙ")
    print("=" * 80)
    print()
    
    modules = {
        'ubx': [],
        'servo': [],
        'ptp': [],
        'ntp': [],
        'nmea': [],
        'phc': [],
        'hostclocks': [],
        'vendors': [],
        'other': []
    }
    
    # Поиск через nm
    result = run_command(["nm", "-D", BINARY_PATH])
    
    if result:
        lines = result.split('\n')
        for line in lines:
            line_lower = line.lower()
            
            # Классификация по модулям
            if 'clocksync' in line_lower:
                match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                full_match = re.search(r'<([^>]+)>', line)
                
                if match and full_match:
                    addr = match.group(1)
                    full_name = full_match.group(1)
                    
                    # Классификация
                    if 'ubx' in line_lower or 'helper/ubx' in line_lower:
                        modules['ubx'].append({'addr': addr, 'name': full_name})
                    elif 'servo' in line_lower:
                        modules['servo'].append({'addr': addr, 'name': full_name})
                    elif 'ptp' in line_lower:
                        modules['ptp'].append({'addr': addr, 'name': full_name})
                    elif 'ntp' in line_lower:
                        modules['ntp'].append({'addr': addr, 'name': full_name})
                    elif 'nmea' in line_lower:
                        modules['nmea'].append({'addr': addr, 'name': full_name})
                    elif 'phc' in line_lower:
                        modules['phc'].append({'addr': addr, 'name': full_name})
                    elif 'hostclock' in line_lower:
                        modules['hostclocks'].append({'addr': addr, 'name': full_name})
                    elif 'vendor' in line_lower:
                        modules['vendors'].append({'addr': addr, 'name': full_name})
                    else:
                        modules['other'].append({'addr': addr, 'name': full_name})
    
    return modules

def analyze_module_completeness(modules):
    """Анализирует полноту анализа каждого модуля"""
    print("=" * 80)
    print("АНАЛИЗ ПОЛНОТЫ")
    print("=" * 80)
    print()
    
    completeness = {}
    
    # Известные функции из анализа
    known_ubx = [
        'UBXTP5Message.ToBytes',
        'UBXGenericMessage.ToBytes',
        'UBXGNSSMessage.ToBytes',
        'UBXMessageHeader.ToBytes',
        'send1PPSOnTimepulsePin',
        'detectUbloxUnit'
    ]
    
    known_servo = [
        'GetClockUsingGetTimeSyscall',
        'StepClockUsingSetTimeSyscall',
        'PerformGranularityMeasurement',
        'GetClockFrequency',
        'SetFrequency',
        'SetOffset',
        'AlgoPID.UpdateClockFreq',
        'Pi.UpdateClockFreq',
        'LinReg.UpdateClockFreq',
        'RunPeriodicAdjustSlaveClocks',
        'ChangeMasterClock',
        'HoldMasterClockElection'
    ]
    
    # Анализ UBX
    ubx_found = 0
    for func in modules['ubx']:
        for known in known_ubx:
            if known.lower() in func['name'].lower():
                ubx_found += 1
                break
    
    ubx_total = len(modules['ubx'])
    ubx_completeness = (ubx_found / len(known_ubx) * 100) if known_ubx else 0
    completeness['ubx'] = {
        'total': ubx_total,
        'known': len(known_ubx),
        'found': ubx_found,
        'percent': ubx_completeness
    }
    
    # Анализ Servo
    servo_found = 0
    for func in modules['servo']:
        for known in known_servo:
            if known.lower() in func['name'].lower():
                servo_found += 1
                break
    
    servo_total = len(modules['servo'])
    servo_completeness = (servo_found / len(known_servo) * 100) if known_servo else 0
    completeness['servo'] = {
        'total': servo_total,
        'known': len(known_servo),
        'found': servo_found,
        'percent': servo_completeness
    }
    
    # Общая статистика
    print("📊 СТАТИСТИКА МОДУЛЕЙ:")
    print()
    for module_name, funcs in modules.items():
        if funcs:
            print(f"  {module_name.upper()}: {len(funcs)} функций")
    
    print()
    print("📈 ПОЛНОТА АНАЛИЗА:")
    print()
    for module_name, stats in completeness.items():
        print(f"  {module_name.upper()}:")
        print(f"    Всего функций: {stats['total']}")
        print(f"    Известных: {stats['known']}")
        print(f"    Найдено: {stats['found']}")
        print(f"    Полнота: {stats['percent']:.1f}%")
        print()
    
    return completeness, modules

def find_missing_functions(modules, completeness):
    """Находит функции, которые еще не проанализированы"""
    print("=" * 80)
    print("ПОИСК НЕПРОАНАЛИЗИРОВАННЫХ ФУНКЦИЙ")
    print("=" * 80)
    print()
    
    # Ключевые слова для важных функций
    important_keywords = [
        'configure', 'config', 'setup', 'init', 'start', 'stop',
        'update', 'adjust', 'sync', 'calibrate', 'measure',
        'get', 'set', 'read', 'write', 'send', 'receive',
        'parse', 'encode', 'decode', 'serialize', 'deserialize'
    ]
    
    important_functions = defaultdict(list)
    
    for module_name, funcs in modules.items():
        for func in funcs:
            func_lower = func['name'].lower()
            for keyword in important_keywords:
                if keyword in func_lower:
                    # Проверяем, не является ли это уже известной функцией
                    is_known = False
                    if module_name == 'ubx' and any(k in func_lower for k in ['ubxtp5', 'tobytes', 'send1pps']):
                        is_known = True
                    elif module_name == 'servo' and any(k in func_lower for k in ['getclock', 'stepclock', 'pid', 'pi']):
                        is_known = True
                    
                    if not is_known:
                        important_functions[module_name].append(func)
                    break
    
    return important_functions

def main():
    print("=" * 80)
    print("ПРОВЕРКА ПОЛНОТЫ АНАЛИЗА ФУНКЦИОНАЛА ПРОГРАММЫ")
    print("=" * 80)
    print()
    
    # Поиск всех функций
    modules = find_all_clocksync_functions()
    
    # Анализ полноты
    completeness, modules = analyze_module_completeness(modules)
    
    # Поиск непроанализированных функций
    missing = find_missing_functions(modules, completeness)
    
    # Вывод непроанализированных функций
    print("🔍 ВАЖНЫЕ ФУНКЦИИ, ТРЕБУЮЩИЕ АНАЛИЗА:")
    print()
    
    total_missing = 0
    for module_name, funcs in missing.items():
        if funcs:
            print(f"  {module_name.upper()} ({len(funcs)} функций):")
            for func in funcs[:10]:  # Первые 10
                print(f"    {func['name']}")
            if len(funcs) > 10:
                print(f"    ... и еще {len(funcs) - 10} функций")
            print()
            total_missing += len(funcs)
    
    if total_missing == 0:
        print("  ✅ Все важные функции проанализированы!")
    else:
        print(f"  ⚠ Найдено {total_missing} важных функций, требующих анализа")
    
    # Сохранение результатов
    output_file = "completeness_check.txt"
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("=" * 80 + "\n")
        f.write("ПРОВЕРКА ПОЛНОТЫ АНАЛИЗА\n")
        f.write("=" * 80 + "\n\n")
        
        f.write("СТАТИСТИКА МОДУЛЕЙ:\n")
        for module_name, funcs in modules.items():
            if funcs:
                f.write(f"  {module_name.upper()}: {len(funcs)} функций\n")
        f.write("\n")
        
        f.write("ПОЛНОТА АНАЛИЗА:\n")
        for module_name, stats in completeness.items():
            f.write(f"  {module_name.upper()}: {stats['percent']:.1f}%\n")
        f.write("\n")
        
        f.write("ВАЖНЫЕ НЕПРОАНАЛИЗИРОВАННЫЕ ФУНКЦИИ:\n")
        for module_name, funcs in missing.items():
            if funcs:
                f.write(f"\n  {module_name.upper()}:\n")
                for func in funcs:
                    f.write(f"    {func['name']} (0x{func['addr']})\n")
    
    print()
    print("=" * 80)
    print("СОХРАНЕНИЕ РЕЗУЛЬТАТОВ")
    print("=" * 80)
    print(f"✅ Результаты сохранены в {output_file}")
    
    # Итоговая оценка
    print()
    print("=" * 80)
    print("ИТОГОВАЯ ОЦЕНКА")
    print("=" * 80)
    
    avg_completeness = sum(s['percent'] for s in completeness.values()) / len(completeness) if completeness else 0
    print(f"\nСредняя полнота анализа: {avg_completeness:.1f}%")
    
    if avg_completeness >= 80:
        print("✅ Отличная полнота анализа!")
    elif avg_completeness >= 60:
        print("✅ Хорошая полнота анализа")
    else:
        print("⚠ Требуется дополнительный анализ")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

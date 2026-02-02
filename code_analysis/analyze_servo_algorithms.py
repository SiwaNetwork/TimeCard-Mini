#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Глубокий анализ Servo алгоритмов
"""

import subprocess
import re
import sys
import os
from collections import defaultdict

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
    
    # Известные имена из графа вызовов (с @@Base суффиксом)
    known_functions = {
        "GetClockUsingGetTimeSyscall": [
            "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockUsingGetTimeSyscall",
            "GetClockUsingGetTimeSyscall@@Base"
        ],
        "StepClockUsingSetTimeSyscall": [
            "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepClockUsingSetTimeSyscall",
            "StepClockUsingSetTimeSyscall@@Base"
        ],
        "PerformGranularityMeasurement": [
            "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.PerformGranularityMeasurement",
            "PerformGranularityMeasurement@@Base"
        ],
    }
    
    found_functions = {}
    
    # Поиск через nm
    print("🔍 Поиск через nm -D:")
    nm_result = run_command(["nm", "-D", BINARY_PATH])
    
    if nm_result and not nm_result.startswith("Ошибка"):
        lines = nm_result.split('\n')
        for line in lines:
            line_lower = line.lower()
            if 'servo' in line_lower and 'adjusttime' in line_lower:
                for short_name, patterns in known_functions.items():
                    # Проверяем все варианты имени
                    found = False
                    for pattern in patterns:
                        if pattern.lower() in line_lower or short_name.lower() in line_lower:
                            found = True
                            break
                    
                    if found and short_name not in found_functions:
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
    
    # Поиск через objdump -T
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
    
    # Поиск всех servo функций
    print("\n🔍 Поиск всех servo функций:")
    servo_patterns = [
        r'servo.*adjusttime',
        r'servo.*clock',
        r'servo.*time',
        r'servo.*sync',
        r'servo.*pid',
        r'servo.*filter',
    ]
    
    all_servo_functions = []
    if nm_result and not nm_result.startswith("Ошибка"):
        lines = nm_result.split('\n')
        for line in lines:
            line_lower = line.lower()
            if 'servo' in line_lower:
                match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                full_name_match = re.search(r'<([^>]+)>', line)
                if match and full_name_match:
                    addr = match.group(1)
                    full_name = full_name_match.group(1)
                    all_servo_functions.append({
                        'addr': addr,
                        'full_name': full_name,
                        'line': line.strip()
                    })
    
    print(f"  Найдено {len(all_servo_functions)} servo функций")
    if all_servo_functions:
        print("\n  Первые 20:")
        for func in all_servo_functions[:20]:
            print(f"    0x{func['addr']}: {func['full_name']}")
    
    return found_functions, all_servo_functions

def disassemble_function(addr, name, size=0x2000):
    """Дизассемблирует функцию"""
    try:
        addr_clean = addr.replace('0x', '').replace('0X', '')
        addr_int = int(addr_clean, 16)
        start_addr = f"0x{addr_int:x}"
        end_addr = f"0x{addr_int + size:x}"
        
        cmd = ["objdump", "-d", "-C", "--start-address", start_addr, 
               "--stop-address", end_addr, BINARY_PATH]
        result = run_command(cmd)
        
        if result and not result.startswith("Ошибка"):
            return result
        return None
    except Exception as e:
        return f"Ошибка дизассемблирования: {str(e)}"

def analyze_assembly(asm_code):
    """Анализирует ассемблерный код"""
    if not asm_code or asm_code.startswith("Ошибка"):
        return {}
    
    analysis = {
        'arithmetic_ops': [],
        'constants': [],
        'calls': [],
        'branches': [],
        'loads': [],
        'stores': [],
        'patterns': {}
    }
    
    lines = asm_code.split('\n')
    
    # Паттерны для поиска
    arithmetic_patterns = [
        (r'\s+(add|sub|mul|div|fadd|fsub|fmul|fdiv)\s+', 'arithmetic'),
        (r'\s+(and|orr|eor|bic)\s+', 'bitwise'),
        (r'\s+(lsl|lsr|asr)\s+', 'shift'),
    ]
    
    constant_patterns = [
        (r'mov\s+[xw]\d+,\s*#0x([0-9a-f]+)', 'hex'),
        (r'movk\s+[xw]\d+,\s*#0x([0-9a-f]+)', 'hex'),
        (r'#0x([0-9a-f]+)', 'hex'),
        (r'#(\d+)', 'decimal'),
    ]
    
    call_patterns = [
        (r'bl\s+([0-9a-f]+)\s+<([^>]+)>', 'direct'),
        (r'blr\s+', 'indirect'),
    ]
    
    branch_patterns = [
        (r'\s+(b|beq|bne|blt|bgt|ble|bge|blo|bhi)\s+', 'conditional'),
        (r'\s+(cbz|cbnz)\s+', 'conditional_zero'),
    ]
    
    for line in lines:
        line_lower = line.lower()
        
        # Арифметические операции
        for pattern, op_type in arithmetic_patterns:
            if re.search(pattern, line_lower):
                analysis['arithmetic_ops'].append({
                    'type': op_type,
                    'line': line.strip()
                })
        
        # Константы
        for pattern, const_type in constant_patterns:
            matches = re.finditer(pattern, line_lower)
            for match in matches:
                value = match.group(1)
                try:
                    if const_type == 'hex':
                        num = int(value, 16)
                    else:
                        num = int(value)
                    analysis['constants'].append({
                        'value': num,
                        'hex': f"0x{num:x}" if num > 0 else f"-0x{abs(num):x}",
                        'line': line.strip()
                    })
                except:
                    pass
        
        # Вызовы функций
        for pattern, call_type in call_patterns:
            matches = re.finditer(pattern, line)
            for match in matches:
                if call_type == 'direct':
                    addr = match.group(1)
                    func_name = match.group(2) if len(match.groups()) > 1 else "unknown"
                    analysis['calls'].append({
                        'addr': addr,
                        'name': func_name,
                        'type': call_type
                    })
        
        # Ветвления
        for pattern, branch_type in branch_patterns:
            if re.search(pattern, line_lower):
                analysis['branches'].append({
                    'type': branch_type,
                    'line': line.strip()
                })
        
        # Загрузки и сохранения
        if re.search(r'\s+ldr\s+', line_lower):
            analysis['loads'].append(line.strip())
        if re.search(r'\s+str\s+', line_lower):
            analysis['stores'].append(line.strip())
    
    # Анализ паттернов
    analysis['patterns'] = {
        'total_instructions': len([l for l in lines if re.search(r'^\s+[0-9a-f]+:', l)]),
        'arithmetic_count': len(analysis['arithmetic_ops']),
        'constant_count': len(analysis['constants']),
        'call_count': len(analysis['calls']),
        'branch_count': len(analysis['branches']),
        'load_count': len(analysis['loads']),
        'store_count': len(analysis['stores']),
    }
    
    return analysis

def extract_pid_patterns(analysis):
    """Извлекает PID-подобные паттерны"""
    pid_patterns = []
    
    # Поиск паттернов: error, integral, derivative
    constants = analysis.get('constants', [])
    arithmetic = analysis.get('arithmetic_ops', [])
    
    # Группируем константы по значениям
    const_values = [c['value'] for c in constants if isinstance(c['value'], (int, float))]
    
    # Ищем коэффициенты (обычно маленькие числа 0.0-1.0 или большие целые)
    coefficients = []
    for val in const_values:
        if 0 < abs(val) < 1000 and (val < 1.0 or val % 1 != 0):
            coefficients.append(val)
        elif 1000 <= abs(val) < 1000000:
            coefficients.append(val)
    
    if coefficients:
        pid_patterns.append({
            'type': 'coefficients',
            'values': sorted(set(coefficients))[:10]
        })
    
    return pid_patterns

def main():
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    print("=" * 80)
    print("ГЛУБОКИЙ АНАЛИЗ SERVO АЛГОРИТМОВ")
    print("=" * 80)
    print()
    
    # Поиск функций
    found_functions, all_servo_functions = find_servo_functions()
    
    if not found_functions:
        print("\n⚠ Ключевые функции не найдены автоматически")
        print("Попробуем найти через альтернативные методы...")
        
        # Поиск по частичным совпадениям
        print("\n🔍 Альтернативный поиск:")
        nm_result = run_command(["nm", "-D", BINARY_PATH])
        if nm_result and not nm_result.startswith("Ошибка"):
            lines = nm_result.split('\n')
            for line in lines:
                if 'adjusttime' in line.lower() or 'clock' in line.lower():
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    full_name_match = re.search(r'<([^>]+)>', line)
                    if match and full_name_match:
                        addr = match.group(1)
                        full_name = full_name_match.group(1)
                        if 'servo' in full_name.lower() or 'adjusttime' in full_name.lower():
                            print(f"  Найдено: 0x{addr} - {full_name}")
                            if 'getclock' in full_name.lower() or 'gettime' in full_name.lower():
                                found_functions['GetClockUsingGetTimeSyscall'] = {
                                    'full_name': full_name,
                                    'addr': addr
                                }
                            elif 'stepclock' in full_name.lower() or 'settime' in full_name.lower():
                                found_functions['StepClockUsingSetTimeSyscall'] = {
                                    'full_name': full_name,
                                    'addr': addr
                                }
                            elif 'granularity' in full_name.lower() or 'measurement' in full_name.lower():
                                found_functions['PerformGranularityMeasurement'] = {
                                    'full_name': full_name,
                                    'addr': addr
                                }
    
    # Анализ найденных функций
    print("\n" + "=" * 80)
    print("АНАЛИЗ SERVO ФУНКЦИЙ")
    print("=" * 80)
    
    results = {}
    
    for name, info in found_functions.items():
        print(f"\n{'=' * 80}")
        print(f"ФУНКЦИЯ: {name}")
        print(f"{'=' * 80}")
        print(f"Адрес: 0x{info['addr']}")
        print(f"Полное имя: {info['full_name']}")
        print()
        
        # Дизассемблирование
        print("📝 Дизассемблирование...")
        asm = disassemble_function(info['addr'], info['full_name'])
        
        if asm and not asm.startswith("Ошибка"):
            # Анализ
            print("🔍 Анализ ассемблерного кода...")
            analysis = analyze_assembly(asm)
            
            results[name] = {
                'info': info,
                'analysis': analysis,
                'asm': asm[:2000] if len(asm) > 2000 else asm  # Первые 2000 символов
            }
            
            # Вывод результатов
            patterns = analysis.get('patterns', {})
            print(f"\n📊 Статистика:")
            print(f"  Всего инструкций: {patterns.get('total_instructions', 0)}")
            print(f"  Арифметических операций: {patterns.get('arithmetic_count', 0)}")
            print(f"  Констант: {patterns.get('constant_count', 0)}")
            print(f"  Вызовов функций: {patterns.get('call_count', 0)}")
            print(f"  Ветвлений: {patterns.get('branch_count', 0)}")
            print(f"  Загрузок: {patterns.get('load_count', 0)}")
            print(f"  Сохранений: {patterns.get('store_count', 0)}")
            
            # Константы
            constants = analysis.get('constants', [])
            if constants:
                print(f"\n🔢 Найденные константы (первые 20):")
                unique_constants = {}
                for c in constants:
                    val = c['value']
                    if val not in unique_constants:
                        unique_constants[val] = c
                for val, c in sorted(unique_constants.items(), key=lambda x: abs(x[0]))[:20]:
                    print(f"  {c['hex']} ({val})")
            
            # Вызовы функций
            calls = analysis.get('calls', [])
            if calls:
                print(f"\n📞 Вызовы функций (первые 10):")
                for call in calls[:10]:
                    print(f"  {call.get('name', 'unknown')} @ {call.get('addr', 'unknown')}")
            
            # PID паттерны
            pid_patterns = extract_pid_patterns(analysis)
            if pid_patterns:
                print(f"\n🎯 PID-подобные паттерны:")
                for pattern in pid_patterns:
                    print(f"  {pattern['type']}: {pattern['values']}")
        else:
            print(f"⚠ Не удалось дизассемблировать функцию")
            results[name] = {
                'info': info,
                'error': 'disassembly_failed'
            }
    
    # Сохранение результатов
    print("\n" + "=" * 80)
    print("СОХРАНЕНИЕ РЕЗУЛЬТАТОВ")
    print("=" * 80)
    
    output_file = "servo_algorithms_analysis.txt"
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("=" * 80 + "\n")
        f.write("АНАЛИЗ SERVO АЛГОРИТМОВ\n")
        f.write("=" * 80 + "\n\n")
        
        for name, result in results.items():
            f.write("=" * 80 + "\n")
            f.write(f"ФУНКЦИЯ: {name}\n")
            f.write("=" * 80 + "\n")
            f.write(f"Адрес: 0x{result['info']['addr']}\n")
            f.write(f"Полное имя: {result['info']['full_name']}\n\n")
            
            if 'analysis' in result:
                patterns = result['analysis'].get('patterns', {})
                f.write("СТАТИСТИКА:\n")
                for key, value in patterns.items():
                    f.write(f"  {key}: {value}\n")
                f.write("\n")
                
                constants = result['analysis'].get('constants', [])
                if constants:
                    f.write("КОНСТАНТЫ:\n")
                    unique_constants = {}
                    for c in constants:
                        val = c['value']
                        if val not in unique_constants:
                            unique_constants[val] = c
                    for val, c in sorted(unique_constants.items(), key=lambda x: abs(x[0])):
                        f.write(f"  {c['hex']} ({val})\n")
                    f.write("\n")
                
                if 'asm' in result:
                    f.write("АССЕМБЛЕРНЫЙ КОД (первые 2000 символов):\n")
                    f.write(result['asm'][:2000])
                    f.write("\n\n")
            else:
                f.write("ОШИБКА: Не удалось проанализировать функцию\n\n")
    
    print(f"✅ Результаты сохранены в {output_file}")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

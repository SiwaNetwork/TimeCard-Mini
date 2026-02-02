#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Глубокий анализ бинарника shiwatime
Извлекает алгоритмы, структуры данных, константы и логику работы
"""

import subprocess
import re
import sys
import os
from collections import defaultdict
from struct import pack, unpack

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

def extract_function_assembly(func_name, func_addr):
    """Извлекает полный ассемблерный код функции"""
    if not func_addr:
        # Пробуем найти адрес по имени
        nm_result = run_command(["nm", "-D", BINARY_PATH])
        if nm_result and not nm_result.startswith("Ошибка"):
            lines = nm_result.split('\n')
            for line in lines:
                if func_name in line:
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    if match:
                        func_addr = match.group(1)
                        break
    
    if not func_addr:
        return None
    
    try:
        # Убираем префикс 0x если есть (может быть двойной)
        addr_clean = func_addr.replace('0x0x', '0x').replace('0X0X', '0x')
        addr_clean = addr_clean.replace('0x', '').replace('0X', '')
        addr_int = int(addr_clean, 16)
        start_addr = f"0x{addr_int:x}"  # objdump принимает 0x формат
        end_addr = f"0x{addr_int + 0x2000:x}"  # +8KB для функции
        
        result = run_command([
            "objdump", "-d", "-C", 
            "--start-address", start_addr,
            "--stop-address", end_addr,
            BINARY_PATH
        ])
        
        if result and not result.startswith("Ошибка"):
            lines = result.split('\n')
            func_lines = []
            in_func = False
            
            for line in lines:
                # Ищем начало функции
                if f'<{func_name}>' in line or (func_addr in line and ':' in line and '<' in line):
                    in_func = True
                    func_lines.append(line)
                    continue
                
                if in_func:
                    if line.strip() == '':
                        continue
                    # Конец функции - другая функция или пустая строка после ret
                    if '<' in line and '>:' in line and func_name not in line:
                        break
                    if 'ret' in line.lower() and len(func_lines) > 10:
                        func_lines.append(line)
                        # Проверяем, есть ли еще код после ret
                        break
                    func_lines.append(line)
            
            return '\n'.join(func_lines) if func_lines else None
    except Exception as e:
        print(f"Ошибка при дизассемблировании: {e}")
        return None
    
    return None

def analyze_arithmetic_operations(assembly):
    """Анализирует арифметические операции в ассемблере"""
    operations = {
        'additions': [],
        'subtractions': [],
        'multiplications': [],
        'divisions': [],
        'shifts': [],
        'bitwise': []
    }
    
    if not assembly:
        return operations
    
    lines = assembly.split('\n')
    
    for line in lines:
        line_lower = line.lower()
        
        # Сложение
        if 'add' in line_lower or 'adc' in line_lower:
            # Извлекаем операнды
            match = re.search(r'add\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', line_lower)
            if match:
                operations['additions'].append({
                    'line': line.strip(),
                    'dest': match.group(1),
                    'src': match.group(2),
                    'imm': match.group(3) if match.group(3) else None
                })
        
        # Вычитание
        if 'sub' in line_lower or 'sbc' in line_lower:
            match = re.search(r'sub\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', line_lower)
            if match:
                operations['subtractions'].append({
                    'line': line.strip(),
                    'dest': match.group(1),
                    'src': match.group(2),
                    'imm': match.group(3) if match.group(3) else None
                })
        
        # Умножение
        if 'mul' in line_lower or 'madd' in line_lower or 'msub' in line_lower:
            operations['multiplications'].append(line.strip())
        
        # Деление (редко в ARM)
        if 'sdiv' in line_lower or 'udiv' in line_lower:
            operations['divisions'].append(line.strip())
        
        # Сдвиги
        if 'lsl' in line_lower or 'lsr' in line_lower or 'asr' in line_lower:
            match = re.search(r'(lsl|lsr|asr)\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', line_lower)
            if match:
                operations['shifts'].append({
                    'type': match.group(1),
                    'dest': match.group(2),
                    'src': match.group(3),
                    'bits': match.group(4) if match.group(4) else None,
                    'line': line.strip()
                })
        
        # Побитовые операции
        if any(op in line_lower for op in ['and', 'orr', 'eor', 'bic', 'orn']):
            operations['bitwise'].append(line.strip())
    
    return operations

def extract_constants_from_assembly(assembly):
    """Извлекает константы из ассемблера"""
    constants = {
        'immediate': [],
        'addresses': [],
        'hex_values': [],
        'decimal_values': []
    }
    
    if not assembly:
        return constants
    
    lines = assembly.split('\n')
    
    for line in lines:
        # Hex значения
        hex_matches = re.findall(r'0x([0-9a-f]+)', line, re.IGNORECASE)
        for hex_val in hex_matches:
            try:
                val = int(hex_val, 16)
                constants['hex_values'].append({
                    'hex': hex_val,
                    'decimal': val,
                    'line': line.strip()
                })
            except:
                pass
        
        # Decimal значения
        dec_matches = re.findall(r'#(\d+)', line)
        for dec_val in dec_matches:
            try:
                val = int(dec_val)
                constants['decimal_values'].append({
                    'decimal': val,
                    'line': line.strip()
                })
            except:
                pass
        
        # Адреса функций
        addr_matches = re.findall(r'<([^>]+)>', line)
        for addr in addr_matches:
            if addr not in constants['addresses']:
                constants['addresses'].append(addr)
    
    return constants

def analyze_control_flow(assembly):
    """Анализирует поток управления"""
    flow = {
        'branches': [],
        'loops': [],
        'calls': [],
        'returns': [],
        'conditions': []
    }
    
    if not assembly:
        return flow
    
    lines = assembly.split('\n')
    addresses = {}
    
    for i, line in enumerate(lines):
        line_stripped = line.strip()
        if not line_stripped:
            continue
        
        # Извлекаем адрес текущей инструкции
        addr_match = re.search(r'^\s*([0-9a-f]+):', line_stripped)
        current_addr = addr_match.group(1) if addr_match else None
        
        # Ветвления
        if any(branch in line_stripped.lower() for branch in ['b.', 'b ', 'bl ', 'bl.']):
            branch_match = re.search(r'(b\.?\w*|bl\.?\w*)\s+([0-9a-f]+|<\w+>)', line_stripped, re.IGNORECASE)
            if branch_match:
                flow['branches'].append({
                    'type': branch_match.group(1),
                    'target': branch_match.group(2),
                    'line': line_stripped,
                    'address': current_addr
                })
        
        # Вызовы функций
        if 'bl ' in line_stripped.lower():
            call_match = re.search(r'bl\s+<([^>]+)>', line_stripped)
            if call_match:
                flow['calls'].append({
                    'function': call_match.group(1),
                    'line': line_stripped,
                    'address': current_addr
                })
        
        # Возвраты
        if 'ret' in line_stripped.lower():
            flow['returns'].append({
                'line': line_stripped,
                'address': current_addr
            })
        
        # Условия
        if 'cmp' in line_stripped.lower() or 'tst' in line_stripped.lower():
            flow['conditions'].append({
                'line': line_stripped,
                'address': current_addr
            })
    
    # Определяем циклы (переходы назад)
    for branch in flow['branches']:
        if branch['address'] and branch['target']:
            try:
                addr_int = int(branch['address'], 16)
                target_int = int(branch['target'], 16)
                if target_int < addr_int:
                    flow['loops'].append(branch)
            except:
                pass
    
    return flow

def extract_data_structures(assembly):
    """Пытается извлечь структуры данных из ассемблера"""
    structures = {
        'memory_access': [],
        'field_offsets': [],
        'struct_patterns': []
    }
    
    if not assembly:
        return structures
    
    lines = assembly.split('\n')
    
    for line in lines:
        # Доступ к памяти через смещения
        # ldr x1, [x0, #offset] - доступ к полю структуры
        mem_match = re.search(r'ldr\s+(\w+),\s*\[(\w+)(?:,\s*#([0-9a-fx]+))?\]', line.lower())
        if mem_match:
            offset = mem_match.group(3)
            if offset:
                try:
                    offset_val = int(offset.replace('0x', ''), 16) if '0x' in offset else int(offset)
                    structures['field_offsets'].append({
                        'offset': offset_val,
                        'register': mem_match.group(2),
                        'line': line.strip()
                    })
                except:
                    pass
        
        # str x1, [x0, #offset] - запись в поле структуры
        str_match = re.search(r'str\s+(\w+),\s*\[(\w+)(?:,\s*#([0-9a-fx]+))?\]', line.lower())
        if str_match:
            offset = str_match.group(3)
            if offset:
                try:
                    offset_val = int(offset.replace('0x', ''), 16) if '0x' in offset else int(offset)
                    structures['field_offsets'].append({
                        'offset': offset_val,
                        'register': str_match.group(2),
                        'line': line.strip(),
                        'type': 'write'
                    })
                except:
                    pass
    
    # Группируем смещения по структурам
    offset_groups = defaultdict(list)
    for field in structures['field_offsets']:
        offset_groups[field['register']].append(field['offset'])
    
    for reg, offsets in offset_groups.items():
        offsets_sorted = sorted(set(offsets))
        if len(offsets_sorted) > 1:
            structures['struct_patterns'].append({
                'register': reg,
                'offsets': offsets_sorted,
                'size': max(offsets_sorted) - min(offsets_sorted) if offsets_sorted else 0
            })
    
    return structures

def analyze_servo_algorithm(func_name, func_addr):
    """Глубокий анализ servo алгоритма"""
    print(f"\n{'='*80}")
    print(f"ГЛУБОКИЙ АНАЛИЗ: {func_name}")
    print(f"{'='*80}")
    
    assembly = extract_function_assembly(func_name, func_addr)
    if not assembly:
        print("⚠ Не удалось извлечь ассемблерный код")
        return None
    
    # Анализ арифметических операций
    arithmetic = analyze_arithmetic_operations(assembly)
    print(f"\n📊 АРИФМЕТИЧЕСКИЕ ОПЕРАЦИИ:")
    print(f"  Сложений: {len(arithmetic['additions'])}")
    if arithmetic['additions']:
        print("  Примеры:")
        for op in arithmetic['additions'][:5]:
            print(f"    {op['line']}")
    
    print(f"  Вычитаний: {len(arithmetic['subtractions'])}")
    if arithmetic['subtractions']:
        print("  Примеры:")
        for op in arithmetic['subtractions'][:5]:
            print(f"    {op['line']}")
    
    print(f"  Умножений: {len(arithmetic['multiplications'])}")
    if arithmetic['multiplications']:
        print("  Примеры:")
        for op in arithmetic['multiplications'][:5]:
            print(f"    {op}")
    
    print(f"  Сдвигов: {len(arithmetic['shifts'])}")
    if arithmetic['shifts']:
        print("  Примеры:")
        for op in arithmetic['shifts'][:5]:
            print(f"    {op['line']} (тип: {op['type']}, бит: {op['bits']})")
    
    # Анализ констант
    constants = extract_constants_from_assembly(assembly)
    print(f"\n🔢 КОНСТАНТЫ:")
    print(f"  Hex значений: {len(constants['hex_values'])}")
    
    # Ищем интересные константы (возможно коэффициенты)
    interesting_constants = []
    for const in constants['hex_values']:
        val = const['decimal']
        # Проверяем, может ли это быть коэффициент (обычно небольшие числа или степени 2)
        if 1 <= val <= 1000000 or (val & (val - 1) == 0):  # Степень 2
            interesting_constants.append(const)
    
    if interesting_constants:
        print("  Интересные константы (возможно коэффициенты):")
        for const in interesting_constants[:10]:
            print(f"    0x{const['hex']} = {const['decimal']} (десятичное)")
    
    # Анализ потока управления
    flow = analyze_control_flow(assembly)
    print(f"\n🔄 ПОТОК УПРАВЛЕНИЯ:")
    print(f"  Ветвлений: {len(flow['branches'])}")
    print(f"  Вызовов функций: {len(flow['calls'])}")
    print(f"  Возвратов: {len(flow['returns'])}")
    print(f"  Условий: {len(flow['conditions'])}")
    print(f"  Циклов: {len(flow['loops'])}")
    
    if flow['calls']:
        print("  Вызываемые функции:")
        for call in flow['calls'][:10]:
            print(f"    - {call['function']}")
    
    # Анализ структур данных
    structures = extract_data_structures(assembly)
    print(f"\n📦 СТРУКТУРЫ ДАННЫХ:")
    print(f"  Доступов к памяти: {len(structures['field_offsets'])}")
    print(f"  Найдено паттернов структур: {len(structures['struct_patterns'])}")
    
    if structures['struct_patterns']:
        print("  Возможные структуры:")
        for struct in structures['struct_patterns'][:5]:
            print(f"    Регистр {struct['register']}:")
            print(f"      Смещения: {struct['offsets']}")
            print(f"      Размер: ~{struct['size']} байт")
    
    # Попытка определить алгоритм
    print(f"\n🧮 АНАЛИЗ АЛГОРИТМА:")
    
    # Проверяем на PID-подобный алгоритм
    if len(arithmetic['additions']) > 5 and len(arithmetic['multiplications']) > 0:
        print("  ⚠ Возможно PID-подобный алгоритм (много сложений и умножений)")
    
    # Проверяем на фильтр
    if len(arithmetic['shifts']) > 3:
        print("  ⚠ Возможно фильтр (много сдвигов)")
    
    # Проверяем на цикл
    if len(flow['loops']) > 0:
        print(f"  ⚠ Найдено {len(flow['loops'])} циклов")
    
    return {
        'arithmetic': arithmetic,
        'constants': constants,
        'flow': flow,
        'structures': structures,
        'assembly': assembly
    }

def analyze_ubx_tp5_structure():
    """Анализирует структуру UBXTP5Message"""
    print(f"\n{'='*80}")
    print("АНАЛИЗ СТРУКТУРЫ UBXTP5Message")
    print(f"{'='*80}")
    
    # Пробуем найти функцию через nm
    func_name = None
    func_addr = None
    
    # Известный адрес из предыдущего анализа
    known_addr = "0x0000000004087e30"
    
    # Ищем через nm
    nm_result = run_command(["nm", "-D", BINARY_PATH])
    if nm_result and not nm_result.startswith("Ошибка"):
        lines = nm_result.split('\n')
        for line in lines:
            if 'ubxtp5' in line.lower() and 'tobytes' in line.lower():
                match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                full_name_match = re.search(r'<([^>]+)>', line)
                if match and full_name_match:
                    func_addr = match.group(1)
                    func_name = full_name_match.group(1)
                    print(f"✓ Найдено: {func_name} (0x{func_addr})")
                    break
    
    # Если не нашли, используем известный адрес
    if not func_addr:
        # Убираем двойной префикс если есть
        known_addr_clean = known_addr.replace('0x0x', '0x').replace('0X0X', '0x')
        func_addr = known_addr_clean
        func_name = "github.com/lasselj/timebeat/beater/clocksync/clients/vendors/helper/ubx.(*UBXTP5Message).ToBytes"
        print(f"⚠ Используем известный адрес: {func_addr}")
    
    result = analyze_servo_algorithm(func_name, func_addr)
    
    if result and result['structures']:
        print("\n📋 ВОЗМОЖНАЯ СТРУКТУРА UBXTP5Message:")
        print("  На основе анализа смещений:")
        
        # Группируем смещения
        all_offsets = []
        for field in result['structures']['field_offsets']:
            all_offsets.append(field['offset'])
        
        offsets_sorted = sorted(set(all_offsets))
        if offsets_sorted:
            print("  Смещения полей:")
            for offset in offsets_sorted[:20]:
                print(f"    offset {offset}: (тип нужно определить)")
    
    return result

def analyze_servo_functions_deep():
    """Глубокий анализ servo функций"""
    print(f"\n{'='*80}")
    print("ГЛУБОКИЙ АНАЛИЗ SERVO ФУНКЦИЙ")
    print(f"{'='*80}")
    
    # Используем полные имена из графа вызовов (с @@Base суффиксом)
    servo_functions = [
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockUsingGetTimeSyscall@@Base", None),
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepClockUsingSetTimeSyscall@@Base", None),
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.PerformGranularityMeasurement@@Base", None),
        # Также пробуем без @@Base
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockUsingGetTimeSyscall", None),
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepClockUsingSetTimeSyscall", None),
        ("github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.PerformGranularityMeasurement", None),
        # И короткие имена
        ("GetClockUsingGetTimeSyscall", None),
        ("StepClockUsingSetTimeSyscall", None),
        ("PerformGranularityMeasurement", None)
    ]
    
    # Находим адреса через nm
    nm_result = run_command(["nm", "-D", BINARY_PATH])
    if nm_result and not nm_result.startswith("Ошибка"):
        lines = nm_result.split('\n')
        for line in lines:
            line_lower = line.lower()
            # Ищем функции servo/adjusttime
            if 'servo' in line_lower and 'adjusttime' in line_lower:
                for i, (name, _) in enumerate(servo_functions):
                    if servo_functions[i][1] is not None:  # Уже нашли
                        continue
                    
                    # Проверяем разные варианты имени
                    name_variants = [
                        name.lower(),
                        name.lower().replace('@@base', ''),
                        name.lower().replace('using', ''),
                        name.split('.')[-1].lower().replace('@@base', '') if '.' in name else name.lower(),
                        name.split('.')[-1].split('@@')[0].lower() if '.' in name and '@@' in name else name.lower()
                    ]
                    
                    for variant in name_variants:
                        if variant and variant in line_lower:
                            match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                            if match:
                                # Извлекаем полное имя функции
                                full_name_match = re.search(r'<([^>]+)>', line)
                                if full_name_match:
                                    full_name = full_name_match.group(1)
                                    addr = match.group(1)
                                    servo_functions[i] = (full_name, addr)
                                    print(f"  ✓ Найдено через nm: {full_name} (0x{addr})")
                                    break
    
    # Альтернативный поиск через objdump -T
    if any(addr is None for _, addr in servo_functions):
        print("\n  Пробуем альтернативный метод через objdump -T...")
        objdump_result = run_command(["objdump", "-T", BINARY_PATH])
        if objdump_result and not objdump_result.startswith("Ошибка"):
            lines = objdump_result.split('\n')
            for line in lines:
                line_lower = line.lower()
                if 'servo' in line_lower and 'adjusttime' in line_lower:
                    for i, (name, _) in enumerate(servo_functions):
                        if servo_functions[i][1] is None:  # Еще не нашли
                            name_variants = [
                                name.lower(),
                                name.lower().replace('using', ''),
                                name.split('.')[-1].lower() if '.' in name else name.lower()
                            ]
                            
                            for variant in name_variants:
                                if variant in line_lower:
                                    match = re.search(r'([0-9a-f]+)\s+.*?\s+<([^>]+)>', line)
                                    if match:
                                        addr = match.group(1)
                                        full_name = match.group(2)
                                        servo_functions[i] = (full_name, addr)
                                        print(f"  ✓ Найдено через objdump: {full_name} (0x{addr})")
                                        break
    
    # Удаляем дубликаты и анализируем только найденные
    found_functions = {}
    for func_name, func_addr in servo_functions:
        if func_addr:
            # Используем полное имя если есть
            short_name = func_name.split('.')[-1] if '.' in func_name else func_name
            if short_name not in found_functions:
                found_functions[short_name] = (func_name, func_addr)
    
    if found_functions:
        for short_name, (full_name, addr) in found_functions.items():
            print(f"\n{'='*60}")
            print(f"Анализ: {short_name}")
            analyze_servo_algorithm(full_name, addr)
    else:
        print("\n⚠ Servo функции не найдены через nm/objdump")
        print("Попробуйте найти вручную:")
        print(f"  nm -D {BINARY_PATH} | grep -i 'servo.*adjusttime'")
        print(f"  objdump -T {BINARY_PATH} | grep -i 'servo.*adjusttime'")

def search_mathematical_patterns():
    """Ищет математические паттерны в бинарнике"""
    print(f"\n{'='*80}")
    print("ПОИСК МАТЕМАТИЧЕСКИХ ПАТТЕРНОВ")
    print(f"{'='*80}")
    
    # Ищем известные математические константы
    math_constants = {
        'PI': [0x40490fdb, 0x400921fb54442d18],  # float и double PI
        'E': [0x402df854, 0x4005bf0a8b145769],   # float и double E
        '2PI': [0x40c90fdb, 0x401921fb54442d18]
    }
    
    # Читаем бинарник
    try:
        with open(BINARY_PATH, 'rb') as f:
            data = f.read()
        
        print("\n🔍 Поиск математических констант:")
        for name, values in math_constants.items():
            for val in values:
                # Ищем в little-endian
                val_bytes = pack('<I', val) if val < 0xFFFFFFFF else pack('<Q', val)
                count = data.count(val_bytes)
                if count > 0:
                    print(f"  {name} (0x{val:x}): найдено {count} раз")
    except Exception as e:
        print(f"Ошибка чтения бинарника: {e}")

def extract_error_handling():
    """Извлекает паттерны обработки ошибок"""
    print(f"\n{'='*80}")
    print("АНАЛИЗ ОБРАБОТКИ ОШИБОК")
    print(f"{'='*80}")
    
    # Ищем строки с ошибками
    strings_result = run_command(["strings", BINARY_PATH])
    if strings_result and not strings_result.startswith("Ошибка"):
        lines = strings_result.split('\n')
        
        error_patterns = []
        for line in lines:
            line_lower = line.lower()
            if any(keyword in line_lower for keyword in ['error', 'fail', 'invalid', 'unable', 'cannot']):
                if len(line) < 200:  # Разумная длина
                    error_patterns.append(line)
        
        print(f"\nНайдено {len(error_patterns)} строк с ошибками:")
        for pattern in error_patterns[:20]:
            print(f"  - {pattern}")

def main():
    print("=" * 80)
    print("ГЛУБОКИЙ АНАЛИЗ БИНАРНИКА SHIWATIME")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # 1. Анализ UBX TP5 структуры
    analyze_ubx_tp5_structure()
    
    # 2. Глубокий анализ servo функций
    analyze_servo_functions_deep()
    
    # 3. Поиск математических паттернов
    search_mathematical_patterns()
    
    # 4. Анализ обработки ошибок
    extract_error_handling()
    
    print("\n" + "=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

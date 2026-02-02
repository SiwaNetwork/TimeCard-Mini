#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Извлечение детальных алгоритмов из бинарника
Фокусируется на математических операциях, формулах и логике
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

def find_function_address(func_name):
    """Находит адрес функции"""
    # Пробуем разные варианты имени
    name_variants = [
        func_name,
        func_name.replace('@@Base', ''),
        func_name.split('.')[-1] if '.' in func_name else func_name,
        func_name.split('.')[-1].split('@@')[0] if '.' in func_name and '@@' in func_name else func_name
    ]
    
    # Метод 1: через nm
    result = run_command(["nm", "-D", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        for line in lines:
            for variant in name_variants:
                if variant and variant in line:
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    if match:
                        return match.group(1)
    
    # Метод 2: через objdump -T
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        for line in lines:
            for variant in name_variants:
                if variant and variant in line:
                    match = re.search(r'([0-9a-f]+)\s+.*?\s+<([^>]+)>', line)
                    if match:
                        return match.group(1)
    
    return None

def disassemble_range(start_addr, end_addr):
    """Дизассемблирует диапазон адресов"""
    try:
        result = run_command([
            "objdump", "-d", "-C",
            "--start-address", f"0x{start_addr}",
            "--stop-address", f"0x{end_addr}",
            BINARY_PATH
        ])
        return result if not result.startswith("Ошибка") else None
    except:
        return None

def extract_register_operations(assembly):
    """Извлекает операции с регистрами для понимания алгоритма"""
    operations = []
    
    if not assembly:
        return operations
    
    lines = assembly.split('\n')
    register_state = defaultdict(list)
    
    for line in lines:
        # Паттерн: операция регистр1, регистр2, значение
        # add x1, x2, #0x100
        patterns = [
            (r'(add|sub|mul)\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', 'arithmetic'),
            (r'(lsl|lsr|asr)\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', 'shift'),
            (r'(and|orr|eor)\s+(\w+),\s*(\w+)(?:,\s*#?([0-9a-fx]+))?', 'bitwise'),
        ]
        
        for pattern, op_type in patterns:
            match = re.search(pattern, line.lower())
            if match:
                operations.append({
                    'type': op_type,
                    'operation': match.group(1),
                    'dest': match.group(2),
                    'src': match.group(3),
                    'imm': match.group(4) if match.group(4) else None,
                    'line': line.strip()
                })
                register_state[match.group(2)].append({
                    'op': match.group(1),
                    'src': match.group(3),
                    'imm': match.group(4)
                })
    
    return operations, register_state

def reconstruct_algorithm(operations, register_state):
    """Пытается восстановить алгоритм из операций"""
    algorithm = {
        'steps': [],
        'variables': set(),
        'constants': [],
        'formula_hints': []
    }
    
    # Собираем переменные
    for op in operations:
        algorithm['variables'].add(op['dest'])
        algorithm['variables'].add(op['src'])
        if op['imm']:
            algorithm['constants'].append(op['imm'])
    
    # Пытаемся найти паттерны
    # Паттерн 1: PID-подобный (error * kp + integral * ki + derivative * kd)
    additions = [op for op in operations if op['type'] == 'arithmetic' and op['operation'] == 'add']
    multiplications = [op for op in operations if op['type'] == 'arithmetic' and op['operation'] == 'mul']
    
    if len(additions) >= 3 and len(multiplications) >= 2:
        algorithm['formula_hints'].append("Возможно PID-подобный алгоритм")
    
    # Паттерн 2: Фильтр (среднее, экспоненциальное сглаживание)
    shifts = [op for op in operations if op['type'] == 'shift']
    if len(shifts) >= 3:
        algorithm['formula_hints'].append("Возможно фильтр (много сдвигов)")
    
    # Паттерн 3: Накопитель (интеграл)
    add_patterns = defaultdict(int)
    for op in additions:
        key = f"{op['dest']} = {op['src']}"
        add_patterns[key] += 1
    
    for key, count in add_patterns.items():
        if count >= 3:
            algorithm['formula_hints'].append(f"Возможно накопитель: {key}")
    
    return algorithm

def analyze_pid_algorithm(func_name):
    """Специальный анализ для PID-подобных алгоритмов"""
    print(f"\n{'='*80}")
    print(f"АНАЛИЗ PID-ПОДОБНОГО АЛГОРИТМА: {func_name}")
    print(f"{'='*80}")
    
    addr = find_function_address(func_name)
    if not addr:
        print(f"⚠ Функция {func_name} не найдена")
        return None
    
    try:
        addr_int = int(addr, 16)
        start = f"{addr_int:016x}"
        end = f"{addr_int + 0x2000:016x}"
        
        assembly = disassemble_range(start, end)
        if not assembly:
            print("⚠ Не удалось дизассемблировать")
            return None
        
        operations, register_state = extract_register_operations(assembly)
        algorithm = reconstruct_algorithm(operations, register_state)
        
        print(f"\n📊 Найдено операций: {len(operations)}")
        print(f"📊 Переменных: {len(algorithm['variables'])}")
        print(f"📊 Констант: {len(set(algorithm['constants']))}")
        
        print(f"\n🔍 Подсказки алгоритма:")
        for hint in algorithm['formula_hints']:
            print(f"  - {hint}")
        
        # Ищем коэффициенты
        print(f"\n🔢 Возможные коэффициенты:")
        unique_constants = sorted(set(algorithm['constants']), key=lambda x: int(x.replace('0x', ''), 16) if '0x' in x else int(x))
        for const in unique_constants[:10]:
            try:
                val = int(const.replace('0x', ''), 16) if '0x' in const else int(const)
                if 1 <= val <= 1000000:
                    print(f"  {const} = {val}")
            except:
                pass
        
        return {
            'operations': operations,
            'algorithm': algorithm,
            'assembly': assembly
        }
    except Exception as e:
        print(f"Ошибка: {e}")
        return None

def analyze_time_calculation():
    """Анализирует вычисления времени"""
    print(f"\n{'='*80}")
    print("АНАЛИЗ ВЫЧИСЛЕНИЙ ВРЕМЕНИ")
    print(f"{'='*80}")
    
    # Ищем функции, связанные с временем
    time_functions = [
        "GetClockUsingGetTimeSyscall",
        "StepClockUsingSetTimeSyscall",
        "GetTimeNow",
        "PerformGranularityMeasurement"
    ]
    
    nm_result = run_command(["nm", "-D", BINARY_PATH])
    if not nm_result or nm_result.startswith("Ошибка"):
        print("⚠ Не удалось получить список символов")
        return
    
    lines = nm_result.split('\n')
    
    for func_name in time_functions:
        print(f"\n🔍 Поиск функции: {func_name}")
        found = False
        
        for line in lines:
            line_lower = line.lower()
            if func_name.lower() in line_lower and 'clocksync' in line_lower:
                match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                full_name_match = re.search(r'<([^>]+)>', line)
                
                if match and full_name_match:
                    full_name = full_name_match.group(1)
                    addr = match.group(1)
                    print(f"  ✓ Найдено: {full_name} (0x{addr})")
                    analyze_pid_algorithm(full_name)
                    found = True
                    break
        
        if not found:
            print(f"  ⚠ Функция {func_name} не найдена")

def extract_ubx_structure_fields():
    """Извлекает поля структуры UBXTP5Message"""
    print(f"\n{'='*80}")
    print("ИЗВЛЕЧЕНИЕ ПОЛЕЙ СТРУКТУРЫ UBXTP5Message")
    print(f"{'='*80}")
    
    func_name = "UBXTP5Message.ToBytes"
    addr = find_function_address(func_name)
    
    if not addr:
        # Пробуем найти через частичное совпадение
        nm_result = run_command(["nm", "-D", BINARY_PATH, "|", "grep", "-i", "ubxtp5"], shell=True)
        if nm_result:
            lines = nm_result.split('\n')
            for line in lines:
                if 'tp5' in line.lower() and 'tobytes' in line.lower():
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    if match:
                        addr = match.group(1)
                        break
    
    if not addr:
        print("⚠ Функция не найдена")
        return
    
    try:
        addr_int = int(addr, 16)
        start = f"{addr_int:016x}"
        end = f"{addr_int + 0x2000:016x}"
        
        assembly = disassemble_range(start, end)
        if not assembly:
            print("⚠ Не удалось дизассемблировать")
            return
        
        # Ищем доступы к полям структуры
        # ldr x1, [x0, #offset] - чтение поля
        # str x1, [x0, #offset] - запись поля
        
        field_accesses = []
        lines = assembly.split('\n')
        
        for line in lines:
            # Чтение
            ldr_match = re.search(r'ldr\s+\w+,\s*\[(\w+)(?:,\s*#([0-9a-fx]+))?\]', line.lower())
            if ldr_match:
                offset = ldr_match.group(2)
                if offset:
                    try:
                        offset_val = int(offset.replace('0x', ''), 16) if '0x' in offset else int(offset)
                        field_accesses.append({
                            'offset': offset_val,
                            'type': 'read',
                            'line': line.strip()
                        })
                    except:
                        pass
            
            # Запись
            str_match = re.search(r'str\s+\w+,\s*\[(\w+)(?:,\s*#([0-9a-fx]+))?\]', line.lower())
            if str_match:
                offset = str_match.group(2)
                if offset:
                    try:
                        offset_val = int(offset.replace('0x', ''), 16) if '0x' in offset else int(offset)
                        field_accesses.append({
                            'offset': offset_val,
                            'type': 'write',
                            'line': line.strip()
                        })
                    except:
                        pass
        
        # Группируем по смещениям
        offset_groups = defaultdict(lambda: {'read': 0, 'write': 0})
        for access in field_accesses:
            offset_groups[access['offset']][access['type']] += 1
        
        print(f"\n📦 Найдено {len(offset_groups)} уникальных смещений:")
        for offset in sorted(offset_groups.keys()):
            group = offset_groups[offset]
            print(f"  offset {offset:4d} (0x{offset:04x}): read={group['read']}, write={group['write']}")
    except Exception as e:
        print(f"Ошибка при анализе: {e}")

def main():
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ ДЕТАЛЬНЫХ АЛГОРИТМОВ")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    # 1. Анализ вычислений времени
    analyze_time_calculation()
    
    # 2. Извлечение полей структуры UBXTP5Message
    extract_ubx_structure_fields()
    
    print("\n" + "=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""
Извлечение алгоритмов через анализ ассемблера
- Понимание логики работы функций
- Извлечение математических операций
- Анализ циклов и условий
- Построение блок-схем алгоритмов

Использование:
    sudo python3 extract_algorithms.py
"""

import os
import sys
import subprocess
import re
from collections import defaultdict

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

def disassemble_function_detailed(func_name):
    """Детальное дизассемблирование функции с анализом"""
    result = run_command(["objdump", "-d", "-C", "-S", BINARY_PATH])
    if result.startswith("Ошибка"):
        return None
    
    lines = result.split('\n')
    in_function = False
    function_code = []
    addresses = []
    
    for line in lines:
        if f'<{func_name}>:' in line:
            in_function = True
            function_code.append(line)
            # Извлекаем адрес
            addr_match = re.search(r'([0-9a-f]+)\s+<', line)
            if addr_match:
                addresses.append(addr_match.group(1))
            continue
        
        if in_function:
            if line.strip() == '':
                continue
            if 'Disassembly' in line and ':' in line and '<' not in line:
                break
            if '<' in line and '>:' in line and func_name not in line:
                # Другая функция
                break
            
            function_code.append(line)
            # Извлекаем адреса инструкций
            addr_match = re.search(r'^\s*([0-9a-f]+):', line)
            if addr_match:
                addresses.append(addr_match.group(1))
    
    return {
        'code': '\n'.join(function_code),
        'addresses': addresses,
        'size': len(addresses)
    }

def analyze_control_flow(assembly_code):
    """Анализирует поток управления в функции"""
    if not assembly_code:
        return None
    
    flow = {
        'branches': [],
        'loops': [],
        'calls': [],
        'returns': []
    }
    
    lines = assembly_code.split('\n')
    addresses = {}
    
    for i, line in enumerate(lines):
        # Извлекаем адрес инструкции
        addr_match = re.search(r'^\s*([0-9a-f]+):', line)
        current_addr = addr_match.group(1) if addr_match else None
        
        line_lower = line.lower()
        
        # Условные переходы
        if any(branch in line_lower for branch in ['je', 'jne', 'jz', 'jnz', 'jl', 'jg', 'jle', 'jge']):
            target_match = re.search(r'([0-9a-f]+)\s+<', line)
            if target_match:
                flow['branches'].append({
                    'from': current_addr,
                    'to': target_match.group(1),
                    'type': 'conditional',
                    'line': line.strip()
                })
        
        # Безусловные переходы
        if 'jmp' in line_lower:
            target_match = re.search(r'([0-9a-f]+)\s+<', line)
            if target_match:
                flow['branches'].append({
                    'from': current_addr,
                    'to': target_match.group(1),
                    'type': 'unconditional',
                    'line': line.strip()
                })
        
        # Вызовы функций
        if 'call' in line_lower or ' bl ' in line_lower:
            call_match = re.search(r'<([^>]+)>', line)
            if call_match:
                flow['calls'].append({
                    'from': current_addr,
                    'to': call_match.group(1),
                    'line': line.strip()
                })
        
        # Возвраты
        if 'ret' in line_lower or 'bx lr' in line_lower:
            flow['returns'].append({
                'from': current_addr,
                'line': line.strip()
            })
    
    # Определяем циклы (переходы назад)
    for branch in flow['branches']:
        if branch['from'] and branch['to']:
            try:
                from_addr = int(branch['from'], 16)
                to_addr = int(branch['to'], 16)
                if to_addr < from_addr:
                    flow['loops'].append(branch)
            except:
                pass
    
    return flow

def extract_mathematical_operations(assembly_code):
    """Извлекает математические операции"""
    if not assembly_code:
        return []
    
    operations = []
    lines = assembly_code.split('\n')
    
    for line in lines:
        line_lower = line.lower()
        
        # Арифметические операции
        if any(op in line_lower for op in ['add', 'sub', 'mul', 'div', 'imul', 'idiv']):
            operations.append({
                'type': 'arithmetic',
                'operation': next(op for op in ['add', 'sub', 'mul', 'div', 'imul', 'idiv'] if op in line_lower),
                'line': line.strip()
            })
        
        # Битовые операции
        if any(op in line_lower for op in ['and', 'or', 'xor', 'shl', 'shr', 'rol', 'ror']):
            operations.append({
                'type': 'bitwise',
                'operation': next(op for op in ['and', 'or', 'xor', 'shl', 'shr', 'rol', 'ror'] if op in line_lower),
                'line': line.strip()
            })
        
        # Сравнения
        if 'cmp' in line_lower:
            operations.append({
                'type': 'comparison',
                'operation': 'cmp',
                'line': line.strip()
            })
    
    return operations

def analyze_function_complexity(assembly_code):
    """Анализирует сложность функции"""
    if not assembly_code:
        return None
    
    lines = assembly_code.split('\n')
    
    complexity = {
        'total_instructions': 0,
        'branches': 0,
        'calls': 0,
        'loops': 0,
        'complexity_score': 0
    }
    
    for line in lines:
        if re.search(r'^\s*[0-9a-f]+:', line):
            complexity['total_instructions'] += 1
            
            line_lower = line.lower()
            
            if any(branch in line_lower for branch in ['je', 'jne', 'jz', 'jnz', 'jl', 'jg', 'jmp']):
                complexity['branches'] += 1
            
            if 'call' in line_lower or ' bl ' in line_lower:
                complexity['calls'] += 1
    
    # Простая оценка сложности
    complexity['complexity_score'] = (
        complexity['total_instructions'] * 1 +
        complexity['branches'] * 2 +
        complexity['calls'] * 3
    )
    
    return complexity

def extract_data_structures(assembly_code):
    """Пытается определить структуры данных"""
    if not assembly_code:
        return []
    
    structures = []
    lines = assembly_code.split('\n')
    
    # Ищем доступ к структурам (смещения)
    for line in lines:
        # Паттерн: mov reg, [reg + offset]
        struct_match = re.search(r'\[([a-z0-9]+)\s*[+\-]\s*([0-9a-fx]+)\]', line, re.IGNORECASE)
        if struct_match:
            offset = struct_match.group(2)
            try:
                offset_val = int(offset, 16) if 'x' in offset.lower() else int(offset)
                if 0 < offset_val < 1000:  # Разумный размер структуры
                    structures.append({
                        'offset': offset_val,
                        'line': line.strip()
                    })
            except:
                pass
    
    return structures

def analyze_ubx_tp5_algorithm():
    """Специальный анализ алгоритма работы с UBX TP5"""
    print("=" * 80)
    print("АНАЛИЗ АЛГОРИТМА UBX TP5")
    print("=" * 80)
    
    # Ищем функции, связанные с TP5
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result.startswith("Ошибка"):
        print(f"Ошибка: {result}")
        return
    
    lines = result.split('\n')
    tp5_functions = []
    
    for line in lines:
        line_lower = line.lower()
        if 'tp5' in line_lower or 'timepulse' in line_lower or 'ubx' in line_lower:
            match = re.search(r'<([^>]+)>', line)
            if match:
                func_name = match.group(1)
                # Проверяем, что это действительно функция из clocksync
                if 'clocksync' in func_name.lower() or 'ubx' in func_name.lower():
                    tp5_functions.append(func_name)
    
    print(f"\nНайдено {len(tp5_functions)} функций, связанных с TP5")
    
    for func_name in tp5_functions[:5]:
        print(f"\n{'='*60}")
        print(f"ФУНКЦИЯ: {func_name}")
        print(f"{'='*60}")
        
        func_data = disassemble_function_detailed(func_name)
        if func_data:
            print(f"Размер: {func_data['size']} инструкций")
            
            # Анализ потока управления
            flow = analyze_control_flow(func_data['code'])
            if flow:
                print(f"\nПоток управления:")
                print(f"  Ветвления: {len(flow['branches'])}")
                print(f"  Циклы: {len(flow['loops'])}")
                print(f"  Вызовы: {len(flow['calls'])}")
                print(f"  Возвраты: {len(flow['returns'])}")
                
                if flow['calls']:
                    print(f"\n  Вызываемые функции:")
                    for call in flow['calls'][:10]:
                        print(f"    - {call['to']}")
            
            # Математические операции
            math_ops = extract_mathematical_operations(func_data['code'])
            if math_ops:
                print(f"\nМатематические операции: {len(math_ops)}")
                op_types = defaultdict(int)
                for op in math_ops:
                    op_types[op['type']] += 1
                for op_type, count in op_types.items():
                    print(f"  {op_type}: {count}")
            
            # Сложность
            complexity = analyze_function_complexity(func_data['code'])
            if complexity:
                print(f"\nСложность:")
                print(f"  Всего инструкций: {complexity['total_instructions']}")
                print(f"  Ветвления: {complexity['branches']}")
                print(f"  Вызовы: {complexity['calls']}")
                print(f"  Оценка сложности: {complexity['complexity_score']}")
            
            # Структуры данных
            structures = extract_data_structures(func_data['code'])
            if structures:
                print(f"\nСтруктуры данных:")
                offsets = sorted(set(s['offset'] for s in structures))
                print(f"  Найдено смещений: {len(offsets)}")
                print(f"  Примеры: {', '.join(map(str, offsets[:10]))}")
    
    print()

def analyze_servo_algorithm():
    """Специальный анализ servo алгоритмов"""
    print("=" * 80)
    print("АНАЛИЗ SERVO АЛГОРИТМОВ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result.startswith("Ошибка"):
        print(f"Ошибка: {result}")
        return
    
    lines = result.split('\n')
    servo_functions = []
    
    keywords = ['servo', 'dpll', 'sync', 'adjust', 'offset', 'frequency', 'phase', 'lock']
    
    for line in lines:
        line_lower = line.lower()
        if any(keyword in line_lower for keyword in keywords):
            match = re.search(r'<([^>]+)>', line)
            if match:
                func_name = match.group(1)
                # Проверяем, что это действительно servo функция из clocksync
                if 'clocksync' in func_name.lower() and any(kw in func_name.lower() for kw in keywords):
                    servo_functions.append(func_name)
    
    print(f"\nНайдено {len(servo_functions)} функций, связанных с servo")
    
    if not servo_functions:
        print("\n⚠ Servo функции не найдены")
        return
    
    for func_name in servo_functions[:10]:
        print(f"\n{'='*60}")
        print(f"ФУНКЦИЯ: {func_name}")
        print(f"{'='*60}")
        
        func_data = disassemble_function_detailed(func_name)
        if func_data:
            print(f"Размер: {func_data['size']} инструкций")
            
            # Анализ
            flow = analyze_control_flow(func_data['code'])
            complexity = analyze_function_complexity(func_data['code'])
            math_ops = extract_mathematical_operations(func_data['code'])
            
            if flow:
                print(f"  Ветвления: {len(flow['branches'])}")
                print(f"  Циклы: {len(flow['loops'])}")
                print(f"  Вызовы: {len(flow['calls'])}")
            
            if complexity:
                print(f"  Сложность: {complexity['complexity_score']}")
            
            if math_ops:
                print(f"  Математических операций: {len(math_ops)}")
                # Ищем операции, связанные с вычислениями
                calc_ops = [op for op in math_ops if op['type'] in ['arithmetic', 'comparison']]
                if calc_ops:
                    print(f"    Вычислительных: {len(calc_ops)}")
    
    print()

def main():
    print("=" * 80)
    print("ИЗВЛЕЧЕНИЕ АЛГОРИТМОВ ЧЕРЕЗ АНАЛИЗ АССЕМБЛЕРА")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # Анализ UBX TP5 алгоритма
    analyze_ubx_tp5_algorithm()
    
    # Анализ Servo алгоритмов
    analyze_servo_algorithm()
    
    print("=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    print()
    print("Для сохранения результатов:")
    print("  sudo python3 extract_algorithms.py > algorithms_analysis.txt 2>&1")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""
Продвинутый анализ бинарника shiwatime
- Дизассемблирование функций clocksync модулей
- Построение полного графа вызовов
- Извлечение алгоритмов через анализ ассемблера
- Понимание логики работы функций

Использование:
    sudo python3 advanced_binary_analysis.py
"""

import os
import sys
import subprocess
import re
from collections import defaultdict, deque
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
    except Exception as e:
        return f"Ошибка: {str(e)}"

def get_all_clocksync_functions():
    """Получает список всех функций из clocksync модулей"""
    print("=" * 80)
    print("1. ПОЛУЧЕНИЕ СПИСКА ФУНКЦИЙ CLOCKSYNC")
    print("=" * 80)
    
    functions = []
    
    # Метод 1: через objdump -T
    result = run_command(["objdump", "-T", BINARY_PATH])
    if not result.startswith("Ошибка"):
        lines = result.split('\n')
        for line in lines:
            line_lower = line.lower()
            if any(module in line_lower for module in TARGET_MODULES) or \
               any(keyword in line_lower for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll']):
                match = re.search(r'([0-9a-f]+)\s+.*?\s+<([^>]+)>', line)
                if match:
                    addr = match.group(1)
                    func_name = match.group(2)
                    # Исключаем AWS SDK и другие нерелевантные модули
                    if any(exclude in func_name.lower() for exclude in ['aws', 'ec2', 'k8s.io', 'kubernetes', 'google.protobuf']):
                        continue
                    # Проверяем, что это действительно функция из clocksync
                    if 'clocksync' in func_name.lower() or \
                       any(module in func_name.lower() for module in TARGET_MODULES) or \
                       any(keyword in func_name.lower() for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll']):
                        functions.append((addr, func_name, line))
    
    # Метод 2: через nm (если objdump не дал результатов)
    if len(functions) == 0:
        print("Пробуем альтернативный метод через nm...")
        result = run_command(["nm", "-D", BINARY_PATH])
        if not result.startswith("Ошибка"):
            lines = result.split('\n')
            for line in lines:
                line_lower = line.lower()
                if any(module in line_lower for module in TARGET_MODULES) or \
                   any(keyword in line_lower for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll']):
                    # Формат nm: адрес тип имя
                    match = re.search(r'([0-9a-f]+)\s+[Tt]\s+(.+)', line)
                    if match:
                        addr = match.group(1)
                        func_name = match.group(2)
                        # Исключаем AWS SDK и другие нерелевантные модули
                        if any(exclude in func_name.lower() for exclude in ['aws', 'ec2', 'k8s.io', 'kubernetes', 'google.protobuf']):
                            continue
                        # Проверяем, что это действительно функция из clocksync
                        if 'clocksync' in func_name.lower() or \
                           any(module in func_name.lower() for module in TARGET_MODULES) or \
                           any(keyword in func_name.lower() for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll']):
                            functions.append((addr, func_name, line))
    
    # Удаляем дубликаты
    seen = set()
    unique_functions = []
    for func in functions:
        if func[1] not in seen:
            seen.add(func[1])
            unique_functions.append(func)
    
    functions = unique_functions
    
    print(f"Найдено {len(functions)} функций")
    if functions:
        print(f"Первые 20:")
        for addr, func_name, _ in functions[:20]:
            print(f"  {addr} - {func_name}")
    else:
        print("⚠ Функции не найдены. Возможно, бинарник stripped или функции имеют другие имена.")
    
    return functions

def disassemble_function(func_name, func_addr=None):
    """Дизассемблирует конкретную функцию по имени или адресу"""
    # Если есть адрес, используем его для более точного поиска
    if func_addr:
        # Пробуем дизассемблировать диапазон вокруг адреса
        try:
            addr_int = int(func_addr, 16)
            start_addr = f"0x{addr_int:016x}"
            end_addr = f"0x{addr_int + 0x1000:016x}"  # +4KB для функции
            result = run_command(["objdump", "-d", "-C", "--start-address", start_addr, 
                                 "--stop-address", end_addr, BINARY_PATH])
            if not result.startswith("Ошибка"):
                return result
        except:
            pass
    
    # Fallback: ищем по имени через objdump с фильтрацией
    # Пробуем дизассемблировать весь файл и найти функцию (может быть медленно)
    # Альтернатива: используем addr2line для получения информации
    try:
        # Пробуем через простой поиск в objdump
        full_result = run_command(["objdump", "-d", "-C", BINARY_PATH])
        if not full_result.startswith("Ошибка"):
            lines = full_result.split('\n')
            in_func = False
            func_lines = []
            for line in lines:
                if f'<{func_name}>:' in line:
                    in_func = True
                    func_lines.append(line)
                    continue
                if in_func:
                    if line.strip() == '' or ('<' in line and '>:' in line and func_name not in line):
                        break
                    func_lines.append(line)
            if func_lines:
                return '\n'.join(func_lines)
    except:
        pass
    if result.startswith("Ошибка") or not result.strip():
        # Последняя попытка: через nm находим адрес и дизассемблируем
        nm_result = run_command(["nm", "-D", BINARY_PATH, "|", "grep", func_name], shell=True)
        if nm_result and not nm_result.startswith("Ошибка"):
            lines = nm_result.split('\n')
            for line in lines:
                if func_name in line:
                    match = re.search(r'([0-9a-f]+)\s+[Tt]', line)
                    if match:
                        addr = match.group(1)
                        return disassemble_function(func_name, addr)
    
    return result if result and not result.startswith("Ошибка") else None

def analyze_assembly_patterns(assembly_code):
    """Анализирует паттерны в ассемблерном коде"""
    patterns = {
        'calls': [],
        'loops': [],
        'conditions': [],
        'constants': [],
        'memory_access': [],
        'key_operations': []
    }
    
    if not assembly_code:
        return patterns
    
    lines = assembly_code.split('\n')
    
    for line in lines:
        # Поиск вызовов функций
        if 'call' in line.lower() or ' bl ' in line.lower():
            call_match = re.search(r'<([^>]+)>', line)
            if call_match:
                patterns['calls'].append(call_match.group(1))
        
        # Поиск циклов (условные переходы назад)
        if 'j' in line.lower() and any(branch in line.lower() for branch in ['jne', 'je', 'jz', 'jnz', 'jl', 'jg']):
            patterns['loops'].append(line.strip())
        
        # Поиск условий
        if 'cmp' in line.lower() or 'test' in line.lower():
            patterns['conditions'].append(line.strip())
        
        # Поиск констант (hex значения)
        hex_match = re.search(r'0x([0-9a-f]+)', line, re.IGNORECASE)
        if hex_match:
            hex_val = hex_match.group(1)
            if len(hex_val) <= 8:  # Разумный размер
                patterns['constants'].append(hex_val)
        
        # Поиск доступа к памяти
        if 'mov' in line.lower() and ('[' in line or 'ptr' in line.lower()):
            patterns['memory_access'].append(line.strip())
        
        # Поиск арифметических операций
        if any(op in line.lower() for op in ['add', 'sub', 'mul', 'div', 'and', 'orr', 'eor', 'lsl', 'lsr']):
            patterns['key_operations'].append('arithmetic')
    
    return patterns

def build_call_graph(functions):
    """Строит граф вызовов функций"""
    print("=" * 80)
    print("2. ПОСТРОЕНИЕ ГРАФА ВЫЗОВОВ")
    print("=" * 80)
    
    call_graph = defaultdict(set)
    function_addrs = {func_name: addr for addr, func_name, _ in functions}
    
    # Дизассемблируем все функции и ищем вызовы
    print("Анализ вызовов функций...")
    
    result = run_command(["objdump", "-d", "-C", BINARY_PATH])
    if result.startswith("Ошибка"):
        print(f"Ошибка: {result}")
        return call_graph
    
    lines = result.split('\n')
    current_function = None
    
    for i, line in enumerate(lines):
        # Определяем начало функции
        func_match = re.search(r'<([^>]+)>:', line)
        if func_match:
            func_name = func_match.group(1)
            if any(module in func_name.lower() for module in TARGET_MODULES) or \
               any(keyword in func_name.lower() for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll', 'clocksync']):
                current_function = func_name
            else:
                current_function = None
            continue
        
        # Ищем вызовы функций
        if current_function:
            if 'call' in line.lower() or ' bl ' in line.lower():
                call_match = re.search(r'<([^>]+)>', line)
                if call_match:
                    called_func = call_match.group(1)
                    # Проверяем, что вызываемая функция тоже из наших модулей
                    if any(module in called_func.lower() for module in TARGET_MODULES) or \
                       any(keyword in called_func.lower() for keyword in ['ubx', 'timepulse', 'tp5', 'servo', 'dpll', 'clocksync']):
                        call_graph[current_function].add(called_func)
    
    print(f"\nПостроен граф с {len(call_graph)} узлами")
    print(f"Всего связей: {sum(len(calls) for calls in call_graph.values())}")
    
    return call_graph

def analyze_key_functions(functions):
    """Анализирует ключевые функции детально"""
    print("=" * 80)
    print("3. ДЕТАЛЬНЫЙ АНАЛИЗ КЛЮЧЕВЫХ ФУНКЦИЙ")
    print("=" * 80)
    
    key_functions = [
        'send1PPSOnTimepulsePin',
        'detectUbloxUnit',
        'sendUbxAidHuiNavMessage',
        'sendUbxNavTimelsMessage',
        'UBXTP5Message',
        'ToBytes'
    ]
    
    found_functions = {}
    for addr, func_name, _ in functions:
        for key_func in key_functions:
            if key_func.lower() in func_name.lower():
                found_functions[func_name] = addr
                break
    
    print(f"\nНайдено {len(found_functions)} ключевых функций для анализа:")
    
    for func_name, addr in list(found_functions.items())[:10]:
        print(f"\n{'='*60}")
        print(f"ФУНКЦИЯ: {func_name}")
        print(f"Адрес: 0x{addr}")
        print(f"{'='*60}")
        
        # Дизассемблируем (ищем адрес функции)
        func_addr = None
        for a, n, _ in functions:
            if n == func_name:
                func_addr = a
                break
        assembly = disassemble_function(func_name, func_addr)
        if assembly:
            lines = assembly.split('\n')
            print(f"Размер: {len(lines)} строк ассемблера")
            print("\nПервые 30 строк:")
            print('\n'.join(lines[:30]))
            
            # Анализируем паттерны
            patterns = analyze_assembly_patterns(assembly)
            print(f"\nПаттерны:")
            print(f"  Вызовы функций: {len(patterns['calls'])}")
            if patterns['calls']:
                print(f"    {', '.join(patterns['calls'][:10])}")
            print(f"  Условия: {len(patterns['conditions'])}")
            print(f"  Константы: {len(set(patterns['constants']))}")
            if patterns['constants']:
                unique_consts = list(set(patterns['constants']))[:10]
                print(f"    Примеры: {', '.join(['0x' + c for c in unique_consts])}")
        else:
            print("Не удалось дизассемблировать")
    
    print()

def extract_algorithm_logic(func_name, assembly_code):
    """Пытается извлечь логику алгоритма из ассемблера"""
    if not assembly_code:
        return None
    
    logic = {
        'function': func_name,
        'steps': [],
        'key_operations': []
    }
    
    lines = assembly_code.split('\n')
    
    for line in lines:
        line_lower = line.lower()
        
        # Определяем операции
        if 'mov' in line_lower:
            logic['key_operations'].append('data_move')
        elif 'add' in line_lower or 'sub' in line_lower:
            logic['key_operations'].append('arithmetic')
        elif 'cmp' in line_lower:
            logic['key_operations'].append('comparison')
        elif 'call' in line_lower:
            logic['key_operations'].append('function_call')
        elif 'j' in line_lower and any(branch in line_lower for branch in ['jne', 'je', 'jz', 'jnz']):
            logic['key_operations'].append('conditional_jump')
    
    return logic

def analyze_ubx_functions(functions):
    """Специальный анализ UBX функций"""
    print("=" * 80)
    print("4. СПЕЦИАЛЬНЫЙ АНАЛИЗ UBX ФУНКЦИЙ")
    print("=" * 80)
    
    ubx_functions = [(addr, name, line) for addr, name, line in functions if 'ubx' in name.lower()]
    
    print(f"\nНайдено {len(ubx_functions)} UBX функций")
    
    # Анализируем ToBytes функции
    tobytes_funcs = [f for f in ubx_functions if 'tobytes' in f[1].lower()]
    
    print(f"\nToBytes функции: {len(tobytes_funcs)}")
    for addr, name, _ in tobytes_funcs[:5]:
        print(f"\n  {name} (0x{addr}):")
        assembly = disassemble_function(name, addr)
        if assembly:
            patterns = analyze_assembly_patterns(assembly)
            print(f"    Вызовов: {len(patterns['calls'])}")
            print(f"    Условий: {len(patterns['conditions'])}")
            print(f"    Констант: {len(set(patterns['constants']))}")
            
            # Ищем UBX константы
            ubx_constants = []
            for const in set(patterns['constants']):
                try:
                    val = int(const, 16)
                    if val == 0xb562 or val == 0x06 or val == 0x31:
                        ubx_constants.append(f"0x{const} (UBX related)")
                except:
                    pass
            
            if ubx_constants:
                print(f"    UBX константы: {', '.join(ubx_constants)}")
    
    print()

def analyze_servo_functions(functions):
    """Специальный анализ Servo функций"""
    print("=" * 80)
    print("5. СПЕЦИАЛЬНЫЙ АНАЛИЗ SERVO ФУНКЦИЙ")
    print("=" * 80)
    
    # Ищем функции, связанные с servo, dpll, sync, adjust
    servo_keywords = ['servo', 'dpll', 'sync', 'adjust', 'offset', 'frequency', 'phase']
    
    servo_functions = []
    for addr, name, line in functions:
        name_lower = name.lower()
        if any(keyword in name_lower for keyword in servo_keywords):
            servo_functions.append((addr, name, line))
    
    print(f"\nНайдено {len(servo_functions)} функций, связанных с servo")
    
    if servo_functions:
        print("\nПервые 20:")
        for addr, name, _ in servo_functions[:20]:
            print(f"  {addr} - {name}")
        
        # Анализируем несколько ключевых
        key_servo = [f for f in servo_functions if any(kw in f[1].lower() for kw in ['update', 'calc', 'adjust'])]
        
        if key_servo:
            print(f"\n\nАнализ ключевых servo функций ({len(key_servo)}):")
            for addr, name, _ in key_servo[:5]:
                print(f"\n  {name}:")
                assembly = disassemble_function(name, addr)
                if assembly:
                    patterns = analyze_assembly_patterns(assembly)
                    print(f"    Размер: ~{len(assembly.split(chr(10)))} строк")
                    print(f"    Вызовов: {len(patterns['calls'])}")
                    print(f"    Условий: {len(patterns['conditions'])}")
                    print(f"    Арифметика: {patterns['key_operations'].count('arithmetic')}")
    else:
        print("\n⚠ Servo функции не найдены через стандартный поиск")
        print("Попробуем поискать по другим критериям...")
    
    print()

def build_full_call_graph(functions, call_graph):
    """Строит полный граф вызовов с визуализацией"""
    print("=" * 80)
    print("6. ПОЛНЫЙ ГРАФ ВЫЗОВОВ")
    print("=" * 80)
    
    # Группируем по модулям
    modules = defaultdict(list)
    for addr, name, _ in functions:
        for module in TARGET_MODULES:
            if module in name.lower():
                modules[module].append(name)
                break
    
    print("\nФункции по модулям:")
    for module, funcs in modules.items():
        print(f"  {module}: {len(funcs)} функций")
    
    # Строим связи между модулями
    print("\n\nСвязи между модулями:")
    module_calls = defaultdict(lambda: defaultdict(int))
    
    for caller, callees in call_graph.items():
        caller_module = None
        for module in TARGET_MODULES:
            if module in caller.lower():
                caller_module = module
                break
        
        if caller_module:
            for callee in callees:
                callee_module = None
                for module in TARGET_MODULES:
                    if module in callee.lower():
                        callee_module = module
                        break
                
                if callee_module and callee_module != caller_module:
                    module_calls[caller_module][callee_module] += 1
    
    for caller_mod, callee_mods in module_calls.items():
        print(f"\n  {caller_mod.upper()} ->")
        for callee_mod, count in sorted(callee_mods.items(), key=lambda x: x[1], reverse=True):
            print(f"    {callee_mod}: {count} вызовов")
    
    # Топ функций по количеству вызовов
    print("\n\nТоп-20 функций по количеству вызовов:")
    call_counts = defaultdict(int)
    for callees in call_graph.values():
        for callee in callees:
            call_counts[callee] += 1
    
    for func, count in sorted(call_counts.items(), key=lambda x: x[1], reverse=True)[:20]:
        print(f"  {func}: {count} вызовов")
    
    print()

def extract_algorithm_flow(func_name, assembly_code):
    """Извлекает поток выполнения алгоритма"""
    if not assembly_code:
        return None
    
    flow = {
        'function': func_name,
        'basic_blocks': [],
        'control_flow': []
    }
    
    lines = assembly_code.split('\n')
    current_block = []
    
    for line in lines:
        line_stripped = line.strip()
        if not line_stripped:
            continue
        
        # Определяем базовые блоки (начало после перехода)
        if ':' in line_stripped and ('<' not in line_stripped or '>' not in line_stripped):
            if current_block:
                flow['basic_blocks'].append(current_block)
            current_block = [line_stripped]
        else:
            current_block.append(line_stripped)
        
        # Ищем переходы управления
        if any(branch in line_stripped.lower() for branch in ['jmp', 'je', 'jne', 'jz', 'jnz', 'call', 'ret']):
            flow['control_flow'].append(line_stripped)
    
    if current_block:
        flow['basic_blocks'].append(current_block)
    
    return flow

def analyze_timepulse_functions(functions):
    """Анализ функций timepulse"""
    print("=" * 80)
    print("7. АНАЛИЗ TIMEPULSE ФУНКЦИЙ")
    print("=" * 80)
    
    tp_functions = [f for f in functions if 'timepulse' in f[1].lower() or 'tp5' in f[1].lower()]
    
    print(f"\nНайдено {len(tp_functions)} timepulse функций")
    
    for addr, name, _ in tp_functions[:5]:
        print(f"\n  {name} (0x{addr}):")
        assembly = disassemble_function(name, addr)
        if assembly:
            # Ищем константы pulse width
            patterns = analyze_assembly_patterns(assembly)
            
            # Проверяем константы на значения pulse width
            pulse_widths = []
            for const in set(patterns['constants']):
                try:
                    val = int(const, 16)
                    # Проверяем, может ли это быть pulse width
                    if val in [100000000, 5000000, 1000000, 10000000]:
                        pulse_widths.append(f"{val} нс")
                    elif val < 1000000000 and val > 1000:
                        # Возможно, это значение в другом формате
                        pulse_widths.append(f"0x{const} (возможно {val} нс)")
                except:
                    pass
            
            if pulse_widths:
                print(f"    Найдены значения pulse width: {', '.join(pulse_widths)}")
            
            print(f"    Размер кода: ~{len(assembly.split(chr(10)))} строк")
            print(f"    Вызовов функций: {len(patterns['calls'])}")
    
    print()

def generate_call_graph_dot(call_graph, output_file):
    """Генерирует DOT файл для визуализации графа"""
    print("=" * 80)
    print("8. ГЕНЕРАЦИЯ DOT ФАЙЛА ДЛЯ ВИЗУАЛИЗАЦИИ")
    print("=" * 80)
    
    dot_content = ["digraph CallGraph {", "  rankdir=LR;", "  node [shape=box];"]
    
    # Группируем по модулям
    nodes = set()
    for caller in call_graph.keys():
        nodes.add(caller)
        nodes.update(call_graph[caller])
    
    # Добавляем узлы с цветами по модулям
    module_colors = {
        'ubx': 'lightblue',
        'gnss': 'lightgreen',
        'ptp': 'lightyellow',
        'ntp': 'lightpink',
        'servo': 'lightcoral',
        'clocksync': 'lightgray'
    }
    
    for node in nodes:
        color = 'white'
        for module, col in module_colors.items():
            if module in node.lower():
                color = col
                break
        
        label = node.split('.')[-1]  # Короткое имя
        dot_content.append(f'  "{node}" [label="{label}", style=filled, fillcolor={color}];')
    
    # Добавляем рёбра
    for caller, callees in call_graph.items():
        for callee in callees:
            dot_content.append(f'  "{caller}" -> "{callee}";')
    
    dot_content.append("}")
    
    try:
        with open(output_file, 'w') as f:
            f.write('\n'.join(dot_content))
        print(f"\n✓ DOT файл сохранен: {output_file}")
        print("Для визуализации используйте:")
        print(f"  dot -Tpng {output_file} -o call_graph.png")
    except Exception as e:
        print(f"✗ Ошибка сохранения: {e}")
    
    print()

def main():
    print("=" * 80)
    print("ПРОДВИНУТЫЙ АНАЛИЗ БИНАРНИКА SHIWATIME")
    print("=" * 80)
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # 1. Получаем все функции
    functions = get_all_clocksync_functions()
    
    if not functions:
        print("✗ Функции не найдены")
        return 1
    
    # 2. Строим граф вызовов
    call_graph = build_call_graph(functions)
    
    # 3. Анализируем ключевые функции
    analyze_key_functions(functions)
    
    # 4. Специальный анализ UBX
    analyze_ubx_functions(functions)
    
    # 5. Специальный анализ Servo
    analyze_servo_functions(functions)
    
    # 6. Анализ timepulse
    analyze_timepulse_functions(functions)
    
    # 7. Полный граф вызовов
    build_full_call_graph(functions, call_graph)
    
    # 8. Генерируем DOT файл (в домашней директории)
    home_dir = os.path.expanduser("~")
    dot_file = os.path.join(home_dir, "clocksync_call_graph.dot")
    generate_call_graph_dot(call_graph, dot_file)
    
    print("=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    print()
    print("Для сохранения результатов:")
    print("  sudo python3 advanced_binary_analysis.py > advanced_analysis.txt 2>&1")
    print()
    print("Для визуализации графа вызовов:")
    print(f"  dot -Tpng {dot_file} -o call_graph.png")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

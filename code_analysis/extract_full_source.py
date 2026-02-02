#!/usr/bin/env python3
"""
Полное извлечение функций и алгоритмов из бинарника timebeat.
Создаёт структуру Go-пакетов с сигнатурами функций.
"""

import subprocess
import re
import os
from collections import defaultdict

BINARY_PATH = "timebeat-extracted/usr/share/timebeat/bin/timebeat"

def run_cmd(cmd):
    try:
        r = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=120)
        return r.stdout
    except:
        return ""

def extract_all_functions():
    """Извлекает все Go функции из бинарника"""
    print("Извлечение всех функций из бинарника...")
    
    # strings для Go путей
    output = run_cmd(f"strings {BINARY_PATH} | grep -E '^github.com/lasselj/timebeat/' | sort -u")
    
    functions = defaultdict(list)
    for line in output.strip().split('\n'):
        if not line:
            continue
        # Парсим путь пакета и имя функции
        match = re.match(r'github\.com/lasselj/timebeat/(.+?)\.(\([^)]+\)\.)?(.+)$', line)
        if match:
            pkg = match.group(1)
            func_name = line.split('.')[-1]
            functions[pkg].append(func_name)
    
    return functions

def create_package_structure(functions, output_dir):
    """Создаёт структуру Go-пакетов"""
    os.makedirs(output_dir, exist_ok=True)
    
    for pkg, funcs in sorted(functions.items()):
        pkg_dir = os.path.join(output_dir, pkg.replace('/', os.sep))
        os.makedirs(pkg_dir, exist_ok=True)
        
        # Создаём файл с сигнатурами
        filename = os.path.basename(pkg) + ".go"
        filepath = os.path.join(pkg_dir, filename)
        
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(f"package {os.path.basename(pkg)}\n\n")
            f.write("// Автоматически извлечено из timebeat-2.2.20\n\n")
            
            # Группируем по типам
            methods = defaultdict(list)
            standalone = []
            
            for func in sorted(set(funcs)):
                if func.startswith('('):
                    # Метод структуры
                    m = re.match(r'\(([^)]+)\)\.(.+)', func)
                    if m:
                        receiver = m.group(1)
                        method = m.group(2)
                        methods[receiver].append(method)
                else:
                    standalone.append(func)
            
            # Записываем standalone функции
            for func in standalone:
                if func.endswith('.go'):
                    continue
                f.write(f"func {func}() {{\n\t// TODO: реконструировать\n}}\n\n")
            
            # Записываем методы
            for receiver, method_list in sorted(methods.items()):
                f.write(f"// {receiver} methods:\n")
                for method in sorted(set(method_list)):
                    if method.endswith('.go') or method.endswith('-fm'):
                        continue
                    f.write(f"// func ({receiver}) {method}()\n")
                f.write("\n")

def extract_servo_algorithms():
    """Извлекает алгоритмы servo через objdump"""
    print("Извлечение servo алгоритмов...")
    
    # Ищем адреса servo функций
    output = run_cmd(f"objdump -t {BINARY_PATH} 2>/dev/null | grep -i servo | head -50")
    
    servo_funcs = []
    for line in output.strip().split('\n'):
        if 'servo' in line.lower():
            match = re.search(r'([0-9a-f]+)\s+.*\s+(\S+)$', line)
            if match:
                addr = match.group(1)
                name = match.group(2)
                servo_funcs.append((addr, name))
    
    return servo_funcs

def disassemble_function(addr, name, lines=100):
    """Дизассемблирует функцию по адресу"""
    output = run_cmd(f"objdump -d {BINARY_PATH} 2>/dev/null | grep -A {lines} '{addr}:'")
    return output[:5000] if output else ""

def main():
    print("=" * 70)
    print("ИЗВЛЕЧЕНИЕ ИСХОДНИКОВ ИЗ TIMEBEAT-2.2.20")
    print("=" * 70)
    
    # 1. Извлекаем все функции
    functions = extract_all_functions()
    print(f"Найдено пакетов: {len(functions)}")
    total_funcs = sum(len(f) for f in functions.values())
    print(f"Всего функций: {total_funcs}")
    
    # 2. Создаём структуру
    output_dir = "extracted_source"
    create_package_structure(functions, output_dir)
    print(f"Структура создана в: {output_dir}/")
    
    # 3. Извлекаем servo
    servo_funcs = extract_servo_algorithms()
    print(f"Найдено servo функций: {len(servo_funcs)}")
    
    # 4. Сохраняем отчёт
    with open("extraction_report.txt", 'w', encoding='utf-8') as f:
        f.write("ОТЧЁТ ОБ ИЗВЛЕЧЕНИИ\n")
        f.write("=" * 50 + "\n\n")
        
        f.write("ПАКЕТЫ:\n")
        for pkg in sorted(functions.keys()):
            f.write(f"  {pkg}: {len(functions[pkg])} функций\n")
        
        f.write("\nSERVO ФУНКЦИИ:\n")
        for addr, name in servo_funcs:
            f.write(f"  0x{addr}: {name}\n")
    
    print("Отчёт: extraction_report.txt")
    print("=" * 70)

if __name__ == "__main__":
    main()

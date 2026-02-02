#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Анализ найденных servo функций по адресам
"""

import subprocess
import re
import sys

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

# Найденные функции из nm вывода
FOUND_FUNCTIONS = {
    "GetClockUsingGetTimeSyscall": {
        "addr": "0x40c7300",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockUsingGetTimeSyscall"
    },
    "StepClockUsingSetTimeSyscall": {
        "addr": "0x40c6ea0",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepClockUsingSetTimeSyscall"
    },
    "PerformGranularityMeasurement": {
        "addr": "0x40c74b0",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.PerformGranularityMeasurement"
    },
    "GetClockFrequency": {
        "addr": "0x40c68c0",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetClockFrequency"
    },
    "SetFrequency": {
        "addr": "0x40c6b30",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.SetFrequency"
    },
    "SetOffset": {
        "addr": "0x40c6cf0",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.SetOffset"
    },
    "GetPreciseTime": {
        "addr": "0x40c7420",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.GetPreciseTime"
    },
    "StepRTCClock": {
        "addr": "0x40c7040",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/adjusttime.StepRTCClock"
    },
    # Servo алгоритмы
    "AlgoPID.UpdateClockFreq": {
        "addr": "0x41c8680",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*AlgoPID).UpdateClockFreq"
    },
    "LinReg.UpdateClockFreq": {
        "addr": "0x41c6c00",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*LinReg).UpdateClockFreq"
    },
    "Pi.UpdateClockFreq": {
        "addr": "0x41c8310",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo/algos.(*Pi).UpdateClockFreq"
    },
    # Servo Controller
    "Controller.RunPeriodicAdjustSlaveClocks": {
        "addr": "0x41e7ff0",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo.(*Controller).RunPeriodicAdjustSlaveClocks"
    },
    "Controller.ChangeMasterClock": {
        "addr": "0x41ec090",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo.(*Controller).ChangeMasterClock"
    },
    "Controller.GetUTCTimeFromMasterClock": {
        "addr": "0x41e7850",
        "full_name": "github.com/lasselj/timebeat/beater/clocksync/servo.(*Controller).GetUTCTimeFromMasterClock"
    },
}

def disassemble_function(addr, name, size=0x3000):
    """Дизассемблирует функцию"""
    try:
        addr_clean = addr.replace('0x', '').replace('0X', '')
        addr_int = int(addr_clean, 16)
        start_addr = f"0x{addr_int:x}"
        end_addr = f"0x{addr_int + size:x}"
        
        cmd = ["objdump", "-d", "-C", "--start-address", start_addr, 
               "--stop-address", end_addr, BINARY_PATH]
        result = subprocess.run(cmd, capture_output=True, text=True, check=False)
        
        if result.returncode == 0 and result.stdout:
            return result.stdout
        return None
    except Exception as e:
        return f"Ошибка: {str(e)}"

def analyze_assembly(asm_code):
    """Анализирует ассемблерный код"""
    if not asm_code or asm_code.startswith("Ошибка"):
        return {}
    
    analysis = {
        'constants': [],
        'calls': [],
        'arithmetic': 0,
        'branches': 0,
        'loads': 0,
        'stores': 0,
    }
    
    lines = asm_code.split('\n')
    
    # Поиск констант
    for line in lines:
        # Hex константы
        hex_matches = re.finditer(r'#0x([0-9a-f]+)', line, re.IGNORECASE)
        for match in hex_matches:
            try:
                val = int(match.group(1), 16)
                analysis['constants'].append({
                    'value': val,
                    'hex': f"0x{val:x}",
                    'line': line.strip()
                })
            except:
                pass
        
        # Decimal константы
        dec_matches = re.finditer(r'#(\d+)', line)
        for match in dec_matches:
            try:
                val = int(match.group(1))
                analysis['constants'].append({
                    'value': val,
                    'hex': f"0x{val:x}",
                    'line': line.strip()
                })
            except:
                pass
        
        # Вызовы функций
        if re.search(r'\s+bl\s+', line, re.IGNORECASE):
            match = re.search(r'bl\s+([0-9a-f]+)\s+<([^>]+)>', line, re.IGNORECASE)
            if match:
                analysis['calls'].append({
                    'addr': match.group(1),
                    'name': match.group(2)
                })
        
        # Подсчет операций
        if re.search(r'\s+(add|sub|mul|fadd|fsub|fmul)\s+', line, re.IGNORECASE):
            analysis['arithmetic'] += 1
        if re.search(r'\s+(b|beq|bne|blt|bgt|cbz|cbnz)\s+', line, re.IGNORECASE):
            analysis['branches'] += 1
        if re.search(r'\s+ldr\s+', line, re.IGNORECASE):
            analysis['loads'] += 1
        if re.search(r'\s+str\s+', line, re.IGNORECASE):
            analysis['stores'] += 1
    
    return analysis

def main():
    print("=" * 80)
    print("АНАЛИЗ НАЙДЕННЫХ SERVO ФУНКЦИЙ")
    print("=" * 80)
    print()
    
    results = {}
    
    for name, info in FOUND_FUNCTIONS.items():
        print(f"{'=' * 80}")
        print(f"ФУНКЦИЯ: {name}")
        print(f"{'=' * 80}")
        print(f"Адрес: {info['addr']}")
        print(f"Полное имя: {info['full_name']}")
        print()
        
        # Дизассемблирование
        print("📝 Дизассемблирование...")
        asm = disassemble_function(info['addr'], info['full_name'])
        
        if asm:
            # Анализ
            print("🔍 Анализ...")
            analysis = analyze_assembly(asm)
            
            results[name] = {
                'info': info,
                'analysis': analysis,
                'asm_preview': asm[:1500] if len(asm) > 1500 else asm
            }
            
            # Вывод статистики
            print(f"\n📊 Статистика:")
            print(f"  Арифметических операций: {analysis.get('arithmetic', 0)}")
            print(f"  Ветвлений: {analysis.get('branches', 0)}")
            print(f"  Загрузок: {analysis.get('loads', 0)}")
            print(f"  Сохранений: {analysis.get('stores', 0)}")
            print(f"  Вызовов функций: {len(analysis.get('calls', []))}")
            print(f"  Констант: {len(analysis.get('constants', []))}")
            
            # Константы
            constants = analysis.get('constants', [])
            if constants:
                print(f"\n🔢 Найденные константы (первые 15):")
                unique_constants = {}
                for c in constants:
                    val = c['value']
                    if val not in unique_constants:
                        unique_constants[val] = c
                for val, c in sorted(unique_constants.items(), key=lambda x: abs(x[0]))[:15]:
                    print(f"  {c['hex']} ({val})")
            
            # Вызовы
            calls = analysis.get('calls', [])
            if calls:
                print(f"\n📞 Вызовы функций (первые 10):")
                for call in calls[:10]:
                    print(f"  {call.get('name', 'unknown')}")
        else:
            print("⚠ Не удалось дизассемблировать")
            results[name] = {
                'info': info,
                'error': 'disassembly_failed'
            }
        
        print()
    
    # Сохранение результатов
    output_file = "servo_functions_detailed_analysis.txt"
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("=" * 80 + "\n")
        f.write("ДЕТАЛЬНЫЙ АНАЛИЗ SERVO ФУНКЦИЙ\n")
        f.write("=" * 80 + "\n\n")
        
        for name, result in results.items():
            f.write("=" * 80 + "\n")
            f.write(f"ФУНКЦИЯ: {name}\n")
            f.write("=" * 80 + "\n")
            f.write(f"Адрес: {result['info']['addr']}\n")
            f.write(f"Полное имя: {result['info']['full_name']}\n\n")
            
            if 'analysis' in result:
                analysis = result['analysis']
                f.write("СТАТИСТИКА:\n")
                f.write(f"  Арифметических операций: {analysis.get('arithmetic', 0)}\n")
                f.write(f"  Ветвлений: {analysis.get('branches', 0)}\n")
                f.write(f"  Загрузок: {analysis.get('loads', 0)}\n")
                f.write(f"  Сохранений: {analysis.get('stores', 0)}\n")
                f.write(f"  Вызовов функций: {len(analysis.get('calls', []))}\n")
                f.write(f"  Констант: {len(analysis.get('constants', []))}\n\n")
                
                constants = analysis.get('constants', [])
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
                
                calls = analysis.get('calls', [])
                if calls:
                    f.write("ВЫЗОВЫ ФУНКЦИЙ:\n")
                    for call in calls:
                        f.write(f"  {call.get('name', 'unknown')}\n")
                    f.write("\n")
                
                if 'asm_preview' in result:
                    f.write("АССЕМБЛЕРНЫЙ КОД (первые 1500 символов):\n")
                    f.write(result['asm_preview'])
                    f.write("\n\n")
            else:
                f.write("ОШИБКА: Не удалось проанализировать функцию\n\n")
    
    print("=" * 80)
    print("СОХРАНЕНИЕ РЕЗУЛЬТАТОВ")
    print("=" * 80)
    print(f"✅ Результаты сохранены в {output_file}")
    print(f"   Проанализировано функций: {len(results)}")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

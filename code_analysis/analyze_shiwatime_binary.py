#!/usr/bin/env python3
"""
Расширенный скрипт для полного анализа бинарника shiwatime
- Структура файла и ELF
- Используемые библиотеки и зависимости
- Структура программы (функции, граф вызовов)
- Поиск значений длительности импульса
- Анализ работы с UBX/Timepulse
- Анализ конфигурации и инициализации
- Поиск точек входа и основных компонентов

Использование:
    sudo python3 analyze_shiwatime_binary.py
    sudo python3 analyze_shiwatime_binary.py > analysis.txt 2>&1
"""

import os
import sys
import subprocess
import re
from struct import pack, unpack

BINARY_PATH = "/usr/share/shiwatime/bin/shiwatime"

def run_command(cmd, shell=False):
    """Выполняет команду и возвращает результат"""
    try:
        # Используем bytes и декодируем вручную для надежности
        result = subprocess.run(
            cmd, 
            shell=shell, 
            capture_output=True, 
            check=True
        )
        # Декодируем с обработкой ошибок
        return result.stdout.decode('utf-8', errors='replace')
    except subprocess.CalledProcessError as e:
        try:
            if isinstance(e.stderr, bytes):
                error_msg = e.stderr.decode('utf-8', errors='replace')
            else:
                error_msg = str(e.stderr)
        except:
            error_msg = str(e.stderr)
        return f"Ошибка: {error_msg}"
    except FileNotFoundError:
        return f"Команда не найдена: {cmd[0] if isinstance(cmd, list) else cmd}"
    except Exception as e:
        return f"Неожиданная ошибка: {str(e)}"

def analyze_file_type():
    """Определяет тип файла"""
    print("=" * 80)
    print("1. ТИП ФАЙЛА")
    print("=" * 80)
    result = run_command(["file", BINARY_PATH])
    print(result)
    print()

def analyze_libraries():
    """Анализирует используемые библиотеки"""
    print("=" * 80)
    print("2. ДИНАМИЧЕСКИЕ БИБЛИОТЕКИ")
    print("=" * 80)
    result = run_command(["ldd", BINARY_PATH])
    print(result)
    print()

def analyze_strings():
    """Ищет строки, связанные с timepulse, TP5, pulse width"""
    print("=" * 80)
    print("3. ПОИСК СТРОК, СВЯЗАННЫХ С TIMEPULSE/PPS")
    print("=" * 80)
    
    keywords = [
        "timepulse", "time pulse", "TP5", "CFG-TP5",
        "pulse", "pulsewidth", "pulse_width", "pulseWidth",
        "100000000", "100 ms", "100ms",
        "5000000", "5 ms", "5ms",
        "ublox", "ubx", "CFG_TP5"
    ]
    
    # Используем strings для поиска
    result = run_command(["strings", BINARY_PATH])
    
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        found = []
        for line in lines:
            line_lower = line.lower()
            for keyword in keywords:
                if keyword.lower() in line_lower:
                    found.append(line)
                    break
        
        if found:
            print(f"Найдено {len(found)} строк:")
            for line in sorted(set(found))[:50]:  # Показываем первые 50 уникальных
                print(f"  {line}")
        else:
            print("Строки не найдены")
    else:
        print("Не удалось выполнить strings")
    print()

def search_binary_values():
    """Ищет значения 100000000 и 5000000 в бинарнике"""
    print("=" * 80)
    print("4. ПОИСК ЗНАЧЕНИЙ ДЛИТЕЛЬНОСТИ ИМПУЛЬСА В БИНАРНИКЕ")
    print("=" * 80)
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Файл {BINARY_PATH} не найден")
        return
    
    # Значения для поиска
    values = {
        100000000: "100 мс (100000000 нс)",
        5000000: "5 мс (5000000 нс)",
        100: "100 (возможно в миллисекундах)",
        5: "5 (возможно в миллисекундах)"
    }
    
    with open(BINARY_PATH, 'rb') as f:
        data = f.read()
    
    print(f"Размер файла: {len(data)} байт")
    print()
    
    for value, description in values.items():
        # Little-endian (4 байта)
        le_bytes = pack('<I', value)
        # Big-endian (4 байта)
        be_bytes = pack('>I', value)
        # Little-endian (8 байт)
        le64_bytes = pack('<Q', value)
        
        positions = []
        
        # Поиск little-endian (4 байта)
        offset = 0
        while True:
            pos = data.find(le_bytes, offset)
            if pos == -1:
                break
            positions.append((pos, "little-endian (uint32)"))
            offset = pos + 1
        
        # Поиск big-endian (4 байта)
        offset = 0
        while True:
            pos = data.find(be_bytes, offset)
            if pos == -1:
                break
            positions.append((pos, "big-endian (uint32)"))
            offset = pos + 1
        
        # Поиск little-endian (8 байт)
        offset = 0
        while True:
            pos = data.find(le64_bytes, offset)
            if pos == -1:
                break
            positions.append((pos, "little-endian (uint64)"))
            offset = pos + 1
        
        if positions:
            print(f"Значение {value} ({description}):")
            for pos, format_type in positions[:10]:  # Показываем первые 10
                # Показываем контекст (16 байт до и после)
                start = max(0, pos - 16)
                end = min(len(data), pos + len(le_bytes) + 16)
                context = data[start:end]
                hex_context = ' '.join(f'{b:02x}' for b in context)
                print(f"  Смещение 0x{pos:X} ({format_type})")
                print(f"    Контекст: {hex_context}")
            print(f"  Всего найдено: {len(positions)} вхождений")
            print()
    print()

def analyze_symbols():
    """Анализирует символы в бинарнике"""
    print("=" * 80)
    print("5. СИМВОЛЫ (если доступны)")
    print("=" * 80)
    
    # Пробуем nm
    result = run_command(["nm", "-D", BINARY_PATH], shell=False)
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        relevant = [l for l in lines if any(kw in l.lower() for kw in ['time', 'pulse', 'tp5', 'ubx', 'cfg'])]
        if relevant:
            print(f"Найдено {len(relevant)} релевантных символов:")
            for line in relevant[:30]:
                print(f"  {line}")
        else:
            print("Релевантные символы не найдены")
    else:
        print("nm недоступен или файл не содержит символов")
    print()

def analyze_sections():
    """Анализирует секции ELF файла"""
    print("=" * 80)
    print("6. СЕКЦИИ ELF ФАЙЛА")
    print("=" * 80)
    result = run_command(["readelf", "-S", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("readelf недоступен или файл не является ELF")
    print()

def analyze_imports():
    """Анализирует импорты (динамические функции)"""
    print("=" * 80)
    print("7. ДИНАМИЧЕСКИЕ ИМПОРТЫ")
    print("=" * 80)
    result = run_command(["readelf", "-d", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result[:2000])  # Первые 2000 символов
    else:
        print("readelf недоступен")
    print()

def search_hex_patterns():
    """Ищет hex паттерны, связанные с CFG-TP5"""
    print("=" * 80)
    print("8. ПОИСК HEX ПАТТЕРНОВ CFG-TP5")
    print("=" * 80)
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Файл {BINARY_PATH} не найден")
        return
    
    with open(BINARY_PATH, 'rb') as f:
        data = f.read()
    
    # UBX sync bytes для CFG-TP5
    # Класс: 0x06 (CFG), ID: 0x31 (TP5)
    patterns = [
        (b'\xb5\x62\x06\x31', "UBX CFG-TP5 заголовок"),
        (b'\x06\x31', "CFG-TP5 класс+ID"),
        (b'\x00\xe1\xf5\x05', "100000000 little-endian"),
        (b'\x40\x4b\x4c\x00', "5000000 little-endian"),
    ]
    
    for pattern, description in patterns:
        positions = []
        offset = 0
        while True:
            pos = data.find(pattern, offset)
            if pos == -1:
                break
            positions.append(pos)
            offset = pos + 1
        
        if positions:
            print(f"Паттерн '{description}': {len(positions)} вхождений")
            for pos in positions[:5]:  # Первые 5
                print(f"  Смещение: 0x{pos:X}")
    print()

def analyze_all_strings():
    """Показывает все строки из бинарника (первые 200)"""
    print("=" * 80)
    print("9. ВСЕ СТРОКИ ИЗ БИНАРНИКА (первые 200)")
    print("=" * 80)
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        print(f"Всего строк: {len(lines)}")
        print("Первые 200 строк:")
        for i, line in enumerate(lines[:200], 1):
            if line.strip():
                print(f"  {i:4d}: {line}")
    print()

def analyze_all_symbols():
    """Показывает все символы"""
    print("=" * 80)
    print("10. ВСЕ СИМВОЛЫ (если доступны)")
    print("=" * 80)
    
    # Пробуем разные варианты
    for cmd in [["nm", "-D", BINARY_PATH], ["nm", BINARY_PATH], ["objdump", "-T", BINARY_PATH]]:
        result = run_command(cmd)
        if result and not result.startswith("Ошибка") and len(result) > 100:
            lines = result.split('\n')
            print(f"Команда: {' '.join(cmd)}")
            print(f"Найдено символов: {len(lines)}")
            print("Первые 100:")
            for line in lines[:100]:
                if line.strip():
                    print(f"  {line}")
            print()
            break
    else:
        print("Символы недоступны (stripped бинарник)")
    print()

def analyze_header():
    """Анализирует ELF заголовок"""
    print("=" * 80)
    print("11. ELF ЗАГОЛОВОК")
    print("=" * 80)
    result = run_command(["readelf", "-h", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("readelf недоступен")
    print()

def analyze_program_headers():
    """Анализирует программные заголовки"""
    print("=" * 80)
    print("12. ПРОГРАММНЫЕ ЗАГОЛОВКИ")
    print("=" * 80)
    result = run_command(["readelf", "-l", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("readelf недоступен")
    print()

def analyze_dynamic_symbols():
    """Анализирует динамические символы"""
    print("=" * 80)
    print("13. ДИНАМИЧЕСКИЕ СИМВОЛЫ")
    print("=" * 80)
    result = run_command(["readelf", "-s", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        print(f"Всего символов: {len(lines)}")
        # Показываем первые 150 строк
        print('\n'.join(lines[:150]))
        if len(lines) > 150:
            print(f"\n... (показано 150 из {len(lines)})")
    else:
        print("readelf недоступен")
    print()

def analyze_version_info():
    """Анализирует информацию о версии"""
    print("=" * 80)
    print("14. ИНФОРМАЦИЯ О ВЕРСИИ")
    print("=" * 80)
    result = run_command(["readelf", "-V", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("readelf недоступен или версии не найдены")
    print()

def analyze_notes():
    """Анализирует заметки (notes)"""
    print("=" * 80)
    print("15. ЗАМЕТКИ (NOTES)")
    print("=" * 80)
    result = run_command(["readelf", "-n", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("readelf недоступен или заметки не найдены")
    print()

def analyze_relocations():
    """Анализирует релокации"""
    print("=" * 80)
    print("16. РЕЛОКАЦИИ (если доступны)")
    print("=" * 80)
    result = run_command(["readelf", "-r", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        if len(lines) > 50:
            print('\n'.join(lines[:50]))
            print(f"\n... (показано 50 из {len(lines)})")
        else:
            print(result)
    else:
        print("readelf недоступен или релокации не найдены")
    print()

def analyze_dependencies():
    """Анализирует зависимости более детально"""
    print("=" * 80)
    print("17. ДЕТАЛЬНЫЙ АНАЛИЗ ЗАВИСИМОСТЕЙ")
    print("=" * 80)
    
    # ldd с подробностями
    result = run_command(["ldd", "-v", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        print(result)
    else:
        print("ldd недоступен")
    print()

def search_config_strings():
    """Ищет строки, связанные с конфигурацией"""
    print("=" * 80)
    print("18. СТРОКИ, СВЯЗАННЫЕ С КОНФИГУРАЦИЕЙ")
    print("=" * 80)
    
    keywords = [
        "config", "configuration", "yaml", "yml",
        "shiwatime", "timebeat", "timecard",
        "gnss", "gps", "glonass", "ublox",
        "ptp", "ntp", "pps", "clock",
        "primary", "secondary", "protocol"
    ]
    
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        found = []
        for line in lines:
            line_lower = line.lower()
            for keyword in keywords:
                if keyword.lower() in line_lower:
                    found.append(line)
                    break
        
        if found:
            print(f"Найдено {len(found)} строк:")
            for line in sorted(set(found))[:100]:
                print(f"  {line}")
        else:
            print("Строки не найдены")
    print()

def analyze_file_info():
    """Дополнительная информация о файле"""
    print("=" * 80)
    print("19. ДОПОЛНИТЕЛЬНАЯ ИНФОРМАЦИЯ О ФАЙЛЕ")
    print("=" * 80)
    
    if os.path.exists(BINARY_PATH):
        stat = os.stat(BINARY_PATH)
        print(f"Размер: {stat.st_size:,} байт ({stat.st_size / 1024 / 1024:.2f} MB)")
        print(f"Права доступа: {oct(stat.st_mode)}")
        print(f"Владелец UID: {stat.st_uid}")
        print(f"Группа GID: {stat.st_gid}")
        print(f"Модифицирован: {stat.st_mtime}")
    print()

def analyze_disassembly():
    """Показывает дизассемблирование (первые функции)"""
    print("=" * 80)
    print("20. ДИЗАССЕМБЛИРОВАНИЕ (первые функции)")
    print("=" * 80)
    
    # Пробуем objdump
    result = run_command(["objdump", "-d", "-C", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        print(f"Всего строк дизассемблирования: {len(lines)}")
        print("Первые 200 строк:")
        print('\n'.join(lines[:200]))
        if len(lines) > 200:
            print(f"\n... (показано 200 из {len(lines)})")
    else:
        print("objdump недоступен или не удалось дизассемблировать")
    print()

def analyze_functions():
    """Список функций в бинарнике"""
    print("=" * 80)
    print("21. СПИСОК ФУНКЦИЙ")
    print("=" * 80)
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        print(f"Найдено функций/символов: {len(lines)}")
        print("Первые 150:")
        for line in lines[:150]:
            if line.strip():
                print(f"  {line}")
        if len(lines) > 150:
            print(f"\n... (показано 150 из {len(lines)})")
    else:
        print("objdump недоступен")
    print()

def analyze_entry_points():
    """Анализирует точки входа программы"""
    print("=" * 80)
    print("22. ТОЧКИ ВХОДА ПРОГРАММЫ")
    print("=" * 80)
    
    # Поиск main и init функций
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        entry_points = []
        for line in lines:
            if any(keyword in line.lower() for keyword in ['main', '_init', '_start', 'constructor', 'destructor']):
                entry_points.append(line)
        
        if entry_points:
            print("Найдены точки входа:")
            for line in entry_points:
                print(f"  {line}")
        else:
            print("Точки входа не найдены явно")
    
    # ELF entry point
    result = run_command(["readelf", "-h", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        for line in result.split('\n'):
            if 'Entry point' in line:
                print(f"\nELF Entry point: {line}")
    print()

def analyze_timepulse_functions():
    """Ищет функции, связанные с timepulse"""
    print("=" * 80)
    print("23. ФУНКЦИИ, СВЯЗАННЫЕ С TIMEPULSE/PPS")
    print("=" * 80)
    
    keywords = [
        'timepulse', 'time_pulse', 'tp5', 'cfg_tp5', 'cfg-tp5',
        'pulse', 'pps', 'pulsewidth', 'pulse_width',
        'ublox', 'ubx', 'gnss', 'gps'
    ]
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        found = []
        for line in lines:
            line_lower = line.lower()
            for keyword in keywords:
                if keyword in line_lower:
                    found.append(line)
                    break
        
        if found:
            print(f"Найдено {len(found)} функций/символов:")
            for line in found[:50]:
                print(f"  {line}")
        else:
            print("Функции не найдены через objdump")
    
    # Также ищем в строках
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        func_strings = []
        for line in lines:
            line_lower = line.lower()
            for keyword in keywords:
                if keyword in line_lower and ('func' in line_lower or 'init' in line_lower or 'config' in line_lower):
                    func_strings.append(line)
                    break
        
        if func_strings:
            print(f"\nНайдено {len(func_strings)} строк, похожих на функции:")
            for line in sorted(set(func_strings))[:30]:
                print(f"  {line}")
    print()

def analyze_call_graph():
    """Анализирует граф вызовов функций"""
    print("=" * 80)
    print("24. ГРАФ ВЫЗОВОВ ФУНКЦИЙ (CALL GRAPH)")
    print("=" * 80)
    
    # Пробуем получить call graph через objdump
    result = run_command(["objdump", "-d", "-C", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        
        # Ищем вызовы функций (call, bl, blr и т.д.)
        calls = []
        current_function = None
        
        for line in lines:
            # Определяем начало функции
            if '>:' in line and '<' in line:
                func_match = re.search(r'<([^>]+)>:', line)
                if func_match:
                    current_function = func_match.group(1)
            
            # Ищем вызовы
            if current_function and ('call' in line.lower() or ' bl ' in line.lower() or ' blr' in line.lower()):
                # Извлекаем имя вызываемой функции
                call_match = re.search(r'<([^>]+)>', line)
                if call_match:
                    called_func = call_match.group(1)
                    if called_func != current_function:
                        calls.append((current_function, called_func))
        
        if calls:
            print(f"Найдено {len(calls)} вызовов функций")
            print("Первые 100 вызовов:")
            
            # Группируем по вызывающим функциям
            call_dict = {}
            for caller, callee in calls[:100]:
                if caller not in call_dict:
                    call_dict[caller] = []
                if callee not in call_dict[caller]:
                    call_dict[caller].append(callee)
            
            for caller, callees in list(call_dict.items())[:20]:
                print(f"\n  {caller} ->")
                for callee in callees[:10]:
                    print(f"    - {callee}")
        else:
            print("Граф вызовов не удалось построить автоматически")
    else:
        print("objdump недоступен")
    print()

def analyze_initialization():
    """Ищет функции инициализации"""
    print("=" * 80)
    print("25. ФУНКЦИИ ИНИЦИАЛИЗАЦИИ")
    print("=" * 80)
    
    init_keywords = [
        'init', 'initialize', 'setup', 'configure',
        'constructor', '_init', 'startup', 'boot'
    ]
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        init_funcs = []
        for line in lines:
            line_lower = line.lower()
            for keyword in init_keywords:
                if keyword in line_lower:
                    init_funcs.append(line)
                    break
        
        if init_funcs:
            print(f"Найдено {len(init_funcs)} функций инициализации:")
            for line in init_funcs[:50]:
                print(f"  {line}")
        else:
            print("Функции инициализации не найдены явно")
    print()

def analyze_file_operations():
    """Ищет операции работы с файлами"""
    print("=" * 80)
    print("26. ОПЕРАЦИИ РАБОТЫ С ФАЙЛАМИ И ПОРТАМИ")
    print("=" * 80)
    
    file_keywords = [
        'open', 'read', 'write', 'close', 'fopen', 'fread', 'fwrite',
        'serial', 'tty', '/dev/', 'socket', 'bind', 'listen', 'accept',
        'ioctl', 'fcntl', 'select', 'poll', 'epoll'
    ]
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        file_ops = []
        for line in lines:
            line_lower = line.lower()
            for keyword in file_keywords:
                if keyword in line_lower:
                    file_ops.append(line)
                    break
        
        if file_ops:
            print(f"Найдено {len(file_ops)} операций с файлами/портами:")
            for line in file_ops[:50]:
                print(f"  {line}")
        else:
            print("Операции не найдены явно")
    
    # Ищем пути к файлам в строках
    result = run_command(["strings", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        file_paths = []
        for line in lines:
            if any(path in line for path in ['/dev/', '/etc/', '/var/', '/usr/', '/tmp/', '.yml', '.yaml', '.conf']):
                file_paths.append(line)
        
        if file_paths:
            print(f"\nНайдено {len(file_paths)} путей к файлам:")
            for path in sorted(set(file_paths))[:50]:
                print(f"  {path}")
    print()

def analyze_network_operations():
    """Ищет сетевые операции"""
    print("=" * 80)
    print("27. СЕТЕВЫЕ ОПЕРАЦИИ")
    print("=" * 80)
    
    network_keywords = [
        'socket', 'bind', 'listen', 'accept', 'connect', 'send', 'recv',
        'sendto', 'recvfrom', 'getaddrinfo', 'gethostbyname',
        'ptp', 'ntp', 'udp', 'tcp', 'ip', 'inet'
    ]
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        network_ops = []
        for line in lines:
            line_lower = line.lower()
            for keyword in network_keywords:
                if keyword in line_lower:
                    network_ops.append(line)
                    break
        
        if network_ops:
            print(f"Найдено {len(network_ops)} сетевых операций:")
            for line in network_ops[:50]:
                print(f"  {line}")
        else:
            print("Сетевые операции не найдены явно")
    print()

def analyze_time_operations():
    """Ищет операции работы со временем"""
    print("=" * 80)
    print("28. ОПЕРАЦИИ РАБОТЫ СО ВРЕМЕНЕМ")
    print("=" * 80)
    
    time_keywords = [
        'time', 'clock', 'gettime', 'settime', 'adjtime', 'adjtimex',
        'timespec', 'timeval', 'nanosleep', 'usleep', 'sleep',
        'phc', 'ptp', 'ntp', 'gps', 'gnss', 'pps'
    ]
    
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        time_ops = []
        for line in lines:
            line_lower = line.lower()
            for keyword in time_keywords:
                if keyword in line_lower:
                    time_ops.append(line)
                    break
        
        if time_ops:
            print(f"Найдено {len(time_ops)} операций со временем:")
            for line in time_ops[:50]:
                print(f"  {line}")
        else:
            print("Операции со временем не найдены явно")
    print()

def analyze_constants():
    """Ищет константы в бинарнике"""
    print("=" * 80)
    print("29. КОНСТАНТЫ И МАГИЧЕСКИЕ ЧИСЛА")
    print("=" * 80)
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Файл {BINARY_PATH} не найден")
        return
    
    with open(BINARY_PATH, 'rb') as f:
        data = f.read()
    
    # Ищем интересные константы
    constants = {
        # Временные константы (в наносекундах)
        1000000000: "1 секунда (нс)",
        100000000: "100 мс (нс)",
        10000000: "10 мс (нс)",
        5000000: "5 мс (нс)",
        1000000: "1 мс (нс)",
        # UBX константы
        0xB562: "UBX sync (big-endian)",
        0x06: "UBX CFG class",
        0x31: "UBX CFG-TP5 ID",
        # Частоты
        9600: "Baud rate 9600",
        115200: "Baud rate 115200",
    }
    
    found_constants = []
    for value, description in constants.items():
        # Разные форматы
        formats = []
        if value < 256:
            formats.append((bytes([value]), "uint8"))
        if value < 65536:
            formats.append((pack('<H', value), "uint16 LE"))
            formats.append((pack('>H', value), "uint16 BE"))
        if value < 4294967296:
            formats.append((pack('<I', value), "uint32 LE"))
            formats.append((pack('>I', value), "uint32 BE"))
        formats.append((pack('<Q', value), "uint64 LE"))
        formats.append((pack('>Q', value), "uint64 BE"))
        
        for pattern, fmt in formats:
            count = data.count(pattern)
            if count > 0:
                found_constants.append((value, description, count, fmt))
                break
    
    if found_constants:
        print("Найдены константы:")
        for value, desc, count, fmt in found_constants:
            print(f"  {value} ({desc}): {count} вхождений как {fmt}")
    else:
        print("Константы не найдены")
    print()

def analyze_program_structure():
    """Анализирует общую структуру программы"""
    print("=" * 80)
    print("30. ОБЩАЯ СТРУКТУРА ПРОГРАММЫ")
    print("=" * 80)
    
    # Получаем все функции
    result = run_command(["objdump", "-T", BINARY_PATH])
    if result and not result.startswith("Ошибка"):
        lines = result.split('\n')
        all_funcs = [l for l in lines if l.strip() and 'F .text' in l]
        
        # Группируем по категориям
        categories = {
            'UBX/GNSS': [],
            'Time/PPS': [],
            'Config': [],
            'Network': [],
            'Init/Setup': [],
            'Other': []
        }
        
        for line in all_funcs:
            line_lower = line.lower()
            categorized = False
            
            if any(kw in line_lower for kw in ['ubx', 'gnss', 'gps', 'ublox']):
                categories['UBX/GNSS'].append(line)
                categorized = True
            if any(kw in line_lower for kw in ['time', 'pps', 'pulse', 'clock', 'phc']):
                categories['Time/PPS'].append(line)
                categorized = True
            if any(kw in line_lower for kw in ['config', 'setup', 'init']):
                categories['Config'].append(line)
                categorized = True
            if any(kw in line_lower for kw in ['socket', 'network', 'ptp', 'ntp']):
                categories['Network'].append(line)
                categorized = True
            if any(kw in line_lower for kw in ['init', 'startup', 'boot', 'main']):
                categories['Init/Setup'].append(line)
                categorized = True
            
            if not categorized:
                categories['Other'].append(line)
        
        print("Функции по категориям:")
        for category, funcs in categories.items():
            if funcs:
                print(f"\n  {category}: {len(funcs)} функций")
                for func in funcs[:10]:
                    # Извлекаем имя функции
                    match = re.search(r'<([^>]+)>', func)
                    if match:
                        print(f"    - {match.group(1)}")
                if len(funcs) > 10:
                    print(f"    ... (еще {len(funcs) - 10})")
    print()

def search_ubx_patterns():
    """Ищет все UBX паттерны в бинарнике"""
    print("=" * 80)
    print("31. ПОИСК ВСЕХ UBX ПАТТЕРНОВ")
    print("=" * 80)
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Файл {BINARY_PATH} не найден")
        return
    
    with open(BINARY_PATH, 'rb') as f:
        data = f.read()
    
    # UBX sync bytes
    ubx_sync = b'\xb5\x62'
    
    # Известные UBX классы
    ubx_classes = {
        0x01: "NAV",
        0x02: "RXM",
        0x04: "INF",
        0x05: "ACK",
        0x06: "CFG",
        0x0A: "MON",
        0x0B: "AID",
        0x0D: "TIM",
        0x10: "ESF",
        0x13: "MGA",
        0x21: "LOG",
        0x27: "SEC",
        0x28: "HNR"
    }
    
    # Известные CFG команды
    cfg_commands = {
        0x00: "PRT",
        0x01: "MSG",
        0x06: "INF",
        0x11: "TP5",
        0x12: "RINV",
        0x13: "ITFM",
        0x17: "GNSS",
        0x23: "TP",
        0x24: "ODO",
        0x31: "TP5",
        0x53: "ANT",
        0x84: "TMODE3",
        0x85: "TP",
        0x86: "TP5",
        0x92: "PMS"
    }
    
    positions = []
    offset = 0
    while True:
        pos = data.find(ubx_sync, offset)
        if pos == -1:
            break
        if pos + 3 < len(data):
            msg_class = data[pos + 2]
            msg_id = data[pos + 3]
            class_name = ubx_classes.get(msg_class, f"0x{msg_class:02X}")
            if msg_class == 0x06:  # CFG
                cmd_name = cfg_commands.get(msg_id, f"0x{msg_id:02X}")
                positions.append((pos, f"{class_name}-{cmd_name}"))
            else:
                positions.append((pos, f"{class_name}-0x{msg_id:02X}"))
        offset = pos + 1
    
    if positions:
        print(f"Найдено {len(positions)} UBX паттернов:")
        for pos, msg_type in positions[:50]:  # Первые 50
            print(f"  Смещение 0x{pos:X}: {msg_type}")
        if len(positions) > 50:
            print(f"\n... (показано 50 из {len(positions)})")
    else:
        print("UBX паттерны не найдены")
    print()

def main():
    print("=" * 80)
    print("РАСШИРЕННЫЙ АНАЛИЗ СТРУКТУРЫ БИНАРНИКА SHIWATIME")
    print("=" * 80)
    print()
    print("Этот скрипт выполняет полный анализ бинарника, включая:")
    print("  - Структуру ELF файла и секции")
    print("  - Используемые библиотеки и зависимости")
    print("  - Все функции и символы")
    print("  - Граф вызовов функций")
    print("  - Точки входа и инициализацию")
    print("  - Операции с файлами, сетью и временем")
    print("  - Поиск констант и паттернов UBX")
    print("  - Структуру программы в целом")
    print()
    
    if not os.path.exists(BINARY_PATH):
        print(f"✗ Ошибка: файл {BINARY_PATH} не найден")
        return 1
    
    if os.geteuid() != 0:
        print("⚠ Для полного анализа рекомендуется запустить с sudo")
        print()
    
    # Базовая информация
    analyze_file_info()
    analyze_file_type()
    analyze_libraries()
    analyze_dependencies()
    
    # ELF структура
    analyze_header()
    analyze_sections()
    analyze_program_headers()
    analyze_imports()
    
    # Символы и строки
    analyze_all_symbols()
    analyze_dynamic_symbols()
    analyze_all_strings()
    analyze_strings()
    search_config_strings()
    
    # Поиск значений
    search_binary_values()
    search_hex_patterns()
    
    # Дополнительно
    analyze_symbols()
    analyze_version_info()
    analyze_notes()
    analyze_relocations()
    
    # Дизассемблирование и функции
    analyze_functions()
    analyze_disassembly()
    
    # Структура программы
    analyze_entry_points()
    analyze_timepulse_functions()
    analyze_call_graph()
    analyze_initialization()
    analyze_file_operations()
    analyze_network_operations()
    analyze_time_operations()
    analyze_constants()
    analyze_program_structure()
    search_ubx_patterns()
    
    print("=" * 80)
    print("АНАЛИЗ ЗАВЕРШЕН")
    print("=" * 80)
    print()
    print("Для сохранения результатов в файл:")
    print("  sudo python3 analyze_shiwatime_binary.py > shiwatime_analysis.txt 2>&1")
    print()
    print("Для просмотра результатов:")
    print("  less shiwatime_analysis.txt")
    print("  или")
    print("  cat shiwatime_analysis.txt | grep -A 5 'TIMEPULSE'")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())

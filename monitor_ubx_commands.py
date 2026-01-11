#!/usr/bin/env python3
"""
Скрипт для мониторинга UBX команд, отправляемых на модуль Ublox
Позволяет отслеживать конфигурацию модуля через протокол UBX

Использование:
    sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600

Требования:
    pip3 install pyserial

Автор: Generated for TimeCard-Mini project
"""

import sys
import serial
import time
from datetime import datetime
from struct import unpack

# UBX sync characters
UBX_SYNC_CHAR1 = 0xB5
UBX_SYNC_CHAR2 = 0x62

# Известные классы UBX сообщений
UBX_CLASSES = {
    0x01: "NAV",      # Navigation
    0x02: "RXM",      # Receiver Manager
    0x04: "INF",      # Information
    0x05: "ACK",      # Acknowledgment
    0x06: "CFG",      # Configuration
    0x0A: "MON",      # Monitoring
    0x0D: "TIM",      # Timing
    0x10: "ESF",      # External Sensor Fusion
    0x21: "LOG",      # Logging
    0x27: "SEC",      # Security
    0x28: "HNR",      # High Rate Navigation
}

# Известные CFG сообщения (класс 0x06)
CFG_MSG_IDS = {
    0x00: "VAR",
    0x01: "VALGET",
    0x02: "VALSET",
    0x03: "VALDEL",
    0x06: "CFG",
    0x11: "MSG",
    0x13: "MSG-RATE",
    0x17: "INF",
    0x1B: "ITFM",
    0x1C: "ITFM-STATUS",
    0x1E: "NAV5",
    0x21: "NAVX5",
    0x23: "SBAS",
    0x24: "SIG",
    0x28: "ODO",
    0x31: "TP5",      # Time Pulse 5 (TIMEPULSE)
    0x38: "RATE",
    0x3E: "TP5",      # Alternative ID
    0x47: "TMODE2",
    0x48: "TMODE3",
    0x53: "USB",
    0x84: "GNSS",
    0x89: "LOGFILTER",
    0x8A: "NAVSPG",
    0x8B: "NAVSPG-EX",
    0x91: "RINV",
    0x92: "RINV-STATUS",
    0x93: "RINV-SUPPORTED",
    0x96: "RINV-RECORD",
    0x9B: "UART1",
    0x9C: "UART2",
    0x9D: "I2C",
    0x9E: "SPI",
    0x9F: "USBSERIALNUMBER",
    0xDD: "TXRX",
}

# TIM сообщения (класс 0x0D)
TIM_MSG_IDS = {
    0x01: "TP",
    0x03: "TM2",
    0x06: "TMODE",
    0x10: "SMEAS",
    0x11: "SVIN",
    0x12: "TOS",
    0x13: "TP2",
    0x14: "TP3",
    0x15: "VCOCAL",
    0x16: "VCOCAL-SLOW",
    0x17: "FCHG",
    0x18: "HOC",
    0x1A: "TOS2",
    0x1B: "TMODE2",
    0x1C: "TOS3",
}

def parse_ubx_packet(data):
    """Парсит UBX пакет и возвращает информацию о нем"""
    if len(data) < 8:
        return None
    
    # Проверка sync chars
    if data[0] != UBX_SYNC_CHAR1 or data[1] != UBX_SYNC_CHAR2:
        return None
    
    msg_class = data[2]
    msg_id = data[3]
    length = unpack('<H', data[4:6])[0]  # Little-endian 16-bit
    
    if len(data) < 6 + length + 2:
        return None
    
    payload = data[6:6+length]
    checksum_a = data[6+length]
    checksum_b = data[6+length+1]
    
    # Проверка контрольной суммы
    ck_a, ck_b = 0, 0
    for byte in data[2:6+length]:
        ck_a = (ck_a + byte) & 0xFF
        ck_b = (ck_b + ck_a) & 0xFF
    
    if ck_a != checksum_a or ck_b != checksum_b:
        return None
    
    class_name = UBX_CLASSES.get(msg_class, f"CLASS_{msg_class:02X}")
    
    msg_name = "UNKNOWN"
    if msg_class == 0x06:  # CFG
        msg_name = CFG_MSG_IDS.get(msg_id, f"ID_{msg_id:02X}")
    elif msg_class == 0x0D:  # TIM
        msg_name = TIM_MSG_IDS.get(msg_id, f"ID_{msg_id:02X}")
    
    return {
        'class': msg_class,
        'class_name': class_name,
        'id': msg_id,
        'msg_name': msg_name,
        'length': length,
        'payload': payload,
        'full_name': f"{class_name}-{msg_name}"
    }

def parse_cfg_tp5(payload):
    """Парсит CFG-TP5 (Time Pulse 5) сообщение"""
    if len(payload) < 32:
        return None
    
    tp_idx = payload[0]
    version = payload[1]
    reserved1 = payload[2:4]
    ant_cable_delay = unpack('<h', payload[4:6])[0]  # nanoseconds
    rf_group_delay = unpack('<h', payload[6:8])[0]   # nanoseconds
    freq_period = unpack('<I', payload[8:12])[0]     # frequency/period
    freq_period_lock = unpack('<I', payload[12:16])[0]
    pulse_len_ratio = unpack('<I', payload[16:20])[0]  # pulse length/ratio
    pulse_len_ratio_lock = unpack('<I', payload[20:24])[0]
    user_config_delay = unpack('<i', payload[24:28])[0]  # nanoseconds
    flags = unpack('<I', payload[28:32])[0]
    
    # Флаги
    active = bool(flags & 0x01)
    lockGnssFreq = bool(flags & 0x02)
    lockedOtherSet = bool(flags & 0x04)
    isFreq = bool(flags & 0x08)
    isLength = bool(flags & 0x10)
    alignToTow = bool(flags & 0x20)
    polarity = bool(flags & 0x40)
    gridUtcGnss = flags & 0x300
    
    # Если isLength = True, то pulse_len_ratio это длительность в наносекундах
    pulse_width_ns = pulse_len_ratio if isLength else None
    pulse_width_ms = pulse_width_ns / 1000000.0 if pulse_width_ns else None
    
    return {
        'tp_idx': tp_idx,
        'version': version,
        'ant_cable_delay_ns': ant_cable_delay,
        'rf_group_delay_ns': rf_group_delay,
        'freq_period': freq_period,
        'pulse_len_ratio': pulse_len_ratio,
        'pulse_width_ns': pulse_width_ns,
        'pulse_width_ms': pulse_width_ms,
        'user_config_delay_ns': user_config_delay,
        'active': active,
        'isFreq': isFreq,
        'isLength': isLength,
        'polarity': polarity,
        'alignToTow': alignToTow,
    }

def format_hex(data, max_len=32):
    """Форматирует данные в hex строку"""
    if len(data) > max_len:
        return data[:max_len].hex() + f" ... (+{len(data)-max_len} bytes)"
    return data.hex()

def monitor_serial(port, baudrate):
    """Мониторит последовательный порт и выводит UBX команды"""
    print(f"Мониторинг UBX команд на {port} (baudrate: {baudrate})")
    print(f"Время запуска: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 80)
    print()
    
    try:
        ser = serial.Serial(port, baudrate, timeout=1)
        print(f"✓ Подключено к {port}")
        print()
    except serial.SerialException as e:
        print(f"✗ Ошибка подключения к {port}: {e}")
        return 1
    
    buffer = bytearray()
    packet_count = 0
    
    try:
        while True:
            # Читаем данные
            data = ser.read(1024)
            if not data:
                time.sleep(0.01)
                continue
            
            buffer.extend(data)
            
            # Ищем UBX пакеты
            while len(buffer) >= 2:
                # Ищем sync characters
                sync_pos = -1
                for i in range(len(buffer) - 1):
                    if buffer[i] == UBX_SYNC_CHAR1 and buffer[i+1] == UBX_SYNC_CHAR2:
                        sync_pos = i
                        break
                
                if sync_pos == -1:
                    # Нет sync chars, очищаем буфер
                    if len(buffer) > 1024:
                        buffer.clear()
                    break
                
                if sync_pos > 0:
                    # Есть данные до sync, возможно это NMEA или другой протокол
                    pre_data = bytes(buffer[:sync_pos])
                    if len(pre_data) > 0:
                        # Пробуем определить NMEA
                        try:
                            text = pre_data.decode('ascii', errors='ignore')
                            if text.strip().startswith('$'):
                                print(f"[NMEA] {text.strip()}")
                        except:
                            pass
                    buffer = buffer[sync_pos:]
                
                if len(buffer) < 8:
                    break
                
                # Получаем длину пакета
                length = unpack('<H', buffer[4:6])[0]
                packet_size = 8 + length  # sync(2) + class(1) + id(1) + length(2) + payload(length) + checksum(2)
                
                if len(buffer) < packet_size:
                    break
                
                # Извлекаем пакет
                packet_data = bytes(buffer[:packet_size])
                buffer = buffer[packet_size:]
                
                # Парсим пакет
                packet_info = parse_ubx_packet(packet_data)
                if packet_info:
                    packet_count += 1
                    timestamp = datetime.now().strftime('%H:%M:%S.%f')[:-3]
                    
                    print(f"[{timestamp}] UBX Packet #{packet_count}")
                    print(f"  Сообщение: {packet_info['full_name']} ({packet_info['class_name']}-{packet_info['msg_name']})")
                    print(f"  Класс: 0x{packet_info['class']:02X}, ID: 0x{packet_info['id']:02X}")
                    print(f"  Длина payload: {packet_info['length']} bytes")
                    
                    # Специальная обработка для CFG-TP5 (Time Pulse 5)
                    if packet_info['full_name'] == 'CFG-TP5':
                        tp5_info = parse_cfg_tp5(packet_info['payload'])
                        if tp5_info:
                            print(f"  ⏱️  TIMEPULSE конфигурация:")
                            print(f"     Индекс TP: {tp5_info['tp_idx']}")
                            print(f"     Активен: {tp5_info['active']}")
                            if tp5_info['pulse_width_ms']:
                                print(f"     Длительность импульса: {tp5_info['pulse_width_ms']:.3f} мс ({tp5_info['pulse_width_ns']} нс)")
                                if tp5_info['pulse_width_ms'] != 5.0:
                                    print(f"     ⚠️  ВНИМАНИЕ: Длительность импульса = {tp5_info['pulse_width_ms']:.1f} мс (требуется 5 мс)")
                            else:
                                print(f"     Длительность: N/A")
                            print(f"     Частота/Период: {tp5_info['freq_period']}")
                            print(f"     isLength: {tp5_info['isLength']}, isFreq: {tp5_info['isFreq']}")
                            print(f"     Полярность: {'HIGH' if tp5_info['polarity'] else 'LOW'}")
                            print(f"     Выравнивание к TOW: {tp5_info['alignToTow']}")
                            print(f"     Задержка кабеля антенны: {tp5_info['ant_cable_delay_ns']} нс")
                            print(f"     Задержка RF группы: {tp5_info['rf_group_delay_ns']} нс")
                    
                    print(f"  Payload (hex): {format_hex(packet_info['payload'])}")
                    print()
                else:
                    # Невалидный пакет, пропускаем один байт
                    buffer = buffer[1:]
    
    except KeyboardInterrupt:
        print()
        print("=" * 80)
        print(f"Остановлено пользователем")
        print(f"Всего обработано UBX пакетов: {packet_count}")
    except Exception as e:
        print(f"\n✗ Ошибка: {e}")
        import traceback
        traceback.print_exc()
        return 1
    finally:
        ser.close()
    
    return 0

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Использование: sudo python3 monitor_ubx_commands.py <port> <baudrate>")
        print("Пример: sudo python3 monitor_ubx_commands.py /dev/ttyS0 9600")
        sys.exit(1)
    
    port = sys.argv[1]
    try:
        baudrate = int(sys.argv[2])
    except ValueError:
        print(f"Ошибка: некорректная скорость {sys.argv[2]}")
        sys.exit(1)
    
    sys.exit(monitor_serial(port, baudrate))


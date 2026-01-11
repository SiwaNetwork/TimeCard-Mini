#!/usr/bin/env python3
"""
Скрипт для конфигурации длительности импульса PPS на модуле Ublox через UBX протокол

Использование:
    sudo python3 configure_ublox_timepulse.py /dev/ttyS0 115200 --pulse-width-ms 5

Требования:
    pip3 install pyserial

Автор: Generated for TimeCard-Mini project
"""

import sys
import serial
import time
import argparse
from struct import pack

# UBX sync characters
UBX_SYNC_CHAR1 = 0xB5
UBX_SYNC_CHAR2 = 0x62

# UBX Class CFG (0x06)
UBX_CFG_CLASS = 0x06

# UBX ID для TIMEPULSE (CFG-TP5)
UBX_CFG_TP5_ID = 0x31

# UBX ID для VALGET (получить значение)
UBX_CFG_VALGET_ID = 0x08

# UBX ID для VALSET (установить значение)
UBX_CFG_VALSET_ID = 0x8A

def calculate_checksum(data):
    """Вычисляет контрольную сумму UBX пакета"""
    ck_a, ck_b = 0, 0
    for byte in data:
        ck_a = (ck_a + byte) & 0xFF
        ck_b = (ck_b + ck_a) & 0xFF
    return ck_a, ck_b

def create_ubx_packet(msg_class, msg_id, payload):
    """Создает UBX пакет"""
    packet = bytearray()
    packet.append(UBX_SYNC_CHAR1)
    packet.append(UBX_SYNC_CHAR2)
    packet.append(msg_class)
    packet.append(msg_id)
    packet.extend(pack('<H', len(payload)))  # Length (little-endian)
    packet.extend(payload)
    
    # Calculate checksum
    ck_a, ck_b = calculate_checksum(packet[2:])
    packet.append(ck_a)
    packet.append(ck_b)
    
    return bytes(packet)

def create_cfg_tp5_packet(tp_idx=0, pulse_width_ms=5.0, freq_period=1000000, 
                         ant_cable_delay_ns=0, rf_group_delay_ns=0,
                         user_config_delay_ns=0, active=True, 
                         align_to_tow=True, polarity=False):
    """
    Создает CFG-TP5 (Time Pulse 5) пакет
    
    Параметры:
    - tp_idx: Индекс time pulse (0-1)
    - pulse_width_ms: Длительность импульса в миллисекундах
    - freq_period: Частота/период в мкГц (по умолчанию 1 Гц = 1000000 мкГц)
    - ant_cable_delay_ns: Задержка кабеля антенны в наносекундах
    - rf_group_delay_ns: Задержка RF группы в наносекундах
    - user_config_delay_ns: Пользовательская задержка в наносекундах
    - active: Активен ли time pulse
    - align_to_tow: Выравнивание к TOW (Time Of Week)
    - polarity: Полярность (False = LOW, True = HIGH)
    """
    payload = bytearray(32)
    
    payload[0] = tp_idx  # tpIdx
    payload[1] = 0  # version
    payload[2:4] = pack('<H', 0)  # reserved1
    
    payload[4:6] = pack('<h', ant_cable_delay_ns)  # antCableDelay (signed 16-bit)
    payload[6:8] = pack('<h', rf_group_delay_ns)   # rfGroupDelay (signed 16-bit)
    
    # freqPeriod (unsigned 32-bit)
    payload[8:12] = pack('<I', freq_period)
    
    # freqPeriodLock (unsigned 32-bit) - то же значение при lock
    payload[12:16] = pack('<I', freq_period)
    
    # pulseLenRatio (unsigned 32-bit) - длительность в наносекундах
    pulse_width_ns = int(pulse_width_ms * 1000000)
    payload[16:20] = pack('<I', pulse_width_ns)
    
    # pulseLenRatioLock (unsigned 32-bit) - то же значение при lock
    payload[20:24] = pack('<I', pulse_width_ns)
    
    # userConfigDelay (signed 32-bit)
    payload[24:28] = pack('<i', user_config_delay_ns)
    
    # Flags (unsigned 32-bit)
    flags = 0
    if active:
        flags |= 0x01  # active
    flags |= 0x02  # lockGnssFreq
    flags |= 0x04  # lockedOtherSet
    flags |= 0x10  # isLength (используем длительность, а не частоту)
    if align_to_tow:
        flags |= 0x20  # alignToTow
    if polarity:
        flags |= 0x40  # polarity
    
    payload[28:32] = pack('<I', flags)
    
    return create_ubx_packet(UBX_CFG_CLASS, UBX_CFG_TP5_ID, payload)

def read_ubx_response(ser, timeout=2.0):
    """Читает UBX ответ от модуля"""
    start_time = time.time()
    buffer = bytearray()
    
    while time.time() - start_time < timeout:
        data = ser.read(1024)
        if data:
            buffer.extend(data)
            # Ищем UBX пакет
            if len(buffer) >= 8:
                for i in range(len(buffer) - 1):
                    if buffer[i] == UBX_SYNC_CHAR1 and buffer[i+1] == UBX_SYNC_CHAR2:
                        if i > 0:
                            buffer = buffer[i:]
                        break
                
                if len(buffer) >= 8:
                    length = int.from_bytes(buffer[4:6], byteorder='little')
                    packet_size = 8 + length
                    if len(buffer) >= packet_size:
                        packet = bytes(buffer[:packet_size])
                        buffer = buffer[packet_size:]
                        return packet
        time.sleep(0.01)
    
    return None

def configure_timepulse(port, baudrate, pulse_width_ms, tp_idx=0, 
                       ant_cable_delay_ns=0, verbose=True):
    """Конфигурирует time pulse на модуле Ublox"""
    
    if verbose:
        print(f"Конфигурация Time Pulse на {port} (baudrate: {baudrate})")
        print(f"Длительность импульса: {pulse_width_ms} мс")
        print(f"Индекс TP: {tp_idx}")
        print()
    
    try:
        ser = serial.Serial(port, baudrate, timeout=1)
        if verbose:
            print(f"✓ Подключено к {port}")
            time.sleep(0.5)  # Даем модулю время инициализироваться
    except serial.SerialException as e:
        print(f"✗ Ошибка подключения к {port}: {e}")
        return False
    
    try:
        # Создаем пакет CFG-TP5
        packet = create_cfg_tp5_packet(
            tp_idx=tp_idx,
            pulse_width_ms=pulse_width_ms,
            ant_cable_delay_ns=ant_cable_delay_ns,
            active=True,
            align_to_tow=True,
            polarity=False
        )
        
        if verbose:
            print(f"Отправка CFG-TP5 пакета...")
            print(f"  Длительность импульса: {pulse_width_ms} мс ({int(pulse_width_ms * 1000000)} нс)")
        
        # Отправляем пакет
        ser.write(packet)
        ser.flush()
        
        # Ждем ответа (ACK или NACK)
        response = read_ubx_response(ser, timeout=2.0)
        
        if response:
            if len(response) >= 10:
                msg_class = response[2]
                msg_id = response[3]
                
                if msg_class == 0x05:  # ACK class
                    if msg_id == 0x01:  # ACK
                        if verbose:
                            print("✓ Конфигурация принята (ACK)")
                        return True
                    elif msg_id == 0x00:  # NACK
                        if verbose:
                            print("✗ Конфигурация отклонена (NACK)")
                        return False
        
        if verbose:
            print("⚠ Ответ не получен или не распознан")
        return False
    
    except Exception as e:
        print(f"✗ Ошибка: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        ser.close()

def main():
    parser = argparse.ArgumentParser(
        description='Конфигурирует длительность импульса PPS на модуле Ublox'
    )
    parser.add_argument('port', help='Последовательный порт (например, /dev/ttyS0)')
    parser.add_argument('baudrate', type=int, help='Скорость передачи (для Raspberry CM4: 9600)')
    parser.add_argument('--pulse-width-ms', type=float, default=5.0,
                       help='Длительность импульса в миллисекундах (по умолчанию: 5.0)')
    parser.add_argument('--tp-idx', type=int, default=0,
                       help='Индекс time pulse (0 или 1, по умолчанию: 0)')
    parser.add_argument('--ant-cable-delay-ns', type=int, default=0,
                       help='Задержка кабеля антенны в наносекундах (по умолчанию: 0)')
    parser.add_argument('--quiet', action='store_true',
                       help='Тихий режим (минимальный вывод)')
    
    args = parser.parse_args()
    
    success = configure_timepulse(
        args.port,
        args.baudrate,
        args.pulse_width_ms,
        args.tp_idx,
        args.ant_cable_delay_ns,
        verbose=not args.quiet
    )
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()


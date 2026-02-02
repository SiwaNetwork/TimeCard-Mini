#!/usr/bin/env python3
"""
Декодирование коэффициентов из hex-дампа coeffs_final_v2.txt.
Использует данные, извлечённые objdump на устройстве.
"""
import struct

# Hex из objdump .noptrdata (little-endian)
# 770b7e0: -1.0, -1.0, -1.0, -1.0
# 770b8a0: коэффициенты (float64)
hex_pairs = [
    ("770b7e0", "000000000000f0bf"),  # -1.0
    ("770b8a0", "000000000000e03f"),  # 0.5
    ("770b8a8", "15b7310afe06e33f"),
    ("770b8b0", "cc3b7f669ea0e63f"),
    ("770b8b8", "acd35a999fe8ea3f"),
]

def hex_to_double(hexstr):
    """Little-endian 8 bytes to double."""
    b = bytes.fromhex(hexstr.replace(" ", ""))
    if len(b) != 8:
        return None
    return struct.unpack("<d", b)[0]

print("=" * 60)
print("ДЕКОДИРОВАНИЕ КОЭФФИЦИЕНТОВ ИЗ HEX (DefaultAlgoCoefficients)")
print("=" * 60)
for addr, hexstr in hex_pairs:
    try:
        val = hex_to_double(hexstr)
        print(f"  {addr}: {hexstr} -> {val}")
    except Exception as e:
        print(f"  {addr}: ошибка {e}")

# Возможные Kp, Ki, Kd (смещения 0, 8, 16 от 770b8a0)
print()
print("Интерпретация как Kp, Ki, Kd (offset 0, 8, 16):")
vals = [
    hex_to_double("000000000000e03f"),
    hex_to_double("15b7310afe06e33f"),
    hex_to_double("cc3b7f669ea0e63f"),
]
print(f"  Kp = {vals[0]}")
print(f"  Ki = {vals[1]}")
print(f"  Kd = {vals[2]}")
print()
print("Альтернатива (offset 8, 16, 24):")
vals2 = [
    hex_to_double("15b7310afe06e33f"),
    hex_to_double("cc3b7f669ea0e63f"),
    hex_to_double("acd35a999fe8ea3f"),
]
print(f"  Kp = {vals2[0]}")
print(f"  Ki = {vals2[1]}")
print(f"  Kd = {vals2[2]}")

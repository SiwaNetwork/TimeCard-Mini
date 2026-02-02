#!/usr/bin/env python3
"""
Извлечение дизассемблера целевых функций из бинарника timebeat.
Нужно для строгой реконструкции: реализация должна повторять бинарник, а не быть «по аналогии».

Использование (в среде с objdump, например WSL):
  python3 code_analysis/extract_disassembly_for_functions.py BINARY [--output DIR] [--functions NAME ...]
  python3 code_analysis/extract_disassembly_for_functions.py timebeat-extracted/usr/share/timebeat/bin/timebeat \\
    --output code_analysis/disassembly \\
    --functions "clocksync/servo/algos" --functions "clients/ntp"

Или без --functions: извлекаются все символы из .text с путём timebeat (длинный вывод).
"""

import argparse
import os
import re
import subprocess
import sys

def run_cmd(cmd, timeout=60):
    if isinstance(cmd, str):
        cmd = ["sh", "-c", cmd]
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        return r.stdout if r.returncode == 0 else ""
    except Exception as e:
        return ""

def get_symbol_table(binary):
    """objdump -t: адрес, размер и имя символа (если бинарник не stripped)."""
    out = run_cmd(["objdump", "-t", binary])
    if not out:
        return []
    # Формат: 0000000000412340 g F .text 0000000000000123 github.com/.../pkg.func
    symbols = []
    for line in out.splitlines():
        parts = line.split(None, 4)
        if len(parts) < 5:
            continue
        try:
            addr = int(parts[0], 16)
            size_hex = parts[3]
            size = int(size_hex, 16) if size_hex != "" else 0
            name = parts[4].strip() if len(parts) > 4 else ""
            if name and (".text" in line or parts[2] == ".text"):
                symbols.append((addr, size, name))
        except (ValueError, IndexError):
            continue
    return symbols

def get_symbol_table_nm(binary):
    """nm -D: адрес и имя (размер 0). Для stripped бинарников, где objdump -t пустой."""
    out = run_cmd(["nm", "-D", binary])
    if not out:
        return []
    # Формат: 0000000004595c00 T github.com/.../pkg.func
    symbols = []
    for line in out.splitlines():
        parts = line.split(None, 2)
        if len(parts) < 3:
            continue
        try:
            addr = int(parts[0], 16)
            symtype = parts[1]
            name = parts[2].strip()
            if name and symtype in ("T", "t"):  # text/code
                symbols.append((addr, 0, name))
        except (ValueError, IndexError):
            continue
    return symbols

def filter_symbols(symbols, patterns):
    """Оставить символы, у которых имя содержит любой из patterns."""
    if not patterns:
        return symbols
    out = []
    for addr, size, name in symbols:
        for p in patterns:
            if p in name:
                out.append((addr, size, name))
                break
    return out

def next_addr(symbols, addr):
    """Ближайший следующий адрес после addr (для оценки границы функции без размера)."""
    sorted_addrs = sorted(set(a for a, _, _ in symbols))
    for a in sorted_addrs:
        if a > addr:
            return a
    return addr + 0x2000  # fallback

def disassemble_range(binary, start, stop):
    return run_cmd(["objdump", "-d", "-C", "--start-address", hex(start), "--stop-address", hex(stop), binary])

def sanitize_filename(name):
    s = re.sub(r'[^\w\-.]', '_', name)
    return s[:120] if len(s) > 120 else s

def main():
    ap = argparse.ArgumentParser(description="Извлечение дизассемблера функций из бинарника timebeat для строгой реконструкции.")
    ap.add_argument("binary", help="Путь к бинарнику (например timebeat-extracted/usr/share/timebeat/bin/timebeat)")
    ap.add_argument("--output", "-o", default="code_analysis/disassembly", help="Каталог для дампов дизассемблера")
    ap.add_argument("--functions", "-f", action="append", default=[], help="Подстрока в имени символа (можно несколько). Пусто = все timebeat символы.")
    ap.add_argument("--max-size", type=int, default=0x3000, help="Макс. размер дампа на функцию (байт)")
    args = ap.parse_args()

    if not os.path.isfile(args.binary):
        print(f"Ошибка: бинарник не найден: {args.binary}", file=sys.stderr)
        sys.exit(1)

    symbols = get_symbol_table(args.binary)
    if not symbols:
        symbols = get_symbol_table_nm(args.binary)
    if not symbols:
        print("Не удалось получить символы (objdump -t и nm -D). Проверьте путь и наличие objdump/nm.", file=sys.stderr)
        sys.exit(1)

    # Фильтр: только timebeat и при необходимости по --functions
    timebeat = [(a, s, n) for a, s, n in symbols if "timebeat" in n or "lasselj" in n]
    if args.functions:
        timebeat = filter_symbols(timebeat, args.functions)
    if not timebeat:
        print("Нет символов по заданным фильтрам.", file=sys.stderr)
        sys.exit(1)

    os.makedirs(args.output, exist_ok=True)
    index_path = os.path.join(args.output, "index.txt")
    index_lines = []

    for addr, size, name in sorted(timebeat, key=lambda x: x[0]):
        stop = addr + (size if size > 0 else args.max_size)
        if size <= 0:
            stop = min(stop, next_addr([(a, 0, "") for a, _, _ in timebeat], addr))
        stop = min(stop, addr + args.max_size)

        disasm = disassemble_range(args.binary, addr, stop)
        if not disasm:
            continue

        fname = sanitize_filename(name) + ".txt"
        fpath = os.path.join(args.output, fname)
        with open(fpath, "w", encoding="utf-8") as f:
            f.write(f"# {name}\n# addr 0x{addr:x} size {size or (stop - addr)} (0x{stop - addr:x})\n\n")
            f.write(disasm)
        index_lines.append(f"0x{addr:x}\t{size or (stop-addr)}\t{name}\t{fname}")

    with open(index_path, "w", encoding="utf-8") as f:
        f.write("\n".join(index_lines))

    print(f"Символов: {len(timebeat)}")
    print(f"Дизассемблер сохранён в: {args.output}/")
    print(f"Индекс: {index_path}")

if __name__ == "__main__":
    main()

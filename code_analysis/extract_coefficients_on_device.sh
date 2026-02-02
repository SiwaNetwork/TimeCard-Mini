#!/bin/bash
# Извлечение коэффициентов из бинарника shiwatime на целевом устройстве (Linux)
# Запускать на машине с установленным shiwatime: /usr/share/shiwatime/bin/shiwatime

BINARY="${1:-/usr/share/shiwatime/bin/shiwatime}"
OUTPUT="${2:-coeffs_extracted.txt}"

if [ ! -f "$BINARY" ]; then
    echo "Бинарник не найден: $BINARY"
    echo "Использование: $0 [путь_к_shiwatime] [файл_вывода]"
    exit 1
fi

echo "=== Извлечение коэффициентов из $BINARY ===" | tee "$OUTPUT"
echo "" | tee -a "$OUTPUT"

# D-массив (0x770a430), 3 float64 = 24 байта
echo "--- D-массив (0x770a430), 3 float64 ---" | tee -a "$OUTPUT"
objdump -s -j .data "$BINARY" 2>/dev/null | grep -A 2 "770a4" | tee -a "$OUTPUT" || echo "Не найдено в .data" | tee -a "$OUTPUT"
objdump -s -j .noptrdata "$BINARY" 2>/dev/null | grep -A 2 "770a4" | tee -a "$OUTPUT" || true
echo "" | tee -a "$OUTPUT"

# DefaultAlgoCoefficients (0x770b7e0)
echo "--- DefaultAlgoCoefficients (0x770b7e0) ---" | tee -a "$OUTPUT"
objdump -s -j .noptrdata "$BINARY" 2>/dev/null | grep -A 20 "770b7e0" | tee -a "$OUTPUT" || \
objdump -s -j .data "$BINARY" 2>/dev/null | grep -A 20 "770b7e0" | tee -a "$OUTPUT" || echo "Не найдено" | tee -a "$OUTPUT"
echo "" | tee -a "$OUTPUT"

# Альтернатива: gdb (если доступен)
if command -v gdb &>/dev/null; then
    echo "--- Чтение через gdb (требует запуска) ---" | tee -a "$OUTPUT"
    echo "gdb -batch -ex 'x/3g 0x770a430' -ex 'x/16g 0x770b7e0' $BINARY 2>/dev/null" | tee -a "$OUTPUT"
    gdb -batch -ex "x/3g 0x770a430" -ex "x/16g 0x770b7e0" "$BINARY" 2>/dev/null | tee -a "$OUTPUT" || true
fi

echo "" | tee -a "$OUTPUT"
echo "Результаты сохранены в $OUTPUT"
echo "См. code_analysis/FOUND_COEFFICIENTS.md для интерпретации"

#!/bin/bash
# Извлечение коэффициентов из локального бинарника timebeat (x86-64 или ARM)
# Использование: ./extract_from_local_binary.sh [путь_к_бинарнику] [файл_вывода]
#
# Примеры:
#   ./extract_from_local_binary.sh timebeat-2.2.20-amd64/timebeat-extracted/usr/share/timebeat/bin/timebeat
#   ./extract_from_local_binary.sh /usr/share/shiwatime/bin/shiwatime

BINARY="${1}"
OUTPUT="${2:-coeffs_extracted.txt}"

if [ -z "$BINARY" ]; then
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    BINARY="$REPO_ROOT/timebeat-extracted/usr/share/timebeat/bin/timebeat"
fi

if [ ! -f "$BINARY" ]; then
    echo "Бинарник не найден: $BINARY"
    echo ""
    echo "Извлеките deb: dpkg-deb -x timebeat-2.2.20-amd64.deb timebeat-extracted"
    echo "Или укажите путь: $0 /path/to/timebeat [output.txt]"
    exit 1
fi

ARCH=$(file "$BINARY" | grep -o 'x86-64\|aarch64\|arm64' || echo "unknown")
echo "=== Извлечение коэффициентов ===" | tee "$OUTPUT"
echo "Бинарник: $BINARY" | tee -a "$OUTPUT"
echo "Архитектура: $ARCH" | tee -a "$OUTPUT"
echo "" | tee -a "$OUTPUT"

# Ищем блок коэффициентов по уникальному паттерну (0.5, 0.5946)
# hex: 00000000 0000e03f 15b7310a fe06e33f
echo "--- DefaultAlgoCoefficients (поиск по паттерну 0.5, 0.5946) ---" | tee -a "$OUTPUT"
objdump -s -j .noptrdata "$BINARY" 2>/dev/null | grep -E "0000e03f 15b7310a|15b7310a fe06e33f" | tee -a "$OUTPUT" || true
objdump -s -j .data "$BINARY" 2>/dev/null | grep -E "0000e03f 15b7310a|15b7310a fe06e33f" | tee -a "$OUTPUT" || true

echo "" | tee -a "$OUTPUT"
echo "--- Блок -1.0 (начало DefaultAlgoCoefficients) ---" | tee -a "$OUTPUT"
objdump -s -j .noptrdata "$BINARY" 2>/dev/null | grep -B1 -A5 "0000f0bf 00000000 0000f0bf" | head -20 | tee -a "$OUTPUT" || true

echo "" | tee -a "$OUTPUT"
echo "Результаты сохранены в $OUTPUT"

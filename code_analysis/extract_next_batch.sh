#!/bin/sh
# Извлечение дизассемблера следующей партии функций для реконструкции (запускать в WSL или среде с objdump/nm).
# См. RECONSTRUCTION_PLAN.md.

BIN="${1:-timebeat-extracted/usr/share/timebeat/bin/timebeat}"
OUT="${2:-code_analysis/disassembly}"

if [ ! -f "$BIN" ]; then
  echo "Бинарник не найден: $BIN"
  exit 1
fi

python3 code_analysis/extract_disassembly_for_functions.py "$BIN" -o "$OUT" \
  -f "NoneGaussianFilter.IsFiltered" \
  -f "IsFiltered" \
  -f "phc.(*PHCDevice).DeterminePTPOffsetBasic" \
  -f "phc.(*PHCDevice).GetPHCToSysClockSamplesBasic"

echo "Готово. Файлы в $OUT/"

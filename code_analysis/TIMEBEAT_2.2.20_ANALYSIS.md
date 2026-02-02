# Анализ timebeat-2.2.20-amd64

Бинарник и deb-пакет из папки `timebeat-2.2.20-amd64/`.

## Содержимое

| Файл | Описание |
|------|----------|
| `timebeat-2.2.20-amd64.deb` | Debian-пакет Timebeat 2.2.20 (amd64) |
| `timebeat-2.2.20-amd64` | tar-архив (распакованный deb) |

## Извлечение бинарника

```bash
dpkg-deb -x timebeat-2.2.20-amd64.deb timebeat-extracted
# Бинарник: timebeat-extracted/usr/share/timebeat/bin/timebeat
```

## Характеристики бинарника

- **Формат:** ELF 64-bit LSB, x86-64
- **Сборка:** Go (BuildID, stripped)
- **Размер:** ~132 MB
- **Пути:** `github.com/lasselj/timebeat/beater/clocksync/...`

## Извлечённые коэффициенты (DefaultAlgoCoefficients)

**Адрес в x86-64:** 0x7c1a040 (в секции .noptrdata)

| Offset | Hex | Float64 |
|--------|-----|---------|
| 0 | 00000000 0000e03f | **0.5** |
| 8 | 15b7310a fe06e33f | **0.5946** |
| 16 | cc3b7f66 9ea0e63f | **0.7071** (= 1/√2) |
| 24 | acd35a99 9fe8ea3f | **0.8409** |

**Интерпретация:** Kp=0.5, Ki=0.5946, Kd=0.7071

Значения **совпадают** с бинарником shiwatime (ARM64 на grandmini) — одна и та же кодовая база.

## Запуск извлечения

```bash
# Из корня репо (после dpkg-deb -x ... timebeat-extracted)
./code_analysis/extract_from_local_binary.sh timebeat-extracted/usr/share/timebeat/bin/timebeat code_analysis/coeffs_extracted.txt
```

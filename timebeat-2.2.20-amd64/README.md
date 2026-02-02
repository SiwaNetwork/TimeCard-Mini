# timebeat-2.2.20-amd64

Официальный бинарник и deb-пакет Timebeat 2.2.20 (amd64).

## Содержимое

- `timebeat-2.2.20-amd64.deb` — Debian-пакет
- `timebeat-2.2.20-amd64` — tar-архив

## Извлечение бинарника

```bash
cd /path/to/TimeCard-Mini
dpkg-deb -x timebeat-2.2.20-amd64.deb timebeat-extracted
```

Бинарник: `timebeat-extracted/usr/share/timebeat/bin/timebeat`

## Анализ коэффициентов

```bash
./code_analysis/extract_from_local_binary.sh timebeat-extracted/usr/share/timebeat/bin/timebeat code_analysis/coeffs_extracted.txt
```

См. `code_analysis/TIMEBEAT_2.2.20_ANALYSIS.md`.

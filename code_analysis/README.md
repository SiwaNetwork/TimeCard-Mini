# Code Analysis

Эта папка содержит все файлы, связанные с анализом кода программы `shiwatime`.

## Структура

### Python скрипты анализа

- **`advanced_binary_analysis.py`** - Продвинутый анализ бинарника
- **`analyze_found_servo_functions.py`** - Анализ найденных servo функций
- **`analyze_servo_algorithms.py`** - Анализ servo алгоритмов
- **`analyze_shiwatime_binary.py`** - Анализ бинарника shiwatime
- **`analyze_ubx_structure.py`** - Анализ структуры UBX
- **`check_completeness.py`** - Проверка полноты анализа
- **`deep_analyze_clocksync.py`** - Глубокий анализ clocksync
- **`deep_binary_analysis.py`** - Глубокий анализ бинарника
- **`extract_algorithm_details.py`** - Извлечение деталей алгоритмов
- **`extract_algorithms.py`** - Извлечение алгоритмов
- **`extract_clocksync_modules.py`** - Извлечение модулей clocksync
- **`extract_coefficients_v2.py`** - Извлечение коэффициентов (версия 2)
- **`extract_coefficients.py`** - Извлечение коэффициентов
- **`find_servo_by_names.py`** - Поиск servo функций по именам
- **`find_servo_functions.py`** - Поиск servo функций
- **`monitor_ubx_commands.py`** - Мониторинг UBX команд

### Markdown документы

- **`ANALYSIS_PLAN_TO_100.md`** - План анализа до 100% готовности
- **`COEFFICIENTS_ANALYSIS.md`** - Анализ коэффициентов
- **`COMPLETE_FUNCTIONALITY_CHECK.md`** - Проверка полной функциональности
- **`COMPLETENESS_ASSESSMENT.md`** - Оценка полноты анализа
- **`EXTRACTED_COEFFICIENTS.md`** - Извлеченные коэффициенты
- **`FOUND_COEFFICIENTS.md`** - Найденные коэффициенты
- **`MASTER_ANALYSIS_REPORT.md`** - Главный отчет анализа
- **`README_ANALYSIS.md`** - README для анализа
- **`RUN_ANALYSIS_ON_DEVICE.md`** - Инструкции по запуску анализа на устройстве

### Результаты анализа

- **`algorithm_details.txt`** - Детали алгоритмов
- **`coeffs_final_v2.txt`** - Финальные коэффициенты (версия 2)
- **`coeffs_simple.txt`** - Простые коэффициенты
- **`completeness_check.txt`** - Результаты проверки полноты
- **`servo_functions_detailed_analysis.txt`** - Детальный анализ servo функций

### Графы и диаграммы

- **`clocksync_call_graph.dot`** - Граф вызовов clocksync

### Shell скрипты

- **`RUN_ANALYSIS.sh`** - Скрипт запуска анализа
- **`RUN_COMPLETE_ANALYSIS.sh`** - Скрипт запуска полного анализа

## Использование

Для запуска анализа на устройстве:

```bash
# Скопировать скрипты на устройство
scp code_analysis/*.py shiwa@grandmini.local:~/

# Запустить анализ
ssh shiwa@grandmini.local
python3 extract_coefficients_v2.py
```

## Статус анализа

Текущий статус анализа можно найти в файле `MASTER_ANALYSIS_REPORT.md`.

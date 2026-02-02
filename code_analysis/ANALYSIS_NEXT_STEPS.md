# План дальнейшего анализа бинарника

Документ описывает шаги для продолжения разбора shiwatime и доведения tc-sync до полного соответствия.

---

## 1. Извлечь числовые коэффициенты

### На целевом устройстве (Linux с shiwatime)

```bash
# Скопировать скрипт на устройство и выполнить
./code_analysis/extract_coefficients_on_device.sh /usr/share/shiwatime/bin/shiwatime coeffs_extracted.txt
```

### Вручную через objdump/gdb

```bash
# D-массив (3 float64) по адресу 0x770a430
objdump -s -j .data /usr/share/shiwatime/bin/shiwatime | grep -A 3 "770a4"

# DefaultAlgoCoefficients по 0x770b7e0
objdump -s -j .noptrdata /usr/share/shiwatime/bin/shiwatime | grep -A 20 "770b7e0"

# Или через gdb (во время выполнения или при загрузке)
gdb -batch -ex "x/3g 0x770a430" /usr/share/shiwatime/bin/shiwatime
gdb -batch -ex "x/16g 0x770b7e0" /usr/share/shiwatime/bin/shiwatime
```

### Куда подставить полученные значения

- **D-массив** → `servo.PID.DCoeffs` в коде или через конфиг `servo.d_coeffs: [d0, d1, d2]`
- **DefaultAlgoCoefficients** → Kp, Ki, Kd в дефолтах конфига

---

## 2. Уже реализовано по анализу

| Компонент | Файл | Описание |
|-----------|------|----------|
| **PI shiwatime-style** | `internal/servo/servo.go` | `algorithm: pi_shiwatime` — формула I += (1e9 - I) * offset_diff / time_diff |
| **PID D-массив** | `internal/servo/servo.go` | Поле `DCoeffs [3]float64` в PID; при задании переопределяет Kd |
| **LinReg окно 64** | `internal/servo/linreg.go` | Реализовано |
| **Константа 1e9** | `internal/servo/servo.go` | IntegralTarget = 1e9 (0x3b9aca00, не 51712 — см. ассемблер) |

---

## 3. Функции для углублённого анализа

### По приоритету

1. **adjustDComponent** (0x41c8bc0) — точная формула индекса для D-массива
2. **enforceAdjustmentLimit** — лимиты max/min adjustment
3. **BestFitFiltered** — фильтр в PID (если используется)
4. **extended_step_limits** — boundary/limit для step (forward/backward)
5. **send1PPSOnTimepulsePin** — вывод 1-PPS
6. **detectUbloxUnit** — автоопределение типа приёмника

### Команды для дизассемблирования

```bash
# Найти функцию по имени
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -B 1 "adjustDComponent"

# Или по адресу
objdump -d /usr/share/shiwatime/bin/shiwatime | grep -A 80 "41c8bc0"
```

---

## 4. Структура конфига для новых опций

Пример добавления в `tc-sync.yml`:

```yaml
servo:
  algorithm: pi_shiwatime  # или pid, pi, linreg
  kp: 0.1
  ki: 0.01
  kd: 0.001
  # d_coeffs: [0.001, 0.002, 0.003]  # опционально, для PID с D-массивом
  interval: 1s
```

---

## 5. Ссылки

- `FOUND_COEFFICIENTS.md` — структуры и смещения
- `COEFFICIENTS_ANALYSIS.md` — детали формул
- `IMPLEMENTATION_GAP.md` — разрыв анализ/реализация
- `program_structure.go` — карта программы

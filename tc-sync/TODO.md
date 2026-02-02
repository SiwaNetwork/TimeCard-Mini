# Что доделать в tc-sync

Краткий список оставшихся и опциональных доработок (по приоритету).

---

## Высокий приоритет

### 1. ~~**step_limit из конфига**~~ — сделано: парсится в `pkg/clocksync.ParseStepLimit`, используется в main и RunDaemon.

### 2. ~~**Graceful shutdown по SIGTERM/SIGINT**~~ — сделано: main вызывает `clocksync.RunDaemon(ctx, ...)` с контекстом, по SIGINT/SIGTERM вызывается `cancel()`, ptp4l и источники останавливаются.

---

## Средний приоритет

### 3. ~~**Коэффициенты servo из анализа**~~ — дефолты Kp/Ki/Kd документированы (см. `FOUND_COEFFICIENTS.md`); при отсутствии в конфиге подставляются через `applyDefaults` (0.1, 0.01, 0.001).

### 4. ~~**Unit-тесты (базовые)**~~ — добавлены: `internal/servo/servo_test.go` (PID, PI, LinReg), `pkg/clocksync/run_test.go` (ParseStepLimit), `internal/ubx/navpvt_test.go` (ParseNAVPVTTime, IsNAVPVTPacket, NAVPVTPayload), `internal/clockselect/select_test.go` (Election Select/GetTimeFromActive).

### 5. ~~**Логирование**~~ — единый вывод через `internal/logger`: префикс "tc-sync: ", Quiet для Info, Error всегда. Используется в main, clocksync, ptp4l. При необходимости: структурированные поля и/или метрики (offset, источник, шаг/slew) для Prometheus.

---

## Низкий приоритет / по необходимости

### 6. **Расширение UBX TP5**
- В анализе — 62 смещения; в tc-sync используются первые 32 байта payload.
- Расширять только если понадобятся дополнительные поля u-blox (остальные 56 полей).

### 7. **send1PPSOnTimepulsePin / detectUbloxUnit**
- Из анализа бинарника; в tc-sync не реализованы.
- Имеет смысл только при интеграции с 1-PPS по пину или автоопределении типа приёмника.

### 8. **phc2sys внутри tc-sync**
- Опционально: запускать `phc2sys` как дочерний процесс (аналогично ptp4l), если нужно синхронизировать PHC → системные часы отдельным демоном вместо нашего servo.

### 9. **Отдельный тип HostClock**
- В анализе есть сущность HostClock (GetTimeNow, StepClock, SlewClock).
- Сейчас логика размазана по clocksync + clockadj; введение HostClock — только для ясности архитектуры, не обязательно для работы.

---

## Уже сделано (для справки)

- GNSS (UBX-NAV-PVT), NMEA (RMC), NTP, PTP+PHC (ptp4l + чтение PHC), PPS (linked_device + /dev/pps на Linux).
- Запуск ptp4l внутри tc-sync (`start_ptp4l` в конфиге).
- Servo: PID, PI, LinReg; clockadj: Step, Slew, SetFrequency, GetFrequency, GranularityNs (Linux).
- Выбор источника (primary → secondary), конфиг в стиле Timebeat, timebeat (Beat на libbeat).
- Сборка под Linux (amd64/arm64), скрипты `ensure-go.ps1`, `build-linux.ps1`.

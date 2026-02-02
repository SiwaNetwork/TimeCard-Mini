# tc-sync + timebeat — аналог Timebeat

## Полный ли это аналог?

**Да, по функциональности синхронизации времени — да.** tc-sync + timebeat покрывают основные возможности Timebeat/shiwatime:

| Область | Покрыто |
|---------|---------|
| **Источники** | GNSS (UBX), NMEA (RMC), NTP, PTP (ptp4l+PHC), PPS (linked_device, /dev/pps) |
| **Выбор источника** | primary → secondary |
| **Servo** | PID, PI, LinReg |
| **Коррекция часов** | step / slew, SetFrequency (adjtimex, clock_settime) |
| **Конфиг** | Формат shiwatime_ru.yml (clock_sync, primary_clocks, secondary_clocks) |
| **Time pulse** | CFG-TP5, configure |
| **Elastic Beat** | timebeat на libbeat v7 |

---

## Чего нет (и нужно ли)

| Возможность | Статус | Нужно? |
|-------------|--------|--------|
| **Elastic Stack UI, Kibana** | ❌ Timebeat публикует события в ES, но метрик синхронизации пока нет | Опционально: при необходимости мониторинга в Kibana |
| **HTTP API** | ❌ Коммерческий Timebeat может иметь REST API | По необходимости |
| **Полные 62 поля UBX TP5** | ⚠️ Используем первые 32 байта (основные поля) | Только если нужны доп. поля u-blox |
| **send1PPSOnTimepulsePin** | ❌ | При интеграции 1-PPS по пину |
| **detectUbloxUnit** | ❌ | При автоопределении типа приёмника |
| **phc2sys** | ❌ Не запускаем phc2sys (PHC→sys) — наш servo сам правит системные часы | Обычно не нужно |
| **PID D-массив, PI 51712** | ⚠️ Упрощённый PID/PI; коэффициенты через конфиг | Для точной копии shiwatime — см. FOUND_COEFFICIENTS |

---

## Итог

**Для синхронизации времени на Linux (GNSS/NTP/PTP/PPS) — tc-sync/timebeat достаточно.** Использование такое же: конфиг в стиле Timebeat, `tc-sync -run` или `./timebeat` как Beat, коррекция adjtimex/clock_settime.

**Дальнейшие шаги (если нужны):**

1. **Метрики в Elasticsearch** — публиковать offset, активный источник, step/slew в события Beat для Kibana.
2. **Расширение UBX TP5** — при необходимости дополнительных полей.
3. **Точные коэффициенты servo** — при желании максимально приблизиться к shiwatime (51712, D-массив).

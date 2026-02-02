package statistics

import (
	"fmt"
	"strconv"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// Автоматически извлечено из timebeat-2.2.20
// EMA.AddValue по дизассемблеру (__EMA_.AddValue.txt): 0x8=count, окно 20 (0x14); при count>=20 — формула newVal=old+alpha*(value-old), alpha из .rodata 54decb8; иначе буфер 0x10+index*8, inc 0x8.

const emaWindowSize = 20 // 0x14 по дизассемблеру

// EMAAlpha — значение из .rodata 54decb8 (для logDebug и AddValue).
const EMAAlpha = 0.1 // приближение; в бинарнике double по 54decb8

// EMA — экспоненциальное скользящее среднее (окно 20, alpha из 54decb8).
// По дизассемблеру logDebug/EnableDebug: 0xb0=debugEnabled, 0xb8=*Logger.
type EMA struct {
	value        int64
	count        int64
	buf          [emaWindowSize]int64
	debugEnabled bool       // 0xb0 по дизассемблеру EnableDebug
	logger       *logging.Logger // 0xb8 по дизассемблеру logDebug
}

func AddValue() {
	// TODO: реконструировать (пакетная заглушка)
}

func NewEMA() *EMA {
	return &EMA{}
}

// GetValue по дизассемблеру (__EMA_.GetValue): mov (%rax),%rax — возврат первого поля (value).
func (e *EMA) GetValue() int64 {
	return e.value
}

// Reset по дизассемблеру (__EMA_.reset): movq $0, 0x8(%rax) — обнуление поля count (0x8).
func (e *EMA) Reset() {
	e.count = 0
}

// getSMA по дизассемблеру (__EMA_.getSMA): копия buf (0x10(rax)), цикл 0..0x14 sum+=buf[i];
// sum*0xcccccccccccccccd, (sum+high)>>4 — среднее за 20 элементов. Возврат sum/20.
func (e *EMA) getSMA() int64 {
	var sum int64
	for i := 0; i < emaWindowSize; i++ {
		sum += e.buf[i]
	}
	return sum / emaWindowSize
}

// EnableDebug по дизассемблеру (__EMA_.EnableDebug): NewLogger(name); 0xb8(ema)=logger; 0xb0(ema)=1.
func (e *EMA) EnableDebug(loggerName string) {
	e.logger = logging.NewLogger(loggerName)
	e.debugEnabled = true
}

// logDebug по дизассемблеру (__EMA_.logDebug): Logger.Debug(strconv(count)), Debug(strconv(value)),
// Debug(prefix+value), (value-diff)*alpha, Debug, fmt.Sprintf float64, Debug, alpha, Debug.
func (e *EMA) logDebug() {
	if e.logger == nil {
		return
	}
	e.logger.Debug(strconv.FormatInt(e.count, 10), 10)
	e.logger.Debug(strconv.FormatInt(e.value, 10), 10)
	e.logger.Debug("ema value "+strconv.FormatInt(e.value, 10), 9)
	e.logger.Debug("ema value "+strconv.FormatInt(e.value, 10), 9)
	diff := e.value - e.getSMA()
	scaled := int64(float64(diff) * EMAAlpha)
	e.logger.Debug(strconv.FormatInt(scaled, 10), 7)
	e.logger.Debug(fmt.Sprintf("%.2f", float64(diff)*EMAAlpha), 2)
	e.logger.Debug(fmt.Sprintf("%.2f", EMAAlpha), 2)
}

// AddValue добавляет значение; при count < 20 — в буфер; при count >= 20 — EMA: new = old + alpha*(value - old), alpha из 54decb8.
func (e *EMA) AddValue(value int64) *EMA {
	// TODO: точная формула по дизассемблеру (alpha 54decb8)
	if e.count < emaWindowSize {
		e.buf[e.count] = value
		e.count++
		return e
	}
	// alpha * (value - old) + old
	e.value = e.value + (value-e.value)/10 // упрощённо; в бинарнике alpha из .rodata
	return e
}

func inittask() {
	// TODO: реконструировать
}


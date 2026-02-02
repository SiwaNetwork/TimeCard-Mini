package algos

import (
	"math"
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
)

// SteeringNames — названия алгоритмов
var SteeringNames = []string{"pid", "pi", "linreg"}

// SteeringCoefficients по дизассемблеру GetCoefficients (24RbfKOo): +0 Kp, +8 Ki, +0x10 Kd, +0x18 = 8, +0x20 = 0x1efe920.
type SteeringCoefficients struct {
	Kp      float64
	Ki      float64
	Kd      float64
	Field18 int64 // 0x18 = 8 по дизассемблеру
	Field20 int64 // 0x20 = 0x1efe920 (33_554_720)
}

// CoefficientStore по дизассемблеру newCoefficientStore/GetCoefficientStore: +0 logger, +8 *SteeringCoefficients (type 1), +0x10 *SteeringCoefficients (type 0).
type CoefficientStore struct {
	logger      *logging.Logger  // 0x00
	coeffType1  *SteeringCoefficients // 0x08 — для algoType 1
	coeffType0  *SteeringCoefficients // 0x10 — для algoType 0
	mu          sync.RWMutex
	coefficients map[string]Coefficients
}

// Coefficients — коэффициенты для алгоритма
type Coefficients struct {
	Kp      float64
	Ki      float64
	Kd      float64
	DCoeffs [MaxDCoeffs]float64
}

var instanceCoefficientStore *CoefficientStore
var once sync.Once

// GetCoefficientStore возвращает singleton
func GetCoefficientStore() *CoefficientStore {
	once.Do(func() {
		instanceCoefficientStore = newCoefficientStore()
	})
	return instanceCoefficientStore
}

// newCoefficientStore по дизассемблеру (newCoefficientStore@@Base 0x457c2a0): NewLogger("coefficient-store"), newobject(CoefficientStore),
// 0(store)=logger; GetCoefficients(store, 1, kp, ki, kd, scale) → 0x8(store); Logger.Info(4 float64); GetCoefficients(store, 0, ...) → 0x10(store).
func newCoefficientStore() *CoefficientStore {
	cs := &CoefficientStore{
		logger:      logging.NewLogger("coefficient-store"),
		coefficients: make(map[string]Coefficients),
	}
	// Дефолты из appConfig (0x7e44c80..0x7e44c98 и 0x7e44ca0..) — используем DefaultAlgoCoefficients
	kp, ki, kd, scale := DefaultAlgoCoefficients.Kp, DefaultAlgoCoefficients.Ki, DefaultAlgoCoefficients.Kd, 1.0
	cs.coeffType1 = cs.getCoefficientsInt(1, kp, ki, kd, scale)
	cs.logger.Info("", 0x2a, kp, ki, kd)
	cs.coeffType0 = cs.getCoefficientsInt(0, kp, ki, kd, scale)
	cs.coefficients["default"] = Coefficients{
		Kp:      DefaultAlgoCoefficients.Kp,
		Ki:      DefaultAlgoCoefficients.Ki,
		Kd:      DefaultAlgoCoefficients.Kd,
		DCoeffs: dComponentLookup,
	}
	return cs
}

// Константы GetCoefficients по дизассемблеру (rodata): type 0 — 54de498 Kp, 54de218 Ki, 54de660 Kd; type 1 — те же;
// type 2 — 54de410 Kp, 54de368 Ki, 54de498 Kd; type 3 — 54de2e0 Kp, 54de260 Ki, 54de498 Kd. Field18=8, Field20=0x1efe920.
var getCoefficientsDefaults = [4]struct{ Kp, Ki, Kd float64 }{
	{0.5, DefaultAlgoCoefficients.Ki, 0.02},             // type 0: 54de498, 54de218, 54de660
	{0.5, DefaultAlgoCoefficients.Ki, 0.02},             // type 1
	{0.5, 0.5946035575013605, 0.5},                      // type 2: 54de410, 54de368, 54de498
	{0.7071067811865475, 0.5946035575013605, 0.5},        // type 3: 54de2e0, 54de260, 54de498
}

// getCoefficientsInt по дизассемблеру GetCoefficients: (receiver, algoType int, kp, ki, kd, scale float64).
// При scale==0 — Logger.Critical. Switch algoType 0/1/2/3: new(SteeringCoefficients), константы из getCoefficientsDefaults (rodata); если kp/ki/kd/scale != 1.0 — перезапись и Logger.Info.
func (cs *CoefficientStore) getCoefficientsInt(algoType int, kp, ki, kd, scale float64) *SteeringCoefficients {
	if scale == 0 {
		cs.logger.Critical("", 0x46)
		return nil
	}
	if algoType < 0 || algoType > 3 {
		return nil
	}
	d := getCoefficientsDefaults[algoType]
	c := &SteeringCoefficients{Kp: d.Kp, Ki: d.Ki, Kd: d.Kd, Field18: 8, Field20: 0x1efe920}
	if kp != 1.0 {
		c.Kp = kp
		cs.logger.Info("", 0x26, kp)
	}
	if ki != 1.0 {
		c.Ki = ki
		cs.logger.Info("", 0x26, ki)
	}
	if kd != 1.0 {
		c.Kd = kd
		cs.logger.Info("", 0x26, kd)
	}
	if scale != 1.0 {
		c.Field18 = int64(scale)
		cs.logger.Info("", 0x24, scale)
	}
	return c
}

// GetCoefficientsInt по дизассемблеру — вызов getCoefficientsInt (для внешнего API по типу int).
func (cs *CoefficientStore) GetCoefficientsInt(algoType int, kp, ki, kd, scale float64) *SteeringCoefficients {
	return cs.getCoefficientsInt(algoType, kp, ki, kd, scale)
}

// GetCoefficientsForTypeInt по дизассемблеру (GetCoefficientsForType@@Base): type 0 → return 0x10(store); type 1 → return 0x8(store);
// type 2/3 → GetCoefficients(store, type, ...), затем масштабирование Ki/Kp/Kd через math.Pow(10, scale).
func (cs *CoefficientStore) GetCoefficientsForTypeInt(algoType int, scale byte) *SteeringCoefficients {
	switch algoType {
	case 0:
		return cs.coeffType0
	case 1:
		return cs.coeffType1
	case 2, 3:
		kp, ki, kd := 1.0, 1.0, 1.0
		coeff := cs.getCoefficientsInt(algoType, kp, ki, kd, 1.0)
		if coeff == nil {
			return nil
		}
		s := float64(scale)
		mult := math.Pow(10, s)
		coeff.Kp *= mult
		coeff.Ki *= mult
		coeff.Kd *= mult
		return coeff
	default:
		return nil
	}
}

// GetCoefficients возвращает коэффициенты по типу
func (cs *CoefficientStore) GetCoefficients(algoType string) Coefficients {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if c, ok := cs.coefficients[algoType]; ok {
		return c
	}
	return cs.coefficients["default"]
}

// GetCoefficientsForType алиас
func (cs *CoefficientStore) GetCoefficientsForType(algoType string) Coefficients {
	return cs.GetCoefficients(algoType)
}

// ChangeSteeringCoefficients по дизассемблеру (ChangeSteeringCoefficients@@Base 0x457cae0): (receiver, algoType int, newCoeffs *SteeringCoefficients).
// algoType==0: Logger.Info(0x35), target=store+0x10 (coeffType0); algoType==1: Logger.Info(0x33), target=store+0x8 (coeffType1); иначе return.
// Копирование из newCoeffs в target: если Kp/Ki/Kd != 0 — записать; если Field18/Field20 != 0 — записать.
func (cs *CoefficientStore) ChangeSteeringCoefficients(algoType string, c Coefficients) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.coefficients[algoType] = c
}

// ChangeSteeringCoefficientsInt по дизассемблеру: algoType 0 или 1; Logger.Info; копирование c в coeffType0/coeffType1 (только ненулевые поля).
func (cs *CoefficientStore) ChangeSteeringCoefficientsInt(algoType int, c *SteeringCoefficients) {
	if c == nil {
		return
	}
	var target *SteeringCoefficients
	switch algoType {
	case 0:
		cs.logger.Info("", 0x35)
		target = cs.coeffType0
	case 1:
		cs.logger.Info("", 0x33)
		target = cs.coeffType1
	default:
		return
	}
	if target == nil {
		return
	}
	if c.Kp != 0 {
		target.Kp = c.Kp
	}
	if c.Ki != 0 {
		target.Ki = c.Ki
	}
	if c.Kd != 0 {
		target.Kd = c.Kd
	}
	if c.Field18 != 0 {
		target.Field18 = c.Field18
	}
	if c.Field20 != 0 {
		target.Field20 = c.Field20
	}
}

// GetLinuxCoefficients возвращает Linux-специфичные коэффициенты
func GetLinuxCoefficients() Coefficients {
	return GetCoefficientStore().GetCoefficients("linux")
}

// IntervalToPPS конвертирует интервал в PPS
func IntervalToPPS(interval float64) float64 {
	if interval <= 0 {
		return 1
	}
	return 1 / interval
}

// CompareInt64 сравнивает два int64
func CompareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Errors
var (
	ErrMismatchedSamples = &algoError{"mismatched samples"}
	ErrSampleSize        = &algoError{"insufficient sample size"}
	ErrZeroVariance      = &algoError{"zero variance"}
)

type algoError struct {
	msg string
}

func (e *algoError) Error() string { return e.msg }

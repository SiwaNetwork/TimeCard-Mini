package servo

import (
	"testing"
	"time"
)

func TestPID_Update(t *testing.T) {
	p := NewPID(0.1, 0.01, 0.001)
	dt := time.Second

	// Положительный offset — положительная коррекция частоты
	out := p.Update(1e6, dt) // 1 ms offset
	if out <= 0 {
		t.Errorf("PID Update(1e6): ожидали положительную коррекцию, получили %v", out)
	}
	if out > 100e-6 {
		t.Errorf("PID Update: коррекция превышает MaxAdjustment %v", out)
	}

	// Отрицательный offset — отрицательная коррекция
	out2 := p.Update(-1e6, dt)
	if out2 >= 0 {
		t.Errorf("PID Update(-1e6): ожидали отрицательную коррекцию, получили %v", out2)
	}

	// Нулевой offset после Reset
	p.Reset()
	out3 := p.Update(0, dt)
	if out3 != 0 {
		t.Errorf("PID Update(0) после Reset: ожидали 0, получили %v", out3)
	}
}

func TestPID_Reset(t *testing.T) {
	p := NewPID(0.1, 0.01, 0.001)
	p.Update(1e9, time.Second)
	p.Reset()
	out := p.Update(0, time.Second)
	if out != 0 {
		t.Errorf("после Reset интеграл должен обнулиться, получили %v", out)
	}
}

func TestPI_Update(t *testing.T) {
	pi := NewPI(0.1, 0.01)
	dt := time.Second
	out := pi.Update(500e6, dt) // 500 ms
	if out <= 0 {
		t.Errorf("PI Update(500e6): ожидали положительную коррекцию, получили %v", out)
	}
	pi.Reset()
	out2 := pi.Update(0, dt)
	if out2 != 0 {
		t.Errorf("PI Update(0) после Reset: ожидали 0, получили %v", out2)
	}
}

func TestPIShiwatime_Update(t *testing.T) {
	pi := NewPIShiwatime(0.1)
	dt := time.Second
	out := pi.Update(500e6, dt)
	if out <= 0 {
		t.Errorf("PIShiwatime Update(500e6): ожидали положительную коррекцию, получили %v", out)
	}
	pi.Reset()
	out2 := pi.Update(100e6, dt)
	if out2 <= 0 {
		t.Errorf("PIShiwatime после Reset и Update(100e6): ожидали положительную коррекцию, получили %v", out2)
	}
}

func TestPID_DCoeffs(t *testing.T) {
	p := NewPID(0.1, 0.01, 0)
	p.DCoeffs = [3]float64{0.001, 0.002, 0.003}
	dt := time.Second
	out := p.Update(1e8, dt)
	if out == 0 {
		t.Error("PID с DCoeffs: ожидали ненулевую коррекцию")
	}
}

func TestLinReg_Update(t *testing.T) {
	lr := NewLinReg()
	dt := time.Second

	// Несколько обновлений с постоянным offset — LinReg накапливает выборку
	for i := 0; i < 10; i++ {
		_ = lr.Update(100e6, dt) // 100 ms
	}
	out := lr.Update(100e6, dt)
	// LinReg возвращает коррекцию частоты (может быть 0 пока окно не заполнено)
	if out != 0 && (out > 1e-3 || out < -1e-3) {
		t.Errorf("LinReg: неожиданно большая коррекция %v", out)
	}
	lr.Reset()
	out2 := lr.Update(0, dt)
	if out2 != 0 {
		t.Errorf("LinReg после Reset: ожидали 0, получили %v", out2)
	}
}

package input

import (
	"math"
	"testing"
)

func length(x, y float32) float32 { return float32(math.Hypot(float64(x), float64(y))) }

func TestJoystickMoveResponseDeadZone(t *testing.T) {
	if x, y := JoystickMoveResponse(0.1, 0.05); x != 0 || y != 0 {
		t.Fatalf("dentro da zona morta devia ser zero, veio (%v, %v)", x, y)
	}
}

// A regra do bug: com o polegar a partir de JoystickFullSpeedAt o celular
// anda na MESMA velocidade do teclado (comprimento 1), em qualquer direcao.
func TestJoystickMoveResponseFullSpeedMatchesKeyboard(t *testing.T) {
	for _, raw := range [][2]float32{{0.55, 0}, {0, -0.6}, {0.5, 0.5}, {-1, 0}, {0.3, -0.9}} {
		x, y := JoystickMoveResponse(raw[0], raw[1])
		if l := length(x, y); math.Abs(float64(l-1)) > 1e-4 {
			t.Errorf("raw %v: comprimento %v, esperado 1", raw, l)
		}
	}
}

func TestJoystickMoveResponseKeepsDirectionAndRamps(t *testing.T) {
	x, y := JoystickMoveResponse(0.21, 0.28) // comprimento 0,35: meio da rampa
	if l := length(x, y); math.Abs(float64(l-0.5)) > 1e-4 {
		t.Errorf("comprimento %v, esperado 0,5", l)
	}
	if math.Abs(float64(x/y-0.75)) > 1e-4 {
		t.Errorf("direcao mudou: (%v, %v)", x, y)
	}
}

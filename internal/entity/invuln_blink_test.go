package entity

import "testing"

// O piscar so existe dentro da janela, alterna, e acelera no fim.
func TestInvulnBlink(t *testing.T) {
	if invulnBlinkGrayAt(0, 1.3) {
		t.Error("sem janela o corpo nao pode piscar")
	}
	changes := func(left float32, dur float64) int {
		n, prev := 0, invulnBlinkGrayAt(left, 0)
		for now := 0.0; now < dur; now += 0.01 {
			if g := invulnBlinkGrayAt(left, now); g != prev {
				n, prev = n+1, g
			}
		}
		return n
	}
	calm, hurry := changes(2, 1), changes(0.3, 1)
	if calm < 6 {
		t.Errorf("no meio da janela o corpo trocou de fase %d vezes em 1 s", calm)
	}
	if hurry <= calm {
		t.Errorf("o fim da janela devia piscar mais rapido (%d trocas contra %d)", hurry, calm)
	}
}

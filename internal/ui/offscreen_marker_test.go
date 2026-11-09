package ui

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestEdgeMarkerPoint(t *testing.T) {
	const sw, sh, inset = 1920, 1080, 40

	if _, _, on := EdgeMarkerPoint(rl.NewVector2(500, 500), sw, sh, inset); !on {
		t.Error("alvo dentro da tela recebeu seta")
	}

	// Direto a direita: a seta fica na borda direita, na altura do centro.
	at, dir, on := EdgeMarkerPoint(rl.NewVector2(5000, sh/2), sw, sh, inset)
	if on || at.X != sw-inset || at.Y != sh/2 || dir.X != 1 {
		t.Errorf("a direita: at %v dir %v on %v", at, dir, on)
	}

	// Em diagonal, a seta fica NA RETA do alvo (nao no canto) e dentro da
	// tela recuada.
	target := rl.NewVector2(sw/2+3000, sh/2-600)
	at, dir, on = EdgeMarkerPoint(target, sw, sh, inset)
	if on {
		t.Fatal("alvo fora da tela tratado como visivel")
	}
	if at.X < inset-0.01 || at.X > sw-inset+0.01 || at.Y < inset-0.01 || at.Y > sh-inset+0.01 {
		t.Errorf("seta fora da tela recuada: %v", at)
	}
	toTarget := rl.Vector2Normalize(rl.Vector2Subtract(target, rl.NewVector2(sw/2, sh/2)))
	if rl.Vector2Distance(dir, toTarget) > 0.001 {
		t.Errorf("seta aponta %v, alvo esta em %v", dir, toTarget)
	}
	fromCenter := rl.Vector2Normalize(rl.Vector2Subtract(at, rl.NewVector2(sw/2, sh/2)))
	if rl.Vector2Distance(fromCenter, toTarget) > 0.001 {
		t.Errorf("seta %v fora da reta do centro ate o alvo", at)
	}
}

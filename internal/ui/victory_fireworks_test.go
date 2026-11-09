package ui

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Nenhum foguete pode descer sem estourar, em nenhuma tela. O caso mais
// dificil e o estouro mais alto que launchFirework sorteia (16% da altura) —
// e foi numa TV 4K que o Gui viu o bug (plans/plan_playtest_27_09.md §3).
func TestFireworksAlwaysBurst(t *testing.T) {
	for _, h := range []float32{720, 1080, 1440, 1600, 2160, 4320} {
		scale := fireworkScale(h)
		fireworks.rockets = []fireworkRocket{{
			pos:    rl.NewVector2(100, h+10),
			vel:    rl.NewVector2(0, -fireworkRocketSpeed*scale),
			burstY: h * 0.16,
		}}
		fireworks.sparks = nil
		burstAt := float32(-1)
		for step := 0; step < 600 && burstAt < 0; step++ {
			updateFireworks(1.0/60, scale)
			if len(fireworks.rockets) == 0 {
				burstAt = float32(step) / 60
			}
		}
		if burstAt < 0 {
			t.Fatalf("tela de %.0f px: o foguete nao estourou em 10 s", h)
		}
		if len(fireworks.sparks) == 0 {
			t.Fatalf("tela de %.0f px: o foguete sumiu sem faiscas", h)
		}
		// Estourou pela LINHA, e nao pela rede de seguranca do topo: com a
		// escala, a subida sempre alcanca os 16% mais altos.
		if top := fireworks.sparks[0].pos.Y; top > h*0.16+fireworkRocketSpeed*scale/60 {
			t.Errorf("tela de %.0f px: estourou em y=%.0f, abaixo da linha %.0f", h, top, h*0.16)
		}
	}
}

// Mesmo sem a escala (um foguete lento demais para a tela), o topo da subida
// estoura: a rede de seguranca de updateFireworks.
func TestFireworkBurstsAtApexWhenTooSlow(t *testing.T) {
	fireworks.rockets = []fireworkRocket{{
		pos:    rl.NewVector2(100, 2170),
		vel:    rl.NewVector2(0, -fireworkRocketSpeed), // escala 1 numa tela 4K
		burstY: 2160 * 0.16,
	}}
	fireworks.sparks = nil
	for step := 0; step < 600 && len(fireworks.rockets) > 0; step++ {
		updateFireworks(1.0/60, 1)
		for _, r := range fireworks.rockets {
			if r.vel.Y > 0 {
				t.Fatalf("o foguete comecou a descer em y=%.0f sem estourar", r.pos.Y)
			}
		}
	}
	if len(fireworks.rockets) != 0 || len(fireworks.sparks) == 0 {
		t.Fatal("o foguete nao estourou no topo da subida")
	}
}

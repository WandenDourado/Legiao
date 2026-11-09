package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// O piscar da imunidade (playtest de 27/09/2026, plans/plan_playtest_27_09.md
// §4): enquanto o host segura uma janela de imunidade para um jogador, o
// corpo alterna entre colorido e o cinza do shader de morte (death_tint.go).
// Reaproveitar o shader e de proposito — nenhum asset novo, e o cinza ja
// significa "fora de combate" neste jogo; piscando, le como "voltando".
//
// Vale para TODA janela concedida, nao so a do renascer: o ultimo suspiro e o
// reerguimento da vitoria tambem passam a piscar, e com isso nenhuma
// imunidade do jogo e mais invisivel.

const (
	// invulnBlinkPeriod e meio ciclo: 0,125 s colorido, 0,125 s cinza.
	invulnBlinkPeriod = 0.125
	// invulnBlinkHurry e o meio ciclo nos ultimos invulnBlinkEnding segundos:
	// o piscar acelera para avisar que a protecao esta acabando.
	invulnBlinkHurry  = 0.06
	invulnBlinkEnding = 0.6
)

// InvulnBlinkGray diz se, NESTE quadro, um corpo com `left` segundos de
// imunidade deve ser desenhado em cinza.
//
// A fase vem do relogio do jogo e nao de `left`: `left` chega do host a
// ~20 Hz (um degrau a cada 0,05 s), e piscar por ele serrilharia o ritmo —
// com o meio ciclo apressado de 0,06 s o degrau quase o engoliria. `left` so
// decide SE pisca e QUAO rapido.
func InvulnBlinkGray(left float32) bool {
	return invulnBlinkGrayAt(left, rl.GetTime())
}

func invulnBlinkGrayAt(left float32, now float64) bool {
	if left <= 0 {
		return false
	}
	half := invulnBlinkPeriod
	if left < invulnBlinkEnding {
		half = invulnBlinkHurry
	}
	return int64(now/half)%2 == 1
}

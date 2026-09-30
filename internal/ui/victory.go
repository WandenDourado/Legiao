package ui

// A cena que fecha a campanha. A Senhora das Trevas (mapa 7) e o unico chefe
// do jogo, entao "tem chefe" ja significa "e o fim da campanha" — nada aqui
// precisa saber o nome do mapa.
//
// Os fogos e o banner de missao cumprida (`missionAccomplished`, chamado por
// `drawMapCleared` em wave_hud.go) so aparecem quando BossState.VictoryReady
// e verdadeiro — e nao no quadro em que a vida do chefe zera. Entre uma coisa
// e outra o host ainda tem de: parar a horda (automatico, EndsWithBoss),
// reerguer quem caiu na luta e deixar a cena final de dialogo terminar de
// falar (game.DialogueDirector, gatilho TriggerWavesCleared). So depois disso
// o host chama MarkBossVictoryReady e VictoryReady vira verdadeiro para o
// host e para todo cliente (ele viaja no mesmo EnemyUpdatePayload da vida do
// chefe — ver internal/network/boss_state.go e protocol.go).

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	missionTitleSize = 64
	missionSubSize   = 24
)

var missionGold = rl.NewColor(255, 214, 120, 255)

// DrawBossVictoryFX toca os fogos da vitoria. Chamado todo quadro pelo
// renderer, ao lado de DrawBossBar; nao desenha nada em mapa sem chefe ou
// enquanto ele ainda esta de pe.
func DrawBossVictoryFX(screenWidth, screenHeight float32) {
	boss := network.GetBossState()
	DrawBossFireworks(boss.Present && boss.VictoryReady, screenWidth, screenHeight)
}

// missionAccomplished e o banner radiante de fim de campanha, desenhado no
// lugar do "Mapa limpo" comum quando o mapa que acabou de limpar tem chefe.
func missionAccomplished(screenWidth, screenHeight float32) {
	pulse := float32(0.85 + 0.15*math.Sin(float64(rl.GetTime())*2.4))

	// Pixels da tela nativa (Px, text.go): em 540p o banner nao dobra.
	titleSize, subSize := int32(Px(missionTitleSize)), int32(Px(missionSubSize))
	title := "MISSAO CONCLUIDA"
	width := rl.MeasureText(title, titleSize)
	x := (int32(screenWidth) - width) / 2
	y := int32(screenHeight/2 - Px(130))

	// Halo por copias deslocadas: raylib desktop nao tem blur de shader
	// disponivel neste passo de HUD, entao o brilho vem de repeticao, nao de
	// pos-processamento.
	for step := int32(10); step >= 2; step -= 2 {
		r := int32(Px(float32(step)))
		glow := rl.Fade(missionGold, (0.04+0.05*pulse)*float32(12-step)/10)
		rl.DrawText(title, x-r, y, titleSize, glow)
		rl.DrawText(title, x+r, y, titleSize, glow)
		rl.DrawText(title, x, y-r, titleSize, glow)
		rl.DrawText(title, x, y+r, titleSize, glow)
	}
	shadow := int32(max(Px(3), 1))
	rl.DrawText(title, x+shadow, y+shadow, titleSize, rl.Fade(rl.Black, 0.7))
	rl.DrawText(title, x, y, titleSize, rl.Fade(missionGold, pulse))

	sub := "a Imperatriz das Trevas caiu. A Legiao venceu."
	subWidth := rl.MeasureText(sub, subSize)
	subY := y + titleSize + int32(Px(14))
	subX := (int32(screenWidth) - subWidth) / 2
	subShadow := int32(max(Px(2), 1))
	rl.DrawText(sub, subX+subShadow, subY+subShadow, subSize, rl.Fade(rl.Black, 0.7))
	rl.DrawText(sub, subX, subY, subSize, rl.RayWhite)
}

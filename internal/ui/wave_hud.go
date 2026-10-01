package ui

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Wave HUD layout. The counter sits top-right so it does not collide with the
// player count and server address already drawn top-left and top-centre.
// Os tamanhos sao BASES: o que se ve na escala 1.0. TextSize (text.go) aplica
// a escala do jogador e o piso. Subidos em 20/09/2026 junto com o resto do
// texto — 22/18 era o contador da horda que os testadores nao liam.
const (
	waveCounterFontBase   = 28
	waveRemainingFontBase = 22
	waveMarginX           = 16
	waveMarginY           = 12
	waveAnnounceFontBase  = 60
	waveSubAnnounceBase   = 26
	waveClearedFontBase   = 46
)

// DrawWaveHUD renders the horde counter and, during a break, the centred
// announcement. Reads the state the network layer publishes, so it works
// identically on host and client.
func DrawWaveHUD(screenWidth, screenHeight float32) {
	state := network.GetWaveState()
	if state.Total == 0 {
		return // No wave run on this map yet.
	}

	drawWaveCounter(state, screenWidth)

	switch network.WavePhase(state.Phase) {
	case network.WavePhaseBreak:
		drawWaveAnnouncement(state, screenWidth, screenHeight)
	case network.WavePhaseCleared:
		drawMapCleared(screenWidth, screenHeight)
	}
}

// waveCounterLeft devolve a borda ESQUERDA do contador "Horda N/M" (e da
// linha "restam N" embaixo dele, a que for mais larga) e o tamanho da fonte
// do titulo. ok e falso quando o contador nao esta na tela — sem horda no
// mapa, ou fase ja limpa. A engrenagem de Configuracoes
// (settings_access_button.go) se encosta aqui, e as duas contas tem de ser a
// mesma que drawWaveCounter usa para desenhar.
func waveCounterLeft(state network.WaveState, screenWidth float32) (left float32, titleSize int32, ok bool) {
	if state.Total == 0 || network.WavePhase(state.Phase) == network.WavePhaseCleared {
		return 0, 0, false
	}
	titleSize = TextSize(waveCounterFontBase)
	width := rl.MeasureText(fmt.Sprintf("Horda %d/%d", state.Index, state.Total), titleSize)
	if network.WavePhase(state.Phase) == network.WavePhaseFighting {
		sub := rl.MeasureText(fmt.Sprintf("restam %d", state.Remaining), TextSize(waveRemainingFontBase))
		width = max(width, sub)
	}
	return screenWidth - float32(width) - Px(waveMarginX), titleSize, true
}

func drawWaveCounter(state network.WaveState, screenWidth float32) {
	if network.WavePhase(state.Phase) == network.WavePhaseCleared {
		return
	}

	titleSize := TextSize(waveCounterFontBase)
	title := fmt.Sprintf("Horda %d/%d", state.Index, state.Total)
	titleWidth := rl.MeasureText(title, titleSize)
	x := int32(screenWidth-Px(waveMarginX)) - titleWidth
	rl.DrawText(title, x, int32(Px(waveMarginY)), titleSize, rl.White)

	// During the break the count would read "restam 0", which is noise; the
	// announcement is already saying what happens next.
	if network.WavePhase(state.Phase) != network.WavePhaseFighting {
		return
	}
	subSize := TextSize(waveRemainingFontBase)
	sub := fmt.Sprintf("restam %d", state.Remaining)
	subWidth := rl.MeasureText(sub, subSize)
	subX := int32(screenWidth-Px(waveMarginX)) - subWidth
	rl.DrawText(sub, subX, int32(Px(waveMarginY+4))+titleSize, subSize, rl.Gray)
}

func drawWaveAnnouncement(state network.WaveState, screenWidth, screenHeight float32) {
	text := state.Announcement
	if text == "" {
		text = state.Name
	}
	if text == "" {
		return
	}

	// Fade in over the first fifth of the break and out over the last fifth so
	// the message does not pop on and off.
	alpha := float32(1)
	elapsed := network.WaveBreakDuration - state.BreakTime
	if fade := network.WaveBreakDuration * 0.2; fade > 0 {
		if elapsed < fade {
			alpha = elapsed / fade
		} else if state.BreakTime < fade {
			alpha = state.BreakTime / fade
		}
	}
	alpha = clamp01(alpha)

	announceSize := TextSize(waveAnnounceFontBase)
	subSize := TextSize(waveSubAnnounceBase)
	width := rl.MeasureText(text, announceSize)
	x := int32((int32(screenWidth) - width) / 2)
	y := int32(screenHeight/2 - Px(90))

	// Drop shadow keeps the text legible over bright grass.
	sh3 := int32(max(Px(3), 1))
	rl.DrawText(text, x+sh3, y+sh3, announceSize, rl.Fade(rl.Black, 0.6*alpha))
	rl.DrawText(text, x, y, announceSize, rl.Fade(rl.RayWhite, alpha))

	countdown := fmt.Sprintf("preparar - %.0f", ceil32(state.BreakTime))
	cWidth := rl.MeasureText(countdown, subSize)
	cX := int32((int32(screenWidth) - cWidth) / 2)
	rl.DrawText(countdown, cX, y+announceSize+int32(Px(10)), subSize, rl.Fade(rl.LightGray, alpha))
}

func drawMapCleared(screenWidth, screenHeight float32) {
	// Um mapa com chefe fecha a campanha, nao so a fase: o aviso generico de
	// "portal aberto" cede lugar ao banner de vitoria (victory.go).
	//
	// Mas so DEPOIS que VictoryReady vira verdadeiro — a horda ja parou
	// (Cleared e o que trouxe a execucao ate aqui), mas o reerguimento dos
	// caidos e a cena final de dialogo ainda podem estar em andamento, e a
	// caixa de dialogo ja ocupa a tela nesse meio-tempo. Ate la, este
	// `drawMapCleared` nao desenha nada: nem o banner de vitoria, nem o
	// generico "Mapa limpo" que faria sentido para um mapa sem chefe.
	if boss := network.GetBossState(); boss.Present {
		if boss.VictoryReady {
			missionAccomplished(screenWidth, screenHeight)
		}
		return
	}

	clearedSize := TextSize(waveClearedFontBase)
	subSize := TextSize(waveSubAnnounceBase)
	text := "Mapa limpo"
	width := rl.MeasureText(text, clearedSize)
	x := int32((int32(screenWidth) - width) / 2)
	y := int32(screenHeight/2 - Px(120))
	sh2 := int32(max(Px(2), 1))
	rl.DrawText(text, x+sh2, y+sh2, clearedSize, rl.Fade(rl.Black, 0.6))
	rl.DrawText(text, x, y, clearedSize, rl.Gold)

	// Clearing the run is what opens the portal, and the portal is at the far
	// end of the map: without naming where it is, the players finish the fight
	// and have nothing telling them where to go.
	sub := "o portal se abriu no fim da estrada de terra"
	sWidth := rl.MeasureText(sub, subSize)
	sX := int32((int32(screenWidth) - sWidth) / 2)
	rl.DrawText(sub, sX, y+clearedSize+int32(Px(8)), subSize, rl.RayWhite)
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ceil32(v float32) float32 {
	i := float32(int(v))
	if v > i {
		return i + 1
	}
	return i
}

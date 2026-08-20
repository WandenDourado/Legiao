package ui

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Wave HUD layout. The counter sits top-right so it does not collide with the
// player count and server address already drawn top-left and top-centre.
const (
	waveCounterFontSize    = 22
	waveRemainingFontSize  = 18
	waveMarginX            = 16
	waveMarginY            = 12
	waveAnnounceFontSize   = 54
	waveSubAnnounceSize    = 22
	waveClearedFontSize    = 40
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

func drawWaveCounter(state network.WaveState, screenWidth float32) {
	if network.WavePhase(state.Phase) == network.WavePhaseCleared {
		return
	}

	title := fmt.Sprintf("Horda %d/%d", state.Index, state.Total)
	titleWidth := rl.MeasureText(title, waveCounterFontSize)
	x := int32(screenWidth) - titleWidth - waveMarginX
	rl.DrawText(title, x, waveMarginY, waveCounterFontSize, rl.White)

	// During the break the count would read "restam 0", which is noise; the
	// announcement is already saying what happens next.
	if network.WavePhase(state.Phase) != network.WavePhaseFighting {
		return
	}
	sub := fmt.Sprintf("restam %d", state.Remaining)
	subWidth := rl.MeasureText(sub, waveRemainingFontSize)
	subX := int32(screenWidth) - subWidth - waveMarginX
	rl.DrawText(sub, subX, waveMarginY+waveCounterFontSize+4, waveRemainingFontSize, rl.Gray)
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

	width := rl.MeasureText(text, waveAnnounceFontSize)
	x := int32((int32(screenWidth) - width) / 2)
	y := int32(screenHeight)/2 - 90

	// Drop shadow keeps the text legible over bright grass.
	rl.DrawText(text, x+3, y+3, waveAnnounceFontSize, rl.Fade(rl.Black, 0.6*alpha))
	rl.DrawText(text, x, y, waveAnnounceFontSize, rl.Fade(rl.RayWhite, alpha))

	countdown := fmt.Sprintf("preparar - %.0f", ceil32(state.BreakTime))
	cWidth := rl.MeasureText(countdown, waveSubAnnounceSize)
	cX := int32((int32(screenWidth) - cWidth) / 2)
	rl.DrawText(countdown, cX, y+waveAnnounceFontSize+10, waveSubAnnounceSize, rl.Fade(rl.LightGray, alpha))
}

func drawMapCleared(screenWidth, screenHeight float32) {
	text := "Mapa limpo"
	width := rl.MeasureText(text, waveClearedFontSize)
	x := int32((int32(screenWidth) - width) / 2)
	y := int32(screenHeight)/2 - 120
	rl.DrawText(text, x+2, y+2, waveClearedFontSize, rl.Fade(rl.Black, 0.6))
	rl.DrawText(text, x, y, waveClearedFontSize, rl.Gold)

	// Clearing the run is what opens the portal, and the portal is at the far
	// end of the map: without naming where it is, the players finish the fight
	// and have nothing telling them where to go.
	sub := "o portal se abriu no fim da estrada de terra"
	sWidth := rl.MeasureText(sub, waveSubAnnounceSize)
	sX := int32((int32(screenWidth) - sWidth) / 2)
	rl.DrawText(sub, sX, y+waveClearedFontSize+8, waveSubAnnounceSize, rl.RayWhite)
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

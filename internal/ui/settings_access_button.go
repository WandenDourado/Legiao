package ui

// O ícone de engrenagem de acesso às Configurações DE DENTRO da partida
// (Passo 6), em toda plataforma — no desktop ele é um segundo caminho ao
// lado do ESC, não uma troca dele.
//
// Mora IMEDIATAMENTE À ESQUERDA do contador "Horda N/M" (wave_hud.go),
// centrado na linha do título, e volta para o canto superior direito quando
// não há contador na tela. Até 28/09/2026 ele ficava sozinho no canto, com
// ~2% da altura da tela de diâmetro: o Gui relatou que era difícil de achar e
// de acertar com o dedo. Encostado no contador ele fica num lugar que o
// jogador já olha, e o diâmetro passou a acompanhar a fonte do contador.

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// settingsGearTeeth is the tooth count of the drawn gear. Eight reads as a
// gear silhouette at this button's small on-screen size without the teeth
// blurring into a fuzzy ring — more would.
const settingsGearTeeth = 8

// settingsGearAlpha is the flat translucency applied to every stroke of the
// icon (backing disc included) — a single fade so the whole glyph reads as
// one soft shape instead of an opaque disc with faded teeth on it. Raised
// from 0.45 to 0.7 on 28/09/2026: "discreet" had become "hard to find".
const settingsGearAlpha = 0.7

// settingsGearGap is the space between the gear and the horde counter.
const settingsGearGap = 12

func settingsAccessRect(sw, sh float32) rl.Rectangle {
	right := sw - Px(waveMarginX)
	titleSize := TextSize(waveCounterFontBase)
	if left, ts, ok := waveCounterLeft(network.GetWaveState(), sw); ok {
		right, titleSize = left-Px(settingsGearGap), ts
	}
	// Diametro: 1,35x a fonte do contador (acompanha a escala do HUD), com
	// piso de 5% da altura para o dedo no celular e de 34 px para uma janela
	// pequena no desktop.
	d := max(float32(titleSize)*1.35, sh*0.05, Px(34))
	cy := Px(waveMarginY) + float32(titleSize)/2
	y := max(cy-d/2, Px(4))
	return rl.NewRectangle(right-d, y, d, d)
}

// DrawSettingsAccessButton paints the gear icon: a backing disc (matching
// the menu panel's palette, main_menu.go) with a toothed gear silhouette on
// top, punched through at the centre so it reads as a gear and not a sun —
// everything faded to settingsGearAlpha.
func DrawSettingsAccessButton(sw, sh float32) {
	rect := settingsAccessRect(sw, sh)
	center := rl.NewVector2(rect.X+rect.Width/2, rect.Y+rect.Height/2)
	radius := rect.Width / 2
	rl.DrawCircleV(center, radius, rl.Fade(rl.NewColor(24, 43, 43, 255), settingsGearAlpha*0.8))
	rl.DrawCircleLinesV(center, radius, rl.Fade(rl.NewColor(202, 170, 92, 255), settingsGearAlpha+0.15))

	gear := rl.Fade(rl.NewColor(238, 242, 248, 255), settingsGearAlpha+0.2)
	hole := rl.Fade(rl.NewColor(24, 43, 43, 255), settingsGearAlpha*0.8)
	bodyR := radius * 0.5
	toothW := radius * 0.26
	toothH := radius * 0.3
	toothDist := bodyR + toothH*0.36
	origin := rl.NewVector2(toothW/2, toothH/2)
	for i := 0; i < settingsGearTeeth; i++ {
		angleDeg := float32(i) * (360.0 / float32(settingsGearTeeth))
		rad := float64(angleDeg) * math.Pi / 180
		tx := center.X + float32(math.Cos(rad))*toothDist
		ty := center.Y + float32(math.Sin(rad))*toothDist
		dst := rl.NewRectangle(tx, ty, toothW, toothH)
		rl.DrawRectanglePro(dst, origin, angleDeg+90, gear)
	}
	rl.DrawCircleV(center, bodyR, gear)
	rl.DrawCircleV(center, bodyR*0.48, hole)
}

// SettingsAccessButtonTapped reports whether this frame's click/tap landed
// on the button (padded by settingsHitPad, not the menu's much larger
// hitPad() — see the comment on that constant in settings_widgets.go).
func SettingsAccessButtonTapped(sw, sh float32) bool {
	return clicked(settingsAccessRect(sw, sh))
}

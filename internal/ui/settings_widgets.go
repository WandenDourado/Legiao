package ui

// Os controles pequenos da tela de Configurações: interruptor, escolha
// fechada (teto de quadro) e botão de mudo. Todos usam clique de mouse —
// exatamente como os botões do menu principal (main_menu.go) já fazem, e
// isso já funciona em toque: no Android este jogo mapeia o toque principal
// para o mouse (é por isso que ShowMenu nunca escaneia toque para CRIAR/
// ENTRAR). Só o Slider (slider.go/slider_input.go) escaneia toque de
// verdade, porque um arraste é mais frágil a essa suposição do que um clique.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// settingsHitPad é o alvo de toque extra desta tela, MENOR de propósito que
// o hitPad() do menu principal (menu.go). hitPad() (18-38px, escala com a
// altura da tela) foi calibrado para os dois botões largos do menu inicial,
// bem separados um do outro. Aqui as linhas ficam ~4-7px umas das outras
// (settings_layout.go, RowGap) e abas/segmentos do ChoiceRow ficam lado a
// lado sem vão nenhum — usar o mesmo pad generoso expandia a área clicável
// de um controle por cima da do vizinho, e um toque perto da borda
// disparava o widget errado (às vezes os dois no mesmo quadro). Este valor
// fica bem abaixo de qualquer vão real do layout, então nunca alcança o
// controle ao lado.
const settingsHitPad float32 = 6

// clicked is the one hit-test every small widget below (and tabRect/
// ChoiceRow) shares: a left click (or its touch-mapped equivalent) landing
// inside rect padded by settingsHitPad.
func clicked(rect rl.Rectangle) bool {
	return rl.IsMouseButtonPressed(rl.MouseLeftButton) && settingsHit(rect, rl.GetMousePosition())
}

// settingsHit is hit() (menu.go) with settingsHitPad instead of hitPad().
func settingsHit(rect rl.Rectangle, point rl.Vector2) bool {
	pad := settingsHitPad
	expanded := rl.NewRectangle(rect.X-pad, rect.Y-pad, rect.Width+2*pad, rect.Height+2*pad)
	return rl.CheckCollisionPointRec(point, expanded)
}

// DrawToggle paints a pill switch: on to the right and filled gold, off to
// the left and dim — the same vocabulary as the volume slider's fill, so a
// bool reads as "a slider with two positions" rather than a different kind
// of control.
func DrawToggle(rect rl.Rectangle, on bool) {
	rl.DrawRectangleRounded(rect, 1, 8, rl.Fade(sliderTrackColor, 0.9))
	knobR := rect.Height * 0.42
	kx := rect.X + rect.Height*0.5
	knobColor := rl.NewColor(150, 150, 150, 255)
	if on {
		kx = rect.X + rect.Width - rect.Height*0.5
		knobColor = sliderFillColor
	}
	rl.DrawCircleV(rl.NewVector2(kx, rect.Y+rect.Height/2), knobR, knobColor)
}

// ToggleRow draws a toggle and reports the new value: clicking it anywhere
// (thanks to hitPad()) flips it. Returns (value, changed).
func ToggleRow(rect rl.Rectangle, value bool) (bool, bool) {
	DrawToggle(rect, value)
	if clicked(rect) {
		return !value, true
	}
	return value, false
}

// ChoiceRow lays out len(labels) equal segments across rect and draws each
// one as a small button, the active one filled. Returns the clicked index,
// or -1 if none was clicked this frame. Used for the frame-cap picker
// (30/60/uncapped) — a closed set of options reads better as buttons than
// as a slider with only three positions.
func ChoiceRow(rect rl.Rectangle, labels []string, active int) int {
	if len(labels) == 0 {
		return -1
	}
	segW := rect.Width / float32(len(labels))
	clickedIdx := -1
	for i, label := range labels {
		seg := rl.NewRectangle(rect.X+float32(i)*segW, rect.Y, segW-4, rect.Height)
		fill := rl.Fade(sliderTrackColor, 0.9)
		textColor := rl.Fade(rl.White, 0.75)
		if i == active {
			fill = sliderFillColor
			textColor = rl.NewColor(24, 43, 43, 255)
		}
		rl.DrawRectangleRounded(seg, 0.3, 6, fill)
		size := int32(max(seg.Height*0.42, Px(12)))
		w := rl.MeasureText(label, size)
		rl.DrawText(label, int32(seg.X+seg.Width/2)-w/2, int32(seg.Y+seg.Height/2)-size/2, size, textColor)
		if clicked(seg) {
			clickedIdx = i
		}
	}
	return clickedIdx
}

// MuteButton is the small speaker glyph next to a volume slider — see
// doc/settings.md, "Botão de mudo por barramento": it silences the bus
// WITHOUT touching the number the slider is at, so unmuting restores exactly
// where the player left it.
func MuteButton(rect rl.Rectangle, muted bool) bool {
	fill := rl.Fade(sliderTrackColor, 0.9)
	if muted {
		fill = rl.NewColor(120, 45, 45, 255)
	}
	rl.DrawRectangleRounded(rect, 0.3, 6, fill)
	glyph := "M"
	if !muted {
		glyph = ")"
	}
	size := int32(max(rect.Height*0.5, Px(12)))
	w := rl.MeasureText(glyph, size)
	rl.DrawText(glyph, int32(rect.X+rect.Width/2)-w/2, int32(rect.Y+rect.Height/2)-size/2, size, rl.Fade(rl.White, 0.9))
	return clicked(rect)
}

// SmallButton is a plain labeled button (restaurar padrões, voltar) — same
// visual family, no icon.
func SmallButton(rect rl.Rectangle, label string) bool {
	hovered := settingsHit(rect, rl.GetMousePosition())
	fill := rl.NewColor(39, 70, 69, 255)
	if hovered {
		fill = rl.NewColor(59, 90, 89, 255)
	}
	rl.DrawRectangleRounded(rect, 0.15, 8, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 8, 2, rl.NewColor(126, 163, 154, 255))
	size := int32(max(rect.Height*0.36, Px(14)))
	w := rl.MeasureText(label, size)
	rl.DrawText(label, int32(rect.X+rect.Width/2)-w/2, int32(rect.Y+rect.Height/2)-size/2, size, rl.NewColor(238, 242, 248, 255))
	return clicked(rect)
}

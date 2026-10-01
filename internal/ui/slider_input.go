package ui

// A metade IMPURA de Slider: le toque, mouse e roda, e desenha. Separada de
// slider.go, cujo conteudo e a matematica sem dependencia de janela que
// slider_test.go testa direto.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Update reads this frame's touch/mouse input against track and returns the
// resolved value. value is the CALLER'S current value (the slider itself is
// stateless about the number — only about which touch owns the drag), so a
// caller that ignores a non-Changed result just keeps using its own value.
func (s *Slider) Update(track rl.Rectangle, value float32) SliderResult {
	res := SliderResult{Value: value}
	// settingsHitPad (settings_widgets.go), nao hitPad() do menu: a trilha
	// fica perto do botao de mudo (audioBusRow, settings_section_audio.go)
	// e do rotulo da linha de baixo, e o pad grande do menu alcancava os
	// dois — um arraste perto da ponta da trilha virava clique no mudo, ou
	// um toque no mudo comecava um arraste.
	pad := settingsHitPad
	hitRect := rl.NewRectangle(track.X-pad, track.Y-pad, track.Width+2*pad, track.Height+2*pad)

	s.updateTouch(track, hitRect, &res)
	s.updateMouseDrag(track, hitRect, &res)
	s.updateWheel(hitRect, &res)
	return res
}

func (s *Slider) updateTouch(track, hitRect rl.Rectangle, res *SliderResult) {
	count := rl.GetTouchPointCount()
	for i := int32(0); i < count; i++ {
		id := rl.GetTouchPointId(i)
		pos := rl.GetTouchPosition(i)
		if s.touchID == sliderNoTouch && id != sliderMouseID && rl.CheckCollisionPointRec(pos, hitRect) {
			// Tocou na trilha: salta pro valor tocado (item 1 do Passo 4),
			// o arraste continua dali no mesmo touch.
			s.touchID = id
			s.applyPos(track, pos.X, res)
			continue
		}
		if s.touchID == id {
			s.applyPos(track, pos.X, res)
		}
	}

	if s.touchID != sliderNoTouch && s.touchID != sliderMouseID {
		stillDown := false
		for i := int32(0); i < count; i++ {
			if rl.GetTouchPointId(i) == s.touchID {
				stillDown = true
				break
			}
		}
		if !stillDown {
			s.touchID = sliderNoTouch
			res.Released = true
		}
	}
}

func (s *Slider) updateMouseDrag(track, hitRect rl.Rectangle, res *SliderResult) {
	if s.touchID == sliderNoTouch && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mp := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mp, hitRect) {
			s.touchID = sliderMouseID
			s.applyPos(track, mp.X, res)
		}
	}
	if s.touchID != sliderMouseID {
		return
	}
	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		s.applyPos(track, rl.GetMousePosition().X, res)
		return
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		s.touchID = sliderNoTouch
		res.Released = true
	}
}

// updateWheel is the desktop-only extra the spec asks for: the mouse wheel
// over the slider, in the same 5% steps as everything else. Arrow keys were
// left out on purpose — see doc/settings.md, "Controle de volume": wiring a
// whole-screen keyboard-focus model for two keys was more machinery than the
// gain justified this round, and the spec explicitly allows cutting it down
// to "just the wheel" when that's the case.
func (s *Slider) updateWheel(hitRect rl.Rectangle, res *SliderResult) {
	if !rl.CheckCollisionPointRec(rl.GetMousePosition(), hitRect) {
		return
	}
	move := rl.GetMouseWheelMove()
	switch {
	case move > 0:
		res.Value = SliderStepUp(res.Value)
		res.Changed = true
		res.Released = true
	case move < 0:
		res.Value = SliderStepDown(res.Value)
		res.Changed = true
		res.Released = true
	}
}

func (s *Slider) applyPos(track rl.Rectangle, x float32, res *SliderResult) {
	v := SliderValueAt(track, x)
	if v != res.Value {
		res.Value = v
		res.Changed = true
	}
}

// sliderTrackColor/sliderFillColor/sliderThumbColor keep the slider in the
// same warm/gold family as drawMenuButton/drawMenuPanel (main_menu.go) so
// the settings screen reads as the same game, not a generic options dialog.
var (
	sliderTrackColor = rl.NewColor(39, 70, 69, 255)
	sliderFillColor  = rl.NewColor(202, 170, 92, 255)
	sliderThumbColor = rl.NewColor(246, 216, 142, 255)
)

// DrawSlider paints the track (grossa, per Passo 4 item 2), the filled
// portion up to value, and a thumb — no numeric label, no mute icon: those
// belong to whoever lays out the row (settings_audio.go), because a slider
// on its own has no idea it is measuring a volume.
func DrawSlider(track rl.Rectangle, value float32) {
	rl.DrawRectangleRounded(track, 0.5, 8, rl.Fade(sliderTrackColor, 0.9))
	fillW := track.Width * clamp01f(value)
	if fillW > 0 {
		fill := rl.NewRectangle(track.X, track.Y, fillW, track.Height)
		rl.DrawRectangleRounded(fill, 0.5, 8, sliderFillColor)
	}
	thumbR := track.Height*0.7 + 2
	thumbCenter := rl.NewVector2(track.X+fillW, track.Y+track.Height/2)
	rl.DrawCircleV(thumbCenter, thumbR, sliderThumbColor)
	rl.DrawCircleLines(int32(thumbCenter.X), int32(thumbCenter.Y), thumbR, rl.NewColor(24, 43, 43, 255))
}

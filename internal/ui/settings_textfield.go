package ui

// TextField edits the player's chosen display name (Passo 5, seção
// Jogador). Keyboard-only: rl.GetCharPressed()/Backspace/Enter, which works
// out of the box on desktop. It does NOT summon Android's on-screen
// keyboard — raylib does not do that for us, and wiring one is real
// platform-specific work this round explicitly cuts (see doc/settings.md,
// "O que ficou de fora"). On a phone with no hardware keyboard attached the
// field still shows the name typed on desktop (it is persisted, per
// internal/settings) — it is only editing ON that device that does not work
// yet.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// TextField holds the one bit of state a text box needs across frames:
// whether it is currently capturing keystrokes.
type TextField struct {
	editing bool
}

// Update reads this frame's keyboard input when the field is being edited,
// and toggles editing on a click inside rect. Returns the (possibly
// unchanged) value and whether it changed this frame.
func (tf *TextField) Update(rect rl.Rectangle, value string, maxRunes int) (string, bool) {
	if clicked(rect) {
		tf.editing = !tf.editing
	}
	if !tf.editing {
		return value, false
	}
	// Clicking outside the field while editing closes it, same as Enter.
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !settingsHit(rect, rl.GetMousePosition()) {
		tf.editing = false
		return value, false
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyEscape) {
		tf.editing = false
		return value, false
	}

	runes := []rune(value)
	changed := false
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch < 32 || len(runes) >= maxRunes {
			continue
		}
		runes = append(runes, ch)
		changed = true
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(runes) > 0 {
		runes = runes[:len(runes)-1]
		changed = true
	}
	if changed {
		value = string(runes)
	}
	return value, changed
}

// Draw paints the field's box, current text, and a blinking caret while
// being edited.
func (tf *TextField) Draw(rect rl.Rectangle, value, placeholder string) {
	border := rl.NewColor(126, 163, 154, 255)
	if tf.editing {
		border = sliderFillColor
	}
	rl.DrawRectangleRounded(rect, 0.2, 8, rl.NewColor(24, 43, 43, 242))
	rl.DrawRectangleRoundedLinesEx(rect, 0.2, 8, 2, border)

	text := value
	color := rl.NewColor(238, 242, 248, 255)
	if text == "" && !tf.editing {
		text = placeholder
		color = rl.Fade(rl.White, 0.4)
	}
	size := int32(max(rect.Height*0.42, Px(14)))
	tx := int32(rect.X + rect.Height*0.35)
	ty := int32(rect.Y+rect.Height/2) - size/2
	rl.DrawText(text, tx, ty, size, color)

	if tf.editing && int(rl.GetTime()*2)%2 == 0 {
		caretX := tx + rl.MeasureText(text, size) + 2
		rl.DrawLineEx(rl.NewVector2(float32(caretX), float32(ty)), rl.NewVector2(float32(caretX), float32(ty)+float32(size)), 2, sliderFillColor)
	}
}

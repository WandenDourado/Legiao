//go:build android

package input

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawJoystickVisual draws the joystick base and knob at the given position and scale.
func DrawJoystickVisual(ts TouchState, centerX, centerY, baseRadius, knobRadius float32) {
	baseCenter := rl.NewVector2(centerX, centerY)
	rl.DrawCircleV(baseCenter, baseRadius, rl.Fade(rl.Gray, 0.5))

	if ts.JoystickTouchID != NoTouch {
		rl.DrawCircleV(ts.JoystickCurrent, knobRadius, rl.Fade(rl.LightGray, 0.8))
	} else {
		rl.DrawCircleV(baseCenter, knobRadius, rl.Fade(rl.LightGray, 0.8))
	}
}

// DrawAimJoystick used to live here. The fire button is now drawn by
// ui.DrawActionButton, together with the two ability buttons, because the
// three are one cluster to the player and drawing them in two packages let
// them drift apart. This package cannot import ui (ui imports this one), so
// the renderer builds the ui.ActionButtonState from the AimJoystick's own
// fields instead. See internal/ui/action_button.go.

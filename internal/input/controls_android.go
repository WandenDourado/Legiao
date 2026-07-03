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

// DrawAimJoystick renders the aim joystick (action button) with aim direction feedback.
func DrawAimJoystick(aj AimJoystick, centerX, centerY, radius float32) {
	center := rl.NewVector2(centerX, centerY)

	if aj.IsActive {
		// Aim mode: draw base at touch origin, direction line, and aim indicator
		rl.DrawCircleV(aj.Origin, radius, rl.Fade(rl.Orange, 0.4))
		rl.DrawCircleLinesV(aj.Origin, radius, rl.Orange)

		// Direction line from origin to current finger position
		rl.DrawLineV(aj.Origin, aj.Current, rl.Yellow)

		// Aim indicator circle at finger position
		indicatorRadius := radius * 0.4
		rl.DrawCircleV(aj.Current, indicatorRadius, rl.Fade(rl.Red, 0.8))

		// Small arrow showing aim direction
		aimEnd := rl.Vector2Add(aj.Origin, rl.Vector2Scale(aj.AimDir, radius*1.5))
		rl.DrawLineV(aj.Origin, aimEnd, rl.Red)
	} else {
		// Resting state: draw as a simple action button
		color := rl.Fade(rl.Red, 0.7)
		rl.DrawCircleV(center, radius, color)
		rl.DrawCircleLinesV(center, radius, rl.White)

		// Draw "FIRE" text
		text := "FIRE"
		textWidth := rl.MeasureText(text, 14)
		rl.DrawText(text,
			int32(center.X)-textWidth/2,
			int32(center.Y)-7,
			14, rl.White)
	}
}

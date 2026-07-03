//go:build !android

package input

// DrawJoystickVisual is a no-op on desktop (touch controls are Android-only).
func DrawJoystickVisual(ts TouchState, centerX, centerY, baseRadius, knobRadius float32) {}

// DrawAimJoystick is a no-op on desktop (touch controls are Android-only).
func DrawAimJoystick(aj AimJoystick, centerX, centerY, radius float32) {}

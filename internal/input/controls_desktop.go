//go:build !android

package input

// DrawJoystickVisual is a no-op on desktop (touch controls are Android-only).
func DrawJoystickVisual(ts TouchState, centerX, centerY, baseRadius, knobRadius float32) {}

// DrawAimJoystick is gone on both platforms — see the note in
// controls_android.go. The fire button is drawn by ui.DrawActionButton, and
// the renderer only reaches that call inside its cfg.FullScreen branch, so
// desktop needs no stub for it.

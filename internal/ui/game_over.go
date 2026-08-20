package ui

// Game Over overlay. The run ends when every player is dead, and only the host
// can start it again — so the button exists on the host's screen and everyone
// else is told who they are waiting for.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// GameOverPanel owns the overlay geometry and the restart button hit-area.
type GameOverPanel struct {
	button rl.Rectangle
}

// NewGameOverPanel creates the overlay. Layout is recomputed every frame, so
// nothing here depends on the screen size at construction time.
func NewGameOverPanel() *GameOverPanel {
	return &GameOverPanel{}
}

// Layout recomputes the restart button from the current screen size, so the
// drawn rectangle and the hit-area can never drift apart.
func (g *GameOverPanel) Layout(sw, sh float32) {
	width := sw * 0.30
	if width < 260 {
		width = 260
	}
	height := sh * 0.10
	if height < 56 {
		height = 56
	}
	g.button = rl.NewRectangle(sw/2-width/2, sh/2+height*0.4, width, height)
}

// Update reports whether the restart button was pressed this frame. It reads
// the mouse, which raylib also feeds from the first touch point, so the same
// code serves desktop and Android.
func (g *GameOverPanel) Update(sw, sh float32) bool {
	g.Layout(sw, sh)
	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return false
	}
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), g.button)
}

// Draw renders the overlay. canReset shows the restart button; without it the
// panel explains that the host is the one who restarts.
func (g *GameOverPanel) Draw(sw, sh float32, canReset bool) {
	g.Layout(sw, sh)

	rl.DrawRectangle(0, 0, int32(sw), int32(sh), rl.Fade(rl.Black, 0.55))

	title := "GAME OVER"
	titleSize := int32(sh * 0.10)
	if titleSize < 48 {
		titleSize = 48
	}
	titleWidth := rl.MeasureText(title, titleSize)
	rl.DrawText(title,
		int32(sw/2)-titleWidth/2,
		int32(sh/2)-titleSize-int32(sh*0.04),
		titleSize, rl.Red)

	subtitle := "Toda a legiao caiu"
	subSize := int32(sh * 0.035)
	if subSize < 20 {
		subSize = 20
	}
	subWidth := rl.MeasureText(subtitle, subSize)
	rl.DrawText(subtitle,
		int32(sw/2)-subWidth/2,
		int32(sh/2)-int32(sh*0.02),
		subSize, rl.Fade(rl.White, 0.9))

	if !canReset {
		wait := "Aguardando o host reiniciar a fase"
		waitWidth := rl.MeasureText(wait, subSize)
		rl.DrawText(wait,
			int32(sw/2)-waitWidth/2,
			int32(g.button.Y+g.button.Height/2)-subSize/2,
			subSize, rl.Fade(rl.White, 0.75))
		return
	}

	hovered := rl.CheckCollisionPointRec(rl.GetMousePosition(), g.button)
	fill := rl.Fade(rl.Maroon, 0.85)
	if hovered {
		fill = rl.Fade(rl.Red, 0.9)
	}
	rl.DrawRectangleRec(g.button, fill)
	rl.DrawRectangleLinesEx(g.button, 3, rl.White)

	label := "REINICIAR FASE (F5)"
	labelSize := int32(g.button.Height * 0.34)
	if labelSize < 18 {
		labelSize = 18
	}
	labelWidth := rl.MeasureText(label, labelSize)
	rl.DrawText(label,
		int32(g.button.X+g.button.Width/2)-labelWidth/2,
		int32(g.button.Y+g.button.Height/2)-labelSize/2,
		labelSize, rl.White)
}

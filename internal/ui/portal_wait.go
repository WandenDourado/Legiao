package ui

// The small HUD shown while the local player is waiting inside an open
// portal (network.LocalPlayerInPortal): the body is not drawn — see
// game/renderer.go — so this is the only thing on screen explaining why,
// plus the way out for whoever stepped in by mistake.

import rl "github.com/gen2brain/raylib-go/raylib"

// PortalWaitPanel owns the "SAIR" button hit-area, laid out fresh every
// frame from the current screen size — same pattern as GameOverPanel.
type PortalWaitPanel struct {
	button rl.Rectangle
}

// NewPortalWaitPanel creates an empty panel; Layout fills it in on first use.
func NewPortalWaitPanel() *PortalWaitPanel {
	return &PortalWaitPanel{}
}

// Layout recomputes the button from the current screen size.
func (p *PortalWaitPanel) Layout(sw, sh float32) {
	width := sw * 0.18
	if width < 140 {
		width = 140
	}
	height := sh * 0.07
	if height < 44 {
		height = 44
	}
	p.button = rl.NewRectangle(sw/2-width/2, sh*0.62, width, height)
}

// Update reports whether the "SAIR" button was pressed this frame. Reading
// the mouse serves desktop click and Android touch alike, exactly like
// GameOverPanel's restart button.
func (p *PortalWaitPanel) Update(sw, sh float32) bool {
	p.Layout(sw, sh)
	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return false
	}
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), p.button)
}

// Draw renders the waiting message and, on a touch build, the SAIR button.
// showButton is false on desktop, where ESC already does the job and a
// button would just duplicate it.
func (p *PortalWaitPanel) Draw(sw, sh float32, showButton bool) {
	msg := "Aguardando o grupo (ESC para sair)"
	if showButton {
		msg = "Aguardando o grupo"
	}
	size := int32(sh * 0.03)
	if size < 20 {
		size = 20
	}
	textWidth := rl.MeasureText(msg, size)
	rl.DrawText(msg, int32(sw/2)-textWidth/2, int32(sh*0.5), size, rl.RayWhite)

	if !showButton {
		return
	}
	p.Layout(sw, sh)
	rl.DrawRectangleRec(p.button, rl.Fade(rl.DarkGray, 0.85))
	rl.DrawRectangleLinesEx(p.button, 2, rl.RayWhite)
	label := "SAIR"
	labelSize := int32(p.button.Height * 0.4)
	labelWidth := rl.MeasureText(label, labelSize)
	rl.DrawText(label,
		int32(p.button.X+p.button.Width/2)-labelWidth/2,
		int32(p.button.Y+p.button.Height/2)-labelSize/2,
		labelSize, rl.RayWhite)
}

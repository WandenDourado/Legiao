package game

// Cancelling out of a portal wait. The host owns the InPortal flag
// (host_portal_presence.go) — it decides who is waiting — but only the
// client that owns this body may move it, so "leaving" is this machine
// nudging its own Position back outside the rectangle. The host notices on
// its own the next tick it recomputes presence; nothing here talks to the
// network directly.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiPortalWait owns the "SAIR" button hit-area, drawn by the renderer and
// read here — same split as uiGameOver.
var uiPortalWait = ui.NewPortalWaitPanel()

// UpdatePortalCancel is called once per frame, only while
// network.LocalPlayerInPortal is true. ESC (desktop) or the SAIR button
// (touch) pushes the player one step outside whichever portal rectangle
// they are standing in, toward the party's centre, and clears the LOCAL
// mirror immediately so control comes back this same frame instead of
// waiting a round trip to the host.
func UpdatePortalCancel(p *entity.Player, w *World, cfg Config, sw, sh float32) {
	cancelled := (!cfg.FullScreen && rl.IsKeyPressed(rl.KeyEscape)) || uiPortalWait.Update(sw, sh)
	if !cancelled {
		return
	}
	stepOutsidePortal(p, w)
	network.LeaveLocalPortal()
}

// stepOutsidePortal moves p just clear of whatever portal rectangle its
// foot box currently overlaps, toward the living party's centre (or away
// from the portal if there is no party to lean on, e.g. solo play).
func stepOutsidePortal(p *entity.Player, w *World) {
	if w == nil {
		return
	}
	box, width, height := entity.PlayerGroundBox(p)
	for _, portal := range w.Portals {
		if !portal.Contains(box, width, height) {
			continue
		}
		centre := portal.Center()
		toward := rl.Vector2Subtract(partyCentreForCancel(p), centre)
		if toward.X == 0 && toward.Y == 0 {
			toward = rl.NewVector2(0, 1)
		} else {
			toward = rl.Vector2Normalize(toward)
		}
		// Half the rectangle's own diagonal, plus a margin, guarantees the
		// step actually clears it regardless of the rectangle's shape.
		clearDistance := rl.Vector2Length(rl.NewVector2(portal.Rect.Width, portal.Rect.Height))/2 + 40
		p.Position = rl.Vector2Add(centre, rl.Vector2Scale(toward, clearDistance))
		return
	}
}

// partyCentreForCancel averages the living party's authoritative positions.
// Solo play (no network role at all) has nobody to average, so it falls
// back to the player's own current position, which stepOutsidePortal reads
// as "no direction, just back away from the portal".
func partyCentreForCancel(p *entity.Player) rl.Vector2 {
	if network.Role == "" {
		return p.Position
	}
	players := network.PresentPlayers()
	if len(players) == 0 {
		return p.Position
	}
	var sum rl.Vector2
	count := 0
	for _, state := range players {
		if state.IsDead {
			continue
		}
		sum = rl.Vector2Add(sum, rl.NewVector2(float32(state.X), float32(state.Y)))
		count++
	}
	if count == 0 {
		return p.Position
	}
	return rl.Vector2Scale(sum, 1/float32(count))
}

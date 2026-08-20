package game

// F4 debug overlay: draws the navigation mesh (free/blocked cells) and
// every bot and monster's current mesh-planned route. F2, F3, F5 and F8 are
// already taken (doc/plan_navegacao_bots_monstros.md §5). Host-only: a
// client never builds a mesh of its own, same restriction DrawFootprintDebug
// already has for enemy footprints.

import (
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var navDebugOn bool

// UpdateNavDebugToggle flips the overlay on F4. Called once per frame from
// input processing, the same place F3's own toggle lives.
func UpdateNavDebugToggle() {
	if rl.IsKeyPressed(rl.KeyF4) {
		navDebugOn = !navDebugOn
	}
}

// DrawNavDebug draws the mesh and every bot/monster route inside the
// world-space camera block, alongside DrawFootprintDebug. visible (the
// camera's world-space rectangle) culls cells far outside what is on
// screen — the mesh can be tens of thousands of cells, and this is a
// per-frame debug draw, not a one-off.
func DrawNavDebug(visible rl.Rectangle) {
	if !navDebugOn || network.Role != "host" || network.CurrentHost == nil {
		return
	}
	host := network.CurrentHost
	grid := host.EntityManager.Nav
	if grid == nil {
		return
	}

	margin := float32(64)
	grid.ForEachCell(func(center rl.Vector2, half float32, free bool) {
		if center.X < visible.X-margin || center.X > visible.X+visible.Width+margin ||
			center.Y < visible.Y-margin || center.Y > visible.Y+visible.Height+margin {
			return
		}
		col := rl.Fade(rl.Red, 0.35)
		if free {
			col = rl.Fade(rl.Lime, 0.10)
		}
		rl.DrawRectangleLines(int32(center.X-half), int32(center.Y-half), int32(half*2), int32(half*2), col)
	})

	for id := range network.GetAllPlayers() {
		drawRoute(host.BotRoute(id))
	}
	for _, e := range host.EntityManager.GetAllEnemies() {
		drawRoute(e.Route())
	}
}

// drawRoute connects an agent's remaining waypoints with a line, dotting
// each one — empty when the agent is walking the straight line instead of
// following a planned route, which is the common case and correctly draws
// nothing.
func drawRoute(path []rl.Vector2) {
	for i, p := range path {
		rl.DrawCircleV(p, 6, rl.SkyBlue)
		if i > 0 {
			rl.DrawLineEx(path[i-1], p, 2, rl.SkyBlue)
		}
	}
}

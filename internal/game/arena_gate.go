package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// arenaGate owns the two one-way rules of an arena visit: crossing the south
// edge seals the route back to the approach corridor, and clearing the map
// opens the physical gate at its north exit. Both reset with the stage run.
type arenaGate struct {
	zone         tilemap.Zone
	armed        bool
	returnLocked bool
	exitOpen     bool
	mapPath      string
	generation   int
}

// bind forgets state from a previous map or stage attempt. The latter matters:
// a failed attempt must begin with the exit gate closed again.
func (g *arenaGate) bind(w *World) bool {
	gen := network.StageGeneration()
	if w != nil && w.Path == g.mapPath && gen == g.generation {
		return false
	}
	changed := false
	if g.exitOpen && w != nil && w.Collision != nil {
		changed = w.Collision.SetFootprintsEnabledOverlapping(g.exitGateArea(), true)
	}
	g.zone, g.armed = tilemap.ArenaLockZone(w.Zones)
	g.returnLocked = false
	g.exitOpen = false
	g.mapPath = w.Path
	g.generation = gen
	return changed
}

// exitGateArea is the single tile strip immediately north of the arena. Map
// authors place the closed gate footprint there; keeping the measurement in
// world units makes it follow the map's own tile scale instead of the screen.
func (g *arenaGate) exitGateArea() rl.Rectangle {
	return rl.NewRectangle(g.zone.Rect.X, g.zone.Rect.Y-128, g.zone.Rect.Width, 128)
}

// UpdateArenaGate applies progression-driven gate changes and the one-way
// entrance lock. It runs after normal collision resolution so it only corrects
// an attempted return through the arena's south edge.
func (w *World) UpdateArenaGate(p *entity.Player) {
	if w == nil || p == nil {
		return
	}
	changed := w.arenaGate.bind(w)
	g := &w.arenaGate
	if network.CurrentHost != nil {
		// The host has no notion of a map zone on its own (doc/tilemap.md
		// "Arena de mão única"): this is the one channel that tells it the
		// zone exists, every frame, so it can apply the same one-way rule to
		// a bot's body — the corpo humano local já obedece porque ESTE
		// próprio código roda no cliente dele; o bot é um corpo que só o
		// host move, e ninguém aplicava a regra a ele antes desta chamada.
		network.CurrentHost.SetArenaLock(g.zone.Rect, g.armed)
	}
	if !g.armed {
		return
	}
	state := network.GetWaveState()
	if !g.exitOpen && state.Total > 0 && network.WavePhase(state.Phase) == network.WavePhaseCleared {
		g.exitOpen = true
		changed = w.Collision.SetFootprintsEnabledOverlapping(g.exitGateArea(), false) || changed
	}
	if changed && network.CurrentHost != nil {
		// As MAGIAS nao precisam ser avisadas. Desde 22/08/2026 elas leem a
		// propria `CollisionGrid` (`Host.SetSolid`, uma vez por mapa), e o
		// portao muda os apoios DENTRO dela: quem le pela grade ja ve a
		// mudanca no quadro seguinte. Antes elas liam uma lista plana de
		// retangulos derivada da grade, e era essa copia que tinha de ser
		// remontada aqui — a linha que existia neste ponto.
		//
		// A malha de navegacao NAO e assim: ela e derivada uma vez no
		// carregamento e nao observa a grade. Um portao que abre em jogo tem
		// de avisa-la, ou a horda continua navegando contra uma porta que ja
		// saiu do lugar.
		network.CurrentHost.RebuildNavArea(g.exitGateArea())
	}

	center, _, height := PlayerFootprint(p)
	if !g.returnLocked && g.zone.Contains(center) {
		g.returnLocked = true
	}
	if !g.returnLocked || center.X < g.zone.Rect.X || center.X >= g.zone.Rect.X+g.zone.Rect.Width {
		return
	}
	// Keep the whole foot box on the arena side of the threshold. The north
	// exit remains unrestricted; only crossing back toward the corridor is
	// rejected.
	limit := g.zone.Rect.Y + g.zone.Rect.Height - height/2
	if center.Y > limit {
		entity.MoveByGroundCorrection(p, center, rl.NewVector2(center.X, limit))
	}
}

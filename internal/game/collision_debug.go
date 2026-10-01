package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawFootprintDebug outlines, under the F3 overlay, the exact boxes movement
// resolution tests — the player's and every enemy's. Both come from the same
// footprint helpers the simulation uses, so what is drawn is what is being
// collided, and a monster clipping a tree is visible instead of guessed.
//
// Only the host simulates enemy movement; a client just renders positions it
// was sent, so it has no footprints of its own to show.
func DrawFootprintDebug(p *entity.Player, grid *tilemap.CollisionGrid) {
	if !tilemap.DebugEnabled() {
		return
	}
	center, width, height := PlayerFootprint(p)
	grid.DrawEntityFootprintDebug(center, width, height)

	if network.Role != "host" || network.CurrentHost == nil {
		return
	}
	for _, e := range network.CurrentHost.EntityManager.GetAllEnemies() {
		enemyW, enemyH := entity.EnemyFootprint(e)
		grid.DrawEntityFootprintDebug(e.Position, enemyW, enemyH)
		// A CAIXA DE ACERTO tambem, em amarelo, e ela nao coincide com a de
		// movimento. O F3 mostrava so a verde — a caixa que resolve
		// deslocamento, que no orc fica nos pes — e por isso a caixa de acerto
		// ficou meses errada sem ninguem ver: o overlay dizia a verdade sobre
		// uma pergunta e ficava calado sobre a outra. Um circulo desenhado onde
		// o tiro e testado responde "por que eu nao acertei isso?" na hora.
		rl.DrawCircleLinesV(e.HitCenter(), e.HitRadius(), rl.Yellow)
	}
}

package game

// Moving between maps. The portal and the F8 stage skip are two different
// reasons to travel, but only one act: swap the whole World at once, so
// renderer, collision, bounds, spawns and portals always come from the same
// file and nothing is left pointing at the map just left behind.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// travelTo loads targetMap, places the player on its arrival point and
// retargets the authoritative simulation. It returns the world it was given
// when the destination cannot be loaded, so the caller can always assign the
// result and a broken destination never ends the session.
//
// targetSpawn is the name of an object in the destination's `spawn` layer;
// empty means the map's own `player_spawn`, which TryLoadWorld already found.
func travelTo(current *World, targetMap, targetSpawn string, p *entity.Player) *World {
	next := TryLoadWorld(targetMap)
	if next == nil {
		// A destination that will not load is a mapping bug, not a reason to
		// end the session. Stay put, keep the old world alive.
		return current
	}
	next.arrived = true
	if targetSpawn != "" {
		// The renderer already holds the parsed destination, so an arrival
		// marker costs a lookup instead of a second parse of the same file.
		if arrival, found := tilemap.NamedSpawnPosition(next.Renderer.Map, targetSpawn); found {
			next.PlayerSpawn = arrival
		}
	}

	current.Unload()
	// Os retratos de dialogo saem com o mapa, e nao com a sessao. Cada fase
	// apresenta oradores novos e o `reference.png` de cada um e ~6 MB na placa;
	// mantidos por sessao, o elenco inteiro da campanha ficava residente ate o
	// fim (doc/performance.md, M5). Depois de `current.Unload()` pelo mesmo
	// motivo que ele: e a linha em que o mapa que ficou para tras devolve tudo
	// o que era so dele.
	ui.UnloadPortraits()
	p.Position = next.PlayerSpawn
	p.Velocity = rl.Vector2{}
	next.ApplyToHost()
	// A festa inteira vai junto, e nao so quem esta nesta maquina. Um jogador
	// vivo se corrigiria sozinho no proximo quadro, ao publicar a posicao; um
	// MORTO nao publica nada, e o host continuaria anunciando um cadaver nas
	// coordenadas do mapa anterior — e ressuscitando-o la.
	if host := network.CurrentHost; host != nil {
		host.PlaceEveryoneAtSpawn()
	}
	return next
}

package game

// O clímax do mapa 3 é uma PORTA, não um limiar de vida.
//
// No mapa 2 a cena dispara quando o grupo está caindo durante uma horda:
// `partyIsFalling` pede `WaveState.Total > 0` e fase de luta. O mapa 3 não tem
// horda nenhuma — a travessia é de guarnição — então aquele gatilho nunca seria
// avaliado ali, e o roteiro `world_03_climax` ficaria em silêncio para sempre.
//
// Aqui a condição é chegar: **todos os jogadores vivos dentro da zona
// `fortaleza`**. Morrer no caminho não abre nada, e o grupo inteiro morto antes
// do portão é Game Over como em qualquer fase. O resgate do último suspiro
// pertence à luta que acontece DEPOIS de o portão ser alcançado.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// stageClimax e a porta da fase em jogo.
//
// Variavel de pacote como o diretor de dialogo e pelo mesmo motivo: ela guarda
// estado que dura UMA visita ao mapa, e `bind` a reinicia sozinha quando o
// caminho do mapa muda — foi o cuidado que faltou uma vez e deixou o roteiro
// do climax marcado como tocado depois de um F5.
var stageClimax climaxGate

// climaxGate watches the fortress zone and reports the frame the party has
// arrived. It is the map's, not the game's: a map without a gate zone never
// fires, which is every map but the third.
type climaxGate struct {
	zone   tilemap.Zone
	armed  bool
	sprung bool
	// mapPath guards against the world changing under us. Without it a portal
	// to another map would leave the previous map's zone armed, and the first
	// player standing at those coordinates in the new map would spring it.
	mapPath string
	// generation e a corrida. O caminho do mapa NAO muda num F5, entao sem
	// isto a porta continuaria marcada como disparada e a segunda tentativa
	// chegaria a fortaleza sem emboscada nenhuma — a fase acabaria em silencio.
	// E exatamente o bug que o diretor de dialogo ja teve com o `played`.
	generation int
}

// bind points the gate at a world. Called every frame; it only does work when
// the map actually changed.
func (g *climaxGate) bind(w *World) {
	gen := network.StageGeneration()
	if w == nil || (w.Path == g.mapPath && gen == g.generation) {
		return
	}
	g.mapPath, g.generation = w.Path, gen
	g.armed, g.sprung = false, false
	zone, ok := tilemap.ClimaxGateZone(w.Zones)
	g.zone = zone
	g.armed = ok
	if ok {
		log.Printf("[Climax] %s: portao do climax em %.0f,%.0f (%.0fx%.0f)",
			w.Path, zone.Rect.X, zone.Rect.Y, zone.Rect.Width, zone.Rect.Height)
	}
}

// partyArrived reports whether every living player stands inside the zone.
//
// Quatro regras, e cada uma delas é uma decisão:
//
//   - Um grupo VAZIO não chegou. Antes do primeiro estado de jogador a lista
//     está vazia e "todos dentro" é verdade sobre ninguém — a cena tocaria no
//     primeiro quadro. É o mesmo cuidado que `partyIsFalling` já toma.
//   - Jogador MORTO não conta. Ele volta pela regra normal e o grupo espera;
//     exigir que um cadáver esteja na esplanada travaria a fase até o respawn,
//     e travaria de vez se ele morresse fora dela.
//   - TODOS, e não a maioria. O clímax é uma emboscada contra o grupo reunido;
//     armá-la com metade do time ainda a caminho entrega a cena a quem chegou
//     e mata quem não chegou.
//   - Quem está DENTRO DE UM PORTAL não conta, nem para segurar nem para
//     abrir. Um corpo em espera no portal está congelado e nem sequer é
//     desenhado (network/host_portal_presence.go), então exigir que ele esteja
//     na esplanada trava a fase para sempre, e ninguém em campo consegue ver
//     por quê. Hoje isso é cinto e suspensório — o portal de um mapa de
//     emboscada não abre antes dela (network/climax_pending.go) —, mas essa
//     combinação foi exatamente a trava relatada na fase 3, e a porta não deve
//     depender de o portal estar fechado para funcionar.
func (g *climaxGate) partyArrived(players map[string]network.PlayerState) bool {
	alive := 0
	for _, p := range players {
		if p.IsDead || p.InPortal {
			continue
		}
		alive++
		if !g.zone.Contains(rl.NewVector2(float32(p.X), float32(p.Y))) {
			return false
		}
	}
	return alive > 0
}

// update springs the gate once, the frame the party is all inside.
//
// Returns true on that single frame. It never fires twice for the same run:
// the ambush is a set piece, and a group that steps out and back in is not a
// second ambush.
func (g *climaxGate) update(w *World) bool {
	g.bind(w)
	if !g.armed || g.sprung {
		return false
	}
	// PresentPlayers: an absent player (mid-reconnect) cannot hold the gate
	// closed for the rest of the party. See host_absence.go.
	if !g.partyArrived(network.PresentPlayers()) {
		return false
	}
	g.sprung = true
	log.Printf("[Climax] o grupo inteiro chegou a fortaleza; a emboscada comeca")
	return true
}

package game

// O portal carrega o GRUPO, não a pessoa.
//
// Enquanto a travessia era local cada um saía quando queria, e o resultado era
// uma partida em dois mapas: o host simulando o antigo, o cliente sozinho no
// novo. A regra agora é a mesma que o clímax do mapa 3 já usa (climax_gate.go),
// porque é a mesma pergunta — "o grupo chegou?" — e vale a pena ela ter uma
// resposta só:
//
//   - TODOS, e não a maioria. Um portal que leva metade do time deixa a outra
//     metade num mapa que ninguém mais simula.
//   - MORTO não conta para a espera. Ele viaja junto e continua morto do outro
//     lado, esperando o mesmo ressurgimento que esperaria aqui; exigir que um
//     cadáver caminhe até o portal travaria a fase para sempre.
//   - Grupo VAZIO não chegou. Antes do primeiro estado de jogador a lista está
//     vazia, e "todos dentro" seria verdade sobre ninguém.
//   - O MESMO portal. Contar por mapa, e não por portal, faria dois portais com
//     destinos diferentes somarem um grupo que nunca esteve junto.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// portalTally is how the party stands in relation to one portal: quantos vivos
// existem e quantos deles estão pisando nele.
type portalTally struct {
	Inside int
	Alive  int
}

// ready reports whether this portal may carry the party.
func (t portalTally) ready() bool {
	return t.Alive > 0 && t.Inside == t.Alive
}

// waiting reports whether the portal has someone standing in it and is holding
// for the rest. It is what the counter on the portal is drawn from: a portal
// nobody is standing in has nothing to announce.
func (t portalTally) waiting() bool {
	return t.Inside > 0 && !t.ready()
}

// countParty tallies the living players standing on each portal, in the same
// order as the portals given.
//
// Positions come from network.GetAllPlayers(), the authoritative snapshot, and
// never from the interpolated one the renderer uses: the drawing is 100 ms old
// on purpose, and deciding a map change with it would mean deciding with 100 ms
// of error. See doc/network.md.
func countParty(portals []tilemap.Portal, players map[string]network.PlayerState) []portalTally {
	tallies := make([]portalTally, len(portals))
	for _, state := range players {
		if state.IsDead {
			continue
		}
		center, width, height := entity.GroundBoxAt(
			rl.NewVector2(float32(state.X), float32(state.Y)),
			entity.CharacterType(state.Character))
		for i, portal := range portals {
			tallies[i].Alive++
			if portal.Contains(center, width, height) {
				tallies[i].Inside++
			}
		}
	}
	return tallies
}

// portalParty is the party the gate counts.
//
// Offline there is no network snapshot at all, so the local player IS the whole
// party and is turned into a one-entry roster instead of being tested by a
// separate path. Measuring solo play with different code is how the two rules
// would drift apart.
func portalParty(p *entity.Player) map[string]network.PlayerState {
	if network.Role != "" {
		// PresentPlayers, not GetAllPlayers: a player mid-reconnect cannot
		// hold the portal door shut for the rest of the party. See
		// network/host_absence.go.
		return network.PresentPlayers()
	}
	return map[string]network.PlayerState{
		network.LocalPlayerID: {
			PlayerID:  network.LocalPlayerID,
			X:         int(p.Position.X),
			Y:         int(p.Position.Y),
			Character: string(p.CharType),
			IsDead:    p.IsDead,
		},
	}
}

// anyoneInside reports whether any portal has someone standing in it.
func anyoneInside(tallies []portalTally) bool {
	for _, t := range tallies {
		if t.Inside > 0 {
			return true
		}
	}
	return false
}

// readyPortal is the portal the whole living party is standing on, if any.
func readyPortal(portals []tilemap.Portal, tallies []portalTally) (tilemap.Portal, bool) {
	for i, portal := range portals {
		if tallies[i].ready() {
			return portal, true
		}
	}
	return tilemap.Portal{}, false
}

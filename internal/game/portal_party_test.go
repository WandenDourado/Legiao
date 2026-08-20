package game

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A caixa testada e a dos PES, ~105 px abaixo de Position, entao um jogador
// "no portal" tem a posicao acima dele. Estes ajudantes escrevem os casos em
// termos de onde o jogador esta EM PE, e nao de aritmetica de deslocamento.

func portalAt(x, y float32, target string) tilemap.Portal {
	return tilemap.Portal{
		Name:      target,
		Rect:      rl.NewRectangle(x, y, 128, 128),
		TargetMap: target,
	}
}

// standingOn devolve o estado de um jogador cujos pes caem no centro do portal.
func standingOn(id string, p tilemap.Portal, dead bool) network.PlayerState {
	center := p.Center()
	def := entity.GetCharacter(entity.CharMago)
	// GroundPoint empurra os pes para BAIXO a partir de Position, entao a
	// posicao que poe os pes no centro do portal e o centro menos o mesmo
	// deslocamento.
	return network.PlayerState{
		PlayerID:  id,
		X:         int(center.X),
		Y:         int(center.Y - entity.GroundOffset(def)),
		Character: string(entity.CharMago),
		IsDead:    dead,
	}
}

func farAway(id string, dead bool) network.PlayerState {
	return network.PlayerState{
		PlayerID:  id,
		X:         -10000,
		Y:         -10000,
		Character: string(entity.CharMago),
		IsDead:    dead,
	}
}

func TestPortalCarriesOnlyTheWholeParty(t *testing.T) {
	portals := []tilemap.Portal{portalAt(1000, 1000, "assets/maps/world_02.json")}

	cases := []struct {
		name    string
		players map[string]network.PlayerState
		inside  int
		alive   int
		ready   bool
	}{
		{
			// Um so, e o grupo e ele: o jogo solo passa pela mesma regra.
			name:    "sozinho no portal atravessa",
			players: map[string]network.PlayerState{"a": standingOn("a", portals[0], false)},
			inside:  1, alive: 1, ready: true,
		},
		{
			name: "metade dentro espera",
			players: map[string]network.PlayerState{
				"a": standingOn("a", portals[0], false),
				"b": farAway("b", false),
			},
			inside: 1, alive: 2, ready: false,
		},
		{
			name: "todos dentro atravessa",
			players: map[string]network.PlayerState{
				"a": standingOn("a", portals[0], false),
				"b": standingOn("b", portals[0], false),
			},
			inside: 2, alive: 2, ready: true,
		},
		{
			// Cadaver nao segura a fase: ele viaja junto e espera o
			// ressurgimento do outro lado.
			name: "morto longe do portal nao trava o grupo",
			players: map[string]network.PlayerState{
				"a": standingOn("a", portals[0], false),
				"b": farAway("b", true),
			},
			inside: 1, alive: 1, ready: true,
		},
		{
			// "Todos dentro" seria verdade sobre ninguem. O primeiro quadro de
			// uma partida tem a lista vazia, e o portal nao pode disparar nele.
			name:    "grupo vazio nao chegou",
			players: map[string]network.PlayerState{},
			inside:  0, alive: 0, ready: false,
		},
		{
			name: "grupo inteiro morto nao atravessa",
			players: map[string]network.PlayerState{
				"a": farAway("a", true),
				"b": farAway("b", true),
			},
			inside: 0, alive: 0, ready: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countParty(portals, tc.players)[0]
			if got.Inside != tc.inside || got.Alive != tc.alive {
				t.Errorf("contagem = %d/%d, esperado %d/%d",
					got.Inside, got.Alive, tc.inside, tc.alive)
			}
			if got.ready() != tc.ready {
				t.Errorf("ready() = %v, esperado %v", got.ready(), tc.ready)
			}
		})
	}
}

// TestPartySplitBetweenPortals: dois portais com destinos diferentes nao somam
// um grupo. Cada um conta so quem esta NELE, entao um time dividido espera —
// que e o contrario de metade do time sumir para outro mapa.
func TestPartySplitBetweenPortals(t *testing.T) {
	portals := []tilemap.Portal{
		portalAt(1000, 1000, "assets/maps/world_02.json"),
		portalAt(4000, 4000, "assets/maps/world_04.json"),
	}
	players := map[string]network.PlayerState{
		"a": standingOn("a", portals[0], false),
		"b": standingOn("b", portals[1], false),
	}

	tallies := countParty(portals, players)
	for i, tally := range tallies {
		if tally.Inside != 1 || tally.Alive != 2 {
			t.Errorf("portal %d: contagem %d/%d, esperado 1/2", i, tally.Inside, tally.Alive)
		}
		if tally.ready() {
			t.Errorf("portal %d levou um grupo dividido", i)
		}
		if !tally.waiting() {
			t.Errorf("portal %d nao anuncia que esta esperando", i)
		}
	}
	if _, ok := readyPortal(portals, tallies); ok {
		t.Error("readyPortal escolheu um portal com o grupo dividido")
	}
}

func TestReadyPortalPicksTheOneWithEveryone(t *testing.T) {
	portals := []tilemap.Portal{
		portalAt(1000, 1000, "assets/maps/world_02.json"),
		portalAt(4000, 4000, "assets/maps/world_04.json"),
	}
	players := map[string]network.PlayerState{
		"a": standingOn("a", portals[1], false),
		"b": standingOn("b", portals[1], false),
	}

	portal, ok := readyPortal(portals, countParty(portals, players))
	if !ok {
		t.Fatal("o grupo inteiro estava no segundo portal e ele nao levou ninguem")
	}
	if portal.TargetMap != "assets/maps/world_04.json" {
		t.Errorf("destino = %s, esperado world_04", portal.TargetMap)
	}
}

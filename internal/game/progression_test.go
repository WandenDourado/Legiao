package game

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// A suprema e ganha por fase E POR PERSONAGEM, e a regra e derivada das duas
// listas que ja existem: `campaignMaps` (a ordem) e `lastStandHeroes` (quem se
// ergue em cada fase). Estes testes travam a derivacao, nao os nomes — uma fase
// nova entra na lista e herda a regra sozinha.

func TestNobodyHasAnUltimateOnTheFirstPhase(t *testing.T) {
	granted := UltimatesGrantedOn(campaignMaps[0])
	if len(granted) != 0 {
		t.Errorf("a primeira fase liberou %v; ela e jogada sem nenhuma suprema", granted)
	}
}

func TestTheHeroOfAPhaseDoesNotArriveWithTheirUltimate(t *testing.T) {
	// A cena da fase e onde o personagem GANHA a suprema. Chegar com ela na mao
	// esvazia a cena: o resgate deixa de ser um resgate e vira um botao que o
	// jogador ja tinha.
	for _, m := range campaignMaps {
		hero, ok := network.LastStandCharacterFor(m)
		if !ok {
			continue
		}
		if UltimatesGrantedOn(m)[hero] {
			t.Errorf("%s: %s ja chega com a suprema, mas e nesta fase que ele se ergue", m, hero)
		}
	}
}

func TestTheHeroOfAPhaseHasTheirUltimateOnTheNextOne(t *testing.T) {
	for i, m := range campaignMaps {
		hero, ok := network.LastStandCharacterFor(m)
		if !ok || i+1 >= len(campaignMaps) {
			continue
		}
		next := campaignMaps[i+1]
		if !UltimatesGrantedOn(next)[hero] {
			t.Errorf("%s: %s se ergueu em %s e deveria ter a suprema aqui", next, hero, m)
		}
	}
}

func TestGrantsOnlyGrow(t *testing.T) {
	// Ninguem PERDE uma suprema ao avancar. Uma tabela escrita a mao poderia
	// esquecer um personagem numa fase do meio; a derivacao nao pode, e este
	// teste e o que avisa se alguem trocar a derivacao por uma tabela.
	var previous map[entity.CharacterType]bool
	for _, m := range campaignMaps {
		current := UltimatesGrantedOn(m)
		for char := range previous {
			if !current[char] {
				t.Errorf("%s: %s perdeu a suprema que ja tinha na fase anterior", m, char)
			}
		}
		previous = current
	}
}

func TestMapOutsideTheCampaignUnlocksEverything(t *testing.T) {
	// Travar um mapa por onde ninguem progride so atrapalharia quem esta
	// experimentando.
	for _, path := range []string{"assets/maps/sandbox.json", ""} {
		granted := UltimatesGrantedOn(path)
		for _, def := range entity.AllCharacters() {
			if !granted[def.Type] {
				t.Errorf("%q: %s travado num mapa fora da campanha", path, def.Type)
			}
		}
	}
}

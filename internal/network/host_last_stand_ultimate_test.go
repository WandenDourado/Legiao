package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// O resgate reergue o heroi da fase, mas a campanha so entrega a suprema dele
// na fase SEGUINTE (game.UltimatesGrantedOn) — sem uma concessao separada, a
// cena anuncia a magia e o host recusa o lancamento. Estes testes travam a
// concessao POR CORRIDA (progression.go) que resolve isso.

func TestRescuePlayingTheHeroUnlocksTheirUltimateForThisRun(t *testing.T) {
	SetUnlockedUltimates(nil) // campanha nao concede nada nesta fase
	defer SetUnlockedUltimates(nil)

	h := &Host{
		stageMap: "assets/maps/world_02.json", // heroi: Necromante
		players: map[string]*PlayerState{
			"necro1": {
				PlayerID: "necro1", Character: string(entity.CharNecromante),
				Health: 20, MaxHealth: 100,
			},
		},
	}

	if UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a suprema ja estava liberada antes do resgate")
	}

	saved := h.reviveHero(entity.CharNecromante)
	if saved != "necro1" {
		t.Fatalf("reviveHero devolveu %q; esperava o jogador do heroi", saved)
	}
	if !UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("o resgate reergueu o heroi mas nao liberou a ultimate dele")
	}
}

func TestRescueGrantsNothingWhenNobodyPlaysTheHero(t *testing.T) {
	SetUnlockedUltimates(nil)
	defer SetUnlockedUltimates(nil)

	h := &Host{
		stageMap: "assets/maps/world_02.json",
		players: map[string]*PlayerState{
			"outro": {PlayerID: "outro", Character: string(entity.CharMago), Health: 20, MaxHealth: 100},
		},
	}

	if saved := h.reviveHero(entity.CharNecromante); saved != "" {
		t.Fatalf("reviveHero devolveu %q sem ninguem jogar o heroi", saved)
	}
	// Ninguem jogando o heroi: o NPC lanca direto (summonHeroNPC), que nao
	// passa pelo gate de suprema. A concessao por corrida nao deveria existir.
	if UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a suprema foi liberada sem ninguem ter sido reerguido como o heroi")
	}
}

func TestRunGrantedUltimateDoesNotSurviveAMapChange(t *testing.T) {
	SetUnlockedUltimates(nil)
	defer SetUnlockedUltimates(nil)

	GrantUltimateForRun(entity.CharNecromante)
	if !UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a concessao da corrida nao pegou")
	}

	// A troca de mapa passa por SetUnlockedUltimates (World.ApplyToHost) —
	// isso, sozinho, tem de apagar a concessao da corrida anterior.
	SetUnlockedUltimates(nil)
	if UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a concessao da corrida atravessou a troca de mapa")
	}
}

func TestRunGrantedUltimateDoesNotSurviveAStageReset(t *testing.T) {
	SetUnlockedUltimates(nil)
	defer SetUnlockedUltimates(nil)

	GrantUltimateForRun(entity.CharNecromante)
	if !UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a concessao da corrida nao pegou")
	}

	// O reinicio de fase (F5) fica no MESMO mapa, entao nunca passa por
	// SetUnlockedUltimates — precisa da limpeza propria.
	ClearRunGrantedUltimates()
	if UltimateUnlockedFor(entity.CharNecromante) {
		t.Fatal("a concessao da corrida sobreviveu ao reinicio de fase")
	}
}

func TestRunGrantedUltimateDoesNotAlterTheCampaignSet(t *testing.T) {
	// A concessao da corrida e uma coisa POR CIMA da campanha, nao uma
	// substituicao: TestTheHeroOfAPhaseDoesNotArriveWithTheirUltimate (game
	// package) depende de UltimatesGrantedOn continuar dizendo que a fase do
	// heroi nao concede a propria suprema.
	SetUnlockedUltimates(nil)
	defer SetUnlockedUltimates(nil)

	GrantUltimateForRun(entity.CharNecromante)
	progressionMu.RLock()
	campaignHasIt := unlockedUltimates[entity.CharNecromante]
	progressionMu.RUnlock()
	if campaignHasIt {
		t.Fatal("a concessao da corrida vazou para o conjunto da campanha")
	}
}

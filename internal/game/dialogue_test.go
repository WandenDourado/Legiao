package game

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/network"
)

// partyIsFalling e a condicao do gatilho on_last_stand. Estes testes travam a
// correcao do defeito relatado: o climax disparava em QUALQUER horda de um
// mapa com corrida, bastando o grupo cair de vida; agora ele so pode disparar
// dentro da janela que o mapa declara (internal/network/climax_window.go).

func resetPartyIsFallingState(t *testing.T) {
	t.Helper()
	network.RemotePlayersMutex.Lock()
	network.RemotePlayers = map[string]network.PlayerState{}
	network.RemotePlayersMutex.Unlock()
	network.SetWaveState(network.WaveState{})
}

func setPlayer(id string, health, max float32, dead bool) {
	network.UpdatePlayerState(network.PlayerState{
		PlayerID: id, Health: health, MaxHealth: max, IsDead: dead,
	})
}

func TestPartyIsFallingDoesNotFireOnTheWrongHorde(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	// world_02 so arma na horda 3 de 3. Todo mundo a 10% na horda 1 nao pode
	// disparar a cena: era exatamente o defeito relatado.
	network.SetWaveState(network.WaveState{Total: 3, Index: 1, Phase: string(network.WavePhaseFighting)})
	setPlayer("p1", 10, 100, false)

	if partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("o climax disparou na horda 1 de 3; so a horda 3 deveria disparar")
	}
}

func TestPartyIsFallingFiresOnTheDeclaredHorde(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	setPlayer("p1", 10, 100, false)

	if !partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("o climax nao disparou na horda declarada (3 de 3), com o grupo caindo")
	}
}

// A regra e do GRUPO somado (28/09/2026): o grupo inteiro saudavel nao
// dispara, mas um personagem quase cheio com o resto caido dispara — o que a
// regra antiga ("todo mundo de pe abaixo de 30%") segurava para sempre.
func TestPartyIsFallingNeedsTheSummedPartyBelowTheThreshold(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	for _, id := range []string{"p1", "p2", "p3", "p4", "p5"} {
		setPlayer(id, 200, 200, false)
	}
	if partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("o climax disparou com o grupo inteiro de vida cheia")
	}

	// Dois cheios e tres caidos: 40%, ainda uma luta em pe.
	setPlayer("p3", 0, 200, true)
	setPlayer("p4", 0, 200, true)
	setPlayer("p5", 0, 200, true)
	if partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("o climax disparou com 40% da vida do grupo")
	}

	// Um quase cheio e quatro caidos: 18%. Era o caso que a regra antiga
	// segurava ate o ultimo morrer.
	setPlayer("p1", 180, 200, false)
	setPlayer("p2", 0, 200, true)
	if !partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("um personagem quase cheio com o resto do grupo caido deveria disparar o climax")
	}
}

// Sem ninguem com vida maxima conhecida nao ha grupo para medir.
func TestPartyHealthShareIgnoresUnknownMaxHealth(t *testing.T) {
	share, ok := partyHealthShare(map[string]network.PlayerState{
		"a": {Health: 50, MaxHealth: 100},
		"b": {Health: 0, MaxHealth: 0},
	})
	if !ok || share != 0.5 {
		t.Errorf("share %.2f ok %v; esperado 0,5 ignorando quem nao tem MaxHealth", share, ok)
	}
	if _, ok := partyHealthShare(map[string]network.PlayerState{"b": {}}); ok {
		t.Error("grupo sem MaxHealth nenhum nao pode ser medido")
	}
}

func TestPartyIsFallingCountsADeadPlayerAsBelowTheThreshold(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	setPlayer("p1", 10, 100, false)
	setPlayer("p2", 0, 100, true)

	if !partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("um jogador morto deveria contar como abaixo do limiar, nao bloquear o climax")
	}
}

func TestPartyIsFallingNeverFiresWithoutADeclaredWindow(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	// world_01 nao tem cena de climax nenhuma. Mesmo com WaveState alto e o
	// grupo inteiro caido, a cena nao pode tocar.
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	setPlayer("p1", 5, 100, false)

	if partyIsFalling("assets/maps/world_01.json", nil) {
		t.Fatal("um mapa sem janela de climax declarada disparou on_last_stand")
	}
}

func TestPartyIsFallingEmptyPartyNeverFires(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})

	if partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("um grupo vazio nao pode contar como grupo caindo")
	}
}

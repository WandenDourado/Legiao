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

func TestPartyIsFallingNeedsEveryoneBelowTheThreshold(t *testing.T) {
	resetPartyIsFallingState(t)
	defer resetPartyIsFallingState(t)
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	setPlayer("p1", 10, 100, false)
	// Um unico jogador ACIMA de 30% impede o disparo — nao e maioria, e TODOS.
	setPlayer("p2", 50, 100, false)

	if partyIsFalling("assets/maps/world_02.json", nil) {
		t.Fatal("o climax disparou com um jogador acima de 30% de vida")
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

package game

// A cena de fim de fase (on_waves_cleared) do corredor dos canhoes.
//
// O world_06 nao tem horda: ate 10/09/2026 o gatilho so perguntava pelo
// WaveState, que ali fica em Total 0 para sempre, e a cena escrita para
// "depois de destruir os canhoes" nunca tocaria. Estes testes guardam os tres
// lados da resposta: corredor (canhoes), mapa de hordas e mapa quieto.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestStageFightOverWaitsForBothCannons(t *testing.T) {
	network.SetWaveState(network.WaveState{})
	// Os dois postos do world_06, nas posicoes reais do mapa.
	oeste := rl.NewVector2(896, 768)
	leste := rl.NewVector2(1664, 768)
	network.SetCannonPosts([]tilemap.SpawnPoint{
		{Name: "enemy_cannon_oeste", Position: oeste},
		{Name: "enemy_cannon_leste", Position: leste},
	})
	defer network.SetCannonPosts(nil)

	if stageFightOver() {
		t.Fatal("os dois canhoes estao de pe e a cena de fim de fase ja tocaria")
	}
	network.SetCannonHealth(oeste, 0)
	if stageFightOver() {
		t.Error("um canhao caiu e a cena tocaria; a fase pede os DOIS")
	}
	network.SetCannonHealth(leste, 0)
	if !stageFightOver() {
		t.Error("os dois canhoes cairam e a cena de depois dos canhoes nao toca")
	}
	// Reinicio de fase: os postos voltam de pe, e a cena da corrida nova so
	// pode tocar quando eles cairem de novo.
	network.ResetCannonBars()
	if stageFightOver() {
		t.Error("depois do reinicio os canhoes estao de pe e a cena ainda tocaria")
	}
}

func TestStageFightOverOnAHordeMap(t *testing.T) {
	network.SetCannonPosts(nil)
	defer network.SetWaveState(network.WaveState{})

	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseFighting)})
	if stageFightOver() {
		t.Error("a ultima horda ainda esta em campo e a cena de fim de fase tocaria")
	}
	network.SetWaveState(network.WaveState{Total: 3, Index: 3, Phase: string(network.WavePhaseCleared)})
	if !stageFightOver() {
		t.Error("a ultima horda morreu e a cena de fim de fase nao toca")
	}
}

func TestStageFightOverQuietMapNeverFires(t *testing.T) {
	network.SetCannonPosts(nil)
	network.SetWaveState(network.WaveState{})
	if stageFightOver() {
		t.Error("um mapa sem horda e sem canhao disparou a cena de fim de fase")
	}
}

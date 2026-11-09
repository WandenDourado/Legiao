package game

// A SAIDA DA FASE 6 E CALAR OS CANHOES.
//
// O corredor final nao tem horda para limpar, entao ate 28/08/2026 o portal
// dele estava aberto desde o primeiro quadro: o que impedia alcanca-lo era so
// o bombardeio, e dava para correr por baixo do fogo com sorte e pular a fase
// inteira. Agora a saida e a recompensa por derrubar os dois postos, como em
// toda outra fase a saida e a recompensa por limpar a horda.
//
// Este teste guarda os dois lados: fechado com canhao de pe, aberto quando o
// ultimo cai, e INEXISTENTE em mapa que nao declara posto nenhum — a regra
// pergunta "ha canhao de pe?", nunca "que mapa e este?".

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestPortalWaitsForBothCannons(t *testing.T) {
	// Os dois postos do world_06, nas posicoes reais do mapa.
	oeste := rl.NewVector2(896, 768)
	leste := rl.NewVector2(1664, 768)
	network.SetCannonPosts([]tilemap.SpawnPoint{
		{Name: "enemy_cannon_oeste", Position: oeste},
		{Name: "enemy_cannon_leste", Position: leste},
	})
	// Mapa seguinte nao pode herdar a trava: limpa ao sair.
	defer network.SetCannonPosts(nil)

	if PortalsUnlocked() {
		t.Fatal("os dois canhoes estao de pe e o portal ja abriu: da para pular o corredor")
	}

	network.SetCannonHealth(oeste, 0)
	if PortalsUnlocked() {
		t.Error("um canhao caiu e o portal abriu; a fase pede os DOIS")
	}

	network.SetCannonHealth(leste, 0)
	if !PortalsUnlocked() {
		t.Error("os dois canhoes cairam e o portal continua fechado: a fase ficou sem saida")
	}
}

// Um reinicio de fase repoe os postos de pe (ResetCannonBars, chamado pelo
// host em RestoreCannons e pelo cliente em applyStageReset). O portal tem de
// fechar junto — senao a segunda tentativa comeca com a saida aberta, que e
// exatamente a trava que climax_pending.go ja existe para impedir no mapa 3.
func TestStageResetShutsThePortalAgain(t *testing.T) {
	pos := rl.NewVector2(896, 768)
	network.SetCannonPosts([]tilemap.SpawnPoint{{Name: "enemy_cannon_oeste", Position: pos}})
	defer network.SetCannonPosts(nil)

	network.SetCannonHealth(pos, 0)
	if !PortalsUnlocked() {
		t.Fatal("o unico canhao caiu e o portal nao abriu")
	}

	network.ResetCannonBars()
	if PortalsUnlocked() {
		t.Error("a fase reiniciou com o canhao de pe e o portal continuou aberto")
	}
}

// Mapa sem canhao nao paga nada por esta regra existir.
func TestMapWithoutCannonsIsNotGatedByThem(t *testing.T) {
	network.SetCannonPosts(nil)
	if network.CannonsStanding() {
		t.Error("mapa sem posto de canhao respondeu que ha canhao de pe")
	}
}

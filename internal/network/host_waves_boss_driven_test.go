package network

// Testes do item 6 (31/08/2026, doc/plan_world07_arena_tuning.md): a horda
// BossDriven perde o relogio proprio e passa a esperar o pedido da chefe, mas
// a PRIMEIRA leva ainda tem de sair sozinha ao entrar na horda — a
// "observacao" do §6.1 do plano, que ja acontecia antes e nao podia quebrar.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TestBossDrivenWaveOnlyReleasesOnRequest prova as duas metades do contrato:
// sem ReleaseBatchNow, nenhum dt (por maior que seja) solta uma leva nova
// depois da primeira; com ele, sai EXATAMENTE uma.
func TestBossDrivenWaveOnlyReleasesOnRequest(t *testing.T) {
	defs := []WaveDef{{
		Name:          "Teste",
		Composition:   map[entity.EnemyType]int{entity.EnemyTypeBasic: 8},
		MaxConcurrent: 20,
		BatchSize:     4,
		SpawnInterval: 30,
		BossDriven:    true,
	}}
	wr := NewWaveRunner(nil, defs)
	spawned := 0
	spawn := func(entity.EnemyType, rl.Vector2) { spawned++ }

	// Fecha o intervalo de abertura (WaveBreakDuration) e arma a horda.
	wr.Update(WaveBreakDuration+0.01, 0, nil, spawn)
	// A primeira leva sai sozinha no proximo quadro, sem pedido nenhum.
	wr.Update(0.016, 0, nil, spawn)
	if spawned != 4 {
		t.Fatalf("primeira leva: spawned=%d, esperado 4 (ela sai sozinha ao entrar "+
			"na horda, ver §6.1)", spawned)
	}

	// Sem relogio proprio: nenhum dt solta a proxima leva sozinho.
	for i := 0; i < 5; i++ {
		wr.Update(1000, 0, nil, spawn)
	}
	if spawned != 4 {
		t.Fatalf("uma horda BossDriven soltou leva sem pedido; spawned=%d, esperado "+
			"continuar em 4", spawned)
	}

	// UM pedido solta EXATAMENTE uma leva.
	wr.ReleaseBatchNow()
	wr.Update(0.016, 0, nil, spawn)
	if spawned != 8 {
		t.Fatalf("depois de ReleaseBatchNow: spawned=%d, esperado 8", spawned)
	}

	// A composicao ja esvaziou; um pedido extra nao tem mais nada para soltar.
	wr.ReleaseBatchNow()
	wr.Update(0.016, 0, nil, spawn)
	if spawned != 8 {
		t.Fatalf("pedido sem pendencia soltou mais alguma coisa; spawned=%d", spawned)
	}
}

// TestWorld07FirstWaveArrivesWithoutTheBoss prova a "observacao" do §6.1 na
// corrida DE VERDADE da arena: o cerco chega ao entrar no mapa mesmo que
// ninguem nunca chame ReleaseBatchNow — a chefe (network/host_boss.go) nao
// participa disto, so das levas SEGUINTES.
func TestWorld07FirstWaveArrivesWithoutTheBoss(t *testing.T) {
	defs := WaveDefsFor("assets/maps/world_07.json")
	if len(defs) == 0 {
		t.Fatal("world_07 sem corrida declarada")
	}
	if !defs[0].BossDriven {
		t.Fatal("a horda do cerco precisa ser BossDriven; o teste nao mede o que pensa")
	}
	wr := NewWaveRunner(nil, defs)
	spawned := 0
	spawn := func(entity.EnemyType, rl.Vector2) { spawned++ }

	wr.Update(WaveBreakDuration+0.01, 0, nil, spawn)
	wr.Update(0.016, 0, nil, spawn)
	if spawned == 0 {
		t.Fatal("a primeira leva do cerco da arena nao saiu sozinha ao entrar no " +
			"mapa; ela nao pode depender da chefe (o pedido original do Gui, §6.1)")
	}
}

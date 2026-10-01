package network

import "testing"

// resetReconcileTestState puts every package var this file touches back to a
// known baseline before each test — they are process-wide, and these tests
// exercise the exact machinery a real client/host pair shares.
func resetReconcileTestState(t *testing.T) {
	t.Helper()
	resetStageReconcileBaseline()
	GameOver = false
	// Drain any reset a previous test left queued.
	ConsumeLocalReset()
}

// TestReconcileFirstSnapshotSetsBaselineWithoutActing is the case the plan
// calls out explicitly: the very first StateUpdatePayload after a fresh join
// has nothing to compare against, and must never be misread as a missed
// reset just because lastSeenStageGen starts at its zero value.
func TestReconcileFirstSnapshotSetsBaselineWithoutActing(t *testing.T) {
	resetReconcileTestState(t)
	c := &Client{}

	state := StateUpdatePayload{StageGen: 7, GameOver: false, Players: []PlayerState{{PlayerID: "p1"}}}
	c.reconcileStageState(state, map[string]PlayerState{"p1": {PlayerID: "p1"}})

	if GameOver {
		t.Error("primeiro snapshot nao pode mexer em GameOver")
	}
	if _, queued := ConsumeLocalReset(); queued {
		t.Error("primeiro snapshot nao pode enfileirar reset nenhum")
	}
}

// TestReconcileAppliesMissedResetWhenStageGenAdvances e o cenario do item 2
// (plan_correcoes_30-08-2026.md): cliente com GameOver=true preso numa
// geracao velha recebe um snapshot de geracao mais nova (o MsgReset que a
// causou nunca chegou) e se corrige sozinho — GameOver volta a false e um
// reset local e enfileirado na posicao que o host reportou para ESTE
// jogador, no mesmo snapshot.
func TestReconcileAppliesMissedResetWhenStageGenAdvances(t *testing.T) {
	resetReconcileTestState(t)
	LocalPlayerID = "p1"
	t.Cleanup(func() { LocalPlayerID = "" })

	c := &Client{}
	// Estabelece a baseline na geracao 3.
	c.reconcileStageState(StateUpdatePayload{StageGen: 3}, nil)

	// O cliente ficou preso mostrando Game Over de uma tentativa anterior.
	GameOver = true

	// A geracao avancou para 4 sem que este cliente tivesse visto o
	// MsgReset — exatamente o defeito relatado.
	players := map[string]PlayerState{"p1": {PlayerID: "p1", X: 555, Y: 666}}
	c.reconcileStageState(StateUpdatePayload{StageGen: 4, GameOver: false, Players: []PlayerState{players["p1"]}}, players)

	if GameOver {
		t.Error("GameOver deveria ter sido limpo pelo reset recuperado")
	}
	spawn, queued := ConsumeLocalReset()
	if !queued {
		t.Fatal("um reset local deveria ter sido enfileirado")
	}
	if spawn.X != 555 || spawn.Y != 666 {
		t.Errorf("spawn recuperado = (%.0f,%.0f), esperado (555,666) — a posicao que o host reportou para este jogador neste snapshot", spawn.X, spawn.Y)
	}
}

// TestReconcileAdoptsHostGameOverWithoutStageChange cobre a metade menor do
// mesmo defeito: a fase NAO reiniciou (mesma geracao), so o MsgGameOver em si
// se perdeu — por exemplo um cliente que reconectou bem no instante em que a
// derrota foi anunciada. Sem geracao nova nao ha nada para "recuperar" alem
// da propria flag.
func TestReconcileAdoptsHostGameOverWithoutStageChange(t *testing.T) {
	resetReconcileTestState(t)
	c := &Client{}
	c.reconcileStageState(StateUpdatePayload{StageGen: 9, GameOver: false}, nil)

	c.reconcileStageState(StateUpdatePayload{StageGen: 9, GameOver: true}, nil)

	if !GameOver {
		t.Error("GameOver deveria ter sido adotado do host")
	}
	if _, queued := ConsumeLocalReset(); queued {
		t.Error("sem mudanca de geracao, nenhum reset deveria ter sido enfileirado")
	}
}

// TestReconcileDoesNothingWhenAlreadyInSync garante que o mecanismo fica
// quieto no caso comum: geracao igual, GameOver igual. Nada de reset,
// nada de log de recuperacao a cada snapshot de 20 Hz.
func TestReconcileDoesNothingWhenAlreadyInSync(t *testing.T) {
	resetReconcileTestState(t)
	c := &Client{}
	c.reconcileStageState(StateUpdatePayload{StageGen: 2, GameOver: false}, nil)

	c.reconcileStageState(StateUpdatePayload{StageGen: 2, GameOver: false}, nil)

	if GameOver {
		t.Error("GameOver nao deveria ter mudado")
	}
	if _, queued := ConsumeLocalReset(); queued {
		t.Error("nenhum reset deveria ter sido enfileirado quando ja esta em dia")
	}
}

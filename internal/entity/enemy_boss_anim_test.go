package entity

import "testing"

// TestBossDodgeWindowMatchesThornTelegraph trava o acoplamento que o item 1
// (31/08/2026, doc/plan_world07_arena_tuning.md) quase quebrou:
// `bossDodgeWindow` (aqui) tem de ser sempre `skill.ThornTelegraph` (em
// boss_thorn.go) MENOS 0,30 — a janela de desvio nao pode dessincronizar do
// telegrafo do espinhao, senao a chefe crava as patas no chao antes ou depois
// de o circulo do dano ter rasgado.
//
// O numero de `skill.ThornTelegraph` fica REPETIDO aqui como literal, e nao
// importado, porque `skill` ja importa `entity` — importar `skill` a partir
// daqui fecharia um ciclo. E por isso que este comentario e o de
// `ThornTelegraph` em boss_thorn.go apontam um para o outro: mexer num sem
// lembrar do outro tem de quebrar ESTE teste, em vez de so desalinhar a
// animacao em jogo sem nenhum aviso.
func TestBossDodgeWindowMatchesThornTelegraph(t *testing.T) {
	const thornTelegraph float32 = 1.2 // == skill.ThornTelegraph; mantenha em sincronia
	want := thornTelegraph - 0.30
	if bossDodgeWindow != want {
		t.Errorf("bossDodgeWindow = %.2f; esperado ThornTelegraph(%.2f) - 0.30 = %.2f",
			bossDodgeWindow, thornTelegraph, want)
	}
}

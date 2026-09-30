package entity

import "testing"

func nearly(a, b float32) bool { return within(a, b, 0.5) }

// within compara com a tolerancia que o numero pede: meio pixel serve para
// velocidade, mas um FrameTime vive na terceira casa decimal e passaria por
// qualquer coisa.
func within(a, b, tol float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}

// TestAttackReachSurvivesGameSpeed e a invariante mais importante do ritmo
// novo: o tiro fica mais rapido, o alcance NAO muda.
//
// Os numeros sao os autorais, escritos a mao: 700 x 1,6 e 420 x 2,0. Se um dia
// a velocidade subir sem o tempo de vida descer, e aqui que aparece — e o
// sintoma em jogo seria o Arqueiro acertando de 1512 px, com o contrato
// numerico das supremas (a Flecha Celestial se mede contra o Orc) mudando
// sozinho, em silencio.
func TestAttackReachSurvivesGameSpeed(t *testing.T) {
	cases := []struct {
		char CharacterType
		want float32
	}{
		{CharArqueiro, 700 * 1.6},
		{CharMago, 420 * 2.0},
		{CharSacerdotisa, 380 * 2.0},
		{CharNecromante, 400 * 2.0},
	}
	for _, c := range cases {
		if got := AttackReach(c.char); !nearly(got, c.want) {
			t.Errorf("AttackReach(%s) = %.1f, esperado %.1f — velocidade e tempo de vida sairam de sincronia",
				c.char, got, c.want)
		}
	}
	if got := LongestAttackReach(); !nearly(got, 1120) {
		t.Errorf("LongestAttackReach = %.1f, esperado 1120 (o raio de ameaca do guarda depende disto)", got)
	}
}

// TestGameSpeedScalesTheCast prova que o fator chegou onde tinha de chegar, e
// com a PROPORCAO preservada: o lobo continua sendo o dobro do orc, que era a
// relacao que a queixa dos testadores explicitamente nao pedia para mudar.
func TestGameSpeedScalesTheCast(t *testing.T) {
	if !nearly(PlayerSpeed, 200*GameSpeed) {
		t.Errorf("PlayerSpeed = %.1f, esperado %.1f", float32(PlayerSpeed), 200*GameSpeed)
	}
	cases := []struct {
		enemy EnemyType
		base  float32
	}{
		{EnemyTypeBasic, 100},
		{EnemyTypeFast, 240},
		{EnemyTypeGarrison, 130},
	}
	for _, c := range cases {
		if got := GetEnemyDef(c.enemy).Speed; !nearly(got, c.base*GameSpeed) {
			t.Errorf("Speed(%s) = %.1f, esperado %.1f", c.enemy, got, c.base*GameSpeed)
		}
	}
	wolf := GetEnemyDef(EnemyTypeFast).Speed
	orc := GetEnemyDef(EnemyTypeGarrison).Speed
	if !nearly(wolf/orc, 240.0/130.0) {
		t.Errorf("lobo/orc = %.3f, esperado %.3f — o fator global nao pode mexer nas proporcoes",
			wolf/orc, 240.0/130.0)
	}
}

// TestStandingEnemiesStayPut: "parado" e uma decisao de design (a Sentinela
// dispara do posto, a Imperatriz luta onde esta), nao um numero pequeno que o
// fator possa inflar.
func TestStandingEnemiesStayPut(t *testing.T) {
	for _, et := range []EnemyType{EnemyTypeCastleSentry, EnemyTypeDarkLady} {
		if got := GetEnemyDef(et).Speed; got != 0 {
			t.Errorf("Speed(%s) = %.1f, esperado 0", et, got)
		}
	}
}

// TestWalkCycleKeepsUpWithTheFeet protege o pe do orc.
//
// O FrameTime do AnimWalk dele nao e gosto: e a unica solucao de
// `2 * 63,5 * RenderScale / (Speed * 10)`, medida na arte. Subir a Speed sem
// encurtar o ciclo devolve o monstro patinando, que foi o defeito que aquele
// numero nasceu para consertar.
func TestWalkCycleKeepsUpWithTheFeet(t *testing.T) {
	walk := GetEnemyDef(EnemyTypeGarrison).AnimDef(AnimWalk)
	if !within(walk.FrameTime*GameSpeed, 0.156, 0.001) {
		t.Errorf("FrameTime do AnimWalk do orc = %.4f, esperado %.4f",
			walk.FrameTime, 0.156/GameSpeed)
	}
	// O idle nao anda, entao a duracao autorada dele fica de pe.
	if idle := GetEnemyDef(EnemyTypeGarrison).AnimDef(AnimIdle); !within(idle.FrameTime, 0.16, 0.001) {
		t.Errorf("FrameTime do AnimIdle do orc = %.4f, esperado 0.16 (parado nao acelera)", idle.FrameTime)
	}
	// E o ciclo do personagem tambem encurtou.
	if ft := GetCharacter(CharMago).FrameTime; !within(ft*GameSpeed, 0.12, 0.001) {
		t.Errorf("FrameTime do Mago = %.4f, esperado %.4f", ft, 0.12/GameSpeed)
	}
}

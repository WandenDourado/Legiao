package entity

// A CURA POR TIRO DA SACERDOTISA E UM NUMERO DE MIRA, e numero de mira some em
// silencio: ninguem olha `HolyAttackRadius = 13` e enxerga "ela precisa acertar
// o aliado em cheio". O defeito so aparece em jogo, como a sensacao de que a
// curandeira nao cura — que foi exatamente o relato do Gui em 31/08/2026.
//
// O teste amarra o raio a uma regua que existe fora dele: o corpo do jogador.

import "testing"

// holyHealTolerance repete a conta que o host faz em checkHolyProjectileHeals
// (`raio do projetil + PlayerSize`). Duplicada aqui de proposito: se alguem
// mudar a formula la, este teste continua medindo a promessa antiga e a
// diferenca aparece como falha — que e o ponto.
func holyHealTolerance() float32 { return HolyAttackRadius + PlayerSize }

// A promessa do buff: o tiro que passa RASPANDO pelo aliado cura. Em numero,
// isso e perdoar pelo menos um corpo inteiro de erro de mira — o alvo se move,
// quem atira tambem, e a horda nao espera.
func TestHolyBoltForgivesAWholeBodyOfAimError(t *testing.T) {
	tolerance := holyHealTolerance()
	if tolerance < 2*PlayerSize {
		t.Errorf("a cura tolera %.0f px de erro e o corpo do jogador tem %.0f de "+
			"raio: a Sacerdotisa voltou a precisar de tiro certeiro para curar",
			tolerance, PlayerSize)
	}
}

// O outro lado da faixa: cura por tiro e MIRA, nao aura. Se a tolerancia
// crescer a ponto de pegar qualquer aliado na direcao geral, o ataque basico
// dela passa a fazer o trabalho do Santuario de graca e sem recarga.
func TestHolyBoltHealingStaysAimAndNotAnAura(t *testing.T) {
	tolerance := holyHealTolerance()
	if tolerance > 4*PlayerSize {
		t.Errorf("a cura tolera %.0f px (mais de quatro raios de corpo): o tiro "+
			"virou aura, e o Santuario perdeu a razao de existir", tolerance)
	}
}

// O que a tela mostra e o que acerta: o corpo desenhado do raio sagrado sai do
// mesmo HolyAttackRadius que resolve dano e cura (drawHolyProjectile usa
// fracoes dele). Uma area de cura maior que o desenho seria mentira visual, e
// e por isso que o buff foi no raio do projetil e nao num raio de cura
// separado (decisao do Gui, 31/08/2026).
func TestHolyBoltIsBigEnoughToBeSeen(t *testing.T) {
	if HolyAttackRadius <= ArrowAttackRadius {
		t.Errorf("o raio sagrado (%.0f) nao e mais largo que uma flecha (%.0f); "+
			"o jogador nao tem como ler na tela que este tiro cura area",
			HolyAttackRadius, ArrowAttackRadius)
	}
}

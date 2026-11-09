package entity

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// O SETOR RESPONDE DUAS PERGUNTAS DIFERENTES, e antes so servia bem a uma.
//
// Ele decide QUEM e problema deste monstro (aquisicao) e ate onde ele vai atras
// (manutencao). Numa linha de barricada as duas coincidem, porque a barricada e
// uma parede. Num corredor aberto nao coincidem: o jogador cruza a divisa so
// por estar caminhando, e o guarda largava a briga no mesmo quadro — foi o
// relato em jogo do mapa 4, "quando o personagem se afasta um pouco, ele ja
// volta para o posto".
//
// `Slack` e quanto o setor vale a MAIS para uma perseguicao ja em curso. Zero
// devolve a regra antiga, que e a que o mapa 3 quer.
func slackGuard(slack float32) Guard {
	post := rl.NewVector2(1000, 1000)
	return Guard{
		Post:      post,
		Territory: rl.NewRectangle(0, 0, 2000, 2000),
		Radius:    2600,
		Slack:     slack,
	}
}

func TestChaseSurvivesLeavingSectorWithinSlack(t *testing.T) {
	g := slackGuard(1400)
	justOutside := rl.NewVector2(1000, 2100) // 100 px alem da divisa

	if g.wants(justOutside, false) {
		t.Error("adquirir fora do setor: o alvo do outro lado da divisa nao e " +
			"problema deste monstro, por perto que esteja")
	}
	if !g.wants(justOutside, true) {
		t.Error("perseguicao em curso morreu a 100 px da divisa, com 1400 de " +
			"folga declarada — e o defeito que a folga existe para corrigir")
	}
}

func TestChaseStillEndsBeyondSlack(t *testing.T) {
	g := slackGuard(1400)
	farOutside := rl.NewVector2(1000, 3500) // 1500 px alem da divisa

	if g.wants(farOutside, true) {
		t.Error("a folga virou coleira infinita; o grupo puxaria a guarnicao " +
			"mapa afora, que e o exploit que o setor existe para impedir")
	}
}

func TestZeroSlackKeepsTheOldRule(t *testing.T) {
	g := slackGuard(0)
	justOutside := rl.NewVector2(1000, 2100)

	if g.wants(justOutside, true) {
		t.Error("sem folga declarada, cruzar a divisa tem de quebrar a " +
			"perseguicao no mesmo quadro — e a regra do mapa 3, e mapa que " +
			"nao declara nada nao pode mudar de comportamento")
	}
}

// O raio tambem e do mapa: 1700 px cobre uma faixa de 7680 de largura de um
// jeito e uma de 3328 de outro. Este teste so fixa que o raio e respeitado como
// dado, nao como constante.
func TestAcquisitionUsesTheDeclaredRadius(t *testing.T) {
	// O setor tem de ser largo o bastante para NAO ser o que reprova aqui,
	// senao o teste passa pelo motivo errado — foi o que aconteceu na primeira
	// versao, com o alvo a 2300 px caindo fora de um retangulo de 2000.
	sector := rl.NewRectangle(0, 0, 6000, 2000)
	target := rl.NewVector2(1000+2300, 1000) // 2300 px do posto, dentro do setor

	near := slackGuard(0)
	near.Territory, near.Radius = sector, 1700
	if near.wants(target, false) {
		t.Error("adquiriu alem do proprio raio")
	}
	wide := slackGuard(0)
	wide.Territory, wide.Radius = sector, 2600
	if !wide.wants(target, false) {
		t.Error("nao adquiriu dentro do raio declarado pelo territorio")
	}
}

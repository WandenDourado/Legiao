package entity

// O duelo que os stats do orc existem para vencer: CINCO orcs contra a Legiao
// Espectral do Necromante. Os trinta espectros tem de morrer; os orcs nao
// precisam sobreviver.
//
// Este teste existe porque a conta e fragil de um jeito particular: ela nao
// depende de UM numero, depende da relacao entre quatro (vida e dano do orc,
// vida e cadencia do espectro). Mexer em qualquer um sozinho quebra o duelo sem
// quebrar mais nada — nao ha crash, nao ha erro de compilacao, e o defeito so
// apareceria com alguem jogando a fase 3 ate o fim e vendo a ultimate ser
// desperdicada.
//
// A SIMULACAO ESPELHA `skill.StepLegions`, e isso ja custou uma versao errada
// deste arquivo. O primeiro modelo supunha que o orc acertava um numero fixo de
// espectros por golpe, como um cleave; o codigo real faz outra coisa: cada
// espectro engajado tem o PROPRIO `hurtTimer`, entao o inimigo revida contra
// todos ao mesmo tempo. Com o modelo errado a resposta foi "dano 60", que no
// codigo de verdade faz UM orc limpar a legiao inteira em quatro quadros.

import "testing"

// Copia de internal/skill/specter.go. Se divergir, TestSpecterNumbersUnchanged
// avisa antes de o duelo dar um resultado que nao quer dizer nada.
const (
	testLegionCount        = 30
	testSpecterMaxHealth   = 60.0
	testSpecterDamage      = 11.0
	testSpecterAttackEvery = 0.18
)

// maxSpectersOnOneOrc e quantos espectros cabem em volta de um orc.
//
// Geometria, nao chute: a circunferencia de contato tem raio
// EnemyOrcRadius + SpecterRadius = 45 + 15 = 60, ou 377 px; cada espectro
// ocupa ~2 x 15 = 30 px de arco. Cabem ~12, mas eles chegam de um lado so e na
// pratica se amontoam em menos. Oito e conservador PARA O ORC — supor mais
// espectros mordendo do que a realidade permite faz o teste exigir stats mais
// duros do que o necessario, que e o lado seguro de errar.
const maxSpectersOnOneOrc = 8

// duelOrcsVersusLegion simula o duelo e devolve quanto durou, quantos orcs
// ficaram de pe e quantos espectros sobraram.
//
// Duas fidelidades que importam mais que o resto:
//
//  1. O revide e POR ESPECTRO, contra todos os engajados, e nao um alvo por
//     golpe. E o que StepLegions faz.
//  2. `hurtTimer` nasce ZERADO, entao o primeiro revide sai no quadro em que o
//     espectro encosta — nao depois de um cooldown. E por isso que dano igual
//     ou maior que a vida do espectro quebra a fase inteira.
//
// O deslocamento e ignorado de proposito: o espectro corre a 340 contra os 130
// do orc, entao o tempo de aproximacao e ruido perto dos segundos que o duelo
// dura, e inclui-lo so favoreceria o orc.
func duelOrcsVersusLegion(orcs int) (seconds float64, orcsLeft, spectersLeft int) {
	def := GetEnemyDef(EnemyTypeGarrison)

	orcHP := make([]float64, orcs)
	for i := range orcHP {
		orcHP[i] = float64(def.Health)
	}
	specterHP := make([]float64, testLegionCount)
	biteTimer := make([]float64, testLegionCount)
	hurtTimer := make([]float64, testLegionCount)
	for i := range specterHP {
		specterHP[i] = testSpecterMaxHealth
	}

	const dt = 0.02
	for t := 0.0; t < 180; t += dt {
		var liveOrcs, liveSpecters []int
		for i, h := range orcHP {
			if h > 0 {
				liveOrcs = append(liveOrcs, i)
			}
		}
		for i, h := range specterHP {
			if h > 0 {
				liveSpecters = append(liveSpecters, i)
			}
		}
		if len(liveOrcs) == 0 || len(liveSpecters) == 0 {
			return t, len(liveOrcs), len(liveSpecters)
		}

		// Cada espectro escolhe um orc; cada orc so aceita os que cabem.
		engagedWith := make(map[int]int, len(liveSpecters))
		count := make(map[int]int, len(liveOrcs))
		for k, si := range liveSpecters {
			oi := liveOrcs[k%len(liveOrcs)]
			if count[oi] < maxSpectersOnOneOrc {
				count[oi]++
				engagedWith[si] = oi
			}
		}

		for si, oi := range engagedWith {
			biteTimer[si] -= dt
			if biteTimer[si] <= 0 {
				biteTimer[si] = testSpecterAttackEvery
				orcHP[oi] -= testSpecterDamage
			}
			hurtTimer[si] -= dt
			if hurtTimer[si] <= 0 {
				hurtTimer[si] = float64(def.AttackCooldown)
				specterHP[si] -= float64(def.AttackDamage)
			}
		}
	}
	return 180, len(orcHP), len(specterHP)
}

// TestFiveOrcsBeatTheLegion e o alvo, dito pelo Gui: cinco orcs derrotam a
// ultimate do Necromante.
func TestFiveOrcsBeatTheLegion(t *testing.T) {
	seconds, orcsLeft, spectersLeft := duelOrcsVersusLegion(5)
	if spectersLeft != 0 {
		t.Errorf("cinco orcs deixaram %d espectro(s) vivos depois de %.1fs; "+
			"o alvo e limpar a legiao inteira", spectersLeft, seconds)
	}
	// Duracao tambem importa. Um duelo longo quer dizer que o orc virou parede
	// de vida em vez de contrapartida, e e o sinal de que alguem consertou o
	// teste engordando a vida em vez de olhar a relacao entre dano e cadencia.
	if seconds > 12 {
		t.Errorf("o duelo levou %.1fs; acima de ~12s o orc e parede de vida, "+
			"nao contrapartida", seconds)
	}
	t.Logf("legiao limpa em %.1fs, %d de 5 orcs de pe", seconds, orcsLeft)
}

// TestFourOrcsLoseToTheLegion e o outro lado do degrau: CINCO e o numero, e um
// numero so e um numero se quatro nao servirem.
func TestFourOrcsLoseToTheLegion(t *testing.T) {
	_, _, spectersLeft := duelOrcsVersusLegion(4)
	if spectersLeft == 0 {
		t.Error("quatro orcs limparam a legiao; o alvo era cinco, e sem o " +
			"degrau o numero perde o sentido")
	}
}

// TestOneOrcLosesToTheLegion e o piso, e ele ja foi violado uma vez.
//
// Com AttackDamage 60 — a vida cheia de um espectro — o revide imediato mata
// tudo que engaja, e um orc sozinho limpa os trinta. O teste existe para que
// essa tentacao (subir o dano ate o duelo "fechar") reprove em vez de passar.
func TestOneOrcLosesToTheLegion(t *testing.T) {
	_, orcsLeft, spectersLeft := duelOrcsVersusLegion(1)
	if orcsLeft != 0 || spectersLeft == 0 {
		t.Errorf("um orc sozinho terminou com %d orc e %d espectros; ele "+
			"deveria perder, ou a Legiao Espectral perde o sentido",
			orcsLeft, spectersLeft)
	}
}

// TestOrcDamageBelowSpecterHealth guarda a premissa que o duelo esconde.
//
// Se o golpe matar um espectro de uma vez, os testes de 4 e de 1 reprovam — mas
// so DEPOIS de alguem rodar a simulacao inteira e ler o resultado. Esta
// verificacao diz o porque em uma linha.
func TestOrcDamageBelowSpecterHealth(t *testing.T) {
	def := GetEnemyDef(EnemyTypeGarrison)
	if float64(def.AttackDamage) >= testSpecterMaxHealth {
		t.Errorf("o golpe do orc (%.0f) mata um espectro (%.0f) de uma vez; "+
			"em StepLegions o revide sai no quadro do contato, entao UM orc "+
			"limparia a legiao", def.AttackDamage, testSpecterMaxHealth)
	}
}

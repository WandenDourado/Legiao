package network

// Numeros de equilibrio que so significam alguma coisa EM RELACAO a outro
// numero. Um teste aqui nao defende o valor — ele defende a RELACAO, que e o
// que se quebra em silencio quando alguem afina um dos dois lados.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// playerMaxHealth e a vida de um personagem PADRAO — quatro dos cinco.
//
// Deixou de ser "qualquer personagem" em 27/08/2026: a Paladina tem 300
// (entity.MaxHealthFor). Os testes daqui que perguntam "quantos golpes para
// derrubar um personagem" continuam falando do padrao, e e por isso que lem a
// constante do pacote entity em vez de repetir o 100 — se o padrao mudar, a
// relacao que eles defendem acompanha.
const playerMaxHealth = entity.DefaultPlayerMaxHealth

// TRES ESFERAS DERRUBAM UM PERSONAGEM.
//
// A gargula bate de 1900, de fora do raio de 720 da Area Angelical: e o unico
// tipo de pressao que a suprema da fase 4 nao responde. Eram quatro acertos
// (25 de dano); no climax do mapa 4 a cadencia real e o tempo de viagem da
// esfera, nao o AttackCooldown, e quatro acertos davam ao grupo tempo de sobra
// para atravessar o saguao absorvendo — a fase nunca derrubava todo mundo
// abaixo de 30% ao mesmo tempo, e a cena do Arqueiro nunca abria.
func TestThreeSentryOrbsKillACharacter(t *testing.T) {
	dmg := entity.GetEnemyDef(entity.EnemyTypeCastleSentry).AttackDamage
	if dmg <= 0 {
		t.Fatal("a sentinela nao causa dano nenhum")
	}
	hits := int(playerMaxHealth / dmg)
	if playerMaxHealth > float32(hits)*dmg {
		hits++
	}
	if hits != 3 {
		t.Errorf("%d esferas para derrubar um personagem (%.0f de dano contra %.0f de vida); esperado 3",
			hits, dmg, playerMaxHealth)
	}
}

// CINCO GOLPES DE ESPADA DERRUBAM UM ORC.
//
// A regua que o Gui deu em 27/08/2026 para o ataque basico da Paladina, e a
// unica coisa que fixa SwordDamage: o orc e o inimigo mais duro que ela
// encontra cara a cara, e vence-lo tem de custar pouco mais de tres segundos
// na cadencia dela (1,5 golpe/s), nao treze.
//
// Os dois lados se afinam JUNTOS: subir a vida do orc sem subir a espada
// devolve a Paladina ao golpe de 20 pancadas, e o contrario faz dela a
// resposta a tudo. Quem mexer num vem parar aqui.
func TestFiveSwordSwingsKillAnOrc(t *testing.T) {
	orc := entity.GetEnemyDef(entity.EnemyTypeGarrison).Health
	if orc <= 0 {
		t.Fatal("o orc nao declara vida; a regua da espada nao existe mais")
	}
	swings := int(orc / skill.SwordDamage)
	if orc > float32(swings)*skill.SwordDamage {
		swings++
	}
	if swings != 5 {
		t.Errorf("%d golpes de espada (%.0f de dano) para os %.0f de vida do orc; "+
			"esperado 5", swings, skill.SwordDamage, orc)
	}
}

// A SUPREMA NAO PODE PIORAR O GOLPE.
//
// O Avatar dos Deuses TRANSFIGURA o ataque basico — espadao dourado, alcance
// maior, dano maior. O dano vem de uma constante separada
// (skill.SwordAvatarDamage), e uma constante separada e uma constante que
// esquece: quando SwordDamage subiu de 30 para 120, os 75 de outrora teriam
// feito a suprema BAIXAR o dano dela. Isso nao aparece lendo o codigo, so
// estranhando em jogo.
func TestTheAvatarNeverWeakensTheSword(t *testing.T) {
	if skill.SwordAvatarDamage <= skill.SwordDamage {
		t.Fatalf("a espada normal da %.0f e a do Avatar da %.0f: a suprema piora "+
			"o ataque basico", skill.SwordDamage, skill.SwordAvatarDamage)
	}
	ratio := skill.SwordAvatarDamage / skill.SwordDamage
	if ratio < 2 {
		t.Errorf("o Avatar multiplica o golpe por %.1f; abaixo de 2x a suprema "+
			"deixa de se notar no ataque", ratio)
	}
}

// UM METEORO TIRA 80% DA VIDA DO ORC.
//
// Duas pedras acertam de morte e uma sozinha nunca resolve. E o contrato que
// mantem a Chuva de Meteoros relevante contra o elenco pesado sem transformar
// a suprema num apagador de fase: o orc e a criatura que ela NAO compra de
// graca (ver entity/orc_legion_test.go para a mesma ideia com a Legiao).
func TestOneMeteorTakesFourFifthsOfAnOrc(t *testing.T) {
	orc := entity.GetEnemyDef(entity.EnemyTypeGarrison).Health
	want := orc * 0.8
	got := skill.MeteorImpactDamage
	if diff := got - want; diff > 1 || diff < -1 {
		t.Errorf("um meteoro tira %.0f; 80%% da vida do Orc (%.0f) e %.0f",
			got, orc, want)
	}
	if got >= orc {
		t.Errorf("um meteoro sozinho ja mata o Orc (%.0f de dano contra %.0f de vida)",
			got, orc)
	}
}

// UMA FLECHA CELESTIAL DERRUBA UM ORC.
//
// A contrapartida da Chuva de Meteoros: a suprema do Mago sorteia o mapa
// inteiro e NAO compra o orc de graca (uma pedra tira 80%); a do Arqueiro
// compra UM orc por flecha, mas so o que ela acertar — ela e apontada.
// Trocar as duas de lugar apaga a diferenca entre os dois personagens.
func TestOneCelestialArrowFellsAnOrc(t *testing.T) {
	orc := entity.GetEnemyDef(entity.EnemyTypeGarrison).Health
	got := skill.CelestialDamage
	if diff := got - orc; diff > 1 || diff < -1 {
		t.Errorf("uma flecha tira %.0f contra %.0f de vida do Orc; "+
			"o contrato e derrubar em UM acerto, sem sobra", got, orc)
	}
}

// A CHEFE DA ARENA E O CRONOMETRO DA FASE.
//
// A corrida do world_07 e `Endless` e so para quando ela cai
// (WaveDef.EndsWithBoss), entao a vida dela decide quanto tempo o grupo segura
// os dois portoes. Ela tem de aguentar bem mais que uma salva de suprema.
func TestTheArenaBossOutlastsASingleUltimate(t *testing.T) {
	boss := entity.GetEnemyDef(entity.EnemyTypeDarkLady).Health
	// As duas flechas do Arqueiro, o golpe mais direto que se pode apontar
	// para ela.
	volley := skill.CelestialDamage * float32(skill.CelestialCharges)
	if boss <= volley*4 {
		t.Errorf("a chefe tem %.0f de vida e uma ativacao de Flechas Celestiais tira %.0f; "+
			"a arena acabaria antes de comecar", boss, volley)
	}
}

// A LEGIAO ESPECTRAL NAO PODE MAIS SER O BOTAO DE VITORIA DA CHEFE.
//
// Ate 30/08/2026 a chefe so revidava (AttackDamage a cada AttackCooldown)
// contra quem estivesse colado nela, e isso deixava a Legiao Espectral
// (skill/legion_manager.go) tirar quase metade dos 30.000 de vida antigos
// numa unica ativacao — duas bastavam para acabar a arena, e uma ultimate que
// vence sozinha nao e mais uma decisao tatica. O anel de contato
// (skill/boss_contact.go, 31/08/2026, doc/plan_world07_arena_tuning.md) e a
// resposta: ele cobra dano de QUEM ENCOSTAR no corpo dela, espectro incluso,
// e agora e ELE — nao mais so o revide — a regua que decide quanto tempo um
// espectro sobrevive colado nela.
//
// Este teste substitui a regua antiga (Flechas Celestiais, acima) pela nova:
// uma ativacao da Legiao inteira tem de ficar entre 3% e 15% da barra —
// acima disso ela ainda e o botao de vitoria; abaixo disso ela deixa de valer
// a pena usar contra a chefe, e uma ultimate que o jogador aprende a nao usar
// e uma ultimate morta (ver a tabela de calibracao do item 5 no plano).
func TestTheArenaBossOutlastsTheSpectralLegion(t *testing.T) {
	darkLady := entity.GetEnemyDef(entity.EnemyTypeDarkLady)
	boss := darkLady.Health

	// Quanto tempo um espectro sobrevive colado no corpo dela: o revide
	// (AttackDamage a cada AttackCooldown) e o anel de contato (BossContactDPS)
	// cobrando ao mesmo tempo, os dois pelo mesmo alvo.
	retaliateDPS := darkLady.AttackDamage / darkLady.AttackCooldown
	specterDPSTaken := retaliateDPS + skill.BossContactDPS
	survival := skill.SpecterMaxHealth / specterDPSTaken

	// Quanto dano UM espectro tira da chefe nesse tempo, mordendo no ritmo
	// mais rapido do jogo (StepLegions, skill/legion_manager.go).
	specterBiteDPS := skill.SpecterDamage / skill.SpecterAttackEvery
	perSpecter := specterBiteDPS * survival
	legionDamage := perSpecter * float32(skill.LegionCount)

	pct := legionDamage / boss
	if pct > 0.15 {
		t.Errorf("a Legiao Espectral tira %.0f de %.0f de vida da chefe (%.0f%% da "+
			"barra); acima de 15%% ela volta a ser o botao de vitoria que o anel de "+
			"contato foi criado para tirar dela", legionDamage, boss, pct*100)
	}
	if pct < 0.03 {
		t.Errorf("a Legiao Espectral tira so %.0f%% da barra da chefe; abaixo de 3%% "+
			"a suprema deixa de valer a pena usar nela", pct*100)
	}
}

package network

// Numeros de equilibrio que so significam alguma coisa EM RELACAO a outro
// numero. Um teste aqui nao defende o valor — ele defende a RELACAO, que e o
// que se quebra em silencio quando alguem afina um dos dois lados.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// playerMaxHealth espelha o que entity.NewPlayer da a qualquer personagem.
const playerMaxHealth float32 = 100

// QUATRO ESFERAS DERRUBAM UM PERSONAGEM.
//
// A gargula bate de 1900, de fora do raio de 720 da Area Angelical: e o unico
// tipo de pressao que a suprema da fase 4 nao responde. Se ela precisar de oito
// acertos, o grupo simplesmente atravessa o corredor absorvendo — e a fase
// deixa de ter pergunta.
func TestFourSentryOrbsKillACharacter(t *testing.T) {
	dmg := entity.GetEnemyDef(entity.EnemyTypeCastleSentry).AttackDamage
	if dmg <= 0 {
		t.Fatal("a sentinela nao causa dano nenhum")
	}
	hits := int(playerMaxHealth / dmg)
	if playerMaxHealth > float32(hits)*dmg {
		hits++
	}
	if hits != 4 {
		t.Errorf("%d esferas para derrubar um personagem (%.0f de dano contra %.0f de vida); esperado 4",
			hits, dmg, playerMaxHealth)
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

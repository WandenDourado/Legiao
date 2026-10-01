package network

// O que o Escudo Sagrado passou a valer (27/08/2026) sao dois numeros de
// EQUILIBRIO, e numero de equilibrio some em silencio: ninguem percebe que a
// cura ficou irrelevante ou que o estouro virou limpa-tela olhando o codigo.
// Cada teste aqui amarra um deles a uma regua que existe fora dele.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// A regua do estouro e o LOBO: o Gui pediu "algo como 80% da vida de um lobo".
// O que esse numero compra e preciso — o estouro nao limpa a matilha em volta
// da Paladina, ele fere todo mundo e termina quem ja estava machucado. Se
// alguem mexer na vida do lobo (enemy_sprite.go) sem revisitar o estouro, o
// significado inverte sem aviso: a 100% da vida dele o escudo quebrado passa a
// matar cada lobo encostado, e quebrar o escudo vira lucro.
func TestShieldBurstIsFourFifthsOfAWolf(t *testing.T) {
	wolf := entity.GetEnemyDef(entity.EnemyTypeFast).Health
	if wolf <= 0 {
		t.Fatal("o lobo nao declara vida; a regua do estouro nao existe mais")
	}
	share := skill.ShieldBreakDamage / wolf
	if share < 0.7 || share > 0.9 {
		t.Errorf("o estouro tira %.0f de %.0f de vida do lobo (%.0f%%); a regua e "+
			"80%%, e fora da faixa 70-90%% ele deixa de ser 'abre espaco' e vira "+
			"outra coisa", skill.ShieldBreakDamage, wolf, share*100)
	}
	if skill.ShieldBreakDamage >= wolf {
		t.Errorf("o estouro (%.0f) mata um lobo (%.0f de vida) sozinho: quebrar o "+
			"escudo passou a ser recompensa, e nao perda", skill.ShieldBreakDamage, wolf)
	}
}

// O estouro sai da BORDA da aura, e nao de um raio proprio, para o jogador
// poder mirar a explosao pelo que ve na tela. Um raio maior que o desenho
// mataria bicho fora do circulo; menor deixaria escapar quem esta encostado
// na borda que o jogo desenhou.
func TestShieldBurstReachesExactlyTheDrawnAura(t *testing.T) {
	if skill.ShieldBreakRadius != skill.ShieldRadius {
		t.Errorf("o estouro alcanca %.0f e a aura desenhada tem %.0f; a explosao "+
			"deixou de casar com o circulo que o jogador ve",
			skill.ShieldBreakRadius, skill.ShieldRadius)
	}
}

// A cura do lancamento tem de valer alguma coisa E nao pode substituir a
// Sacerdotisa. A faixa: pelo menos uma mordida de lobo por lancamento (senao
// e enfeite), e no maximo um terco da vida (senao a Paladina se sustenta
// sozinha e a classe de cura perde a razao de existir).
func TestShieldCastHealIsWorthCastingButNotASubstituteHealer(t *testing.T) {
	// A vida da PALADINA, e nao a padrao: a cura e uma fracao da vida maxima
	// de quem lanca, e ela tem 300.
	healed := entity.MaxHealthFor(entity.CharPaladina) * skill.ShieldCastHeal
	wolfBite := entity.GetEnemyDef(entity.EnemyTypeFast).AttackDamage
	if healed < wolfBite {
		t.Errorf("o Q cura %.0f e uma mordida de lobo tira %.0f: a cura nao repoe "+
			"nem um golpe, entao lancar com a vida cheia ou no fim volta a dar no "+
			"mesmo", healed, wolfBite)
	}
	if skill.ShieldCastHeal > 0.34 {
		t.Errorf("o Q cura %.0f%% da vida a cada %.1fs de recarga; a Paladina passa "+
			"a se sustentar sozinha e a Sacerdotisa perde o lugar dela no grupo",
			skill.ShieldCastHeal*100, skill.ShieldCooldown)
	}
}

// O empurrao do estouro tem de tirar o inimigo DE DENTRO da aura: quem quebrou
// o escudo cercado precisa do passo para relancar, e um empurrao menor que o
// raio deixaria o orc encostado exatamente como estava.
func TestShieldBurstPushesEnemiesOutOfTheAura(t *testing.T) {
	if skill.ShieldBreakKnockback <= skill.ShieldBreakRadius {
		t.Errorf("o estouro empurra %.0f px e alcanca %.0f px: o inimigo termina "+
			"dentro da aura, no mesmo lugar em que estava",
			skill.ShieldBreakKnockback, skill.ShieldBreakRadius)
	}
	// E nao pode virar reposicionamento: perder o escudo continua sendo
	// perder o escudo, nao ganhar uma ferramenta de controle de horda.
	if skill.ShieldBreakKnockback > skill.ShieldBreakRadius*3 {
		t.Errorf("o estouro joga o inimigo %.0f px, mais de tres vezes o raio da "+
			"aura (%.0f): quebrar o escudo virou vantagem",
			skill.ShieldBreakKnockback, skill.ShieldBreakRadius)
	}
}

// A PALADINA E A UNICA QUE SAI DO PADRAO, e o motivo e a distancia: ela e a
// unica que precisa estar a alcance de espada para trabalhar. Se um dia todo
// mundo tiver 300, a vantagem dela some sem ninguem apagar linha nenhuma.
func TestPaladinaOutlivesTheRangedClasses(t *testing.T) {
	paladina := entity.MaxHealthFor(entity.CharPaladina)
	if paladina <= entity.DefaultPlayerMaxHealth {
		t.Errorf("a Paladina tem %.0f de vida e o padrao e %.0f: a linha de frente "+
			"nao aguenta mais do que quem atira de longe",
			paladina, entity.DefaultPlayerMaxHealth)
	}
	for _, char := range []entity.CharacterType{
		entity.CharMago, entity.CharArqueiro, entity.CharSacerdotisa, entity.CharNecromante,
	} {
		if entity.MaxHealthFor(char) >= paladina {
			t.Errorf("%s tem %.0f de vida contra %.0f da Paladina: uma classe de "+
				"longe deixou de pagar pela distancia",
				char, entity.MaxHealthFor(char), paladina)
		}
	}
}

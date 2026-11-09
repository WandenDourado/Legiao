package network

// A caçada da Paladina bot aos canhoes (bot/paladina.go, huntCannon) e feita
// de numeros que moram em TRES pacotes: a vida e o alcance do canhao aqui, o
// dano e a duracao da suprema em skill, e a janela e as distancias em bot —
// que nao pode importar nenhum dos dois. Este arquivo e a costura: e o unico
// lugar do repositorio que enxerga os tres.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// A janela da caçada e a duracao do Avatar dos Deuses copiada para tuning.go,
// porque bot nao importa skill (mesmo arranjo de celestialMemory). Copia sem
// verificacao envelhece: se alguem encurtar o Avatar, a Paladina bot continua
// se achando imune por 15 s e morre parada em frente ao canhao.
func TestBotCannonHuntWindowMatchesAvatarDuration(t *testing.T) {
	if bot.CannonHuntWindow != skill.AvatarDuration {
		t.Errorf("a janela da caçada e %.1fs e o Avatar dos Deuses dura %.1fs; "+
			"a Paladina bot atravessa o corredor achando que ainda esta imune",
			bot.CannonHuntWindow, skill.AvatarDuration)
	}
}

// Ela so assume a missao a partir do alcance do canhao: mais perto do que isso
// ela ja esta sangrando de qualquer jeito. Se as duas se separarem, ou ela
// marcha de longe demais (e chega sem suprema), ou fica parada tomando tiro
// sem nunca decidir atravessar.
func TestBotCannonHuntRangeMatchesCannonRange(t *testing.T) {
	if bot.CannonHuntRange != CannonRange {
		t.Errorf("a Paladina bot assume a missao a %.0f px e o canhao alcanca "+
			"%.0f px; ou ela marcha cedo demais, ou toma tiro sem reagir",
			bot.CannonHuntRange, CannonRange)
	}
}

// O orcamento da travessia: levantar a suprema a CannonAvatarRange tem de
// deixar tempo para andar ate la E derrubar os dois postos. Se nao deixar, a
// imunidade acaba com ela em campo aberto e a fase fica sem saida ate o Avatar
// recarregar — 60 s dentro do alcance dos canhoes.
func TestBotCannonApproachFitsInsideTheAvatar(t *testing.T) {
	// Velocidade do jogador: literal em entity/player.go, sem constante
	// exportada. 200 e o valor de hoje.
	const playerSpeed float32 = 200
	// Cadencia da Paladina: AttackSpeed 1.5 em entity/character.go.
	const swingsPerSecond float32 = 1.5
	// Os dois postos do mapa 6 ficam a 768 px um do outro.
	const betweenPosts float32 = 768

	swingsPerCannon := CannonMaxHP / skill.SwordAvatarDamage
	swinging := 2 * swingsPerCannon / swingsPerSecond
	walking := (bot.CannonAvatarRange + betweenPosts) / playerSpeed
	if walking+swinging > skill.AvatarDuration {
		t.Errorf("a travessia custa %.1fs de caminhada + %.1fs de espada = %.1fs, "+
			"e o Avatar dos Deuses dura %.1fs: a Paladina bot fica sem imunidade "+
			"antes de calar os dois canhoes",
			walking, swinging, walking+swinging, skill.AvatarDuration)
	}
}

// E ela tem de conseguir ENCOSTAR: parar a CannonStrikeRange do meio do plinto
// e ainda ter o alvo dentro do arco. O arco alcanca SwordRange (ou o dobro
// largo sob a suprema) mais o raio do alvo.
func TestBotStopsInsideSwordReachOfTheCannon(t *testing.T) {
	if bot.CannonStrikeRange > skill.SwordRange+cannonHitHalfWidth {
		t.Errorf("a Paladina bot para a %.0f px do plinto e a espada normal "+
			"alcanca %.0f px (%.0f de arco + %.0f de meia-largura do alvo): ela "+
			"bateria no vazio", bot.CannonStrikeRange, skill.SwordRange+cannonHitHalfWidth,
			skill.SwordRange, cannonHitHalfWidth)
	}
}

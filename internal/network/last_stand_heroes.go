package network

// Quem responde ao ultimo suspiro, por MAPA.
//
// Isto era o Necromante, escrito no codigo em oito lugares. Funcionou enquanto
// existia uma fase com clima: no mapa 3 quem se ergue e a SACERDOTISA, e a
// unica forma de descobrir isso lendo o codigo antigo era procurar a palavra
// "necromante" espalhada por host_last_stand.go, client.go e o desenho do NPC.
//
// A mesma correcao que waveRuns levou, e pelo mesmo motivo: uma fase nova nao
// pode depender de alguem lembrar de editar uma constante global. Aqui a fase
// declara o heroi dela e o resto do sistema le a declaracao.
//
// A TABELA ENCOLHEU quando o NPC do ultimo suspiro saiu (ver host_last_stand.go).
// Ela carregava, por heroi, um `npcID`, um `cast`, um `alive`, um `anchor` e um
// `endSignal` — cinco campos que existiam so para um personagem invocado poder
// ser dono de um efeito, mante-lo ancorado e avisar o cliente quando sumir.
// Com toda classe vaga preenchida por um bot, quem lanca a suprema e um
// JOGADOR: a magia passa pelo caminho normal de qualquer magia, e o que a fase
// ainda precisa declarar e so QUEM se ergue e QUAL magia devolver carregada.

import "github.com/WandenDourado/Legiao/internal/entity"

// lastStandHero e quem se ergue quando o grupo cai, e qual suprema a cena
// devolve carregada e destravada para ele.
type lastStandHero struct {
	character entity.CharacterType
	skillID   string
}

var (
	necromancerLastStand = lastStandHero{
		character: entity.CharNecromante,
		skillID:   "spectral_legion",
	}
	priestessLastStand = lastStandHero{
		character: entity.CharSacerdotisa,
		skillID:   "angelic_area",
	}
	archerLastStand = lastStandHero{
		character: entity.CharArqueiro,
		skillID:   "celestial_arrows",
	}
	mageLastStand = lastStandHero{
		character: entity.CharMago,
		skillID:   "meteor_rain",
	}
	paladinaLastStand = lastStandHero{
		character: entity.CharPaladina,
		skillID:   "divine_avatar",
	}
)

// lastStandHeroes e o heroi de cada mapa, pelo caminho a partir da raiz do
// repo — a mesma chave que World.Path, campaignMaps e waveRuns usam.
var lastStandHeroes = map[string]lastStandHero{
	"assets/maps/world_02.json": necromancerLastStand,
	"assets/maps/world_03.json": priestessLastStand,
	"assets/maps/world_04.json": archerLastStand,
	"assets/maps/world_05.json": mageLastStand,
	"assets/maps/world_06.json": paladinaLastStand,
}

// LastStandHeroFor devolve o heroi do mapa. Mapa que nao declara um cai no
// Necromante, que e o comportamento que existia antes desta tabela: um mapa de
// teste que chegue a armar a cena continua tendo uma saida em vez de travar.
func LastStandHeroFor(mapPath string) lastStandHero {
	if hero, ok := lastStandHeroes[mapPath]; ok {
		return hero
	}
	return necromancerLastStand
}

// LastStandCharacterFor devolve QUEM se ergue neste mapa, e se alguem se ergue.
//
// E irma de `LastStandHeroFor` e existe pela diferenca no `ok`: aquela precisa
// de um heroi sempre, porque um mapa que arma a cena e nao tem saida trava; esta
// precisa da verdade, porque quem a le e a progressao, e um mapa sem cena nao
// pode entregar a suprema do Necromante por causa de um fallback. O mapa 1 nao
// tem cena, e antes desta funcao o fallback dizia que tinha.
//
// A progressao le isto em vez de uma tabela propria de proposito: a fase ja
// declara o heroi dela, e uma segunda lista dizendo a mesma coisa e uma lista
// que vai divergir no primeiro conserto.
func LastStandCharacterFor(mapPath string) (entity.CharacterType, bool) {
	hero, ok := lastStandHeroes[mapPath]
	if !ok {
		return "", false
	}
	return hero.character, true
}

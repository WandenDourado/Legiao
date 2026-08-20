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

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// lastStandHero e quem se ergue quando o grupo cai, e o que ele lanca.
//
// `npcID` e o dono da magia quando NINGUEM no grupo joga com o personagem. Ele
// nao e um jogador: nunca entra em h.players, entao nao conta no HUD, nao pesa
// no Game Over e nao precisa morrer nem ser sincronizado. Precisa so ser dono
// de um efeito e ser desenhado.
//
// `alive` e `anchor` separam as duas naturezas de ultimate que existem hoje, e
// a diferenca e de MANUTENCAO, nao de poder:
//
//   - A Legiao Espectral e um bando que SEGUE o dono, entao ela tem `anchor`:
//     o host reancora a legiao no NPC a cada quadro.
//   - A Area Angelical e um ALTAR consagrado no chao. Ela nao segue ninguem,
//     entao nao tem `anchor` — mas tem `alive` como qualquer outra, porque o
//     NPC precisa sair de campo quando o efeito se gasta. Uma versao anterior
//     disto so pulava a manutencao para quem nao fosse legiao, e o resultado
//     era a Sacerdotisa invocada ficando plantada na esplanada ate o fim da
//     fase.
//
// `endSignal` e a mensagem que avisa o cliente para parar de desenhar o NPC.
type lastStandHero struct {
	character entity.CharacterType
	skillID   string
	npcID     string
	cast      func(*skill.Manager, string, rl.Vector2)
	alive     func(*skill.Manager, string) bool
	anchor    func(*skill.Manager, string, rl.Vector2)
	endSignal string
}

var (
	necromancerLastStand = lastStandHero{
		character: entity.CharNecromante,
		skillID:   "spectral_legion",
		npcID:     "npc_necromante",
		cast:      skill.ActivateLegion,
		alive:     (*skill.Manager).HasLegion,
		anchor:    (*skill.Manager).SetLegionAnchor,
		endSignal: "legion_end",
	}
	priestessLastStand = lastStandHero{
		character: entity.CharSacerdotisa,
		skillID:   "angelic_area",
		npcID:     "npc_sacerdotisa",
		cast:      skill.ActivateAngelic,
		alive:     (*skill.Manager).HasAngelic,
		// sem anchor: o altar fica onde foi consagrado
		endSignal: "angelic_end",
	}
	archerLastStand = lastStandHero{
		character: entity.CharArqueiro,
		skillID:   "celestial_arrows",
		npcID:     "npc_arqueiro",
		// Map 4 owns this cast because it must target the two declared stream
		// sentries, not an arbitrary aim point. ResolveLastStand invokes it.
		cast:      func(*skill.Manager, string, rl.Vector2) {},
		alive:     func(*skill.Manager, string) bool { return false },
		endSignal: "celestial_end",
	}
	mageLastStand = lastStandHero{
		character: entity.CharMago,
		skillID:   "meteor_rain",
		npcID:     "npc_mago",
		cast:      skill.StartMeteorRain,
		alive:     (*skill.Manager).HasMeteorRain,
		endSignal: "meteor_rain_end",
	}
	paladinaLastStand = lastStandHero{
		character: entity.CharPaladina,
		skillID:   "divine_avatar",
		npcID:     "npc_paladina",
		// O cast puro (imunidade + visual) e so metade do resgate do mapa 6:
		// o julgamento que alcanca e destroi os dois canhoes e roteirizado a
		// parte, em castCannonJudgment — a mesma divisao que o mapa 4 faz
		// entre "conceder a suprema" e "acertar as sentinelas declaradas".
		// ResolveLastStand chama os dois.
		cast:  skill.ActivateAvatar,
		alive: (*skill.Manager).HasAvatar,
		// sem anchor: o NPC nao anda, e tickAvatars so reancora avatares de
		// JOGADOR (h.players) mesmo — um avatar de NPC fica parado onde foi
		// concedido, que e o centro do grupo.
		endSignal: "avatar_end",
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

// isLastStandNPC diz se o dono de um efeito e o NPC do ultimo suspiro de
// qualquer mapa. O cliente precisa disto: ele aprende que ha um NPC em campo
// pela propria mensagem da magia, sem mensagem nova de protocolo para uma
// coisa que acontece uma vez por fase.
func isLastStandNPC(ownerID string) bool {
	for _, hero := range lastStandHeroes {
		if hero.npcID == ownerID {
			return true
		}
	}
	return false
}

// noteLastStandNPC poe o NPC em campo se o dono do efeito for um deles.
//
// Chamado do cliente em cada ultimate que algum mapa usa no ultimo suspiro.
// Para o dono ser um jogador de verdade nao faz nada, que e o caso comum.
func noteLastStandNPC(ownerID string, pos rl.Vector2) {
	for _, hero := range lastStandHeroes {
		if hero.npcID == ownerID {
			setLastStandNPCAs(pos, true, ownerID, hero.character)
			return
		}
	}
}

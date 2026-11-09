package network

// Progressao da campanha: o que o grupo ja destravou.
//
// Hoje ha uma coisa so aqui, a ultimate, mas o lugar existe porque ela nao vai
// ser a unica: o grupo comeca sem os poderes supremos e ganha um a cada fase
// vencida. O estado e do HOST, como todo o resto que decide o que pode ser
// lancado, e o cliente nunca precisa saber - ele ja recebe a recusa.
//
// A LIBERACAO E POR PERSONAGEM, E NAO DA FESTA INTEIRA.
//
// Isto era um booleano: "a partir do mapa 2, todo mundo tem a sua". Mas a
// campanha nao entrega as supremas de uma vez — cada personagem ganha a dele na
// fase SEGUINTE a cena em que ele se ergue (`lastStandHeroes`), e e isso que
// da a cada fase uma regua diferente de dificuldade. Com o booleano, o grupo
// chegava ao mapa 3 com a Area Angelical e as Flechas Celestiais na mao, e as
// hordas dos mapas 3, 4 e 5 — calibradas cada uma contra as supremas que
// deveriam existir ali — ficavam calibradas contra outra coisa.

import (
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
)

var (
	progressionMu     sync.RWMutex
	unlockedUltimates map[entity.CharacterType]bool
	// runGrantedUltimates sao supremas liberadas SO PARA A CORRIDA ATUAL, por
	// cima do conjunto da campanha acima. O resgate do ultimo suspiro concede
	// a suprema do heroi da fase sem tocar em `unlockedUltimates`: a tabela de
	// progressao (game.UltimatesGrantedOn) continua dizendo que o personagem
	// so ganha a dele na fase SEGUINTE, e a regua de dificuldade dessa fase
	// seguinte continua calibrada contra o conjunto que ela espera.
	runGrantedUltimates map[entity.CharacterType]bool
)

// SetUnlockedUltimates declara QUAIS supremas estao liberadas na fase atual.
// Chamado quando o mapa e carregado (World.ApplyToHost), no host e no cliente:
// o host para recusar o lancamento, o cliente para nao desenhar um botao que
// vai levar nao.
//
// O conjunto e COPIADO. Quem chama monta o mapa a cada carregamento e nao tem
// motivo para saber que este pacote guarda a referencia.
//
// Toda troca de mapa passa por aqui, e por isso a concessao DA CORRIDA e
// apagada junto: ela nao pode atravessar um portal para a fase seguinte, que
// tem a propria regua de dificuldade calibrada sem essa suprema extra.
func SetUnlockedUltimates(chars map[entity.CharacterType]bool) {
	progressionMu.Lock()
	defer progressionMu.Unlock()
	unlockedUltimates = make(map[entity.CharacterType]bool, len(chars))
	for c, ok := range chars {
		if ok {
			unlockedUltimates[c] = true
		}
	}
	runGrantedUltimates = nil
}

// UltimateUnlockedFor reporta se ESTE personagem ja tem a suprema dele na fase
// atual — pela campanha, ou porque o resgate a concedeu nesta corrida. Lido
// pelo gate do host e pelo HUD.
func UltimateUnlockedFor(char entity.CharacterType) bool {
	progressionMu.RLock()
	defer progressionMu.RUnlock()
	return unlockedUltimates[char] || runGrantedUltimates[char]
}

// GrantUltimateForRun libera a suprema de char so para a corrida atual, por
// cima do que a campanha ja concede. Chamado quando o resgate reergue o heroi
// da fase e alguem do grupo joga com ele (Host.reviveHero) — o caso em que a
// cena PRECISA que o lancamento funcione, no desktop e no celular.
func GrantUltimateForRun(char entity.CharacterType) {
	progressionMu.Lock()
	defer progressionMu.Unlock()
	if runGrantedUltimates == nil {
		runGrantedUltimates = make(map[entity.CharacterType]bool, 1)
	}
	runGrantedUltimates[char] = true
}

// ClearRunGrantedUltimates apaga as concessoes da corrida atual.
//
// SetUnlockedUltimates ja cobre a troca de mapa; esta existe para o outro
// caso que o esquece por natureza — o reinicio de fase (F5), que fica no
// MESMO mapa e por isso nunca chama SetUnlockedUltimates. Sem isto uma fase
// perdida DEPOIS do resgate reiniciaria com a suprema do heroi ainda
// destravada, mesmo que a proxima tentativa nao passe pela cena de novo.
func ClearRunGrantedUltimates() {
	progressionMu.Lock()
	defer progressionMu.Unlock()
	runGrantedUltimates = nil
}

// RunGrantedUltimatesSnapshot devolve quem tem a suprema so PARA A CORRIDA
// atual. Usado para reenviar a concessao a um cliente que entra no meio ou
// reconecta (Host.sendUltimateGrantsTo) — sem isto o jogador veria o botao
// apagado ate a proxima vez que alguem lancasse a magia.
func RunGrantedUltimatesSnapshot() []entity.CharacterType {
	progressionMu.RLock()
	defer progressionMu.RUnlock()
	out := make([]entity.CharacterType, 0, len(runGrantedUltimates))
	for c := range runGrantedUltimates {
		out = append(out, c)
	}
	return out
}

// BroadcastUltimateGrant avisa todo peer que a suprema de char foi liberada
// para esta corrida. Chamado uma vez pelo host, de Host.reviveHero.
func (h *Host) BroadcastUltimateGrant(char entity.CharacterType) {
	h.broadcast(Message{Type: MsgUltimateGrant, Payload: MustMarshal(UltimateGrantPayload{
		Character: string(char),
	})})
}

// sendUltimateGrantsTo reenvia toda concessao da corrida atual a UMA conexao
// — um cliente que entrou no meio da partida ou reconectou nunca viu o
// broadcast original, e sem isto o botao da ultimate ficaria apagado ate a
// magia ser lancada de novo.
func (h *Host) sendUltimateGrantsTo(c *ClientConn) {
	for _, char := range RunGrantedUltimatesSnapshot() {
		h.sendTo(c, Message{Type: MsgUltimateGrant, Payload: MustMarshal(UltimateGrantPayload{
			Character: string(char),
		})})
	}
}

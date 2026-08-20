package network

// Desenhar o passado, para o presente chegar liso.
//
// O host publica a 20 Hz (broadcast_rate.go) e a tela desenha a 60: sem nada
// entre os dois, cada posicao remota daria tres quadros parada e um salto —
// e o salto so cresce se um snapshot atrasar. O cliente entao NAO desenha o
// ultimo snapshot recebido: ele desenha interpDelay atras dele, com dois
// snapshots na mao, interpolando entre os dois.
//
// O atraso e o preco: o que se ve de um jogador remoto esta ~100 ms velho.
// Isso e correto para este jogo. O host e autoritativo e ja decide todo o
// combate; o cliente nunca precisou que a posicao remota fosse instantanea,
// precisou que ela fosse CONTINUA. Suavizar sem atraso exigiria extrapolar, e
// extrapolacao erra a direcao toda vez que alguem muda de rumo — o defeito
// vira personagem escorregando e voltando, que e pior que 100 ms.
//
// O jogador LOCAL nunca passa por aqui: ele e simulado no proprio quadro.
// O HOST tambem nao: ele tem o estado de verdade a 60 Hz.

import (
	"sync"
	"time"
)

// interpDelay e quanto o desenho fica atras do ultimo snapshot.
//
// Dois intervalos de snapshot a 20 Hz sao 100 ms: um para ter sempre um par
// para interpolar, e a folga do segundo para um pacote atrasado nao esvaziar
// o buffer. Menos que isso e a interpolacao vira extrapolacao no primeiro
// engasgo da rede.
const interpDelay = 100 * time.Millisecond

// snapshot e um estado recebido e o instante em que chegou.
type snapshot struct {
	at      time.Time
	players map[string]PlayerState
	enemies map[string]EnemyState
}

var (
	interpMutex sync.Mutex
	// prev e curr sao os dois ultimos snapshots. A interpolacao acontece
	// entre eles; nada mais e guardado, porque um buffer maior so adiaria o
	// descarte do que ja nao vai ser desenhado.
	prevSnap, currSnap *snapshot
)

// recordSnapshot guarda o estado recebido para a interpolacao.
//
// Recebe os dois mapas juntos porque eles chegam em mensagens diferentes e
// precisam de um instante comum: interpolar jogador e inimigo em linhas de
// tempo separadas faria um monstro adiantado bater num jogador atrasado.
func recordSnapshot(players map[string]PlayerState, enemies map[string]EnemyState) {
	interpMutex.Lock()
	defer interpMutex.Unlock()

	next := &snapshot{at: time.Now(), players: players, enemies: enemies}
	if currSnap != nil && players == nil {
		next.players = currSnap.players
	}
	if currSnap != nil && enemies == nil {
		next.enemies = currSnap.enemies
	}
	prevSnap, currSnap = currSnap, next
}

// interpAlpha e onde estamos entre prev e curr, de 0 a 1, para o instante que
// deve aparecer na tela agora.
//
// Devolve false quando nao ha par para interpolar (comeco da sessao, ou um
// snapshot so). O chamador desenha o mais recente cru, que e o comportamento
// antigo — pior que interpolar, melhor que nao desenhar.
func interpAlpha() (float32, bool) {
	if prevSnap == nil || currSnap == nil {
		return 0, false
	}
	span := currSnap.at.Sub(prevSnap.at)
	if span <= 0 {
		return 0, false
	}
	target := time.Now().Add(-interpDelay)
	alpha := float32(target.Sub(prevSnap.at)) / float32(span)
	// Preso ao intervalo: passar de 1 seria extrapolar, e e o que acontece
	// quando o proximo snapshot atrasa. Segurar no ultimo conhecido faz o
	// personagem PARAR por um instante em vez de deslizar para o lugar errado
	// e voltar.
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return alpha, true
}

// interpPlayersBuf e interpEnemiesBuf sao os buffers reaproveitados pelas
// duas funcoes abaixo. Cada uma e chamada mais de uma vez por quadro
// (renderer.go desenha os jogadores remotos e depois usa a mesma lista para
// mirar os inimigos), sempre pelo desenho, na mesma goroutine, e sempre
// consumida antes da proxima chamada — nunca guardada entre quadros. Um mapa
// novo por chamada e alocacao com 83 inimigos em campo (doc/performance.md).
var (
	interpPlayersBuf = make(map[string]PlayerState)
	interpEnemiesBuf = make(map[string]EnemyState)
)

// InterpolatedPlayers e o mapa de jogadores como deve APARECER agora.
//
// Interpola so o que e continuo: posicao e velocidade. Vida, morte, quadro de
// animacao e identidade vem do snapshot mais recente sem misturar — meia morte
// nao existe, e um quadro de sprite interpolado seria um numero entre dois
// desenhos.
func InterpolatedPlayers() map[string]PlayerState {
	if Role != "client" {
		return GetAllPlayers()
	}
	interpMutex.Lock()
	defer interpMutex.Unlock()

	alpha, ok := interpAlpha()
	if !ok || currSnap == nil {
		return GetAllPlayers()
	}
	clear(interpPlayersBuf)
	for id, cur := range currSnap.players {
		old, had := prevSnap.players[id]
		if !had {
			// Quem acabou de aparecer nao tem de onde vir: desenhar no lugar
			// dele e melhor que desenhar vindo da origem do mundo.
			interpPlayersBuf[id] = cur
			continue
		}
		cur.X = lerpInt(old.X, cur.X, alpha)
		cur.Y = lerpInt(old.Y, cur.Y, alpha)
		cur.VelX = lerp(old.VelX, cur.VelX, alpha)
		cur.VelY = lerp(old.VelY, cur.VelY, alpha)
		interpPlayersBuf[id] = cur
	}
	return interpPlayersBuf
}

// InterpolatedEnemies e o mesmo pelos inimigos.
func InterpolatedEnemies() map[string]EnemyState {
	interpMutex.Lock()
	defer interpMutex.Unlock()

	alpha, ok := interpAlpha()
	if !ok || currSnap == nil {
		RemoteEnemiesMutex.Lock()
		defer RemoteEnemiesMutex.Unlock()
		clear(interpEnemiesBuf)
		for id, e := range RemoteEnemies {
			interpEnemiesBuf[id] = e
		}
		return interpEnemiesBuf
	}
	clear(interpEnemiesBuf)
	for id, cur := range currSnap.enemies {
		old, had := prevSnap.enemies[id]
		if !had {
			interpEnemiesBuf[id] = cur
			continue
		}
		cur.X = lerpInt(old.X, cur.X, alpha)
		cur.Y = lerpInt(old.Y, cur.Y, alpha)
		interpEnemiesBuf[id] = cur
	}
	return interpEnemiesBuf
}

// ResetInterpolation esquece os snapshots guardados. Chamado quando o mundo
// mirrorado e limpo (reset de fase): interpolar da posicao antiga para a nova
// faria todo mundo DESLIZAR do campo de batalha ate o ponto de spawn.
func ResetInterpolation() {
	interpMutex.Lock()
	defer interpMutex.Unlock()
	prevSnap, currSnap = nil, nil
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func lerpInt(a, b int, t float32) int {
	return a + int(float32(b-a)*t)
}

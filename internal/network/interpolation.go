package network

// Desenhar o passado, para o presente chegar liso.
//
// O host publica a 20 Hz (broadcast_rate.go) e a tela desenha a 60: sem nada
// entre os dois, cada posicao remota daria tres quadros parada e um salto. O
// cliente entao desenha um pouco ATRAS do ultimo snapshot, entre os dois
// snapshots que cercam esse instante.
//
// Ate 28/09/2026 isso nao funcionava: guardavam-se so os DOIS ultimos
// snapshots (50 ms de historia) e desenhava-se 100 ms atras - fora do
// intervalo. O fator de mistura saia sempre 0 e tudo andava aos saltos de
// 20 Hz. Alem disso state_update e enemy_update, que chegam juntos, viravam
// dois snapshots a 0 ms um do outro. Agora:
//
//   - os snapshots ficam num buffer curto, marcados com o RELOGIO DO HOST
//     (HostT em state_update/enemy_update), nao com a hora de chegada: o 4G
//     entrega em rajadas, e usar a chegada transformaria cada rajada num
//     tranco. As duas mensagens do mesmo tique viram UM snapshot;
//   - o atraso se adapta a rede (interpolation_clock.go): ~100 ms na LAN,
//     mais quando a chegada oscila, e muda devagar para nao dar solavanco.
//
// Continua valendo: interpola so posicao e velocidade (vida, morte, quadro de
// animacao e identidade vem do snapshot mais novo), e NUNCA extrapola - se o
// proximo snapshot atrasa, o boneco para no ultimo ponto conhecido em vez de
// deslizar para o lugar errado e voltar. O jogador LOCAL e o HOST nao passam
// por aqui.

import (
	"sync"
)

// interpBufferLen cobre bem mais que o maior atraso (maxInterpDelay) a 20 Hz.
const interpBufferLen = 16

// sameTickMs: mensagens com HostT tao proximo sao do mesmo tique.
const sameTickMs = 5

// snapshot e o estado de um tique do host.
type snapshot struct {
	t       int64 // ms no relogio do host (ou de chegada, com host antigo)
	players map[string]PlayerState
	enemies map[string]EnemyState
}

var (
	interpMutex sync.Mutex
	snaps       []snapshot // em ordem de t, no maximo interpBufferLen
)

// recordSnapshot guarda o estado recebido. players ou enemies nil = esta
// mensagem nao fala deles (vem da mensagem irma do mesmo tique, ou do tique
// anterior). hostT 0 = host antigo, sem relogio: usa a hora de chegada.
func recordSnapshot(players map[string]PlayerState, enemies map[string]EnemyState, hostT int64) {
	interpMutex.Lock()
	defer interpMutex.Unlock()

	t := observeArrival(hostT)
	if n := len(snaps); n > 0 {
		last := &snaps[n-1]
		if t-last.t <= sameTickMs && t >= last.t-sameTickMs {
			if players != nil {
				last.players = players
			}
			if enemies != nil {
				last.enemies = enemies
			}
			return
		}
		if t < last.t {
			return // fora de ordem (nao acontece em TCP; defensivo)
		}
		if players == nil {
			players = last.players
		}
		if enemies == nil {
			enemies = last.enemies
		}
	}
	snaps = append(snaps, snapshot{t: t, players: players, enemies: enemies})
	if len(snaps) > interpBufferLen {
		snaps = append(snaps[:0], snaps[len(snaps)-interpBufferLen:]...)
	}
}

// bracket acha os dois snapshots em volta do instante de desenho e o fator de
// mistura entre eles. ok falso sem snapshot nenhum.
func bracket() (a, b *snapshot, alpha float32, ok bool) {
	n := len(snaps)
	if n == 0 {
		return nil, nil, 0, false
	}
	if n == 1 {
		return &snaps[0], &snaps[0], 1, true
	}
	rt := renderTime()
	if rt >= snaps[n-1].t { // o proximo atrasou: segura no ultimo, sem extrapolar
		return &snaps[n-1], &snaps[n-1], 1, true
	}
	if rt <= snaps[0].t {
		return &snaps[0], &snaps[0], 1, true
	}
	for i := n - 2; i >= 0; i-- {
		if snaps[i].t <= rt {
			a, b = &snaps[i], &snaps[i+1]
			return a, b, float32(rt-a.t) / float32(b.t-a.t), true
		}
	}
	return &snaps[0], &snaps[0], 1, true
}

// interpPlayersBuf e interpEnemiesBuf sao reaproveitados entre chamadas: um
// mapa novo por chamada e alocacao com 83 inimigos em campo
// (doc/performance.md). Sempre consumidos antes da proxima chamada.
var (
	interpPlayersBuf = make(map[string]PlayerState)
	interpEnemiesBuf = make(map[string]EnemyState)
)

// InterpolatedPlayers e o mapa de jogadores como deve APARECER agora.
func InterpolatedPlayers() map[string]PlayerState {
	if Role != "client" {
		return GetAllPlayers()
	}
	interpMutex.Lock()
	defer interpMutex.Unlock()
	a, b, alpha, ok := bracket()
	if !ok || b.players == nil {
		return GetAllPlayers()
	}
	clear(interpPlayersBuf)
	for id, cur := range b.players {
		if old, had := a.players[id]; had && a != b {
			cur.X = lerpInt(old.X, cur.X, alpha)
			cur.Y = lerpInt(old.Y, cur.Y, alpha)
			cur.VelX = lerp(old.VelX, cur.VelX, alpha)
			cur.VelY = lerp(old.VelY, cur.VelY, alpha)
		}
		interpPlayersBuf[id] = cur
	}
	return interpPlayersBuf
}

// InterpolatedEnemies e o mesmo pelos inimigos.
func InterpolatedEnemies() map[string]EnemyState {
	interpMutex.Lock()
	defer interpMutex.Unlock()
	clear(interpEnemiesBuf)
	a, b, alpha, ok := bracket()
	if !ok || b.enemies == nil {
		RemoteEnemiesMutex.Lock()
		defer RemoteEnemiesMutex.Unlock()
		for id, e := range RemoteEnemies {
			interpEnemiesBuf[id] = e
		}
		return interpEnemiesBuf
	}
	for id, cur := range b.enemies {
		if old, had := a.enemies[id]; had && a != b {
			cur.X = lerpInt(old.X, cur.X, alpha)
			cur.Y = lerpInt(old.Y, cur.Y, alpha)
		}
		interpEnemiesBuf[id] = cur
	}
	return interpEnemiesBuf
}

// ResetInterpolation esquece os snapshots. Chamado quando o mundo espelhado e
// limpo (reset de fase, reconexao): interpolar da posicao antiga para a nova
// faria todo mundo DESLIZAR ate o ponto de spawn.
func ResetInterpolation() {
	interpMutex.Lock()
	defer interpMutex.Unlock()
	snaps = snaps[:0]
	resetInterpClock()
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func lerpInt(a, b int, t float32) int {
	return a + int(float32(b-a)*t)
}

package skill

import (
	"fmt"
	"sync"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// cannonSeq é um contador atômico pela mesma razão do orbSeq da sentinela:
// com dois canhões disparando ao longo do corredor inteiro, colisão de ID por
// `generateID()` (260 resultados possíveis) é questão de tempo.
var cannonSeq uint64

func nextCannonBallID() string {
	return fmt.Sprintf("cb%d", atomic.AddUint64(&cannonSeq, 1))
}

var cannonMutex sync.RWMutex

// cannonState espelha sentryState: as duas coleções andam juntas (o impacto
// tira uma bola e põe uma explosão no lugar dela).
type cannonState struct {
	Balls  map[string]*CannonBall
	Bursts []*CannonBallBurst
}

var cannon = cannonState{Balls: map[string]*CannonBall{}}

// clientCannon é o espelho do cliente. Host e cliente não podem dividir a
// mesma coleção pela mesma razão da sentinela: quando o processo é host, os
// dois caminhos existem juntos.
var clientCannon = cannonState{Balls: map[string]*CannonBall{}}
var clientCannonMutex sync.RWMutex

func cannonStore(host bool) (*cannonState, *sync.RWMutex) {
	if host {
		return &cannon, &cannonMutex
	}
	return &clientCannon, &clientCannonMutex
}

// SpawnCannonBall põe uma bola em campo e devolve o ID gerado, que o chamador
// precisa para replicar aos clientes.
func SpawnCannonBall(host bool, id, cannonID string, start, dir rl.Vector2) string {
	if id == "" {
		id = nextCannonBallID()
	}
	st, mu := cannonStore(host)
	mu.Lock()
	defer mu.Unlock()
	st.Balls[id] = NewCannonBall(id, cannonID, start, dir)
	return id
}

// RemoveCannonBall tira uma bola de campo e, se burst, deixa a explosão no
// lugar dela.
func RemoveCannonBall(host bool, id string, burst bool) {
	st, mu := cannonStore(host)
	mu.Lock()
	defer mu.Unlock()
	b, ok := st.Balls[id]
	if !ok {
		return
	}
	delete(st.Balls, id)
	if burst {
		st.Bursts = append(st.Bursts, NewCannonBallBurst(b.Position))
	}
}

// AddCannonBurstAt deixa uma explosão numa posição sem que haja bola local —
// o caminho do cliente, que pode receber o impacto antes de ter a bola.
func AddCannonBurstAt(host bool, pos rl.Vector2) {
	st, mu := cannonStore(host)
	mu.Lock()
	defer mu.Unlock()
	st.Bursts = append(st.Bursts, NewCannonBallBurst(pos))
}

// GetCannonBalls devolve um retrato das bolas ativas.
func GetCannonBalls(host bool) []*CannonBall {
	st, mu := cannonStore(host)
	mu.RLock()
	defer mu.RUnlock()
	list := make([]*CannonBall, 0, len(st.Balls))
	for _, b := range st.Balls {
		list = append(list, b)
	}
	return list
}

// StepCannonBalls avança as bolas do HOST e devolve as que expiraram por
// tempo sem acertar ninguém. Colisão com jogador fica fora: quem sabe onde os
// jogadores estão, e quem pode tirar vida deles, é a camada de rede
// (internal/network/host_cannon.go).
func StepCannonBalls(dt float32) []string {
	expired := make([]string, 0)
	for _, b := range GetCannonBalls(true) {
		if !b.Update(dt) {
			expired = append(expired, b.ID)
		}
	}
	return expired
}

// AdvanceCannonBalls faz o mesmo no CLIENTE, sem decidir remoção por
// impacto — só poda as órfãs por TTL.
func AdvanceCannonBalls(dt float32) {
	clientCannonMutex.Lock()
	defer clientCannonMutex.Unlock()
	for id, b := range clientCannon.Balls {
		b.AdvanceVisual(dt)
		if b.Expired() {
			delete(clientCannon.Balls, id)
		}
	}
}

// UpdateCannonBursts anima as explosões e descarta as que acabaram.
func UpdateCannonBursts(host bool, dt float32) {
	st, mu := cannonStore(host)
	mu.Lock()
	defer mu.Unlock()
	kept := st.Bursts[:0]
	for _, b := range st.Bursts {
		if b.Update(dt) {
			kept = append(kept, b)
		}
	}
	st.Bursts = kept
}

// DrawCannonBalls desenha bolas e explosões em espaço de mundo.
func DrawCannonBalls(host bool) {
	st, mu := cannonStore(host)
	mu.RLock()
	defer mu.RUnlock()
	for _, b := range st.Balls {
		b.Draw()
	}
	for _, b := range st.Bursts {
		b.Draw()
	}
}

// ResetCannonBalls limpa as duas coleções. O reset da fase tem que passar por
// aqui pela mesma razão da sentinela: uma bola que sobrevive ao reinício
// persegue... não persegue nada, mas ainda voaria rumo a um alvo que já não
// está mais onde estava.
func ResetCannonBalls() {
	cannonMutex.Lock()
	cannon.Balls = map[string]*CannonBall{}
	cannon.Bursts = nil
	cannonMutex.Unlock()

	clientCannonMutex.Lock()
	clientCannon.Balls = map[string]*CannonBall{}
	clientCannon.Bursts = nil
	clientCannonMutex.Unlock()
}

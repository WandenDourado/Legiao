package skill

import (
	"fmt"
	"sync"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// orbSeq e um contador atomico, e nao o generateID() do particle.go, porque
// aquele tem 260 resultados possiveis: com duas sentinelas disparando a cada
// 1,35s durante o climax inteiro, colisao de ID e questao de tempo - e duas
// esferas com o mesmo ID fazem uma sumir quando a outra acerta.
var orbSeq uint64

func nextSentryOrbID() string {
	return fmt.Sprintf("so%d", atomic.AddUint64(&orbSeq, 1))
}

// sentryMutex guarda as duas colecoes da sentinela. Elas andam juntas (o
// impacto tira uma esfera e poe um estouro), entao dividem um lock so.
var sentryMutex sync.RWMutex

// SentryOrbs e SentryBursts vivem no Manager como as demais colecoes. Ficam
// declaradas aqui, e nao no fireball_manager.go, para nao empurrar mais um
// campo de monstro para dentro do struct das magias de personagem - o
// Manager ja e grande.
type sentryState struct {
	Orbs   map[string]*SentryOrb
	Bursts []*SentryOrbBurst
}

var sentry = sentryState{Orbs: map[string]*SentryOrb{}}

// clientSentry e o espelho do cliente. Host e cliente NAO podem dividir a
// mesma colecao: quando o jogo roda como host, os dois caminhos existem no
// mesmo processo e um limparia o outro.
var clientSentry = sentryState{Orbs: map[string]*SentryOrb{}}
var clientSentryMutex sync.RWMutex

func sentryStore(host bool) (*sentryState, *sync.RWMutex) {
	if host {
		return &sentry, &sentryMutex
	}
	return &clientSentry, &clientSentryMutex
}

// SpawnSentryOrb poe uma esfera em campo e devolve o ID gerado, que o chamador
// precisa para replicar aos clientes. ttl <= 0 cai para SentryOrbTTL —
// ver NewSentryOrb.
func SpawnSentryOrb(host bool, id, sentryID, targetID string, start, target rl.Vector2, ttl float32) string {
	if id == "" {
		id = nextSentryOrbID()
	}
	st, mu := sentryStore(host)
	mu.Lock()
	defer mu.Unlock()
	st.Orbs[id] = NewSentryOrb(id, sentryID, targetID, start, target, ttl)
	return id
}

// SentryHasLiveOrb reports whether the sentry identified by sentryID already
// has an orb in flight — the "one orb per sentry" rule
// (doc/plan_avanco_bots_e_gargula.md §B2): with global range and a slow
// travel time, cadence alone would otherwise stack a dozen orbs chasing the
// same player.
func SentryHasLiveOrb(host bool, sentryID string) bool {
	st, mu := sentryStore(host)
	mu.RLock()
	defer mu.RUnlock()
	for _, o := range st.Orbs {
		if o.SentryID == sentryID {
			return true
		}
	}
	return false
}

// RemoveSentryOrb tira uma esfera de campo e, se burst, deixa o estouro no
// lugar dela.
func RemoveSentryOrb(host bool, id string, burst bool) {
	st, mu := sentryStore(host)
	mu.Lock()
	defer mu.Unlock()
	o, ok := st.Orbs[id]
	if !ok {
		return
	}
	delete(st.Orbs, id)
	if burst {
		st.Bursts = append(st.Bursts, NewSentryOrbBurst(o.Position))
	}
}

// AddSentryBurstAt deixa um estouro numa posicao sem que haja esfera local -
// e o caminho do cliente, que pode receber o impacto antes de ter a esfera.
func AddSentryBurstAt(host bool, pos rl.Vector2) {
	st, mu := sentryStore(host)
	mu.Lock()
	defer mu.Unlock()
	st.Bursts = append(st.Bursts, NewSentryOrbBurst(pos))
}

// GetSentryOrbs devolve um retrato das esferas ativas.
func GetSentryOrbs(host bool) []*SentryOrb {
	st, mu := sentryStore(host)
	mu.RLock()
	defer mu.RUnlock()
	list := make([]*SentryOrb, 0, len(st.Orbs))
	for _, o := range st.Orbs {
		list = append(list, o)
	}
	return list
}

// StepSentryOrbs avanca as esferas do HOST e devolve as que expiraram por
// tempo. Perseguicao e colisao com jogador ficam fora: quem sabe onde os
// jogadores estao, e quem pode tirar vida deles, e a camada de rede.
//
// targets mapeia player_id -> posicao atual. Um alvo ausente do mapa quer
// dizer que morreu ou saiu: a esfera segue reto ate o TTL.
func StepSentryOrbs(dt float32, targets map[string]rl.Vector2) []string {
	expired := make([]string, 0)
	for _, o := range GetSentryOrbs(true) {
		pos, ok := targets[o.TargetID]
		if !o.Update(dt, pos, ok) {
			expired = append(expired, o.ID)
		}
	}
	return expired
}

// AdvanceSentryOrbs faz o mesmo no CLIENTE, sem decidir remocao por impacto -
// so poda as orfas por TTL, para que uma esfera cujo evento de impacto se
// perdeu nao voe para sempre.
func AdvanceSentryOrbs(dt float32, targets map[string]rl.Vector2) {
	clientSentryMutex.Lock()
	defer clientSentryMutex.Unlock()
	for id, o := range clientSentry.Orbs {
		pos, ok := targets[o.TargetID]
		o.AdvanceVisual(dt, pos, ok)
		if o.Expired() {
			delete(clientSentry.Orbs, id)
		}
	}
}

// UpdateSentryBursts anima os estouros e descarta os que acabaram.
func UpdateSentryBursts(host bool, dt float32) {
	st, mu := sentryStore(host)
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

// DrawSentryOrbs desenha esferas e estouros em espaco de mundo.
func DrawSentryOrbs(host bool) {
	st, mu := sentryStore(host)
	mu.RLock()
	defer mu.RUnlock()
	for _, o := range st.Orbs {
		o.Draw()
	}
	for _, b := range st.Bursts {
		b.Draw()
	}
}

// ResetSentryOrbs limpa as duas colecoes. O reset da fase tem que passar por
// aqui: uma esfera que sobrevive ao reinicio persegue um jogador que ja
// renasceu em outro lugar, e ninguem consegue explicar de onde ela veio.
func ResetSentryOrbs() {
	sentryMutex.Lock()
	sentry.Orbs = map[string]*SentryOrb{}
	sentry.Bursts = nil
	sentryMutex.Unlock()

	clientSentryMutex.Lock()
	clientSentry.Orbs = map[string]*SentryOrb{}
	clientSentry.Bursts = nil
	clientSentryMutex.Unlock()
}

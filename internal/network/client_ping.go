package network

import (
	"encoding/json"
	"log"
	"sync/atomic"
	"time"
)

// Latencia medida pelo cliente, para o HUD (ui/online_badge.go).
//
// A cada pingInterval o cliente manda MsgPing com a propria hora; o host
// devolve igual (MsgPong) e a diferenca e o tempo de ida e volta. Media movel
// para o numero nao pular a cada pacote. Um host antigo nao responde: o HUD
// simplesmente nao mostra nada.

const pingInterval = 2 * time.Second

var (
	pingEpoch = time.Now()
	rttMicros atomic.Int64 // media movel; 0 = ainda sem medida
)

// Ping e a latencia de ida e volta ate o host (0 = sem medida ainda).
func Ping() time.Duration { return time.Duration(rttMicros.Load()) * time.Microsecond }

// startPinger mede a latencia enquanto c for o cliente desta sessao.
func startPinger(c *Client) {
	rttMicros.Store(0)
	go func() {
		t := time.NewTicker(pingInterval)
		defer t.Stop()
		for range t.C {
			if CurrentClient != c || SessionEnded() {
				return
			}
			if AwaitingHost() {
				continue
			}
			id := time.Since(pingEpoch).Microseconds()
			SendMessage(Message{Type: MsgPing, Payload: MustMarshal(PingPayload{ID: id})})
		}
	}()
}

func handlePong(payload json.RawMessage) {
	var p PingPayload
	if json.Unmarshal(payload, &p) != nil || p.ID <= 0 {
		return
	}
	sample := time.Since(pingEpoch).Microseconds() - p.ID
	if sample < 0 {
		return
	}
	if old := rttMicros.Load(); old != 0 {
		sample = (old*3 + sample) / 4
	}
	rttMicros.Store(sample)
}

// handleJoinRejected: o host recusou a entrada (sala cheia, versao
// diferente). Volta ao menu com a frase certa, sem tentar reconectar - a
// resposta seria a mesma.
func handleJoinRejected(payload json.RawMessage) {
	var p JoinRejectedPayload
	json.Unmarshal(payload, &p)
	text := "O anfitriao recusou a entrada."
	switch p.Reason {
	case joinRejectFull:
		text = "A partida esta cheia."
	case joinRejectVersion:
		text = "Sua versao do jogo e diferente da do anfitriao. Atualizem o jogo."
	}
	log.Printf("[Cliente] entrada recusada pelo host: %s", p.Reason)
	setFailureText(text)
	failSession(FailureConnect)
}

package network

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"time"
)

// Protecoes do host, agora que qualquer pessoa com o app pode entrar numa
// sala online (a LAN tambem ganha, de graca).
//
//   - Linha maxima (maxClientLine): o host lia JSON sem limite de tamanho; uma
//     linha gigante enchia a memoria do aparelho de quem hospeda.
//   - Taxa maxima por jogador (clientMsgRate): o input legitimo e ~20/s
//     (client_input_rate.go) mais ataques e magias; uma enxurrada de
//     mensagens tomaria a CPU do host. O excesso e descartado.
//   - Sala cheia (MaxPartySize) e versao diferente (JoinPayload.Proto) sao
//     recusadas com MsgJoinRejected, e o cliente le a frase certa no menu.

// MaxPartySize e o teto de jogadores humanos numa partida, com o host.
// Casa com o relay (10 amigos por sala, relay.NewServer).
const MaxPartySize = 11

// maxClientLine e o tamanho do bufio.Reader de cada conexao (acceptLoop): uma
// linha que nao cabe nele derruba a conexao. As mensagens do cliente tem
// poucas centenas de bytes.
const maxClientLine = 64 << 10

const (
	clientMsgRate  = 120.0 // mensagens por segundo, sustentado
	clientMsgBurst = 240.0
)

var errLineTooLong = errors.New("mensagem do cliente grande demais")

// readClientMessage le uma mensagem do cliente respeitando maxClientLine.
func readClientMessage(r *bufio.Reader) (Message, error) {
	var msg Message
	line, err := r.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return msg, errLineTooLong
	}
	if err != nil {
		return msg, err
	}
	return msg, json.Unmarshal(line, &msg)
}

// allowMessage e um balde de fichas por conexao. So a goroutine de leitura
// daquela conexao chama, entao nao precisa de lock.
func (c *ClientConn) allowMessage(now time.Time) bool {
	if c.rateAt.IsZero() {
		c.rateAt, c.tokens = now, clientMsgBurst
	}
	c.tokens = min(clientMsgBurst, c.tokens+now.Sub(c.rateAt).Seconds()*clientMsgRate)
	c.rateAt = now
	if c.tokens < 1 {
		if now.Sub(c.rateWarned) > 5*time.Second {
			log.Printf("[Host] %s mandando mensagens demais; descartando o excesso", c.conn.RemoteAddr())
			c.rateWarned = now
		}
		return false
	}
	c.tokens--
	return true
}

// admitJoin decide se o join pode seguir. Recusa (e fecha a conexao depois de
// entregar o motivo) versao diferente e sala cheia; quem ja tem vaga (uma
// reconexao) sempre volta.
func (h *Host) admitJoin(c *ClientConn, payload json.RawMessage) bool {
	var join JoinPayload
	if err := json.Unmarshal(payload, &join); err != nil {
		return true // handleJoin registra o erro, como sempre fez
	}
	reason := ""
	if join.Proto != "" && join.Proto != OnlineGameTag {
		reason = joinRejectVersion
	} else {
		h.playersMutex.Lock()
		_, known := h.players[join.PlayerID]
		humans := 0
		for id := range h.players {
			if !isBotID(id) {
				humans++
			}
		}
		h.playersMutex.Unlock()
		if !known && humans >= MaxPartySize {
			reason = joinRejectFull
		}
	}
	if reason == "" {
		return true
	}
	log.Printf("[Host] entrada de %s recusada: %s", c.conn.RemoteAddr(), reason)
	data, _ := json.Marshal(Message{Type: MsgJoinRejected, Payload: MustMarshal(JoinRejectedPayload{Reason: reason})})
	c.sendThenClose(data)
	return false
}

// sendThenClose entrega uma ultima mensagem e fecha a conexao depois dela.
func (c *ClientConn) sendThenClose(data []byte) {
	if c.out == nil {
		c.sendSync(data)
		c.conn.Close()
		return
	}
	c.out.push(outItem{data: data, typ: messageType(data), closeAfter: true})
}

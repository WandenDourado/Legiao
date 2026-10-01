package network

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

// Envio do host para UM jogador, fora do laco do jogo.
//
// Antes, broadcast escrevia e dava Flush em cada conexao dentro do proprio
// laco do jogo, segurando peersMutex. Na LAN isso nunca apareceu; pela
// internet, um celular com 4G ruim enche o buffer TCP, o Flush BLOQUEIA - e o
// quadro do host inteiro para junto, travando a partida de todo mundo por
// causa de um jogador. Agora cada conexao tem uma fila e uma goroutine que
// escreve; o laco do jogo so empilha e segue.
//
// A fila respeita tres regras:
//   - SNAPSHOT (estado substituivel: jogadores, inimigos, projeteis,
//     cooldowns) nao acumula: um snapshot novo substitui o anterior do mesmo
//     tipo que ainda nao saiu. Quem esta atrasado recebe o estado mais
//     recente, nao uma fila de estados velhos - o mesmo principio de
//     client_backlog.go, agora na origem.
//   - EVENTO (combate, dialogo, viagem, reset...) nunca e descartado e nunca
//     troca de ordem; um snapshot nao pula por cima de um evento.
//   - Tudo o que esta na fila sai num lote so, com UM Flush: as tres
//     mensagens de um tique viram um pacote em vez de tres.
//
// Uma conexao que nao aceita um lote em peerWriteTimeout, ou que acumula
// maxPendingEvents, e fechada: o cliente reconecta (reconnect.go) e o resto
// da partida nunca espera por ele.

const maxPendingEvents = 1024

// peerWriteTimeout e variavel so para o teste do jogador lento nao esperar 5 s.
var peerWriteTimeout = 5 * time.Second

var errPeerClosed = errors.New("conexao com o jogador encerrada")

type outItem struct {
	data     []byte
	typ      MessageType
	snapshot bool
	deflate  bool // marcador: daqui em diante o fluxo e comprimido
	// closeAfter: fechar a conexao depois de entregar este item (recusa de
	// entrada, host_guard.go).
	closeAfter bool
}

type outbox struct {
	mu      sync.Mutex
	items   []outItem
	closed  bool
	wake    chan struct{}
	timeout time.Duration // peerWriteTimeout lido uma vez, na criacao
}

// push empilha sem bloquear. Devolve erro se a conexao ja foi encerrada.
func (o *outbox) push(it outItem) error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return errPeerClosed
	}
	replaced := false
	if it.snapshot {
		// So entre os itens DEPOIS do ultimo evento: substituir antes dele
		// faria o snapshot novo sair antes de um evento que veio antes.
		for i := len(o.items) - 1; i >= 0 && o.items[i].snapshot; i-- {
			if o.items[i].typ == it.typ {
				o.items[i], replaced = it, true
				break
			}
		}
	}
	if !replaced {
		if len(o.items) >= maxPendingEvents {
			o.closed = true
			o.mu.Unlock()
			return errPeerClosed
		}
		o.items = append(o.items, it)
	}
	o.mu.Unlock()
	select {
	case o.wake <- struct{}{}:
	default:
	}
	return nil
}

// take devolve tudo o que esta na fila; ok falso quando a fila foi fechada.
func (o *outbox) take() ([]outItem, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	items := o.items
	o.items = nil
	return items, !o.closed
}

func (o *outbox) shut() {
	o.mu.Lock()
	o.closed = true
	o.mu.Unlock()
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

// startWriter liga o envio assincrono desta conexao. acceptLoop chama para
// toda conexao real; os testes que montam um ClientConn a mao continuam no
// envio sincrono (sendSync).
func (c *ClientConn) startWriter() {
	c.out = &outbox{wake: make(chan struct{}, 1), timeout: peerWriteTimeout}
	go c.writeLoop()
}

func (c *ClientConn) stopWriter() {
	if c.out != nil {
		c.out.shut()
	}
}

func (c *ClientConn) writeLoop() {
	for {
		select {
		case <-c.out.wake:
		case <-c.done:
			return
		}
		batch, ok := c.out.take()
		if !ok {
			return
		}
		if err := c.writeBatch(batch); err != nil {
			log.Printf("[Host] envio para %s falhou (%v); fechando a conexao", c.conn.RemoteAddr(), err)
			c.out.shut()
			c.conn.Close()
			return
		}
		for _, it := range batch {
			if it.closeAfter {
				c.out.shut()
				c.conn.Close()
				return
			}
		}
	}
}

func (c *ClientConn) writeBatch(batch []outItem) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(c.out.timeout))
	for _, it := range batch {
		if it.deflate {
			if err := c.startDeflateLocked(); err != nil {
				return err
			}
			continue
		}
		if err := c.writeLineLocked(it.data); err != nil {
			return err
		}
	}
	if c.z != nil {
		if err := c.z.Flush(); err != nil {
			return err
		}
	}
	return c.writer.Flush()
}

func (c *ClientConn) writeLineLocked(data []byte) error {
	if c.z != nil {
		if _, err := c.z.Write(data); err != nil {
			return err
		}
		_, err := c.z.Write(newline)
		return err
	}
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	return c.writer.WriteByte('\n')
}

var newline = []byte{'\n'}

// startDeflateLocked escreve o marcador ainda em texto puro e liga a
// compressao para tudo o que vier depois nesta conexao.
func (c *ClientConn) startDeflateLocked() error {
	marker, _ := json.Marshal(Message{Type: MsgCompress, Payload: MustMarshal(CompressPayload{Algo: compressDeflate})})
	if err := c.writeLineLocked(marker); err != nil {
		return err
	}
	// BestSpeed: medido com o tique real (11 jogadores, 83 inimigos), ~57 us
	// por tique por jogador e ~6,7x menos bytes; o nivel padrao comprime so
	// um pouco mais (9,4x) por 3,5x mais CPU. Ver doc/network.md.
	z, err := flate.NewWriter(c.writer, flate.BestSpeed)
	if err != nil {
		return err
	}
	c.z = z
	return nil
}

// enableCompression pede a compressao desta conexao. O marcador entra na fila
// como um evento, entao o que ja estava na fila sai em texto puro antes dele.
func (c *ClientConn) enableCompression() {
	if c.out == nil {
		return // envio sincrono (testes): continua em texto puro
	}
	c.out.push(outItem{deflate: true})
}

// messageType le o tipo do JSON de uma Message ja serializada. O campo type
// e o primeiro da struct, entao basta olhar o comeco.
func messageType(data []byte) MessageType {
	const prefix = `{"type":"`
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return ""
	}
	rest := data[len(prefix):]
	if i := bytes.IndexByte(rest, '"'); i >= 0 {
		return MessageType(rest[:i])
	}
	return ""
}

package network

import (
	"bufio"
	"compress/flate"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func line(t MessageType, n int) []byte {
	b, _ := json.Marshal(Message{Type: t, Payload: MustMarshal(map[string]int{"n": n})})
	return b
}

func TestOutboxKeepsOnlyLatestSnapshotButNeverReordersEvents(t *testing.T) {
	o := &outbox{wake: make(chan struct{}, 1)}
	push := func(typ MessageType, n int) {
		o.push(outItem{data: line(typ, n), typ: typ, snapshot: isSnapshotType(typ)})
	}
	push(MsgStateUpdate, 1)
	push(MsgEnemyUpdate, 1)
	push(MsgStateUpdate, 2) // substitui o 1
	push(MsgCombatEvent, 1) // barreira
	push(MsgStateUpdate, 3) // NAO pode substituir o 2 (esta antes do evento)
	push(MsgStateUpdate, 4) // substitui o 3
	items, _ := o.take()
	var got []string
	for _, it := range items {
		got = append(got, string(it.data))
	}
	want := []string{string(line(MsgStateUpdate, 2)), string(line(MsgEnemyUpdate, 1)), string(line(MsgCombatEvent, 1)), string(line(MsgStateUpdate, 4))}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("fila =\n%v\nquer\n%v", got, want)
	}
}

func TestOutboxOverflowClosesInsteadOfGrowing(t *testing.T) {
	o := &outbox{wake: make(chan struct{}, 1)}
	for i := 0; i < maxPendingEvents; i++ {
		if err := o.push(outItem{data: line(MsgCombatEvent, i), typ: MsgCombatEvent}); err != nil {
			t.Fatal(err)
		}
	}
	if err := o.push(outItem{data: line(MsgCombatEvent, -1), typ: MsgCombatEvent}); err == nil {
		t.Fatal("fila passou do teto sem fechar a conexao")
	}
}

func newQueuedConn(t *testing.T) (*ClientConn, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	c := &ClientConn{conn: server, writer: bufio.NewWriter(server), reader: bufio.NewReader(server), done: make(chan struct{})}
	c.startWriter()
	t.Cleanup(func() { server.Close(); client.Close() })
	return c, client
}

// O fluxo: linhas em texto puro, o marcador, e dai em diante deflate - lido
// pelo MESMO startInflate que o cliente usa.
func TestCompressedStreamRoundTrip(t *testing.T) {
	c, peer := newQueuedConn(t)
	c.send(line(MsgCombatEvent, 1))
	c.enableCompression()
	for i := 2; i <= 50; i++ {
		c.send(line(MsgCombatEvent, i)) // eventos: nenhum pode sumir
	}

	peer.SetReadDeadline(time.Now().Add(3 * time.Second))
	cl := &Client{}
	r := bufio.NewReader(peer)
	n, compressed := 0, false
	for n < 50 {
		raw, err := r.ReadBytes('\n')
		if err != nil {
			t.Fatalf("depois de %d mensagens: %v", n, err)
		}
		msg, ok := parseMessage(raw)
		if !ok {
			t.Fatalf("linha corrompida: %q", raw)
		}
		if msg.Type == MsgCompress {
			r, compressed = cl.startInflate(r), true
			continue
		}
		n++
		var p map[string]int
		json.Unmarshal(msg.Payload, &p)
		if p["n"] != n {
			t.Fatalf("mensagem %d chegou fora de ordem: %v", n, p)
		}
	}
	if !compressed {
		t.Fatal("o fluxo nunca foi comprimido")
	}
}

// Um jogador que para de ler nao pode travar quem envia: o laco do jogo so
// empilha, e a conexao travada e fechada em peerWriteTimeout.
func TestStalledPeerNeverBlocksTheHost(t *testing.T) {
	prev := peerWriteTimeout
	peerWriteTimeout = 200 * time.Millisecond
	defer func() { peerWriteTimeout = prev }()
	c, _ := newQueuedConn(t) // ninguem le do outro lado

	// Abaixo do teto da fila: nada aqui pode esperar a rede.
	start := time.Now()
	for i := 0; i < maxPendingEvents/2; i++ {
		c.send(line(MsgStateUpdate, i))
		c.send(line(MsgCombatEvent, i))
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("enviar para um jogador travado levou %v", d)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.send(line(MsgCombatEvent, 0)); err == errPeerClosed {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("a conexao travada nunca foi fechada")
}

// O marcador pode chegar no meio da drenagem de snapshots atrasados: a
// drenagem para nele e avisa o readLoop.
func TestDrainStopsAtCompressionMarker(t *testing.T) {
	var buf strings.Builder
	buf.Write(line(MsgStateUpdate, 2))
	buf.WriteByte('\n')
	marker, _ := json.Marshal(Message{Type: MsgCompress, Payload: MustMarshal(CompressPayload{Algo: compressDeflate})})
	buf.Write(marker)
	buf.WriteByte('\n')
	var z strings.Builder
	zw, _ := flate.NewWriter(&z, flate.BestSpeed)
	zw.Write(append(line(MsgStateUpdate, 3), '\n'))
	zw.Flush()
	buf.WriteString(z.String())

	cl := &Client{reader: bufio.NewReader(strings.NewReader(buf.String()))}
	cl.reader.Peek(1) // como no readLoop: a leitura anterior ja encheu o buffer
	got := cl.drainStaleSnapshots(Message{Type: MsgStateUpdate})
	if !cl.inflatePending {
		t.Fatal("a drenagem nao parou no marcador")
	}
	var p map[string]int
	json.Unmarshal(got.Payload, &p)
	if p["n"] != 2 {
		t.Fatalf("drenagem devolveu %v, quer o snapshot 2", p)
	}
	r := cl.startInflate(cl.reader)
	raw, err := r.ReadBytes('\n')
	if err != nil || messageType(raw) != MsgStateUpdate {
		t.Fatalf("depois do marcador: %q, %v", raw, err)
	}
}

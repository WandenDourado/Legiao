package network

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func TestReadClientMessageRejectsHugeLines(t *testing.T) {
	r := bufio.NewReaderSize(strings.NewReader(strings.Repeat("x", 100)+"\n"), 16)
	if _, err := readClientMessage(r); err != errLineTooLong {
		t.Fatalf("erro = %v, quer errLineTooLong", err)
	}
	ok := bufio.NewReaderSize(strings.NewReader(`{"type":"ping","payload":{"id":1}}`+"\n"), 64)
	if msg, err := readClientMessage(ok); err != nil || msg.Type != MsgPing {
		t.Fatalf("mensagem normal: %v, %v", msg, err)
	}
}

func TestMessageFloodIsThrottled(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	c := &ClientConn{conn: server}
	now := time.Unix(0, 0)
	allowed := 0
	for i := 0; i < 1000; i++ { // 1000 mensagens no mesmo instante
		if c.allowMessage(now) {
			allowed++
		}
	}
	if allowed != int(clientMsgBurst) {
		t.Fatalf("rajada permitiu %d, quer %d", allowed, int(clientMsgBurst))
	}
	// Ritmo normal de jogo (60/s) nunca e cortado.
	for i := 0; i < 600; i++ {
		now = now.Add(time.Second / 60)
		if !c.allowMessage(now) && i > int(clientMsgBurst) {
			t.Fatalf("mensagem %d de um cliente normal foi descartada", i)
		}
	}
}

func joinMsg(id, proto string) json.RawMessage {
	return MustMarshal(JoinPayload{PlayerID: id, Color: "red", Character: "mago", Proto: proto})
}

func TestAdmitJoinRejectsFullPartyAndOtherVersions(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{}}
	for i := 0; i < MaxPartySize; i++ {
		id := string(rune('a' + i))
		h.players[id] = &PlayerState{PlayerID: id}
	}
	c, received := newRecordingClientConn(t)
	if h.admitJoin(c, joinMsg("novo", OnlineGameTag)) {
		t.Fatal("entrou numa partida cheia")
	}
	msg := <-received
	var p JoinRejectedPayload
	json.Unmarshal(msg.Payload, &p)
	if msg.Type != MsgJoinRejected || p.Reason != joinRejectFull {
		t.Fatalf("recusa = %v %v", msg.Type, p)
	}
	c2, _ := newRecordingClientConn(t)
	if !h.admitJoin(c2, joinMsg("a", OnlineGameTag)) {
		t.Fatal("reconexao de quem ja tinha vaga foi recusada")
	}
	h2 := &Host{players: map[string]*PlayerState{}}
	c3, received3 := newRecordingClientConn(t)
	if h2.admitJoin(c3, joinMsg("x", "net999")) {
		t.Fatal("versao diferente entrou")
	}
	msg = <-received3
	json.Unmarshal(msg.Payload, &p)
	if p.Reason != joinRejectVersion {
		t.Fatalf("motivo = %q", p.Reason)
	}
	c4, _ := newRecordingClientConn(t)
	if !h2.admitJoin(c4, joinMsg("antigo", "")) {
		t.Fatal("cliente antigo (sem Proto) foi recusado")
	}
}

func TestPongAveragesRoundTrip(t *testing.T) {
	prev := pingEpoch
	pingEpoch = time.Now().Add(-time.Hour)
	defer func() { pingEpoch = prev }()
	rttMicros.Store(0)
	id := time.Since(pingEpoch).Microseconds() - 40_000 // "mandado" ha 40 ms
	handlePong(MustMarshal(PingPayload{ID: id}))
	if p := Ping(); p < 35*time.Millisecond || p > 60*time.Millisecond {
		t.Fatalf("ping = %v, quer ~40 ms", p)
	}
}

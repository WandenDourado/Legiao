package network

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// newRecordingClientConn is like newPipeClientConn, but instead of silently
// draining the other half of the pipe it decodes every newline-delimited
// message the host writes and hands them back through the returned channel,
// so a test can assert on WHICH messages a join produced.
func newRecordingClientConn(t *testing.T) (*ClientConn, <-chan Message) {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })

	received := make(chan Message, 32)
	go func() {
		scanner := bufio.NewScanner(client)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			var msg Message
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			received <- msg
		}
		close(received)
	}()

	return &ClientConn{
		conn:   server,
		writer: bufio.NewWriter(server),
		reader: bufio.NewReader(server),
		done:   make(chan struct{}),
	}, received
}

// drainForTravel closes the server half of c's pipe (there is nothing more
// this test will send) so the recording goroutine hits EOF, drains
// `received` to completion — the goroutine only closes it after EOF, so a
// full range blocks exactly as long as it needs to and never races the
// write — and returns the first MsgTravel found, if any.
func drainForTravel(t *testing.T, c *ClientConn, received <-chan Message) (TravelPayload, bool) {
	t.Helper()
	c.conn.Close()
	var travel TravelPayload
	found := false
	for msg := range received {
		if found || msg.Type != MsgTravel {
			continue
		}
		if err := json.Unmarshal(msg.Payload, &travel); err != nil {
			t.Fatalf("MsgTravel payload nao decodifica: %v", err)
		}
		found = true
	}
	return travel, found
}

// TestFreshJoinReceivesCurrentMap prova o defeito do item 3 (A2): antes do
// conserto, sendCurrentMapTo so era chamado no ramo isReconnect, entao um
// jogador novo nunca recebia MsgTravel e ficava com o mapa que a maquina dele
// carregou antes do menu (world_01) desenhado por baixo das coordenadas do
// mapa em que o host realmente esta (aqui, world_03).
func TestFreshJoinReceivesCurrentMap(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players:       map[string]*PlayerState{},
		PlayerSpawn:   rl.NewVector2(10, 20),
		stageMap:      "assets/maps/world_03.json",
	}
	c, received := newRecordingClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "novo", Color: "green", Character: "mago"}))

	travel, ok := drainForTravel(t, c, received)
	if !ok {
		t.Fatal("join novo nao recebeu MsgTravel; fica preso no mapa que a maquina carregou antes do menu")
	}
	if travel.TargetMap != "assets/maps/world_03.json" {
		t.Errorf("TargetMap = %q, esperado assets/maps/world_03.json", travel.TargetMap)
	}
	// Um join novo nao e uma reconexao: o PlayerState acabou de ser criado em
	// h.PlayerSpawn (e sendStateToClient ja mandou essa posicao antes deste
	// MsgTravel), entao Reconnect deve ser false — dizer a verdade sobre o
	// que aconteceu, mesmo que ApplyPendingTravel trate os dois casos do
	// mesmo jeito quando o mapa de origem ja e o mesmo.
	if travel.Reconnect {
		t.Error("Reconnect = true para um join novo; isso nao e uma reconexao")
	}
}

// TestReconnectStillMarksTravelAsReconnect prova que o caminho antigo
// continua funcionando: quem reconecta ainda recebe Reconnect=true, que
// ApplyPendingTravel usa para pousar na posicao preservada pelo host em vez
// do spawn do mapa.
func TestReconnectStillMarksTravelAsReconnect(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"p1": {PlayerID: "p1", X: 777, Y: 888, Color: "blue", Character: "mago"},
		},
		stageMap: "assets/maps/world_04.json",
	}
	h.markAbsent("p1")
	c, received := newRecordingClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "p1", Color: "red", Character: "arqueiro"}))

	travel, ok := drainForTravel(t, c, received)
	if !ok {
		t.Fatal("reconexao nao recebeu MsgTravel")
	}
	if !travel.Reconnect {
		t.Error("Reconnect = false para uma reconexao; ApplyPendingTravel nao vai pousar na posicao preservada")
	}
}

// TestJoinWithoutStageMapSendsNoTravel garante que a ausencia de mapa atual
// (host ainda no menu, ou stageMap nunca setado) continua nao mandando
// MsgTravel nenhum, em vez de um travel para "".
func TestJoinWithoutStageMapSendsNoTravel(t *testing.T) {
	h := &Host{EntityManager: entity.NewEntityManager(), players: map[string]*PlayerState{}, PlayerSpawn: rl.NewVector2(0, 0)}
	c, received := newRecordingClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "novo", Color: "green", Character: "mago"}))

	if _, ok := drainForTravel(t, c, received); ok {
		t.Error("MsgTravel enviado sem stageMap definido")
	}
}

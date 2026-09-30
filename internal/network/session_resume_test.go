package network

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// REINICIAR E VOLTAR, pela internet (29/09/2026). Os dois lados que o teste
// em rede local do Gui nao cobria, de ponta a ponta sobre um relay de verdade.

func startTestRelay(t *testing.T) (string, *relay.Server) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	s := relay.NewServer()
	go s.Serve(ln)
	useTestRelay(t, ln.Addr().String())
	return ln.Addr().String(), s
}

func startOnlineTestHost(t *testing.T, id, code, secret string) (*Host, *OnlineRoom) {
	t.Helper()
	MarkFrame()
	h, err := StartHost(0, id, "#ffffff", "mago", rl.NewVector2(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	prev := CurrentHost
	CurrentHost = h
	t.Cleanup(func() { CurrentHost = prev })
	room, err := OpenOnlineRoomAs("Gui", code, secret)
	if err != nil {
		h.Close()
		t.Fatal(err)
	}
	h.ServeOnline(room)
	return h, room
}

// joinOnline entra na sala como um cliente de verdade: disca pelo relay e
// manda o MsgJoin em JSON puro (sem pedir compressao).
func joinOnline(t *testing.T, code, playerID string) net.Conn {
	t.Helper()
	c, err := dialGame(OnlineRoomAddr(code), 2*time.Second)
	if err != nil {
		t.Fatalf("%s nao entrou na sala %s: %v", playerID, code, err)
	}
	line, _ := json.Marshal(Message{Type: MsgJoin, Payload: MustMarshal(JoinPayload{
		PlayerID: playerID, Color: "green", Character: "arqueiro", Proto: OnlineGameTag,
	})})
	if _, err := c.Write(append(line, '\n')); err != nil {
		t.Fatal(err)
	}
	// O host escreve snapshots; ninguem le do lado do teste, e o pipe do
	// relay nao pode entupir.
	go func() {
		buf := make([]byte, 32*1024)
		for {
			if _, err := c.Read(buf); err != nil {
				return
			}
		}
	}()
	return c
}

func waitResume(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(what)
}

func playerSnapshot(h *Host, id string) (PlayerState, bool) {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	p, ok := h.players[id]
	if !ok {
		return PlayerState{}, false
	}
	return *p, true
}

// CLIENTE reinicia: o processo morre (a conexao do relay cai sem aviso) e o
// processo novo disca o MESMO codigo com o MESMO PlayerID. O host tem de ver
// uma reconexao — posicao e vida de antes — e nao um jogador novo.
func TestOnlineClientRestartKeepsItsSlot(t *testing.T) {
	startTestRelay(t)
	h, room := startOnlineTestHost(t, "host-resume-client", "", "")
	defer h.Close()
	defer ShutdownOnline()
	code := room.Code()

	first := joinOnline(t, code, "guest-1")
	waitResume(t, "o host nao registrou o convidado", func() bool {
		_, ok := playerSnapshot(h, "guest-1")
		return ok
	})
	h.playersMutex.Lock()
	h.players["guest-1"].X, h.players["guest-1"].Y, h.players["guest-1"].Health = 777, 888, 42
	h.playersMutex.Unlock()

	first.Close() // o app morreu no REINICIAR
	waitResume(t, "o host nao marcou o convidado como ausente", func() bool {
		p, _ := playerSnapshot(h, "guest-1")
		return p.Absent
	})

	second := joinOnline(t, code, "guest-1")
	defer second.Close()
	waitResume(t, "o convidado nao voltou", func() bool {
		p, _ := playerSnapshot(h, "guest-1")
		return !p.Absent
	})
	p, _ := playerSnapshot(h, "guest-1")
	if p.X != 777 || p.Y != 888 || p.Health != 42 {
		t.Fatalf("voltou como jogador novo: X=%d Y=%d vida=%v", p.X, p.Y, p.Health)
	}
}

// HOST reinicia: a sala cai com ele, o processo novo pede o MESMO codigo com
// o segredo guardado, e o amigo — que so conhece o codigo antigo — entra.
func TestOnlineHostRestartReopensSameCode(t *testing.T) {
	addr, _ := startTestRelay(t)
	h1, room1 := startOnlineTestHost(t, "host-resume-1", "", "")
	code, secret := room1.Code(), OnlineRoomSecret()
	if secret == "" {
		t.Fatal("o segredo da sala nao foi guardado para o bilhete")
	}
	ShutdownOnline() // o controle fecha, como quando o processo morre
	h1.Close()
	waitResume(t, "a sala antiga nao saiu do relay", func() bool {
		rooms, _ := relay.List(addr, OnlineGameTag, time.Second)
		return len(rooms) == 0
	})

	h2, room2 := startOnlineTestHost(t, "host-resume-2", code, secret)
	defer h2.Close()
	defer ShutdownOnline()
	if room2.Code() != code {
		t.Fatalf("a sala voltou com %s; os amigos discam %s", room2.Code(), code)
	}

	guest := joinOnline(t, code, "guest-2")
	defer guest.Close()
	waitResume(t, "o amigo nao entrou na sala reaberta", func() bool {
		_, ok := playerSnapshot(h2, "guest-2")
		return ok
	})
}

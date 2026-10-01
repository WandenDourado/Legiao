package network

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Um amigo entra pela sala online e chega ao MESMO acceptLoop da LAN, com um
// RemoteAddr proprio - e um segundo amigo nao pisa no primeiro em h.peers.
func TestOnlineRoomReachesHostAcceptLoop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go relay.NewServer().Serve(ln)

	useTestRelay(t, ln.Addr().String())

	MarkFrame() // o laco do jogo "rodando": sem isso o host nao responde PING
	h, err := StartHost(0, "host-online-test", "#ffffff", "mago", rl.NewVector2(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	room, err := OpenOnlineRoom("Gui")
	if err != nil {
		t.Fatal(err)
	}
	h.ServeOnline(room)
	defer ShutdownOnline()
	code := room.Code()
	if rooms, _ := relay.List(ln.Addr().String(), OnlineGameTag, time.Second); len(rooms) != 1 || rooms[0].Code != code {
		t.Fatalf("sala nao listada: %+v", rooms)
	}

	for i := 0; i < 2; i++ {
		c, err := dialGame(OnlineRoomAddr(code), time.Second)
		if err != nil {
			t.Fatalf("amigo %d: %v", i, err)
		}
		defer c.Close()
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		h.peersMutex.Lock()
		n := 0
		for addr := range h.peers {
			if strings.HasPrefix(addr, "relay/") {
				n++
			}
		}
		h.peersMutex.Unlock()
		if n == 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("o host nao registrou os dois amigos vindos do relay")
}

// Fim do jogo: a sala sai da lista na hora, sem esperar o sinal de vida.
func TestShutdownOnlineRemovesRoomImmediately(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go relay.NewServer().Serve(ln)
	useTestRelay(t, ln.Addr().String())

	MarkFrame()
	h, err := StartHost(0, "host-shutdown-test", "#ffffff", "mago", rl.NewVector2(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	prevHost := CurrentHost
	CurrentHost = h
	defer func() { CurrentHost = prevHost }()
	room, err := OpenOnlineRoom("Gui")
	if err != nil {
		t.Fatal(err)
	}
	h.ServeOnline(room)
	ShutdownOnline()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if rooms, _ := relay.List(ln.Addr().String(), OnlineGameTag, time.Second); len(rooms) == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("sala continuou listada depois do fim do jogo")
}

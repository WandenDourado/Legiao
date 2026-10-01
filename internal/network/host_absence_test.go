package network

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// newPipeClientConn returns a ClientConn backed by a real connection (a
// net.Pipe half) so handleJoin's sendStateToClient/sendCurrentMapTo/
// sendDialogueTo calls have something to write into instead of panicking on
// a nil writer. The other half is drained in the background for the life of
// the test.
func newPipeClientConn(t *testing.T) *ClientConn {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := client.Read(buf); err != nil {
				return
			}
		}
	}()
	return &ClientConn{
		conn:   server,
		writer: bufio.NewWriter(server),
		reader: bufio.NewReader(server),
		done:   make(chan struct{}),
	}
}

func TestRejoinPreservesPositionHealthAndDeath(t *testing.T) {
	h := &Host{EntityManager: entity.NewEntityManager(), players: map[string]*PlayerState{
		"p1": {
			PlayerID:  "p1",
			X:         777,
			Y:         888,
			Color:     "blue",
			Character: "mago",
			Health:    0,
			MaxHealth: 100,
			IsDead:    true,
			RespawnIn: 12.5,
		},
	}}
	h.markAbsent("p1") // simulates the connection having dropped earlier

	c := newPipeClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "p1", Color: "red", Character: "arqueiro"}))

	h.playersMutex.Lock()
	p := *h.players["p1"]
	h.playersMutex.Unlock()

	if p.X != 777 || p.Y != 888 {
		t.Errorf("posicao = (%d,%d), esperado (777,888): rejoin nao pode teleportar", p.X, p.Y)
	}
	if !p.IsDead || p.RespawnIn != 12.5 {
		t.Errorf("IsDead=%v RespawnIn=%v, esperado morto com 12.5s restantes", p.IsDead, p.RespawnIn)
	}
	if p.Health != 0 {
		t.Errorf("Health = %v, esperado 0 (nao curado pelo rejoin)", p.Health)
	}
	if p.Absent {
		t.Error("rejoin deveria limpar a ausencia")
	}
	if c.playerID != "p1" {
		t.Errorf("c.playerID = %q, esperado p1", c.playerID)
	}
}

func TestRejoinIsNotConfusedWithFreshJoin(t *testing.T) {
	h := &Host{EntityManager: entity.NewEntityManager(), players: map[string]*PlayerState{}, PlayerSpawn: rl.NewVector2(10, 20)}
	c := newPipeClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "novo", Color: "green", Character: "mago"}))

	h.playersMutex.Lock()
	p, ok := h.players["novo"]
	h.playersMutex.Unlock()
	if !ok {
		t.Fatal("join novo nao criou o jogador")
	}
	if p.IsDead || p.Health != 100 {
		t.Errorf("jogador novo deveria nascer vivo com 100 de vida, veio IsDead=%v Health=%v", p.IsDead, p.Health)
	}
}

func TestAbsenceExpiresAfterGraceAndOnlyThen(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{"p1": {PlayerID: "p1"}}}
	h.markAbsent("p1")

	// Quase no limite: nao pode ter sumido ainda.
	h.playersMutex.Lock()
	h.players["p1"].absentSince = time.Now().Add(-(ReconnectGrace - time.Second))
	h.playersMutex.Unlock()
	h.tickAbsence()
	h.playersMutex.Lock()
	_, stillThere := h.players["p1"]
	h.playersMutex.Unlock()
	if !stillThere {
		t.Fatal("removido antes do fim da janela de reconexao")
	}

	// Passou da janela: agora sim.
	h.playersMutex.Lock()
	h.players["p1"].absentSince = time.Now().Add(-(ReconnectGrace + time.Second))
	h.playersMutex.Unlock()
	h.tickAbsence()
	h.playersMutex.Lock()
	_, stillThere = h.players["p1"]
	h.playersMutex.Unlock()
	if stillThere {
		t.Error("nao removido depois de exceder a janela de reconexao")
	}
}

func TestPresentPlayersExcludesAbsent(t *testing.T) {
	RemotePlayersMutex.Lock()
	RemotePlayers = map[string]PlayerState{
		"a": {PlayerID: "a"},
		"b": {PlayerID: "b", Absent: true},
	}
	RemotePlayersMutex.Unlock()
	t.Cleanup(func() {
		RemotePlayersMutex.Lock()
		RemotePlayers = nil
		RemotePlayersMutex.Unlock()
	})

	present := PresentPlayers()
	if _, ok := present["b"]; ok {
		t.Error("jogador ausente apareceu em PresentPlayers")
	}
	if _, ok := present["a"]; !ok {
		t.Error("jogador presente sumiu de PresentPlayers")
	}
	all := GetAllPlayers()
	if _, ok := all["b"]; !ok {
		t.Error("GetAllPlayers deve continuar mostrando o ausente (desenho/HUD)")
	}
}

func TestCheckGameOverIgnoresAbsentPlayers(t *testing.T) {
	// Um ausente VIVO e parado nao pode segurar o fim: so quem esta presente
	// conta.
	players := map[string]*PlayerState{
		"absent-alive": {PlayerID: "absent-alive", IsDead: false, Absent: true},
		"present-dead": {PlayerID: "present-dead", IsDead: true},
	}
	if !checkGameOver(players) {
		t.Error("Game Over deveria disparar: o unico jogador PRESENTE esta morto")
	}

	// Ninguem presente: nao e uma derrota, e a partida esperando alguem
	// voltar.
	onlyAbsent := map[string]*PlayerState{
		"a": {PlayerID: "a", Absent: true},
	}
	if checkGameOver(onlyAbsent) {
		t.Error("sem ninguem presente nao e Game Over")
	}

	// Um presente vivo continua segurando o fim normalmente.
	mixed := map[string]*PlayerState{
		"absent-dead": {PlayerID: "absent-dead", IsDead: true, Absent: true},
		"present-ok":  {PlayerID: "present-ok", IsDead: false},
	}
	if checkGameOver(mixed) {
		t.Error("jogador presente e vivo nao pode dar Game Over")
	}
}

func TestDuplicateConnectionClosesOldOneWithoutTouchingNewSlot(t *testing.T) {
	h := &Host{EntityManager: entity.NewEntityManager(), players: map[string]*PlayerState{}, peers: map[string]*ClientConn{}}

	oldConn := newPipeClientConn(t)
	oldConn.playerID = "p1"
	h.peers["old-addr"] = oldConn

	newConn := newPipeClientConn(t)
	h.peers["new-addr"] = newConn

	h.handleJoin(newConn, MustMarshal(JoinPayload{PlayerID: "p1", Color: "red", Character: "mago"}))

	if !oldConn.superseded.Load() {
		t.Error("a conexao antiga deveria ter sido marcada como substituida")
	}
	if newConn.superseded.Load() {
		t.Error("a conexao nova nao pode se marcar substituida")
	}

	// A conexao antiga "cai" (o decode dela falha, como acontece de verdade
	// quando o Close() do supersedeConnection chega). O defer do
	// handleClient dela nao pode apagar nem marcar ausente o slot que agora
	// pertence a conexao nova.
	h.playersMutex.Lock()
	if oldConn.playerID != "" && !oldConn.superseded.Load() {
		h.markAbsent(oldConn.playerID)
	}
	h.playersMutex.Unlock()

	h.playersMutex.Lock()
	p, ok := h.players["p1"]
	h.playersMutex.Unlock()
	if !ok {
		t.Fatal("o slot do jogador reconectado sumiu")
	}
	if p.Absent {
		t.Error("o slot da conexao NOVA foi marcado ausente pela limpeza da conexao antiga")
	}
}

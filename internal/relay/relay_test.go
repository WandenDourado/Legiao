package relay

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const tag = "net1"

func startServer(t *testing.T, tune func(*Server)) (string, *Server) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer()
	s.PingInterval = 30 * time.Millisecond
	s.HostSilence = 200 * time.Millisecond
	if tune != nil {
		tune(s)
	}
	go s.Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), s
}

func openRoom(t *testing.T, addr, name string, alive func() bool) *Listener {
	t.Helper()
	l, err := Host(addr, tag, name, time.Second, alive)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go echoHost(l)
	return l
}

// echoHost aceita conexoes e devolve cada linha com o RemoteAddr na frente.
func echoHost(l *Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			r := bufio.NewReader(c)
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				io.WriteString(c, c.RemoteAddr().String()+" "+line)
			}
		}(c)
	}
}

func roundTrip(t *testing.T, c net.Conn, msg string) string {
	t.Helper()
	c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(c, msg+"\n"); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(line)
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(what)
}

func rooms(t *testing.T, addr, tg string) []Room {
	t.Helper()
	r, err := List(addr, tg, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestJoinPipesBytesAndGivesEachPeerItsOwnAddr(t *testing.T) {
	addr, _ := startServer(t, nil)
	l := openRoom(t, addr, "Sala do Gui", nil)
	a, err := Join(addr, l.Code(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Join(addr, l.Code(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	ra, rb := roundTrip(t, a, `{"type":"join"}`), roundTrip(t, b, `{"type":"join"}`)
	if !strings.HasPrefix(ra, "relay/") || !strings.HasSuffix(ra, `{"type":"join"}`) {
		t.Fatalf("eco inesperado: %q", ra)
	}
	if strings.Fields(ra)[0] == strings.Fields(rb)[0] {
		t.Fatalf("dois jogadores com o mesmo RemoteAddr: %q", ra)
	}
	got := rooms(t, addr, tag)
	if len(got) != 1 || got[0].Players != 2 || got[0].Max != 10 {
		t.Fatalf("lista = %+v", got)
	}
}

func TestListFiltersByGameTagAndCleansName(t *testing.T) {
	addr, _ := startServer(t, nil)
	l := openRoom(t, addr, "  Sala\tdo Gui  ", nil)
	if got := rooms(t, addr, tag); len(got) != 1 || got[0].Code != l.Code() || got[0].Name != "Salado Gui" {
		t.Fatalf("salas = %+v", got)
	}
	if got := rooms(t, addr, "net2"); len(got) != 0 {
		t.Fatalf("outra versao do jogo viu a sala: %+v", got)
	}
}

func TestClosingTheHostKillsRoomAndPlayers(t *testing.T) {
	addr, _ := startServer(t, nil)
	l := openRoom(t, addr, "Gui", nil)
	c, err := Join(addr, l.Code(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	roundTrip(t, c, "oi")
	l.Close()
	eventually(t, "sala continuou listada", func() bool { return len(rooms(t, addr, tag)) == 0 })
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.Read(make([]byte, 1)); err == nil {
		t.Fatal("jogador continuou conectado a uma sala encerrada")
	}
}

// App em segundo plano ou travado: processo vivo, conexao aberta, mas o jogo
// nao roda. Sem PONG, a sala tem de morrer - e nao voltar sozinha.
func TestFrozenHostLosesRoomAndDoesNotReRegister(t *testing.T) {
	addr, _ := startServer(t, nil)
	var running atomic.Bool
	running.Store(true)
	l := openRoom(t, addr, "Gui", running.Load)
	code := l.Code()
	running.Store(false)
	eventually(t, "sala de host parado continuou viva", func() bool { return len(rooms(t, addr, tag)) == 0 })
	time.Sleep(300 * time.Millisecond)
	if len(rooms(t, addr, tag)) != 0 {
		t.Fatal("host parado registrou a sala de novo")
	}
	running.Store(true)
	eventually(t, "sala nao voltou com o jogo", func() bool {
		r := rooms(t, addr, tag)
		return len(r) == 1 && r[0].Code == code
	})
}

func TestHostReRegistersWithSameCode(t *testing.T) {
	addr, s := startServer(t, nil)
	l := openRoom(t, addr, "Gui", nil)
	code := l.Code()
	s.mu.Lock()
	s.rooms[code].ctrl.Close() // queda de rede no controle
	s.mu.Unlock()
	eventually(t, "host nao voltou com o mesmo codigo", func() bool {
		c, err := Join(addr, code, time.Second)
		if err != nil {
			return false
		}
		defer c.Close()
		return strings.HasSuffix(roundTrip(t, c, "oi"), "oi")
	})
}

func TestCodeCannotBeTakenWithoutSecret(t *testing.T) {
	addr, _ := startServer(t, nil)
	l := openRoom(t, addr, "Gui", nil)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	writeLine(c, "HOST "+Version+" "+l.Code()+" "+NewSecret()+" "+tag+" Intruso")
	if line, _ := readLine(c); line == "ROOM "+l.Code() {
		t.Fatal("outro host tomou o codigo sem o segredo")
	}
}

func TestLimits(t *testing.T) {
	t.Run("sala cheia", func(t *testing.T) {
		addr, _ := startServer(t, func(s *Server) { s.MaxPlayersPerRoom = 1 })
		l := openRoom(t, addr, "Gui", nil)
		c, err := Join(addr, l.Code(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := Join(addr, l.Code(), time.Second); err == nil || !strings.Contains(err.Error(), "cheia") {
			t.Fatalf("segundo jogador entrou numa sala de 1: %v", err)
		}
	})
	t.Run("maximo de salas", func(t *testing.T) {
		addr, _ := startServer(t, func(s *Server) { s.MaxRooms = 1 })
		openRoom(t, addr, "A", nil)
		if _, err := Host(addr, tag, "B", time.Second, nil); err == nil {
			t.Fatal("segunda sala aberta com MaxRooms=1")
		}
	})
	t.Run("teto mensal", func(t *testing.T) {
		addr, s := startServer(t, func(s *Server) { s.MonthlyByteBudget = 10 })
		l := openRoom(t, addr, "Gui", nil)
		c, err := Join(addr, l.Code(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip(t, c, "passa do teto de 10 bytes")
		c.Close()
		if _, err := Join(addr, l.Code(), time.Second); err == nil || !strings.Contains(err.Error(), "limite mensal") {
			t.Fatalf("entrou depois do teto mensal: %v (%s)", err, s.Stats())
		}
	})
	t.Run("conexoes por ip", func(t *testing.T) {
		addr, _ := startServer(t, func(s *Server) { s.MaxConnsPerIP = 2 })
		var open []net.Conn
		for i := 0; i < 3; i++ {
			c, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatal(err)
			}
			open = append(open, c)
		}
		defer func() {
			for _, c := range open {
				c.Close()
			}
		}()
		open[2].SetReadDeadline(time.Now().Add(time.Second))
		if _, err := open[2].Read(make([]byte, 1)); err == nil {
			t.Fatal("terceira conexao do mesmo IP foi aceita")
		}
	})
	t.Run("partida ociosa", func(t *testing.T) {
		addr, _ := startServer(t, func(s *Server) { s.PipeIdle = 150 * time.Millisecond })
		l := openRoom(t, addr, "Gui", nil)
		c, err := Join(addr, l.Code(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := c.Read(make([]byte, 1)); err == nil {
			t.Fatal("conexao ociosa nao foi derrubada")
		}
		eventually(t, "vaga da conexao ociosa nao foi liberada", func() bool {
			r := rooms(t, addr, tag)
			return len(r) == 1 && r[0].Players == 0
		})
	})
}

func TestStatsAndOldProtocol(t *testing.T) {
	addr, _ := startServer(t, nil)
	openRoom(t, addr, "Gui", nil)
	st, err := Stats(addr, time.Second)
	if err != nil || !strings.Contains(st, "salas=1/50") {
		t.Fatalf("stats = %q, %v", st, err)
	}
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	writeLine(c, "JOIN LEGIAO1 ABCDE")
	c.SetDeadline(time.Now().Add(time.Second))
	if line, _ := readLine(c); !strings.HasPrefix(line, "ERR") {
		t.Fatalf("protocolo antigo aceito: %q", line)
	}
}

func TestReadLineDoesNotConsumePastNewline(t *testing.T) {
	r := strings.NewReader("OK\n{\"type\":\"state\"}\n")
	if line, _ := readLine(r); line != "OK" {
		t.Fatalf("linha = %q", line)
	}
	if rest, _ := io.ReadAll(r); string(rest) != "{\"type\":\"state\"}\n" {
		t.Fatalf("readLine engoliu bytes do jogo: %q", rest)
	}
}

func TestRefusalsAreTyped(t *testing.T) {
	addr, _ := startServer(t, nil)
	_, err := Join(addr, "ZZZZZ", time.Second)
	var r *RefusedError
	if !errors.As(err, &r) || r.Reason != ReasonRoomNotFound {
		t.Fatalf("erro = %#v", err)
	}
}

func TestConnectedTracksTheControlConnection(t *testing.T) {
	addr, s := startServer(t, nil)
	l := openRoom(t, addr, "Gui", nil)
	if !l.Connected() {
		t.Fatal("sala recem-registrada nao aparece como conectada")
	}
	s.mu.Lock()
	s.rooms[l.Code()].ctrl.Close()
	s.mu.Unlock()
	eventually(t, "queda do controle nao foi notada", func() bool { return !l.Connected() })
	eventually(t, "nao voltou a conectar", l.Connected)
}

// O host reiniciou o app: a sala antiga caiu com ele e o processo novo pede o
// MESMO codigo. Tem de voltar com ele — e so assim que os amigos, discando o
// codigo antigo dentro da janela de reconexao, reencontram a partida.
func TestHostAsReclaimsCodeAfterRoomDied(t *testing.T) {
	addr, s := startServer(t, nil)
	first := openRoom(t, addr, "Gui", nil)
	code, secret := first.Code(), first.Secret()
	first.Close()
	eventually(t, "sala antiga nao caiu", func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.rooms[code] == nil
	})

	again, err := HostAs(addr, tag, "Gui", code, secret, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if again.Code() != code {
		t.Fatalf("voltou com %s, queria %s", again.Code(), code)
	}
}

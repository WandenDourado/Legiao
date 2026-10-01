package relay

import (
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Listener e o lado do HOST: um net.Listener cujas conexoes chegam pelo relay.
// network.Host o usa ao lado do listener TCP da LAN, e nada mais no host
// precisa saber de onde veio cada jogador.
type Listener struct {
	server, tag, name, secret string
	timeout                   time.Duration
	// alive diz se o jogo do host esta rodando de verdade. Sem ele vivo o
	// Listener nao responde PING nem se registra de novo, e a sala morre no
	// servidor em HostSilence.
	alive func() bool

	// connected diz se a sala esta registrada no servidor AGORA. Falso
	// entre uma queda do controle e o novo registro (network mostra isso ao
	// host: "servidor online fora - reconectando").
	connected atomic.Bool

	mu    sync.Mutex
	code  string
	ctrl  net.Conn
	conns chan net.Conn
	quit  chan struct{}
	once  sync.Once
}

// Host registra uma sala em server e devolve o listener. Volta so depois do
// primeiro registro, para quem chama saber na hora se o servidor respondeu;
// quedas posteriores sao tratadas em segundo plano, sempre com o MESMO codigo,
// para que um cliente reconectando (network/reconnect.go) ache a sala.
//
// tag separa versoes do jogo (network.OnlineGameTag). alive pode ser nil
// (sempre vivo).
func Host(server, tag, name string, timeout time.Duration, alive func() bool) (*Listener, error) {
	return HostAs(server, tag, name, "", "", timeout, alive)
}

// HostAs e Host pedindo um codigo JA USADO, com o segredo de quando ele foi
// aberto. E a volta de um host que reiniciou o app (network, "REINICIAR E
// VOLTAR", 29/09/2026): o servidor devolve o mesmo codigo — reassumindo a
// sala se ela ainda existe (mesmo segredo) ou pegando o codigo livre de novo
// se ela ja caiu — e os amigos, ainda na janela de reconexao discando esse
// codigo, voltam sozinhos. Codigo vazio = um novo; segredo vazio = um novo.
// Se o codigo foi tomado nesse meio tempo, a sala nasce com outro.
func HostAs(server, tag, name, code, secret string, timeout time.Duration, alive func() bool) (*Listener, error) {
	if alive == nil {
		alive = func() bool { return true }
	}
	if secret == "" {
		secret = NewSecret()
	}
	l := &Listener{
		server:  server,
		tag:     tag,
		alive:   alive,
		name:    CleanName(name),
		secret:  secret,
		code:    code,
		timeout: timeout,
		conns:   make(chan net.Conn, 8),
		quit:    make(chan struct{}),
	}
	if err := l.register(); err != nil {
		return nil, err
	}
	go l.run()
	return l, nil
}

// Connected diz se a sala esta registrada no servidor neste momento.
func (l *Listener) Connected() bool { return l.connected.Load() }

// Secret e o segredo que prova a posse do codigo — guardado por quem quer
// voltar com a MESMA sala depois de reiniciar (HostAs).
func (l *Listener) Secret() string { return l.secret }

// Code e o codigo da sala no relay.
func (l *Listener) Code() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.code
}

func (l *Listener) register() error {
	c, err := net.DialTimeout("tcp", l.server, l.timeout)
	if err != nil {
		return err
	}
	keepAlive(c)
	want := l.Code()
	if want == "" {
		want = "-"
	}
	c.SetDeadline(time.Now().Add(l.timeout))
	if err := writeLine(c, strings.Join([]string{"HOST", Version, want, l.secret, l.tag, l.name}, " ")); err != nil {
		c.Close()
		return err
	}
	line, err := readLine(c)
	if err != nil {
		c.Close()
		return err
	}
	if !strings.HasPrefix(line, "ROOM ") {
		c.Close()
		return refused(line)
	}
	c.SetDeadline(time.Time{})
	l.mu.Lock()
	l.code, l.ctrl = strings.TrimPrefix(line, "ROOM "), c
	l.mu.Unlock()
	l.connected.Store(true)
	log.Printf("[relay] sala online %s registrada em %s", l.Code(), l.server)
	return nil
}

// run le a conexao de controle; se ela cair, registra de novo com recuo.
func (l *Listener) run() {
	backoff := time.Second
	for {
		l.mu.Lock()
		c := l.ctrl
		l.mu.Unlock()
		if c != nil {
			l.serveControl(c)
			backoff = time.Second
		}
		select {
		case <-l.quit:
			return
		case <-time.After(backoff):
		}
		// Jogo parado (app em segundo plano, travado): nao volta ao servidor.
		// A sala ja morreu la; ela renasce com o mesmo codigo quando o jogo
		// voltar a rodar, e um cliente ainda dentro da janela de reconexao a
		// encontra.
		if !l.alive() {
			backoff = 2 * time.Second
			continue
		}
		if err := l.register(); err != nil {
			log.Printf("[relay] sem conexao com %s: %v", l.server, err)
			l.mu.Lock()
			l.ctrl = nil
			l.mu.Unlock()
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}
}

func (l *Listener) serveControl(c net.Conn) {
	defer func() {
		l.connected.Store(false)
		c.Close()
	}()
	for {
		// O servidor manda PING a cada 15 s: silencio de 60 s e conexao morta.
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		line, err := readLine(c)
		if err != nil {
			return
		}
		switch {
		case line == "PING":
			if l.alive() {
				c.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if writeLine(c, "PONG") != nil {
					return
				}
			}
		case strings.HasPrefix(line, "OPEN "):
			if l.alive() {
				go l.open(strings.TrimPrefix(line, "OPEN "))
			}
		}
	}
}

// open atende um OPEN: nova conexao de dados, identificada pelo id.
func (l *Listener) open(id string) {
	c, err := net.DialTimeout("tcp", l.server, l.timeout)
	if err != nil {
		log.Printf("[relay] falha ao abrir conexao de dados: %v", err)
		return
	}
	keepAlive(c)
	if writeLine(c, "DATA "+id) != nil {
		c.Close()
		return
	}
	select {
	case l.conns <- &relayConn{Conn: c, remote: relayAddr("relay/" + id)}:
	case <-l.quit:
		c.Close()
	}
}

// Accept implementa net.Listener.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.quit:
		return nil, net.ErrClosed
	}
}

// Close encerra a sala: o relay a remove quando a conexao de controle cai.
func (l *Listener) Close() error {
	l.once.Do(func() {
		close(l.quit)
		l.mu.Lock()
		if l.ctrl != nil {
			l.ctrl.Close()
		}
		l.mu.Unlock()
	})
	return nil
}

// Addr implementa net.Listener.
func (l *Listener) Addr() net.Addr { return relayAddr(l.server) }

// relayConn da a cada jogador um RemoteAddr proprio. Sem isso todas as
// conexoes teriam o endereco do relay, e network.Host, que indexa os peers por
// RemoteAddr, guardaria um jogador por cima do outro.
type relayConn struct {
	net.Conn
	remote net.Addr
}

func (c *relayConn) RemoteAddr() net.Addr { return c.remote }

type relayAddr string

func (a relayAddr) Network() string { return "relay" }
func (a relayAddr) String() string  { return string(a) }

func keepAlive(c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(15 * time.Second)
	}
}

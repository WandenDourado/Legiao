package relay

import (
	"log"
	"net"
	"time"
)

// host registra a sala e segura a conexao de controle ate ela morrer.
//
// A sala so vive enquanto o host PROVA que esta vivo: ele responde PONG a
// cada PING, e o app so responde enquanto o laco do jogo esta rodando
// (network.MarkFrame). App fechado, em segundo plano ou sem rede: sem PONG
// por HostSilence, e a sala some, levando junto as conexoes dos jogadores.
// Sem isso uma queda de rede sem FIN deixava a sala viva por ~15 minutos.
//
// Um host que volta com o mesmo codigo E o mesmo segredo reassume a MESMA
// sala (com os jogadores que ela ja tem): o controle antigo pode ainda nao ter
// sido dado como morto. Sem o segredo, ninguem sequestra um codigo alheio.
func (s *Server) host(c net.Conn, want, secret, tag, name string) {
	if len(secret) < 16 || !validTag(tag) {
		writeLine(c, "ERR "+ReasonBadRegister)
		c.Close()
		return
	}
	s.mu.Lock()
	r, oldCtrl := s.rooms[want], net.Conn(nil)
	switch {
	case r != nil && r.secret == secret && validCode(want):
		oldCtrl = r.ctrl
		r.wmu.Lock()
		r.ctrl, r.name = c, name
		r.wmu.Unlock()
	case len(s.rooms) >= s.MaxRooms:
		s.mu.Unlock()
		writeLine(c, "ERR "+ReasonServerFull)
		c.Close()
		return
	default:
		r = &room{name: name, secret: secret, tag: tag, ctrl: c, pipes: map[net.Conn]net.Conn{}}
		if validCode(want) && s.rooms[want] == nil {
			r.code = want
		}
		for r.code == "" {
			if code := newCode(); s.rooms[code] == nil {
				r.code = code
			}
		}
		s.rooms[r.code] = r
	}
	s.mu.Unlock()
	if oldCtrl != nil {
		oldCtrl.Close()
	}
	if r.send("ROOM "+r.code) != nil {
		s.drop(r, c)
		return
	}
	c.SetWriteDeadline(time.Time{})
	log.Printf("[relay] sala %s aberta (%q, %s)", r.code, r.name, c.RemoteAddr())

	dead := make(chan struct{})
	go func() {
		defer close(dead)
		for {
			c.SetReadDeadline(time.Now().Add(s.HostSilence))
			if _, err := readLine(c); err != nil {
				return
			}
		}
	}()
	t := time.NewTicker(s.PingInterval)
	defer t.Stop()
	for {
		select {
		case <-dead:
			s.drop(r, c)
			return
		case <-t.C:
			if !s.ping(r, c) {
				s.drop(r, c)
				return
			}
		}
	}
}

// ping manda PING pelo controle c, se c ainda for o controle da sala.
func (s *Server) ping(r *room, c net.Conn) bool {
	r.wmu.Lock()
	defer r.wmu.Unlock()
	if r.ctrl != c {
		return false
	}
	c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return writeLine(c, "PING") == nil
}

// drop encerra a sala cujo controle e c. Se c ja foi substituido por um
// registro novo do mesmo host, so fecha c: a sala continua com o controle novo.
func (s *Server) drop(r *room, c net.Conn) {
	c.Close()
	s.mu.Lock()
	if r.ctrl != c || s.rooms[r.code] != r {
		s.mu.Unlock()
		return
	}
	delete(s.rooms, r.code)
	pipes := r.pipes
	r.pipes = map[net.Conn]net.Conn{}
	s.mu.Unlock()
	for a, b := range pipes {
		a.Close()
		b.Close()
	}
	log.Printf("[relay] sala %s fechada (%d jogador(es) desconectado(s))", r.code, len(pipes))
}

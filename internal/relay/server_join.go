package relay

import (
	"net"
	"time"
)

// join reserva uma vaga, pede ao host uma conexao de dados e junta as duas.
func (s *Server) join(c net.Conn, code string) {
	s.mu.Lock()
	r := s.rooms[code]
	var why string
	switch {
	case r == nil:
		why = ReasonRoomNotFound
	case s.overBudgetLocked():
		why = ReasonMonthlyLimit
	case r.slots >= s.MaxPlayersPerRoom:
		why = ReasonRoomFull
	case s.players >= s.MaxPlayers:
		why = ReasonServerFull
	}
	if why != "" {
		s.mu.Unlock()
		writeLine(c, "ERR "+why)
		c.Close()
		return
	}
	r.slots++
	s.players++
	id := newID()
	p := &pairing{ch: make(chan net.Conn, 1), done: make(chan struct{})}
	s.pending[id] = p
	s.mu.Unlock()
	defer close(p.done)

	hc := s.awaitHost(r, id, p.ch)
	if hc == nil || writeLine(c, "OK") != nil {
		if hc != nil {
			hc.Close()
		} else {
			writeLine(c, "ERR "+ReasonHostTimeout)
		}
		c.Close()
		s.freeSlot(r)
		return
	}
	s.mu.Lock()
	alive := s.rooms[code] == r
	if alive {
		r.pipes[c] = hc
	}
	s.mu.Unlock()
	if !alive { // a sala morreu durante o pareamento
		c.Close()
		hc.Close()
		s.freeSlot(r)
		return
	}
	c.SetDeadline(time.Time{})
	hc.SetDeadline(time.Time{})
	s.pipe(c, hc)
	s.mu.Lock()
	delete(r.pipes, c)
	s.mu.Unlock()
	s.freeSlot(r)
}

// pairing liga um JOIN ao DATA que o host abre em resposta.
type pairing struct {
	ch   chan net.Conn
	done chan struct{} // fechado quando join acabou com a conexao do host
}

// awaitHost manda OPEN e espera o DATA correspondente; nil se nao veio.
func (s *Server) awaitHost(r *room, id string, ch chan net.Conn) net.Conn {
	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()
	if r.send("OPEN "+id) != nil {
		return nil
	}
	select {
	case hc := <-ch:
		return hc
	case <-time.After(s.PairTimeout):
		return nil
	}
}

func (s *Server) freeSlot(r *room) {
	s.mu.Lock()
	r.slots--
	s.players--
	s.mu.Unlock()
}

// data entrega a conexao de dados do host ao join que a pediu e espera ele
// terminar com ela. handle nao pode voltar antes disso: release() descontaria
// a conexao do limite enquanto a partida ainda corre.
func (s *Server) data(c net.Conn, id string) {
	s.mu.Lock()
	p := s.pending[id]
	delete(s.pending, id)
	s.mu.Unlock()
	if p == nil {
		c.Close()
		return
	}
	p.ch <- c // buffer de 1: nunca bloqueia
	<-p.done
	// Fecha de novo aqui porque um DATA que chegou colado no prazo pode ter
	// caido no canal depois de join desistir; sem isto ele viraria um peer
	// fantasma no host, recebendo todo broadcast da partida.
	c.Close()
}

// pipe copia nos dois sentidos, conta os bytes enviados e fecha as duas
// pontas quando uma acaba ou fica PipeIdle sem trafego.
func (s *Server) pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		buf := make([]byte, 32*1024)
		for {
			src.SetReadDeadline(time.Now().Add(s.PipeIdle))
			n, err := src.Read(buf)
			if n > 0 {
				if _, werr := dst.Write(buf[:n]); werr != nil {
					break
				}
				s.countSent(n)
			}
			if err != nil { // fim, erro ou PipeIdle sem trafego
				break
			}
		}
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	a.Close()
	b.Close()
	<-done
}

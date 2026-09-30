package relay

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Server e o lado publico do relay (cmd/relay). Estado so em memoria:
// reiniciar derruba as salas, e os hosts se registram de novo sozinhos com o
// mesmo codigo (Listener.run).
//
// Os limites existem para a VM Always Free (1/8 de OCPU, 1 GB, ate 50 Mbps,
// 10 TB/mes de saida) nunca ser o gargalo nem sair do gratuito. Medido em
// 29/09/2026 com um gerador de carga contra este servidor (doc/network.md,
// "Capacidade do relay"): com o fluxo comprimido (~33 KB/s por amigo no pior
// caso, a 20 Hz) cada amigo custa ~0,12% de um nucleo de CPU e ~0,27 Mbps.
// 64 amigos = ~8% de um nucleo (65% do 1/8 garantido) e ~17 Mbps. Agora e a
// CPU, e nao a banda, que limita; uma sala vazia continua custando quase nada.
type Server struct {
	MaxRooms          int
	MaxPlayersPerRoom int // amigos por sala, sem contar o host
	MaxPlayers        int // amigos no servidor inteiro
	MaxConns          int
	MaxConnsPerIP     int
	// MonthlyByteBudget e o teto de bytes ENVIADOS pelo relay no mes (UTC).
	// Passou dele, ninguem novo entra ate virar o mes. Contado em memoria:
	// reiniciar o processo zera a conta, o que so erra para menos.
	MonthlyByteBudget int64

	HandshakeTimeout time.Duration // primeira linha de qualquer conexao
	PairTimeout      time.Duration // host responder um OPEN
	PingInterval     time.Duration
	HostSilence      time.Duration // host sem PONG por isto: sala morta
	PipeIdle         time.Duration // partida sem um byte por isto: conexao morta

	mu      sync.Mutex
	rooms   map[string]*room
	pending map[string]*pairing
	conns   int
	perIP   map[string]int
	players int // vagas ocupadas (pareando ou jogando)
	month   string
	sent    int64 // bytes enviados neste mes
	sentAll int64
	started time.Time
	// Taxa de envio recente, para STATS: bytes da janela atual e a taxa da
	// ultima janela fechada (10 s).
	winStart time.Time
	winBytes int64
	rateKBps int64
	rejected int64
}

type room struct {
	code, name, secret, tag string
	ctrl                    net.Conn
	wmu                     sync.Mutex // serializa OPEN e PING no controle
	slots                   int        // vagas ocupadas nesta sala
	pipes                   map[net.Conn]net.Conn
}

func (r *room) send(line string) error {
	r.wmu.Lock()
	defer r.wmu.Unlock()
	r.ctrl.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return writeLine(r.ctrl, line)
}

// NewServer devolve um servidor com os limites pensados para a VM gratuita.
func NewServer() *Server {
	return &Server{
		MaxRooms:          50,      // salas custam quase nada; o teto real e MaxPlayers
		MaxPlayersPerRoom: 10,      // 11 com o host
		MaxPlayers:        64,      // ~8% de um nucleo e ~17 Mbps com fluxo comprimido
		MaxConns:          512,     // 2 por amigo + controle das salas + consultas LIST
		MaxConnsPerIP:     40,      // um host com sala cheia usa 11 sozinho
		MonthlyByteBudget: 8 << 40, // 8 TiB, abaixo dos 10 TB gratuitos
		HandshakeTimeout:  10 * time.Second,
		PairTimeout:       5 * time.Second,
		PingInterval:      15 * time.Second,
		HostSilence:       45 * time.Second,
		PipeIdle:          90 * time.Second,
		rooms:             map[string]*room{},
		pending:           map[string]*pairing{},
		perIP:             map[string]int{},
		started:           time.Now(),
	}
}

// Serve atende ln ate ele fechar.
func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		ip := hostOf(c.RemoteAddr())
		if !s.admit(ip) {
			c.Close()
			continue
		}
		keepAlive(c)
		go func() {
			defer s.release(ip)
			s.handle(c)
		}()
	}
}

func (s *Server) admit(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conns >= s.MaxConns || s.perIP[ip] >= s.MaxConnsPerIP {
		s.rejected++
		return false
	}
	s.conns++
	s.perIP[ip]++
	return true
}

func (s *Server) release(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns--
	if s.perIP[ip]--; s.perIP[ip] <= 0 {
		delete(s.perIP, ip)
	}
}

func hostOf(a net.Addr) string {
	h, _, err := net.SplitHostPort(a.String())
	if err != nil {
		return a.String()
	}
	return h
}

// handle le o comando e fica com a conexao ate ela acabar: o limite de
// conexoes (admit/release) cobre a vida inteira dela, partida incluida.
func (s *Server) handle(c net.Conn) {
	c.SetDeadline(time.Now().Add(s.HandshakeTimeout))
	line, err := readLine(c)
	if err != nil {
		c.Close()
		return
	}
	f := strings.SplitN(line, " ", 6)
	switch {
	case f[0] == "HOST" && len(f) == 6 && f[1] == Version:
		s.host(c, f[2], f[3], f[4], CleanName(f[5]))
	case f[0] == "JOIN" && len(f) == 3 && f[1] == Version:
		s.join(c, f[2])
	case f[0] == "DATA" && len(f) == 2:
		s.data(c, f[1])
	case f[0] == "LIST" && len(f) == 3 && f[1] == Version:
		s.list(c, f[2])
	case f[0] == "STATS" && len(f) == 2 && f[1] == Version:
		writeLine(c, s.Stats())
		c.Close()
	default:
		writeLine(c, "ERR "+ReasonBadCommand)
		c.Close()
	}
}

func (s *Server) list(c net.Conn, tag string) {
	defer c.Close()
	s.mu.Lock()
	lines := make([]string, 0, len(s.rooms)+1)
	for _, r := range s.rooms {
		if r.tag == tag {
			lines = append(lines, fmt.Sprintf("R %s %d %d %s", r.code, len(r.pipes), s.MaxPlayersPerRoom, r.name))
		}
	}
	s.mu.Unlock()
	for _, l := range append(lines, "END") {
		if writeLine(c, l) != nil {
			return
		}
	}
}

// Stats e uma linha legivel com a saude do servidor.
func (s *Server) Stats() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rollMonthLocked()
	now := time.Now()
	s.rollRateLocked(now)
	rate := s.rateKBps
	if now.Sub(s.winStart) > 20*time.Second {
		rate = 0 // nada passou nas ultimas duas janelas
	}
	return fmt.Sprintf("salas=%d/%d jogadores=%d/%d conexoes=%d/%d recusadas=%d taxa_kbs=%d mes=%s enviado_mes_mb=%d teto_mes_mb=%d enviado_total_mb=%d no_ar=%s",
		len(s.rooms), s.MaxRooms, s.players, s.MaxPlayers, s.conns, s.MaxConns, s.rejected, rate,
		s.month, s.sent>>20, s.MonthlyByteBudget>>20, s.sentAll>>20, time.Since(s.started).Round(time.Second))
}

// rollMonthLocked zera a conta do mes quando o mes (UTC) vira.
func (s *Server) rollMonthLocked() {
	if m := time.Now().UTC().Format("2006-01"); m != s.month {
		s.month, s.sent = m, 0
	}
}

func (s *Server) countSent(n int) {
	s.mu.Lock()
	s.rollMonthLocked()
	s.sent += int64(n)
	s.sentAll += int64(n)
	s.winBytes += int64(n)
	s.rollRateLocked(time.Now())
	s.mu.Unlock()
}

// rollRateLocked fecha a janela de 10 s da taxa de envio.
func (s *Server) rollRateLocked(now time.Time) {
	if s.winStart.IsZero() {
		s.winStart = now
		return
	}
	if el := now.Sub(s.winStart); el >= 10*time.Second {
		s.rateKBps = int64(float64(s.winBytes) / el.Seconds() / 1024)
		s.winStart, s.winBytes = now, 0
	}
}

func (s *Server) overBudgetLocked() bool {
	s.rollMonthLocked()
	return s.MonthlyByteBudget > 0 && s.sent >= s.MonthlyByteBudget
}

package network

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
)

// A lista de salas online, do lado de quem procura partida. Consultada em
// segundo plano, no mesmo ritmo da descoberta da LAN, para que o laco do menu
// nunca espere a rede para desenhar.

const onlineListInterval = 3 * time.Second

// OnlineStatus diz o que a tela de entrar pode contar sobre o servidor.
type OnlineStatus int

const (
	OnlineOff         OnlineStatus = iota // build sem OnlineConfigURL
	OnlineChecking                        // primeira consulta em andamento
	OnlineReachable                       // ultima consulta respondeu
	OnlineUnreachable                     // ultima consulta falhou
	OnlineDisabled                        // o arquivo de configuracao desligou o online
)

var (
	onlineRoomsMu  sync.Mutex
	onlineRooms    []relay.Room
	onlineState    = OnlineOff
	onlineStop     chan struct{}
	onlineRoomCode atomic.Value // codigo da sala que ESTE host serve
)

// OnlineRoomCode e o codigo da sala online deste host, ou "".
func OnlineRoomCode() string {
	s, _ := onlineRoomCode.Load().(string)
	return s
}

// StartOnlineRoomPoll comeca a consultar o relay. Idempotente.
func StartOnlineRoomPoll() {
	if !OnlineConfigured() {
		return
	}
	onlineRoomsMu.Lock()
	defer onlineRoomsMu.Unlock()
	if onlineStop != nil {
		return
	}
	stop := make(chan struct{})
	onlineStop, onlineState = stop, OnlineChecking
	go func() {
		for {
			rooms, state := listOnlineRooms()
			onlineRoomsMu.Lock()
			if onlineStop == stop {
				onlineRooms, onlineState = rooms, state
			}
			onlineRoomsMu.Unlock()
			select {
			case <-stop:
				return
			case <-time.After(onlineListInterval):
			}
		}
	}()
}

// StopOnlineRoomPoll para a consulta e esquece a lista.
func StopOnlineRoomPoll() {
	onlineRoomsMu.Lock()
	defer onlineRoomsMu.Unlock()
	if onlineStop != nil {
		close(onlineStop)
		onlineStop = nil
	}
	onlineRooms, onlineState = nil, OnlineOff
}

// GetOnlineRooms devolve a ultima lista recebida do relay e o estado dele.
func GetOnlineRooms() ([]relay.Room, OnlineStatus) {
	onlineRoomsMu.Lock()
	defer onlineRoomsMu.Unlock()
	return append([]relay.Room(nil), onlineRooms...), onlineState
}

func listOnlineRooms() ([]relay.Room, OnlineStatus) {
	server, err := relayAddress()
	if errors.Is(err, errOnlineDisabled) {
		return nil, OnlineDisabled
	}
	if err == nil {
		var rooms []relay.Room
		if rooms, err = relay.List(server, OnlineGameTag, onlineDialTimeout); err == nil {
			return rooms, OnlineReachable
		}
		forgetRelayAddress()
	}
	log.Printf("[Online] lista de salas indisponivel: %v", err)
	return nil, OnlineUnreachable
}

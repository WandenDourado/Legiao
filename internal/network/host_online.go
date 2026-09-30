package network

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/relay"
)

// OnlineRoom e uma sala registrada no relay, ainda sem host ou ja servindo.
// A UI a abre ANTES de iniciar a partida (ui/online_room_setup.go), para o
// jogador saber na hora se o servidor respondeu em vez de hospedar uma
// partida que ninguem de fora consegue ver.
type OnlineRoom struct {
	ln   *relay.Listener
	name string
}

// OpenOnlineRoom registra a sala name no relay. Bloqueia ate onlineDialTimeout;
// a UI chama em segundo plano.
func OpenOnlineRoom(name string) (*OnlineRoom, error) {
	return OpenOnlineRoomAs(name, "", "")
}

// OpenOnlineRoomAs abre a sala pedindo de volta um codigo que este jogador ja
// tinha, com o segredo dele (relay.HostAs) — a volta do host depois de um
// REINICIAR (ui/resume_session.go). Os amigos, reconectando ao codigo antigo,
// reencontram a partida sem passar pela lista.
func OpenOnlineRoomAs(name, code, secret string) (*OnlineRoom, error) {
	// Conta como quadro: entre abrir a sala e o primeiro quadro da partida
	// ainda ha a troca de tela, e a sala nao pode nascer "parada".
	MarkFrame()
	server, err := relayAddress()
	if err != nil {
		return nil, err
	}
	ln, err := relay.HostAs(server, OnlineGameTag, name, code, secret, onlineDialTimeout, gameRunning)
	if err != nil {
		forgetRelayAddress()
		return nil, err
	}
	return &OnlineRoom{ln: ln, name: name}, nil
}

// Code e o codigo unico da sala no relay (ex.: K7M2Q).
func (r *OnlineRoom) Code() string { return r.ln.Code() }

// Close encerra a sala no relay na hora (o relay derruba quem estava nela).
func (r *OnlineRoom) Close() { r.ln.Close() }

// ServeOnline passa a atender, pelo relay, os jogadores da sala. As conexoes
// entram no MESMO acceptLoop da LAN; cada uma traz um RemoteAddr proprio
// ("relay/<id>"), entao h.peers nao mistura jogadores.
func (h *Host) ServeOnline(r *OnlineRoom) {
	h.peersMutex.Lock()
	h.onlineListener = r.ln
	h.peersMutex.Unlock()
	onlineRoomCode.Store(r.Code())
	onlineRoomName.Store(r.name)
	onlineRoomSecret.Store(r.ln.Secret())
	log.Printf("[Host] partida online aberta: sala %s", r.Code())
	go h.acceptLoop(r.ln)
}

// ShutdownOnline fecha a sala online, se houver. Chamado quando o jogo
// termina (game.Run): o relay tira a sala da lista no mesmo instante, sem
// esperar os 45 s do sinal de vida.
func ShutdownOnline() {
	if h := CurrentHost; h != nil {
		h.peersMutex.Lock()
		ln := h.onlineListener
		h.onlineListener = nil
		h.peersMutex.Unlock()
		if ln != nil {
			ln.Close()
		}
	}
	onlineRoomCode.Store("")
	onlineRoomName.Store("")
	onlineRoomSecret.Store("")
}

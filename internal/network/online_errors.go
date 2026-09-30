package network

import (
	"errors"
	"net"
	"sync"

	"github.com/WandenDourado/Legiao/internal/relay"
)

// OnlineErrorText traduz uma falha do jogo online para o jogador: o que
// aconteceu e, quando da, o que ele pode fazer. A UI nunca mostra o erro cru.
//
// A distincao que importa: o servidor RESPONDEU e recusou (sala cheia, sala
// encerrada, servidor lotado - relay.RefusedError), ou nem respondeu (rede do
// jogador, servidor fora do ar).
func OnlineErrorText(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errOnlineDisabled) {
		return "O jogo online esta desativado no momento. A rede local continua funcionando."
	}
	var r *relay.RefusedError
	if errors.As(err, &r) {
		switch r.Reason {
		case relay.ReasonRoomNotFound:
			return "Esta sala foi encerrada pelo anfitriao."
		case relay.ReasonRoomFull:
			return "Esta sala esta cheia."
		case relay.ReasonServerFull:
			return "O servidor online esta lotado. Tente de novo em alguns minutos."
		case relay.ReasonMonthlyLimit:
			return "O servidor online atingiu o limite do mes. A rede local continua funcionando."
		case relay.ReasonHostTimeout:
			return "O anfitriao nao respondeu. Tente de novo."
		case relay.ReasonBadCommand, relay.ReasonBadRegister:
			return "Sua versao do jogo nao e compativel com o servidor. Atualize o jogo."
		}
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "O servidor online demorou para responder. Verifique sua internet."
	}
	return "Falha na comunicacao com o servidor online. Verifique sua internet e tente de novo."
}

var (
	onlineErrMu      sync.Mutex
	failureDetail    string // frase pronta para o menu, "" = usar a padrao
	lastReconnectErr error  // ultima tentativa de reconexao que falhou
)

func setFailureDetail(addr string, err error) {
	onlineErrMu.Lock()
	defer onlineErrMu.Unlock()
	failureDetail = ""
	if IsOnlineRoom(addr) {
		failureDetail = OnlineErrorText(err)
	}
}

// setFailureText guarda uma frase pronta, valida tambem na LAN (ex.: o host
// recusou a entrada, client_ping.go).
func setFailureText(text string) {
	onlineErrMu.Lock()
	failureDetail = text
	onlineErrMu.Unlock()
}

// LastFailureDetail e a frase especifica da ultima falha de sessao online,
// ou "" (o menu usa a frase padrao de ui/menu.go).
func LastFailureDetail() string {
	onlineErrMu.Lock()
	defer onlineErrMu.Unlock()
	return failureDetail
}

func setReconnectErr(err error) {
	onlineErrMu.Lock()
	lastReconnectErr = err
	onlineErrMu.Unlock()
}

// ReconnectHint explica, na tela de "Reconectando...", por que ainda nao
// voltou - so para sala online, onde o motivo e conhecido. "" na LAN ou antes
// da primeira tentativa falhar.
func ReconnectHint() string {
	onlineErrMu.Lock()
	err := lastReconnectErr
	onlineErrMu.Unlock()
	if err == nil || !IsOnlineRoom(ServerAddress) {
		return ""
	}
	var r *relay.RefusedError
	if errors.As(err, &r) && r.Reason == relay.ReasonRoomNotFound {
		// O host fechou o app ou perdeu a rede; se ele voltar dentro da
		// janela, a sala renasce com o mesmo codigo e a reconexao pega.
		return "O anfitriao saiu ou perdeu a conexao. Aguardando ele voltar..."
	}
	return OnlineErrorText(err)
}

// OnlineHostStatus e o estado da sala online DESTE host, para o HUD: codigo,
// se esta registrada no servidor agora, e se existe sala online.
func OnlineHostStatus() (code string, connected, active bool) {
	h := CurrentHost
	if h == nil {
		return "", false, false
	}
	h.peersMutex.Lock()
	ln := h.onlineListener
	h.peersMutex.Unlock()
	rl, ok := ln.(*relay.Listener)
	if !ok || rl == nil {
		return "", false, false
	}
	return rl.Code(), rl.Connected(), true
}

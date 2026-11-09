package network

// Abrir a conexao de cliente — e o estado em que ela AINDA NAO EXISTE.
//
// Este era o ponto cego da camada de rede. `Role` virava "client" no menu,
// antes de qualquer socket existir, e `CurrentClient` so era atribuido quando
// o dial voltava. Entre os dois instantes a maquina rodava um jogo COMPLETO de
// papel cliente sem conexao nenhuma: papel de cliente nao simula nada (so o
// host chama UpdateSimulation, game/loop.go), entao a tela mostrava o mapa SEM
// inimigos e SEM os outros jogadores, com o jogador local andando normalmente.
// E nada vigiava: TickReconnect devolvia na hora enquanto CurrentClient fosse
// nil, entao o relogio de silencio de 5 s (client_liveness.go) nunca comecava
// a contar; o host, que nunca recebeu MsgJoin, nem sabia que alguem tentou
// entrar. O relato era exatamente "entrei numa fase vazia, sozinho, e o host
// nao me ve" — uma falha de rede desenhada como um jogo funcionando.
//
// A correcao tem tres partes:
//
//  1. O dial tem PRAZO (ConnectClient usa dialTimeout, client.go). Sem prazo,
//     um SYN DESCARTADO em silencio — o que um firewall de entrada faz, ao
//     contrario de uma porta fechada, que responde RST na hora — deixava o
//     Android preso os ~2 minutos do tempo limite do sistema operacional.
//  2. Enquanto o dial esta pendente a sessao esta EM ESPERA e diz isso:
//     AwaitingHost congela o quadro e o renderer desenha a sobreposicao, do
//     mesmo jeito que uma reconexao ja fazia. O jogador le "Conectando ao
//     anfitriao...", nao um mundo vazio.
//  3. Falhar volta ao menu com o motivo (failSession, reconnect.go), em vez do
//     log.Fatalf que fechava o aplicativo na cara de quem so errou a rede.

import (
	"log"
	"sync"
	"time"
)

// connectCeiling e o teto do estado "conectando", independente do prazo do
// dial. ConnectClient ja volta em dialTimeout, entao na pratica este relogio
// nunca vence: ele existe para que o ponto cego descrito acima nao possa
// voltar a existir por OUTRO caminho (uma resolucao de nome presa, um dial que
// nao respeite o prazo em alguma plataforma). Por isso e folgado em relacao a
// dialTimeout em vez de colado nele — apertar os dois faria este teto virar o
// detector, e o detector certo e o prazo do dial.
const connectCeiling = 10 * time.Second

var (
	connectMu        sync.Mutex
	connecting       bool
	connectStartedAt time.Time
	// connectGen identifica a tentativa de conexao em curso. O dial roda numa
	// goroutine e pode voltar DEPOIS de o teto acima ter encerrado a sessao;
	// sem a geracao, essa conexao atrasada viraria o CurrentClient de uma
	// sessao que o jogador ja abandonou no menu.
	connectGen int
)

// StartClientSession abre a sessao de cliente: guarda a identidade, entra no
// estado "conectando" e disca em segundo plano.
//
// Volta na hora, de proposito. Quem chama e o laco do menu, e ele nao pode
// ficar sem desenhar quadro pelo tempo de um dial — o preco de nao esperar e
// justamente o estado "conectando" que este arquivo existe para tornar
// visivel e vigiado.
func StartClientSession(addr, playerID, color, charType string) {
	Role = "client"
	LocalPlayerID = playerID
	ServerAddress = addr
	// Guardados aqui, e nao so dentro do join: uma reconexao (reconnect.go)
	// reenvia exatamente esta identidade em vez de pedir ao jogador para
	// escolher de novo.
	LocalPlayerColor = color
	LocalPlayerCharacter = charType

	setFailureDetail("", nil)
	setReconnectErr(nil)
	connectMu.Lock()
	connectGen++
	gen := connectGen
	connecting = true
	connectStartedAt = time.Now()
	connectMu.Unlock()

	go func() {
		log.Printf("[Cliente] conectando a %s...", addr)
		c, err := ConnectClient(addr)
		if err != nil {
			log.Printf("[Cliente] falha ao conectar a %s: %v", addr, err)
			setFailureDetail(addr, err)
			if finishConnect(gen) {
				failSession(FailureConnect)
			}
			return
		}
		if !finishConnect(gen) {
			// O teto ja encerrou esta tentativa: a conexao que acabou de
			// nascer nao pertence a sessao nenhuma, e deixa-la aberta daria
			// ao host um peer que ninguem esta jogando.
			c.Close()
			return
		}
		CurrentClient = c
		SendMessage(Message{Type: MsgJoin, Payload: MustMarshal(JoinPayload{
			PlayerID:  playerID,
			Color:     color,
			Character: charType,
			Compress:  compressDeflate,
			Proto:     OnlineGameTag,
		})})
		startPinger(c)
		log.Printf("[Cliente] conectado a %s; join enviado", addr)
	}()
}

// finishConnect sai do estado "conectando" e diz se a tentativa gen ainda era
// a atual. Falso significa que ela foi abandonada (teto vencido, ou sessao
// reiniciada por ResetClientSession) e que quem chamou nao deve publicar nada.
func finishConnect(gen int) bool {
	connectMu.Lock()
	defer connectMu.Unlock()
	if gen != connectGen || !connecting {
		return false
	}
	connecting = false
	return true
}

// Connecting reporta que o dial inicial esta pendente: ja existe papel de
// cliente, ainda nao existe conexao.
func Connecting() bool {
	connectMu.Lock()
	defer connectMu.Unlock()
	return connecting
}

// AwaitingHost e "este cliente nao tem ligacao utilizavel agora", pelos dois
// motivos possiveis: a primeira conexao ainda nao abriu, ou ela abriu e caiu.
// O laco do jogo congela o quadro com ela (game/loop.go) e o renderer desenha
// a sobreposicao. Os dois casos merecem o mesmo tratamento porque nos dois o
// mundo na tela nao esta sendo governado por ninguem — o que os separa e so o
// texto da sobreposicao.
func AwaitingHost() bool { return Connecting() || Reconnecting() }

// tickConnect vigia o teto do estado "conectando". Chamado por TickReconnect,
// que continua sendo o unico ponto por quadro da maquina de estados do
// cliente.
func tickConnect() {
	connectMu.Lock()
	stuck := connecting && time.Since(connectStartedAt) >= connectCeiling
	if stuck {
		connecting = false
		// Invalida a tentativa: se o dial voltar depois disto, finishConnect
		// devolve falso e a conexao e fechada em vez de adotada.
		connectGen++
	}
	connectMu.Unlock()
	if !stuck {
		return
	}
	log.Printf("[Cliente] conexao a %s nao completou em %s; desistindo", ServerAddress, connectCeiling)
	failSession(FailureConnect)
}

// resetConnectState limpa o estado de conexao para que a proxima sessao comece
// do zero. Ver ResetClientSession (reconnect.go).
func resetConnectState() {
	connectMu.Lock()
	connecting = false
	connectGen++
	connectMu.Unlock()
}

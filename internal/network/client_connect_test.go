package network

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// resetSessionForTest isola o estado global de sessao de cliente: estes testes
// mexem em Role, CurrentClient e nos dois relogios (conexao e reconexao), que
// sao variaveis de pacote. Sem isto, um teste deixaria o proximo comecando no
// meio de uma sessao.
func resetSessionForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if CurrentClient != nil {
			CurrentClient.Close()
			CurrentClient = nil
		}
		Role = ""
		resetConnectState()
		reconnectMu.Lock()
		reconnecting = false
		failure = FailureNone
		reconnectMu.Unlock()
	})
}

// waitFor espera uma condicao virar verdadeira, cutucando a cada milissegundo.
// A abertura da conexao acontece numa goroutine (StartClientSession volta na
// hora, de proposito), entao o teste tem de esperar por ela em vez de dormir
// um tempo fixo e torcer.
func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("esperava %s em %s, nao aconteceu", what, timeout)
}

// closedLoopbackAddr devolve um endereco de loopback com CERTEZA sem ninguem
// escutando: abre e fecha um listener, e entrega a porta que acabou de vagar.
// Loopback recusa na hora (RST), entao este e o caminho de falha RAPIDO — o
// caminho lento (SYN descartado por firewall) e o que dialTimeout cobre, e nao
// da para simular num teste sem um firewall de verdade.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("nao consegui abrir listener temporario: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func TestStartClientSessionSendsJoinOnceConnected(t *testing.T) {
	resetSessionForTest(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("nao consegui abrir o host de teste: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- conn
	}()

	StartClientSession(ln.Addr().String(), "player_teste", "#ff0000", "mago")

	waitFor(t, "a conexao abrir", 3*time.Second, func() bool {
		return !Connecting() && CurrentClient != nil
	})
	if AwaitingHost() {
		t.Fatal("com a conexao aberta, o cliente nao deveria mais estar esperando o host")
	}
	if GaveUp() {
		t.Fatalf("conexao bem-sucedida nao deveria encerrar a sessao (motivo=%v)", LastFailure())
	}

	var conn net.Conn
	select {
	case conn = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("o host de teste nao recebeu conexao nenhuma")
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Fatalf("nao chegou mensagem nenhuma do cliente: %v", err)
	}
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		t.Fatalf("a primeira mensagem nao decodifica: %v", err)
	}
	if msg.Type != MsgJoin {
		t.Fatalf("a primeira mensagem tinha de ser %s, foi %s", MsgJoin, msg.Type)
	}
	var join JoinPayload
	if err := json.Unmarshal(msg.Payload, &join); err != nil {
		t.Fatalf("o join nao decodifica: %v", err)
	}
	if join.PlayerID != "player_teste" || join.Color != "#ff0000" || join.Character != "mago" {
		t.Fatalf("o join tem de levar a identidade escolhida no menu, levou %+v", join)
	}
	// A mesma identidade fica guardada para uma reconexao poder reenviar
	// exatamente ela (reconnect.go, rejoin).
	if LocalPlayerColor != "#ff0000" || LocalPlayerCharacter != "mago" {
		t.Fatalf("identidade nao foi guardada para a reconexao: cor=%q personagem=%q", LocalPlayerColor, LocalPlayerCharacter)
	}

	// Fecha por dentro ANTES de o listener cair: assim o readLoop ve que a
	// conexao dele ja nao e a atual e sai calado, em vez de disparar uma
	// reconexao em cima do proximo teste.
	CurrentClient.Close()
}

func TestStartClientSessionFailsWhenNobodyIsListening(t *testing.T) {
	resetSessionForTest(t)

	StartClientSession(closedLoopbackAddr(t), "player_teste", "#ff0000", "mago")

	// O ponto do teste: a falha TERMINA a sessao. Era aqui que o jogo entrava
	// numa fase vazia e ficava — sem conexao, sem aviso e sem ninguem olhando.
	waitFor(t, "a sessao terminar com falha", 5*time.Second, SessionEnded)
	if got := LastFailure(); got != FailureConnect {
		t.Fatalf("motivo tinha de ser FailureConnect, foi %v", got)
	}
	if Connecting() {
		t.Fatal("depois de falhar o dial, o cliente nao pode continuar em estado de conexao")
	}
	if CurrentClient != nil {
		t.Fatal("um dial que falhou nao deveria ter publicado um CurrentClient")
	}
}

func TestConnectCeilingEndsSessionWhenDialNeverReturns(t *testing.T) {
	resetSessionForTest(t)

	// Um dial pendurado: o estado existe, mas ja passou do teto. E o cenario
	// que dialTimeout torna improvavel e que este teto cobre de qualquer jeito.
	Role = "client"
	ServerAddress = "192.0.2.1:9000"
	connectMu.Lock()
	connectGen++
	gen := connectGen
	connecting = true
	connectStartedAt = time.Now().Add(-connectCeiling - time.Second)
	connectMu.Unlock()

	tickConnect()

	if Connecting() {
		t.Fatal("o teto venceu: o estado de conexao tinha de ter sido abandonado")
	}
	if !SessionEnded() || LastFailure() != FailureConnect {
		t.Fatalf("o teto tinha de encerrar a sessao com FailureConnect, deu ended=%v motivo=%v", SessionEnded(), LastFailure())
	}
	// E a conexao ATRASADA, se voltar agora, nao pode ser adotada por uma
	// sessao que o jogador ja perdeu.
	if finishConnect(gen) {
		t.Fatal("uma conexao que volta depois do teto nao deveria ser aceita")
	}
}

func TestTickReconnectWatchesTheInitialConnect(t *testing.T) {
	resetSessionForTest(t)

	// Exatamente o estado do bug: papel de cliente, NENHUM CurrentClient, e um
	// dial que nao volta. TickReconnect devolvia na primeira linha por causa do
	// nil, entao o relogio de silencio nunca comecava e nada notava.
	Role = "client"
	CurrentClient = nil
	ServerAddress = "192.0.2.1:9000"
	connectMu.Lock()
	connectGen++
	connecting = true
	connectStartedAt = time.Now().Add(-connectCeiling - time.Second)
	connectMu.Unlock()

	TickReconnect()

	if !SessionEnded() {
		t.Fatal("TickReconnect tem de vigiar a conexao inicial mesmo com CurrentClient nil")
	}
}

func TestAwaitingHostCoversBothWaits(t *testing.T) {
	resetSessionForTest(t)

	if AwaitingHost() {
		t.Fatal("sem conexao pendente e sem reconexao, nao ha espera")
	}

	connectMu.Lock()
	connecting = true
	connectMu.Unlock()
	if !AwaitingHost() {
		t.Fatal("a primeira conexao pendente e uma espera pelo host")
	}
	resetConnectState()

	reconnectMu.Lock()
	reconnecting = true
	reconnectMu.Unlock()
	if !AwaitingHost() {
		t.Fatal("uma reconexao em curso e uma espera pelo host")
	}
}

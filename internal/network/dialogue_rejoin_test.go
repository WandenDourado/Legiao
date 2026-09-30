package network

import (
	"encoding/json"
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// drainForDialogue e o irmao de drainForTravel (host_rejoin_test.go): fecha a
// metade do host para o leitor da outra ponta ver EOF, esvazia o canal ate o
// fim e devolve o PRIMEIRO MsgDialogue encontrado.
func drainForDialogue(t *testing.T, c *ClientConn, received <-chan Message) (DialogueState, bool) {
	t.Helper()
	c.conn.Close()
	var state DialogueState
	found := false
	for msg := range received {
		if found || msg.Type != MsgDialogue {
			continue
		}
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			t.Fatalf("MsgDialogue payload nao decodifica: %v", err)
		}
		found = true
	}
	return state, found
}

// TestRejoinClearsEndedDialogue prova o conserto do bug de 20/09/2026.
//
// O cenario e o relatado: alguem cai DURANTE um dialogo, o host termina a cena
// enquanto essa pessoa esta fora, e ela volta. Antes, sendDialogueTo tinha um
// `if !d.Active { return }` e o host nao mandava nada — a maquina dela ficava
// com o Active:true de antes da queda, e como DialogueActive e o portao de
// pausa de input e simulacao, ela voltava congelada numa caixa que so o host
// podia avancar. O host ja tinha encerrado a cena; ninguem podia.
func TestRejoinClearsEndedDialogue(t *testing.T) {
	// O estado e global do pacote: devolver ao que estava evita contaminar
	// qualquer teste que rode depois.
	t.Cleanup(func() { SetDialogueState(DialogueState{}) })
	// A cena ja acabou — exatamente o que DialogueDirector.end publica.
	SetDialogueState(DialogueState{Active: false})

	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"p1": {PlayerID: "p1", X: 100, Y: 200, Color: "blue", Character: "mago"},
		},
		stageMap: "assets/maps/world_04.json",
	}
	h.markAbsent("p1")
	c, received := newRecordingClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "p1", Color: "blue", Character: "mago"}))

	state, ok := drainForDialogue(t, c, received)
	if !ok {
		t.Fatal("rejoin nao recebeu MsgDialogue; o cliente fica preso na cena de antes da queda")
	}
	if state.Active {
		t.Errorf("Active = true depois de a cena encerrar; esperado false")
	}
}

// TestRejoinResendsRunningDialogue guarda o motivo pelo qual sendDialogueTo
// existe desde o inicio: quem entra (ou volta) no meio de uma cena tem de
// pousar ja pausado, em vez de correr pelo mapa enquanto o grupo le.
func TestRejoinResendsRunningDialogue(t *testing.T) {
	t.Cleanup(func() { SetDialogueState(DialogueState{}) })
	SetDialogueState(DialogueState{
		Active:   true,
		ScriptID: "world_04",
		Speaker:  "Mago",
		Text:     "A torre ainda respira.",
		Index:    2,
		Total:    5,
	})

	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"p1": {PlayerID: "p1", X: 100, Y: 200, Color: "blue", Character: "mago"},
		},
		stageMap: "assets/maps/world_04.json",
	}
	h.markAbsent("p1")
	c, received := newRecordingClientConn(t)
	h.handleJoin(c, MustMarshal(JoinPayload{PlayerID: "p1", Color: "blue", Character: "mago"}))

	state, ok := drainForDialogue(t, c, received)
	if !ok {
		t.Fatal("rejoin no meio da cena nao recebeu MsgDialogue")
	}
	if !state.Active || state.Text != "A torre ainda respira." || state.Index != 2 {
		t.Errorf("estado reenviado = %+v; esperado a linha corrente da cena", state)
	}
}

// TestTriggerReconnectClearsDialogue cobre a outra metade do conserto: o
// cliente nao guarda uma cena que ele nao tem como avancar nem encerrar
// enquanto a conexao esta caida. Sem isto, a caixa continuaria desenhada por
// cima do overlay de reconexao.
func TestTriggerReconnectClearsDialogue(t *testing.T) {
	prevRole, prevClient := Role, CurrentClient
	t.Cleanup(func() {
		Role, CurrentClient = prevRole, prevClient
		SetDialogueState(DialogueState{})
		reconnectMu.Lock()
		reconnecting = false
		failure = FailureNone
		reconnectMu.Unlock()
	})

	Role = "client"
	// Sem conexao: dropConn trata conn nil, e o que este teste observa e o
	// estado do dialogo, nao o socket.
	CurrentClient = &Client{}
	SetDialogueState(DialogueState{Active: true, Text: "linha no ar quando a rede caiu"})

	triggerReconnect("teste")

	if DialogueActive() {
		t.Error("dialogo continua ativo depois da queda; a pessoa fica congelada numa cena que so o host avanca")
	}
}

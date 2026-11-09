package network

// O laco do jogo republica o jogador local a cada quadro, e o que ele publica
// e o que ESTA maquina prediz — nao os veredictos do host. Ver
// UpdateLocalPlayerState em globals.go.

import "testing"

func withEmptyRoster(t *testing.T) func() {
	t.Helper()
	RemotePlayersMutex.Lock()
	RemotePlayers = map[string]PlayerState{}
	RemotePlayersMutex.Unlock()
	return func() {
		RemotePlayersMutex.Lock()
		RemotePlayers = nil
		RemotePlayersMutex.Unlock()
	}
}

func TestLocalUpdateKeepsTheHostsPortalVerdict(t *testing.T) {
	defer withEmptyRoster(t)()

	// O host disse: este corpo esta parado dentro do portal.
	UpdatePlayerState(PlayerState{PlayerID: "eu", InPortal: true, Absent: true})

	// O quadro seguinte do laco local nao sabe disso e nem tem como saber.
	UpdateLocalPlayerState(PlayerState{PlayerID: "eu", X: 10, Y: 20})

	got := GetAllPlayers()["eu"]
	if !got.InPortal {
		t.Error("o quadro local apagou o InPortal do host; e isso que fazia o " +
			"personagem continuar desenhado e o aviso piscar")
	}
	if !got.Absent {
		t.Error("o quadro local apagou o Absent do host")
	}
	if got.X != 10 || got.Y != 20 {
		t.Errorf("a posicao predita nao foi publicada: (%d, %d)", got.X, got.Y)
	}
}

func TestLocalUpdateStillCreatesAPlayerThatDidNotExist(t *testing.T) {
	defer withEmptyRoster(t)()

	UpdateLocalPlayerState(PlayerState{PlayerID: "novo", X: 1})
	if _, ok := GetAllPlayers()["novo"]; !ok {
		t.Error("o primeiro quadro de um jogador novo nao entrou no espelho")
	}
}

func TestLeaveLocalPortalClearsBothTheFlagAndTheMirror(t *testing.T) {
	defer withEmptyRoster(t)()

	LocalPlayerID = "eu"
	defer func() { LocalPlayerID = "" }()
	UpdatePlayerState(PlayerState{PlayerID: "eu", InPortal: true})
	LocalPlayerInPortal = true

	LeaveLocalPortal()

	if LocalPlayerInPortal {
		t.Error("a flag local continuou marcada depois do ESC/SAIR")
	}
	// O espelho tambem, ou o proximo quadro repunha a flag a partir dele e o
	// jogador ficava congelado ate o host recalcular a presenca.
	if GetAllPlayers()["eu"].InPortal {
		t.Error("o espelho continuou dizendo que o jogador esta no portal")
	}
}

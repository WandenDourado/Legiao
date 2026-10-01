package game

// An absent player (mid-reconnect, network/host_absence.go) cannot hold
// group decisions hostage. This is the one call site in this package that is
// cheap to exercise directly; partyIsFalling and climaxGate.partyArrived
// share the exact same primitive (network.PresentPlayers), covered in
// internal/network/host_absence_test.go.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/network"
)

func TestPortalPartyExcludesAbsentPlayers(t *testing.T) {
	network.Role = "host"
	defer func() { network.Role = "" }()

	network.RemotePlayersMutex.Lock()
	network.RemotePlayers = map[string]network.PlayerState{
		"a": {PlayerID: "a"},
		"b": {PlayerID: "b", Absent: true},
	}
	network.RemotePlayersMutex.Unlock()
	defer func() {
		network.RemotePlayersMutex.Lock()
		network.RemotePlayers = nil
		network.RemotePlayersMutex.Unlock()
	}()

	party := portalParty(nil)
	if _, ok := party["b"]; ok {
		t.Error("jogador ausente nao pode contar para a porta do portal")
	}
	if _, ok := party["a"]; !ok {
		t.Error("jogador presente sumiu da contagem do portal")
	}
}

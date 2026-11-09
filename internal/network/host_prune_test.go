package network

import "testing"

// TestBroadcastRosterPrunesRemovedPlayers verifies that RemotePlayers does
// not retain IDs that are no longer in h.players. Without pruning, a player
// removed from h.players (tickAbsence, a bot taken over) would linger in
// RemotePlayers forever since BroadcastStateUpdate/BroadcastRoster only ever
// write entries.
func TestBroadcastRosterPrunesRemovedPlayers(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{
		"p1": {PlayerID: "p1"},
	}}

	RemotePlayersMutex.Lock()
	RemotePlayers = map[string]PlayerState{
		"p1":    {PlayerID: "p1"},
		"ghost": {PlayerID: "ghost"},
	}
	RemotePlayersMutex.Unlock()

	h.BroadcastRoster()

	RemotePlayersMutex.Lock()
	defer RemotePlayersMutex.Unlock()
	if _, ok := RemotePlayers["ghost"]; ok {
		t.Fatalf("expected ghost to be pruned from RemotePlayers, got %+v", RemotePlayers)
	}
	if _, ok := RemotePlayers["p1"]; !ok {
		t.Fatalf("expected p1 to remain in RemotePlayers")
	}
}

// TestBroadcastStateUpdatePrunesRemovedPlayers mirrors the above for the
// per-tick publication path.
func TestBroadcastStateUpdatePrunesRemovedPlayers(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{
		"p1": {PlayerID: "p1"},
	}, announce: newAnnouncer()}

	RemotePlayersMutex.Lock()
	RemotePlayers = map[string]PlayerState{
		"p1":    {PlayerID: "p1"},
		"ghost": {PlayerID: "ghost"},
	}
	RemotePlayersMutex.Unlock()

	h.BroadcastStateUpdate()

	RemotePlayersMutex.Lock()
	defer RemotePlayersMutex.Unlock()
	if _, ok := RemotePlayers["ghost"]; ok {
		t.Fatalf("expected ghost to be pruned from RemotePlayers, got %+v", RemotePlayers)
	}
}

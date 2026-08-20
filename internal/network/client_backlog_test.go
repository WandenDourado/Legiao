package network

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
)

// primeBuffered forces r to fill its internal buffer from the underlying
// reader without consuming anything, so Buffered() reports what a live
// socket already delivered to the kernel instead of 0 from a bufio.Reader
// that has not been read from yet.
//
// Pre-existing gap: this test package did not compile at all without it,
// which blocked every test in internal/network — not just the ones here.
// Unrelated to the climax/ultimate fix; added only so the package's tests
// (including the new ones this change adds) can run.
func primeBuffered(r *bufio.Reader) {
	r.Peek(r.Size())
}

func newlineJoined(t *testing.T, msgs ...Message) string {
	t.Helper()
	var b strings.Builder
	for _, m := range msgs {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}

func stateUpdateMsg(t *testing.T, playerID string) Message {
	t.Helper()
	payload, err := json.Marshal(StateUpdatePayload{
		Players: []PlayerState{{PlayerID: playerID}},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return Message{Type: MsgStateUpdate, Payload: payload}
}

// TestDrainStaleSnapshotsKeepsOnlyTheNewestOfTheSameType reproduces the
// resume-from-background backlog: several state_update snapshots already
// sitting in the read buffer, none of which was ever going to reach the
// screen. Only the last one should survive.
func TestDrainStaleSnapshotsKeepsOnlyTheNewestOfTheSameType(t *testing.T) {
	first := stateUpdateMsg(t, "p1")
	second := stateUpdateMsg(t, "p2")
	third := stateUpdateMsg(t, "p3")

	body := newlineJoined(t, second, third)
	c := &Client{reader: bufio.NewReader(strings.NewReader(body))}
	primeBuffered(c.reader)

	got := c.drainStaleSnapshots(first)

	var state StateUpdatePayload
	if err := json.Unmarshal(got.Payload, &state); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(state.Players) != 1 || state.Players[0].PlayerID != "p3" {
		t.Fatalf("esperava sobrar so o ultimo snapshot (p3), ficou %+v", state.Players)
	}
}

// TestDrainStaleSnapshotsStopsAtEmptyBuffer confirms the drain never blocks
// on the network: with nothing buffered, the message passed in comes back
// untouched.
func TestDrainStaleSnapshotsStopsAtEmptyBuffer(t *testing.T) {
	msg := stateUpdateMsg(t, "solo")
	c := &Client{reader: bufio.NewReader(strings.NewReader(""))}

	got := c.drainStaleSnapshots(msg)

	var state StateUpdatePayload
	if err := json.Unmarshal(got.Payload, &state); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(state.Players) != 1 || state.Players[0].PlayerID != "solo" {
		t.Fatalf("sem backlog, a mensagem devolvida deveria ser a mesma que entrou, ficou %+v", state.Players)
	}
}

// TestDrainStaleSnapshotsIgnoresNonSnapshotTypes confirms the fast path: a
// message that is not a snapshot type is returned as-is, without reading
// ahead in the buffer at all (an event is never dropped, so there is nothing
// to coalesce).
func TestDrainStaleSnapshotsIgnoresNonSnapshotTypes(t *testing.T) {
	dialogue := Message{Type: MsgDialogue, Payload: json.RawMessage(`{}`)}
	// A message sitting right after it in the buffer that must NOT be
	// consumed by this call.
	body := newlineJoined(t, stateUpdateMsg(t, "after"))
	c := &Client{reader: bufio.NewReader(strings.NewReader(body))}

	got := c.drainStaleSnapshots(dialogue)
	if got.Type != MsgDialogue {
		t.Fatalf("mensagem de evento nao deveria ser trocada, veio %s", got.Type)
	}
	if c.reader.Buffered() == 0 {
		t.Fatal("drainStaleSnapshots nao deveria ter consumido o buffer para um tipo que nao e snapshot")
	}
}

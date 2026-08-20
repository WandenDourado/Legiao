package network

import "sync"

// DialogueState is the line currently on screen, or Active=false when no
// dialogue is running. It is the whole payload of MsgDialogue.
//
// The text travels instead of a script ID plus a line number on purpose: a
// client then needs no dialogue file of its own, so a client with an older
// assets folder cannot show a different line than the host is narrating.
type DialogueState struct {
	Active   bool   `json:"active"`
	ScriptID string `json:"script_id,omitempty"`
	Speaker  string `json:"speaker,omitempty"`
	// Portrait is a character key ("mago", "paladina", ...); empty means the
	// line is narration and no image should be drawn.
	Portrait string `json:"portrait,omitempty"`
	Text     string `json:"text,omitempty"`
	// Index is 1-based, Total is the script length. Presentation only.
	Index int `json:"index,omitempty"`
	Total int `json:"total,omitempty"`
}

var (
	// CurrentDialogue is the shared render state, exactly like CurrentWave:
	// written by the network layer, read by the HUD every frame.
	CurrentDialogue      DialogueState
	CurrentDialogueMutex sync.Mutex
)

// SetDialogueState stores the latest line locally.
func SetDialogueState(d DialogueState) {
	CurrentDialogueMutex.Lock()
	defer CurrentDialogueMutex.Unlock()
	CurrentDialogue = d
}

// GetDialogueState returns a copy of the latest line.
func GetDialogueState() DialogueState {
	CurrentDialogueMutex.Lock()
	defer CurrentDialogueMutex.Unlock()
	return CurrentDialogue
}

// DialogueActive reports whether a dialogue is holding the game. Every pause
// decision (input, simulation, respawn timers) reads this one flag, so host
// and client freeze on the same condition.
func DialogueActive() bool {
	return GetDialogueState().Active
}

// PublishDialogue stores the line locally and, on a host, pushes it to every
// peer. Solo play (no role) still works: the state is set locally and nothing
// is sent.
//
// It is called on every change rather than every frame because the host's
// per-frame broadcast lives in UpdateSimulation, which is exactly what a
// dialogue freezes.
func PublishDialogue(d DialogueState) {
	SetDialogueState(d)
	if Role != "host" || CurrentHost == nil {
		return
	}
	CurrentHost.broadcast(Message{Type: MsgDialogue, Payload: MustMarshal(d)})
}

// sendDialogueTo pushes the running dialogue to one connection. A client that
// joins mid-scene would otherwise keep playing while everyone else reads.
func (h *Host) sendDialogueTo(c *ClientConn) {
	d := GetDialogueState()
	if !d.Active {
		return
	}
	h.sendTo(c, Message{Type: MsgDialogue, Payload: MustMarshal(d)})
}

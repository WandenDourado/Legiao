// Package dialogue holds the narrative scripts a map can play and the cursor
// that walks through one of them. It is pure data: it does not draw, does not
// touch the network and does not decide when a script fires. The host decides
// (see internal/game/dialogue.go), the network layer publishes the current
// line, and the UI draws it.
package dialogue

// Trigger is the moment in a map's run that starts a script.
type Trigger string

const (
	// TriggerMapStart fires once, the first frame the map is live.
	TriggerMapStart Trigger = "on_map_start"
	// TriggerWavesCleared fires once, when every horde of the map is dead.
	TriggerWavesCleared Trigger = "on_waves_cleared"
	// TriggerLastStand fires once, when the party is about to be wiped: every
	// player dead, or every player still standing below LastStandHealth.
	//
	// It is the only trigger that fires DURING a fight, and that is the point
	// — it exists for the moment a map is designed to be lost, so the scene
	// can happen instead of the Game Over.
	TriggerLastStand Trigger = "on_last_stand"
)

// PerRun reports whether a trigger belongs to ONE run of the stage rather than
// to the map visit.
//
// A cena de abertura e do MAPA: reiniciar a fase nao devolve o grupo a
// floresta, entao repetir a conversa de chegada a cada tentativa so cansaria.
// O ultimo suspiro e o fim de fase sao da CORRIDA: eles falam sobre uma luta
// especifica, e quem perdeu e recomecou vai lutar de novo.
//
// Isto ja custou um bug: o roteiro do climax ficava marcado como tocado, entao
// na segunda tentativa a cena nao abria — e como e ela que segura o Game Over,
// o grupo caia e perdia na hora, sem resgate nenhum.
func (t Trigger) PerRun() bool {
	switch t {
	case TriggerLastStand, TriggerWavesCleared:
		return true
	}
	return false
}

// LastStandHealth is the share of maximum health below which a player counts
// as "about to fall" for TriggerLastStand.
//
// It is not zero on purpose: waiting for a full wipe would put the scene after
// the Game Over the host announces, and a rescue that arrives after the run
// ended is not a rescue. 30% of the bar is late enough that the party has
// visibly lost the fight and early enough that someone is still on their feet
// to hear the line.
const LastStandHealth = 0.30

// Line is one spoken (or narrated) beat.
//
// Portrait is a character key ("mago", "paladina", ...), not an image path:
// the portrait comes from that character's reference art, so a character whose
// art changes does not need every script rewritten. An empty Portrait means no
// image at all, which is what narration and off-screen voices want.
type Line struct {
	Speaker  string `json:"speaker"`
	Portrait string `json:"portrait"`
	Text     string `json:"text"`
}

// Script is one uninterrupted sequence of lines bound to a trigger.
type Script struct {
	// ID is unique across the whole game, not just the map. It is what marks
	// a script as already played, so it must not repeat between maps.
	ID      string  `json:"id"`
	Trigger Trigger `json:"trigger"`
	Lines   []Line  `json:"lines"`
}

// File is one map's whole narrative, as stored in assets/dialogues/<map>.json.
type File struct {
	Map     string   `json:"map"`
	Scripts []Script `json:"scripts"`
}

// ByTrigger returns the first script bound to the given trigger, if any.
// One script per trigger is deliberate: two scripts firing on the same beat
// would race for the screen, and a longer scene is just more lines.
func (f File) ByTrigger(t Trigger) (Script, bool) {
	for _, s := range f.Scripts {
		if s.Trigger == t && len(s.Lines) > 0 {
			return s, true
		}
	}
	return Script{}, false
}

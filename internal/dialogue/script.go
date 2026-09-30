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
	// TriggerLastStand fires once, when the party is about to be wiped: the
	// whole party's health, summed, at or below LastStandPartyHealth of its
	// summed maximum (the dead count as zero).
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

// LastStandPartyHealth is the share of the PARTY's health — every present
// member's health summed (the dead count as zero) over their maximums summed —
// at or below which the party counts as "about to fall" for TriggerLastStand.
//
// ATE 28/09/2026 A REGRA ERA POR JOGADOR: todo mundo de pe abaixo de 30% da
// propria barra. Na pratica um unico personagem saudavel — quase sempre um bot
// que ficou longe da briga — segurava a cena, e o grupo tinha de morrer
// inteiro para o apice da fase comecar (relato do playtest). Somar o grupo e o
// pedido do Gui: um personagem quase cheio com o resto do time caido JA e um
// grupo perdendo; so o grupo inteiro de vida cheia nao e.
//
// Por que 35%. Com os cinco lugares ocupados (vaga de classe vira bot), um
// personagem cheio e quatro caidos da 20%; dois cheios e tres caidos, 40% —
// ainda uma luta em pe. Todos a um terco da barra tambem dispara, e esta
// certo: e um grupo visivelmente perdendo. Nao e zero pelo mesmo motivo de
// sempre: esperar o wipe poria a cena depois do Game Over, e resgate que chega
// depois do fim nao e resgate.
const LastStandPartyHealth = 0.35

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

// PortraitKeys lista, sem repetir, todo retrato que este arquivo pode pedir.
//
// Existe para o PRECARREGAMENTO. O retrato de um orador e a `reference.png`
// dele — 1536x1024, ~6 MB depois de decodificada —, e ela era carregada na
// primeira fala em que aparecia, ou seja DENTRO do quadro que abre a caixa de
// dialogo: um quadro isolado de ~40 ms, medido no F3 do mapa 3, toda vez que
// um personagem novo falava pela primeira vez.
//
// E o mesmo defeito que `entity.PreloadEnemyTextures` ja resolveu para as
// folhas de monstro, e a mesma solucao: descobrir o elenco no carregamento do
// mapa e pagar o custo la, onde o jogador ja espera uma pausa.
//
// A lista sai do ARQUIVO inteiro e nao do roteiro que vai tocar agora, porque
// os tres gatilhos de um mapa (abertura, hordas limpas, ultimo suspiro)
// disparam em momentos que ninguem consegue prever no carregamento — e o
// ultimo suspiro dispara no meio de uma luta perdida, que e o pior momento
// possivel para parar o quadro e ler um PNG do disco.
func (f File) PortraitKeys() []string {
	seen := make(map[string]bool)
	keys := make([]string, 0, 4)
	for _, s := range f.Scripts {
		for _, l := range s.Lines {
			if l.Portrait == "" || seen[l.Portrait] {
				continue
			}
			seen[l.Portrait] = true
			keys = append(keys, l.Portrait)
		}
	}
	return keys
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

package game

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/dialogue"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DialogueDirector decides when a scene starts and when it moves on. Only the
// machine that owns the narrative runs those decisions — the host, or the
// single player in a solo run. A client's director stays idle and the client
// simply displays whatever arrives over the network.
type DialogueDirector struct {
	// mapPath is the map the loaded scripts belong to. Comparing it against
	// the live world is how a portal transition is noticed without the portal
	// code having to know that dialogues exist.
	mapPath string
	scripts dialogue.File
	// zones e o layer `zones` do mapa vivo, guardado so para o gatilho do
	// ultimo suspiro consultar um eventual `corridor_checkpoint` (mapa 6). Os
	// outros gatilhos nao precisam dele.
	zones []tilemap.Zone
	// played is keyed by script ID (unique game-wide), so walking back into a
	// map does not replay its opening scene.
	played map[string]bool
	runner dialogue.Runner
	// stageGeneration e a corrida da fase que o `played` abaixo descreve.
	// Quando ela muda, as cenas DA CORRIDA sao esquecidas.
	stageGeneration int
	// runningLastStand marca que a cena no ar e a do resgate. Guardar isso no
	// diretor, e nao comparar o id do roteiro com uma constante, mantem o
	// codigo independente do NOME que cada mapa da a sua cena: o que importa
	// e o gatilho, e o gatilho e quem escolheu comecar.
	runningLastStand bool
}

// NewDialogueDirector returns a director with nothing loaded yet; the first
// Update binds it to whatever map is live.
func NewDialogueDirector() *DialogueDirector {
	return &DialogueDirector{played: map[string]bool{}}
}

// Update advances the narrative for one frame and reports whether the dialogue
// owned that frame. When it returns true the caller must skip input, movement
// and simulation: the game is paused.
//
// It returns true for the frame that ends a scene as well, so the tap or Enter
// that closes the box cannot also fire an attack.
func (d *DialogueDirector) Update(w *World) bool {
	if d == nil {
		return false
	}
	// A client never starts, advances or ends a scene. It only mirrors the
	// host, which is what makes the host the narrator.
	if network.Role == "client" {
		return network.DialogueActive()
	}

	d.syncMap(w)
	d.syncStageGeneration()
	wasActive := d.runner.Active()

	if !wasActive {
		d.startTriggered()
		if !d.runner.Active() {
			return false
		}
		// A scene that just opened must not consume this frame's advance
		// input, or the key that was still down would skip the first line.
		return true
	}

	if d.advanceRequested() {
		if d.runner.Advance() {
			d.publish()
		} else {
			endedLastStand := d.runningLastStand
			d.end()
			// O resgate acontece quando a ULTIMA LINHA fecha, nao quando a
			// cena abre: "Ergam-se" e a deixa, e ela e a ultima fala.
			if endedLastStand && network.CurrentHost != nil {
				network.CurrentHost.ResolveLastStand()
			}
		}
	}
	return true
}

// Paused reports whether a dialogue is currently holding the game.
func (d *DialogueDirector) Paused() bool { return network.DialogueActive() }

// syncMap reloads the scripts when the live world is not the one they came
// from, and fires the new map's opening scene on the next check.
func (d *DialogueDirector) syncMap(w *World) {
	if w == nil || w.Path == d.mapPath {
		return
	}
	d.mapPath = w.Path
	d.scripts = dialogue.LoadForMap(w.Path)
	d.zones = w.Zones
	// O elenco do mapa sobe AGORA, no quadro da troca, e nao na primeira fala
	// de cada personagem. Ver dialogue.File.PortraitKeys: sem isto, cada
	// orador novo custava um quadro de ~40 ms no meio da cena — e o gatilho do
	// ultimo suspiro cobraria esse quadro durante uma luta perdida.
	//
	// Roda em TODA maquina e nao so na que dirige a narrativa: o cliente nao
	// decide quando a cena comeca, mas desenha a mesma caixa com o mesmo
	// retrato, entao paga a mesma trava se nao precarregar.
	ui.PreloadPortraits(d.scripts.PortraitKeys())
	warnIfLastStandHasNoWindow(w.Path, d.scripts)
}

// warnIfLastStandHasNoWindow logs when a map's script has an `on_last_stand`
// scene but the map never declared a climax window
// (internal/network/climax_window.go). A script that never fires is silent
// otherwise — the same care StartWaveRun already takes for a map with
// enemy_spawn_* markers and no run.
func warnIfLastStandHasNoWindow(mapPath string, scripts dialogue.File) {
	if _, ok := scripts.ByTrigger(dialogue.TriggerLastStand); !ok {
		return
	}
	if _, ok := network.ClimaxWindowFor(mapPath); ok {
		return
	}
	log.Printf("[Climax] %s tem roteiro on_last_stand mas nenhuma janela em "+
		"climaxWindows; a cena nunca vai tocar", mapPath)
}

// syncStageGeneration forgets the scenes that belong to a single run when the
// stage restarts.
//
// Sem isso o roteiro do climax continuava marcado como tocado depois de um
// F5: a cena nao abria na segunda tentativa, e como e ela que segura o Game
// Over, o grupo caia e perdia direto. A cena de abertura NAO e esquecida, por
// isso a pergunta e por gatilho e nao "limpa tudo".
func (d *DialogueDirector) syncStageGeneration() {
	gen := network.StageGeneration()
	if gen == d.stageGeneration {
		return
	}
	d.stageGeneration = gen
	forgotten := 0
	for _, s := range d.scripts.Scripts {
		if s.Trigger.PerRun() && d.played[s.ID] {
			delete(d.played, s.ID)
			forgotten++
		}
	}
	if forgotten > 0 {
		log.Printf("[Dialogo] fase reiniciada; %d cena(s) da corrida podem tocar de novo", forgotten)
	}
}

// startTriggered starts the first script whose trigger has fired and that has
// not played yet.
func (d *DialogueDirector) startTriggered() {
	if d.triggerFired(dialogue.TriggerMapStart) {
		return
	}
	// O ultimo suspiro vem antes do fim de fase: ele acontece DURANTE a luta,
	// e perguntar por ele depois seria perguntar tarde demais.
	if d.triggerFired(dialogue.TriggerLastStand) {
		return
	}
	d.triggerFired(dialogue.TriggerWavesCleared)
}

// triggerFired starts the script bound to t when its condition holds, and
// reports whether a scene is now running.
func (d *DialogueDirector) triggerFired(t dialogue.Trigger) bool {
	script, ok := d.scripts.ByTrigger(t)
	if !ok || d.played[script.ID] {
		return false
	}
	if !d.conditionMet(t) {
		return false
	}
	if !d.runner.Start(script) {
		return false
	}
	// O ultimo suspiro segura o Game Over a partir do quadro em que a cena
	// abre. Armar aqui e nao no fim e o que importa: entre a queda do grupo e
	// a ultima linha o host anunciaria o fim, e o resgate chegaria tarde.
	d.runningLastStand = t == dialogue.TriggerLastStand
	if d.runningLastStand {
		network.ArmLastStand()
	}
	d.played[script.ID] = true
	log.Printf("[Dialogo] iniciando %s (%s)", script.ID, t)
	d.publish()
	return true
}

// conditionMet answers the trigger's question about the current run.
func (d *DialogueDirector) conditionMet(t dialogue.Trigger) bool {
	switch t {
	case dialogue.TriggerMapStart:
		// syncMap has just bound this map, so being asked at all means the
		// map is live.
		return true
	case dialogue.TriggerWavesCleared:
		// The wave runner reports "cleared" only after the last horde is dead,
		// and keeps reporting it; the played set is what stops the replay.
		state := network.GetWaveState()
		return state.Total > 0 && network.WavePhase(state.Phase) == network.WavePhaseCleared
	case dialogue.TriggerLastStand:
		return partyIsFalling(d.mapPath, d.zones)
	}
	return false
}

// partyIsFalling reports whether the group has visibly lost the fight: nobody
// left standing, or everybody still up below LastStandHealth.
//
// Two guards, and each is a decision:
//
//   - The window has to be OPEN. `network.ClimaxWindowOpen` answers a
//     question the MAP declares (internal/network/climax_window.go): which
//     horde the climax belongs to, whether it is the ambush itself, or
//     whether the party has reached a checkpoint. A map that declares no
//     window never opens one, which is what stops the old defect — any horde
//     of any map with a run used to satisfy a bare `WaveState.Total > 0`
//     check, so falling during the FIRST horde of world_02 or world_05 fired
//     the scene meant for the last one.
//   - An empty party is not a falling party. Before the first player state
//     arrives the list is empty, and "every player is below the threshold" is
//     vacuously true of nobody — which would fire the scene on frame one.
func partyIsFalling(mapPath string, zones []tilemap.Zone) bool {
	// PresentPlayers, not GetAllPlayers: an absent player parked mid-field
	// cannot hold this scene hostage, and their last-known health should not
	// count toward "everyone is below the threshold" either.
	players := network.PresentPlayers()
	if len(players) == 0 {
		return false
	}
	if !network.ClimaxWindowOpen(mapPath, zones) {
		return false
	}
	for _, p := range players {
		if p.IsDead {
			continue
		}
		if p.MaxHealth <= 0 || p.Health > p.MaxHealth*dialogue.LastStandHealth {
			return false
		}
	}
	return true
}

// advanceRequested reads the narrator's "next line" input: Enter or Space on a
// keyboard, a tap inside the box on a touch screen. Both are accepted on both
// platforms, so a host on a tablet and a host on a desktop behave the same.
func (d *DialogueDirector) advanceRequested() bool {
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) {
		return true
	}
	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return false
	}
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	// Only a tap on the box advances. Tapping elsewhere does nothing, so a
	// stray touch on the joystick area cannot skip half a scene.
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), ui.DialogueBoxRect(sw, sh))
}

// publish pushes the line on screen to every machine, including this one.
func (d *DialogueDirector) publish() {
	line, index, total := d.runner.Current()
	network.PublishDialogue(network.DialogueState{
		Active:   true,
		ScriptID: d.runner.ScriptID(),
		Speaker:  line.Speaker,
		Portrait: line.Portrait,
		Text:     line.Text,
		Index:    index,
		Total:    total,
	})
}

// end closes the scene everywhere and hands the game back to the players.
func (d *DialogueDirector) end() {
	d.runner.Stop()
	d.runningLastStand = false
	network.PublishDialogue(network.DialogueState{Active: false})
	log.Printf("[Dialogo] cena encerrada")
}

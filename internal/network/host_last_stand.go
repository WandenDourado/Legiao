package network

// O ultimo suspiro: o resgate roteirizado que salva o grupo da matilha.
//
// A cena tem tres momentos e cada um mora num lugar diferente, de proposito:
//
//	o gatilho   game.partyIsFalling -> o diretor de dialogo pede ArmLastStand
//	a fala      o dialogo congela o mundo, como qualquer cena
//	o resgate   ResolveLastStand, quando a ultima linha fecha
//
// Entre armar e resolver o Game Over fica SEGURADO. Sem isso a cena perderia
// a corrida: o grupo cai, o host anuncia o fim, e o resgate chega depois do
// jogo ter acabado — que e o mesmo que nao chegar.
//
// QUEM SE ERGUE E SEMPRE ALGUEM DA PARTIDA.
//
// Aqui morava um NPC de recurso: um "heroi invocado" que entrava em campo
// quando ninguem no grupo jogava com a classe da fase, apenas para ser dono
// de um efeito e ser desenhado. Ele existia porque a cena nao podia depender
// das escolhas de personagem do grupo — e essa dependencia acabou quando toda
// classe vaga passou a ser preenchida por um BOT (host_bots.go). Nao existe
// mais partida sem Sacerdotisa, sem Arqueiro ou sem Paladina: a cena sempre
// encontra um corpo de verdade para reerguer, e a suprema e lancada por quem
// joga com ela, humano ou bot.
//
// O que se ganha ao remove-lo nao e so codigo. O NPC nao era um jogador — nao
// contava no HUD, nao pesava no Game Over, nao morria e nao era sincronizado —
// e cada uma dessas excecoes era uma regra a menos valendo dentro de uma cena
// que e justamente o momento mais delicado da fase.

import (
	"log"
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// O resgate atende duas pessoas diferentes, com dois tamanhos.
//
// O HEROI DA FASE e quem age: ele volta INTEIRO e com alguns segundos, porque
// precisa sair dos dentes, escolher a direcao e lancar. O resto do grupo so
// precisa nao morrer no proximo golpe — mas precisa disso de verdade, porque a
// cena dispara com todo mundo abaixo de um quarto da vida e devolver o jogo
// naquele estado e devolve-lo perdido.
const (
	// LastStandInvulnerability is how long the rescued hero cannot be hurt.
	//
	// Eram dois segundos, e em jogo isso nao dava. A cena termina com o heroi
	// no meio da horda que acabou de derrubar o grupo — na quinta onda do mapa
	// 5 sao 35 simultaneos — e dois segundos e o tempo de tomar um golpe, nao
	// de sair dos dentes, escolher a direcao e lancar. Cinco.
	LastStandInvulnerability float32 = 5.0
	// LastStandPartyHeal is the share of maximum health everyone else is
	// brought back to.
	LastStandPartyHeal float32 = 0.30
	// LastStandPartyInvulnerability is the breath the rest of the party gets
	// to reposition — half the caster's, because they are not the ones who
	// have to aim anything.
	LastStandPartyInvulnerability float32 = 1.0
)

type lastStandState struct {
	mu sync.RWMutex
	// armed is true from the moment the scene is requested until it resolves.
	armed bool
	// done marks the rescue as already spent for this run of the stage.
	done bool
}

var lastStand lastStandState

// LastStandPending reports whether the rescue is owed. The Game Over check
// consults it, which is the whole reason the flag exists.
func LastStandPending() bool {
	lastStand.mu.RLock()
	defer lastStand.mu.RUnlock()
	return lastStand.armed
}

// LastStandDone reports whether the rescue already happened in this run, so
// the scene cannot play twice on a stage the party keeps losing.
func LastStandDone() bool {
	lastStand.mu.RLock()
	defer lastStand.mu.RUnlock()
	return lastStand.done
}

// ArmLastStand holds the Game Over back because the scene is about to play.
// Called by the dialogue director the frame the scene starts.
func ArmLastStand() {
	lastStand.mu.Lock()
	defer lastStand.mu.Unlock()
	if lastStand.done {
		return
	}
	lastStand.armed = true
	log.Printf("[UltimoSuspiro] cena armada; Game Over segurado")
}

// ResetLastStand clears the scene for a fresh run of the stage.
//
// The fields are cleared one by one instead of assigning a zero struct: the
// state carries its own mutex, and overwriting the struct would copy a lock —
// which go vet rejects, and which here would also replace the very lock being
// held to do it.
func ResetLastStand() {
	lastStand.mu.Lock()
	defer lastStand.mu.Unlock()
	lastStand.armed = false
	lastStand.done = false
}

// ResolveLastStand performs the rescue as the scene's last line closes.
//
// Uma forma so, desde que os bots existem: quem joga a classe do heroi — humano
// ou bot — volta de pe, com vida cheia, alguns segundos de imunidade e a
// suprema recarregada e destravada. A CENA ENTREGA O MOMENTO A QUEM ESTA
// JOGANDO; ela nao o encena por ninguem.
//
// QUEM e o heroi vem de LastStandHeroFor(h.stageMap) — no mapa 2 o
// Necromante, no 3 a Sacerdotisa, no 4 o Arqueiro.
func (h *Host) ResolveLastStand() {
	lastStand.mu.Lock()
	if lastStand.done {
		lastStand.mu.Unlock()
		return
	}
	lastStand.done = true
	lastStand.armed = false
	lastStand.mu.Unlock()

	hero := LastStandHeroFor(h.stageMap)
	saved := h.reviveHero(hero.character)
	// Todo mundo que NAO e o heroi recebe o resgate menor. Sem isso a
	// cena devolvia o jogo com o grupo ainda abaixo de um quarto da vida, e o
	// primeiro lobo desfazia o que ela acabou de fazer.
	h.reviveParty(saved)

	// O mapa 6 tem um SEGUNDO ATO, e ele nao e uma magia: os dois canhoes do
	// corredor sao destruidos pelo roteiro. Isto continua roteirizado — e nao
	// entregue a Paladina como as outras supremas — porque o Avatar dos Deuses
	// e imunidade total, nao um ataque a distancia: sem esta parte o resgate do
	// mapa 6 devolveria o grupo vivo dentro do mesmo corredor bombardeado.
	// Ver castCannonJudgment.
	if h.stageMap == "assets/maps/world_06.json" && saved != "" {
		h.castCannonJudgment(saved)
	}

	if saved == "" {
		// Nao deveria acontecer: ReconcileBots mantem exatamente um bot por
		// classe sem humano, entao toda classe esta sempre ocupada. Se
		// acontecer, a cena passa sem resgate — e isso precisa aparecer no log,
		// nao em silencio no meio de um Game Over.
		log.Printf("[UltimoSuspiro] ninguem em campo joga com %s; a cena "+
			"passou sem resgate (bot da classe faltando?)", hero.character)
		return
	}
	log.Printf("[UltimoSuspiro] %s se ergueu; a ultimate e dele", saved)
}

// castCannonJudgment e o resgate do mapa 6: com o Avatar dos Deuses erguido,
// a Paladina alcanca o fim do corredor e destroi os dois canhoes de uma vez.
//
// Isto e roteirizado a parte da magia selecionavel — Avatar dos Deuses e
// imunidade total, nao um ataque a distancia — porque o resgate precisa de um
// contrato PRECISO: os canhoes declarados pelo mapa, e nao um alvo que o
// jogador teria de mirar. A Paladina nao anda fisicamente ate la — a narracao
// do dialogo e que conta a corrida, e a explosao acontece em cima do canhao,
// que e onde o jogador precisa ve-la.
func (h *Host) castCannonJudgment(ownerID string) {
	hit := h.DestroyCannons()
	for i, pos := range hit {
		h.broadcastCannonBall("impact", "judgment"+ownerID+string(rune('a'+i)), pos, rl.Vector2{})
	}
	if len(hit) == 0 {
		log.Printf("[UltimoSuspiro] julgamento da Paladina nao encontrou canhao de pe")
		return
	}
	log.Printf("[UltimoSuspiro] julgamento da Paladina destruiu %d canhao(oes); o caminho para a porta esta livre", len(hit))
}

// reviveParty brings everyone except `except` back to LastStandPartyHeal of
// their health, with a short window to get out of the way.
//
// A dead player is raised too: the scene is a rescue, and leaving a body on
// the ground for the thirty-second respawn while the ultimate clears the field
// would mean the player watches their own rescue.
func (h *Host) reviveParty(except string) {
	var revived []PlayerState
	h.playersMutex.Lock()
	for id, p := range h.players {
		if id == except {
			continue
		}
		p.IsDead = false
		p.RespawnIn = 0
		if healed := p.MaxHealth * LastStandPartyHeal; p.Health < healed {
			p.Health = healed
		}
		revived = append(revived, *p)
	}
	h.playersMutex.Unlock()

	for _, p := range revived {
		h.GrantInvulnerability(p.PlayerID, LastStandPartyInvulnerability)
		h.broadcastRespawn(p)
	}
	if len(revived) > 0 {
		log.Printf("[UltimoSuspiro] %d jogador(es) reerguidos a %.0f%% com %.0fs de folga",
			len(revived), LastStandPartyHeal*100, LastStandPartyInvulnerability)
	}
}

// reviveHero puts the map's hero back on their feet at full health, and
// returns their id — or "" when nobody in the party plays that character.
//
// It revives even a hero who is still standing: the scene fires with the
// party under a quarter of health, so "alive" here means barely, and sending
// them into the rescue on that sliver would waste the moment.
func (h *Host) reviveHero(character entity.CharacterType) string {
	var revived *PlayerState
	h.playersMutex.Lock()
	// Map iteration order is random, and a class CAN have more than one
	// player of it (plan §3, "classe duplicada"). Pick deterministically:
	// a present human first, then an absent one, then the bot last — a bot
	// invariably losing this pick if a human is in the running at all.
	best := -1
	for id, p := range h.players {
		if entity.CharacterType(p.Character) != character {
			continue
		}
		rank := 0
		switch {
		case isBotID(id):
			rank = 2
		case p.Absent:
			rank = 1
		}
		if best == -1 || rank < best {
			best, revived = rank, p
		}
	}
	if revived != nil {
		revived.IsDead = false
		revived.RespawnIn = 0
		revived.Health = revived.MaxHealth
		snapshot := *revived
		revived = &snapshot
	}
	h.playersMutex.Unlock()
	if revived == nil {
		return ""
	}
	h.GrantInvulnerability(revived.PlayerID, LastStandInvulnerability)
	// E a ultimate, que e o motivo da cena existir. Reerguer o heroi sem
	// devolver a magia dele entregava um resgate mudo: ele volta inteiro,
	// invulneravel, e a habilidade que o dialogo acabou de anunciar continua
	// em recarga. Ver ClearSkillCooldown.
	h.ClearSkillCooldown(revived.PlayerID, LastStandHeroFor(h.stageMap).skillID)
	// A campanha so entrega a suprema dele na fase SEGUINTE
	// (game.UltimatesGrantedOn), entao sem isto a cena anunciava a magia e o
	// host recusava o lancamento: o gate (skillUnlocked) e o mesmo de sempre,
	// e ele nao sabia que o resgate acabou de acontecer. A concessao vale so
	// PARA ESTA CORRIDA (ver progression.go) e precisa chegar ao cliente, ou o
	// botao do celular continua apagado e o R do desktop continua mudo.
	GrantUltimateForRun(character)
	h.BroadcastUltimateGrant(character)
	h.broadcastRespawn(*revived)
	return revived.PlayerID
}

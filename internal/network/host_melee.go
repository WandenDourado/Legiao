package network

// Host-side handling for the Paladina's basic melee attack (sword sweep):
// spawn + instant arc damage, anchor sync so the blade follows the owner,
// and the broadcast that lets clients replicate the swing visual.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/skill"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// performSwordAttack resolves the Paladina's basic attack on the host:
// spawns the sweep visual, applies SwordDamage once to every enemy inside the
// 120° arc, and broadcasts the swing to all clients.
// Caller must hold playersMutex.
func (h *Host) performSwordAttack(playerID string, origin, dir rl.Vector2) {
	sweep := skill.NewSwordSweep(playerID, origin, dir)
	// Divine Avatar transfigures the basic attack too: a much larger golden
	// greatsword with empowered damage, for as long as the ultimate lasts.
	sweep.Empowered = h.Skills.HasAvatar(playerID)
	h.Skills.AddSword(sweep)

	// Instant arc damage: the sweep visual plays over ~0.3s, but the hit is
	// resolved at cast time for responsiveness and authoritative simplicity.
	dmg := sweep.Damage()
	for _, e := range h.EntityManager.GetAllEnemies() {
		// HitCenter, e nao Position: o arco tem de alcancar o corpo, nao o pe.
		if !sweep.InArc(e.HitCenter(), e.HitRadius()) {
			continue
		}
		if e.TakeDamage(dmg) {
			log.Printf("[Host] Enemy %s died from sword sweep", e.ID)
			h.EntityManager.RemoveEnemy(e.ID)
			h.broadcastCombatEvent("death", e.ID, "enemy", 0, playerID)
		} else {
			h.broadcastCombatEvent("damage", e.ID, "enemy", dmg, playerID)
		}
	}

	// O mesmo arco derruba canhao (mapa 6). Fica aqui, e nao num sistema a
	// parte, porque para quem joga e UM golpe: a espada acerta o que estiver
	// na frente dela, e nao importa se aquilo e bicho ou pedra armada.
	hitCannons, fallen := h.damageCannonsInSweep(sweep, dmg)
	h.publishCannonState(hitCannons)
	h.announceCannonsFallen(playerID, fallen)

	h.broadcastMelee(playerID, origin, dir, sweep.Empowered)
}

// announceCannonsFallen mostra a queda dos postos que zeraram: a explosao em
// cima de cada um, nas duas pontas. A vida e a troca de arte ja foram por
// publishCannonState; aqui e so o estouro.
func (h *Host) announceCannonsFallen(playerID string, fallen []rl.Vector2) {
	for i, pos := range fallen {
		// O estouro na maquina do host tambem: broadcastCannonBall so fala com
		// os clientes, e quem derrubou o canhao esta olhando para ele.
		skill.AddCannonBurstAt(true, pos)
		h.broadcastCannonBall("impact", "cannonfall"+playerID+string(rune('a'+i)), pos, rl.Vector2{})
	}
}

// broadcastMelee replicates a sword sweep on every client.
func (h *Host) broadcastMelee(ownerID string, origin, dir rl.Vector2, empowered bool) {
	payload := MeleePayload{
		OwnerID:   ownerID,
		X:         int(origin.X),
		Y:         int(origin.Y),
		DirX:      dir.X,
		DirY:      dir.Y,
		Empowered: empowered,
	}
	h.broadcast(Message{Type: MsgMelee, Payload: MustMarshal(payload)})
}

// handleSwordTick keeps every sweep glued to its owner and advances/prunes
// the sweep animations (host authoritative visuals).
func (h *Host) handleSwordTick(dt float32) {
	h.playersMutex.RLock()
	anchors := make(map[string]rl.Vector2, len(h.players))
	for id, p := range h.players {
		anchors[id] = rl.NewVector2(float32(p.X), float32(p.Y))
	}
	h.playersMutex.RUnlock()

	for id, pos := range anchors {
		h.Skills.SetSwordAnchor(id, pos)
	}
	h.Skills.AdvanceSwords(dt)
}

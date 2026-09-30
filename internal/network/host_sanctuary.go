package network

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// HandleSanctuarySkill casts the Sacerdotisa's Sanctuary skill for playerID.
// It is gated by the per-player cooldown and validates that the caster is a
// Sacerdotisa who is still alive. Mirrors HandleFireSkill flow (host-only).
func (h *Host) HandleSanctuarySkill(playerID string) {
	p, ok := h.players[playerID]
	if !ok || p.IsDead {
		return
	}
	if entity.CharacterType(p.Character) != entity.CharSacerdotisa {
		return
	}
	if cd, ok := h.sanctuaryCooldowns[playerID]; ok && cd > 0 {
		return // on cooldown
	}
	h.castSanctuary(playerID)
	h.sanctuaryCooldowns[playerID] = skill.SanctuaryCooldown
}

// castSanctuary spawns the aura ON the caster and broadcasts the spawn so
// clients start drawing it. Nao ha mais direcao nem distancia a resolver: a
// aura e ancorada na Sacerdotisa e handleSanctuaryTick a reancora todo quadro.
func (h *Host) castSanctuary(playerID string) {
	p := h.players[playerID]
	center := rl.NewVector2(float32(p.X), float32(p.Y))
	skill.SpawnSanctuary(h.Skills, playerID, center)
	h.broadcastSanctuary(playerID, center)
}

// broadcastSanctuary tells clients to render a sanctuary at a world position.
func (h *Host) broadcastSanctuary(ownerID string, center rl.Vector2) {
	payload := SanctuaryPayload{
		SanctuaryID: ownerID, // one sanctuary per caster; reuse owner as key
		OwnerID:     ownerID,
		X:           int(center.X),
		Y:           int(center.Y),
	}
	msg := Message{Type: MsgSanctuary, Payload: MustMarshal(payload)}
	h.broadcast(msg)
}

// handleSanctuaryTick advances all sanctuaries on the host: ages them,
// heals living allies inside the area, and ticks cooldowns. Healing is
// authoritative here and synced to clients via "heal" combat events.
func (h *Host) handleSanctuaryTick(dt float32) {
	h.anchorSanctuaries()
	allies := h.allyHealTargets()
	events := h.Skills.UpdateSanctuaries(dt, allies)
	for _, ev := range events {
		if p, ok := h.players[ev.PlayerID]; ok {
			p.Health += ev.Amount
			if p.Health > p.MaxHealth {
				p.Health = p.MaxHealth
			}
			h.broadcastCombatEvent("heal", ev.PlayerID, "player", p.Health, "")
		}
	}
	for id, cd := range h.sanctuaryCooldowns {
		if cd > 0 {
			h.sanctuaryCooldowns[id] -= dt
			if h.sanctuaryCooldowns[id] < 0 {
				h.sanctuaryCooldowns[id] = 0
			}
		}
	}
}

// anchorSanctuaries cola cada aura no dono dela antes de a cura ser cobrada.
//
// A ORDEM E A REGRA INTEIRA: reancorar DEPOIS de curar cobraria a cura deste
// quadro no circulo do quadro passado, e com a Sacerdotisa andando (ainda por
// cima 25% mais rapido) a borda medida ficaria sempre um passo atras do que o
// jogador ve na tela. E a mesma razao de tickAvatars ancorar antes de
// UpdateAvatars.
//
// O dono morto perde a aura em ClearOwner (skill/clear_owner.go), junto com
// todo o resto que estava preso a ele; aqui basta seguir quem esta de pe.
func (h *Host) anchorSanctuaries() {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		h.Skills.SetSanctuaryAnchor(id, rl.NewVector2(float32(p.X), float32(p.Y)))
	}
}

// allyHealTargets builds a heal-target view for every living player.
func (h *Host) allyHealTargets() map[string]skill.PlayerHealTarget {
	targets := make(map[string]skill.PlayerHealTarget)
	for id, p := range h.players {
		targets[id] = skill.PlayerHealTarget{
			X:         float32(p.X),
			Y:         float32(p.Y),
			Health:    p.Health,
			MaxHealth: p.MaxHealth,
			IsDead:    p.IsDead,
		}
	}
	return targets
}

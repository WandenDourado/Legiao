package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"
)

// ShieldSkill is the Paladina's Escudo Sagrado ability: a holy energy barrier
// that follows the caster and absorbs damage until its strength is depleted.
// Recasting while active regenerates it to full (never stacks).
type ShieldSkill struct{}

// NewShieldSkill registers the shield skill in the global registry.
func NewShieldSkill() *ShieldSkill {
	s := &ShieldSkill{}
	RegisterSkill(s)
	return s
}

func (ShieldSkill) ID() string { return "shield" }

func (ShieldSkill) Cooldown() float32 { return skill.ShieldCooldown }

func (ShieldSkill) Cast(ctx *CastContext) {
	// Untargeted: the aura is anchored on the caster; anchor then follows the
	// owner every tick (host: handleShieldTick, client: SyncShieldAnchors).
	skill.ActivateShield(ctx.Host.SkillManager(), ctx.PlayerID, ctx.Position)
	ctx.Host.BroadcastSkill("shield", ctx.PlayerID, ctx.Position)
}

func (ShieldSkill) Draw(m *skill.Manager) { m.DrawShields() }

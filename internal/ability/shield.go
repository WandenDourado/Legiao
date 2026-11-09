package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"
)

// ShieldSkill is the Paladina's Escudo Sagrado ability: a holy energy barrier
// that follows the caster and absorbs damage until its strength is depleted.
// Recasting while active regenerates it to full (never stacks).
//
// Lancar tambem CURA a Paladina em ShieldCastHeal da vida maxima. A cura sai
// no lancamento e nao ao longo do escudo de proposito: e o que transforma o Q
// na decisao de segurar a linha em vez de recuar, e vale igual no relancamento
// — quem gasta a recarga para renovar o escudo antes dele estourar paga por
// isso em tempo de recarga, e recebe a mesma vida.
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
	ctx.Host.HealPlayerFraction(ctx.PlayerID, skill.ShieldCastHeal)
	ctx.Host.BroadcastSkill("shield", ctx.PlayerID, ctx.Position)
}

func (ShieldSkill) Draw(m *skill.Manager) { m.DrawShields() }

package ability

// The Necromante's two skills: Cemitério (Q) and Legião Espectral (R).
// Thin Strategy wrappers over their internal/skill implementations.

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Necromante Q: Cemitério ---

// GraveyardSkill raises a rectangle of rotting, cursed ground in the aimed
// direction: skeletal hands claw out of the dirt while enemies crossing it
// take damage over time and are slowed.
type GraveyardSkill struct{}

func NewGraveyardSkill() *GraveyardSkill { s := &GraveyardSkill{}; RegisterSkill(s); return s }

func (GraveyardSkill) ID() string        { return "graveyard" }
func (GraveyardSkill) Cooldown() float32 { return skill.GraveyardCooldown }

func (GraveyardSkill) Cast(ctx *CastContext) {
	dir := rl.Vector2Subtract(ctx.Aim, ctx.Position)
	if rl.Vector2Length(dir) < 1 {
		dir = rl.NewVector2(0, 1)
	}
	dir = rl.Vector2Normalize(dir)
	origin := rl.Vector2Add(ctx.Position, rl.Vector2Scale(dir, skill.GraveyardOffset))
	skill.SpawnGraveyard(ctx.Host.SkillManager(), ctx.PlayerID, origin, dir)
	ctx.Host.BroadcastSkillDir("graveyard", ctx.PlayerID, origin, dir)
}

func (GraveyardSkill) Draw(m *skill.Manager) { m.DrawGraveyards() }

// --- Necromante R: Legião Espectral ---

// SpectralLegionSkill summons 30 fast specters in a circle around the
// necromancer. They guard him (leashed to his position) and dive at enemies
// that come close, each specter consuming itself to strike. The legion lasts
// until every specter has died; recasting summons a fresh circle.
type SpectralLegionSkill struct{}

func NewSpectralLegionSkill() *SpectralLegionSkill {
	s := &SpectralLegionSkill{}
	RegisterSkill(s)
	return s
}

func (SpectralLegionSkill) ID() string        { return "spectral_legion" }
func (SpectralLegionSkill) Cooldown() float32 { return skill.LegionCooldown }

func (SpectralLegionSkill) Cast(ctx *CastContext) {
	skill.ActivateLegion(ctx.Host.SkillManager(), ctx.PlayerID, ctx.Position)
	ctx.Host.BroadcastSkill("spectral_legion", ctx.PlayerID, ctx.Position)
}

func (SpectralLegionSkill) Draw(m *skill.Manager) { m.DrawLegions() }

package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// FireballSkill is the Mago's targeted Bola de Fogo ability.
type FireballSkill struct{}

// NewFireballSkill registers the fireball skill in the global registry.
func NewFireballSkill() *FireballSkill {
	s := &FireballSkill{}
	RegisterSkill(s)
	return s
}

func (FireballSkill) ID() string { return "fireball" }

func (FireballSkill) Cooldown() float32 { return skill.FireballCooldown }

func (FireballSkill) Cast(ctx *CastContext) {
	start := ctx.Position
	dir := rl.Vector2Subtract(ctx.Aim, start)
	if rl.Vector2Length(dir) < 1 {
		return
	}
	skill.SpawnFireball(ctx.Host.SkillManager(), ctx.PlayerID, start, dir)
	// Aimed broadcast: clients need origin + direction to replicate the
	// traveling fireball (not just a point event).
	ctx.Host.BroadcastSkillDir("fireball", ctx.PlayerID, start, dir)
}

func (FireballSkill) Draw(m *skill.Manager) { m.DrawFire() }

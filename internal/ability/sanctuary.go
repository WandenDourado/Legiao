package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// SanctuarySkill is the Sacerdotisa's ground-heal ability.
type SanctuarySkill struct{}

// NewSanctuarySkill registers the sanctuary skill in the global registry.
func NewSanctuarySkill() *SanctuarySkill {
	s := &SanctuarySkill{}
	RegisterSkill(s)
	return s
}

func (SanctuarySkill) ID() string { return "sanctuary" }

func (SanctuarySkill) Cooldown() float32 { return skill.SanctuaryCooldown }

func (SanctuarySkill) Cast(ctx *CastContext) {
	// Sanctuary drops at the caster's feet, offset to match the original
	// placement (area appears just ahead, not on top, of the priestess).
	center := rl.Vector2Add(ctx.Position, rl.NewVector2(0, skill.SanctuaryOffset))
	skill.SpawnSanctuary(ctx.Host.SkillManager(), ctx.PlayerID, center)
	ctx.Host.BroadcastSkill("sanctuary", ctx.PlayerID, center)
}

func (SanctuarySkill) Draw(m *skill.Manager) { m.DrawSanctuaries() }

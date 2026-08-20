package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ArrowVolleySkill is the Arqueiro's Rajada de Flechas ability: five arrows
// leave the archer together and fan out over a cone toward the aim point.
type ArrowVolleySkill struct{}

// NewArrowVolleySkill registers the arrow volley skill in the global registry.
func NewArrowVolleySkill() *ArrowVolleySkill {
	s := &ArrowVolleySkill{}
	RegisterSkill(s)
	return s
}

func (ArrowVolleySkill) ID() string { return "arrow_volley" }

func (ArrowVolleySkill) Cooldown() float32 { return skill.ArrowVolleyCooldown }

func (ArrowVolleySkill) Cast(ctx *CastContext) {
	start := ctx.Position
	dir := rl.Vector2Subtract(ctx.Aim, start)
	if rl.Vector2Length(dir) < 1 {
		return
	}
	skill.SpawnArrowVolley(ctx.Host.SkillManager(), ctx.PlayerID, start, dir)
	ctx.Host.BroadcastSkillDir("arrow_volley", ctx.PlayerID, start, dir)
}

func (ArrowVolleySkill) Draw(m *skill.Manager) { m.DrawArrows() }

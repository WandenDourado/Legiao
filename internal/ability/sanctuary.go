package ability

import (
	"github.com/WandenDourado/Legiao/internal/skill"
)

// SanctuarySkill is the Sacerdotisa's mobile healing aura.
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
	// A aura nasce EM CIMA da conjuradora, nao a frente dela: ela e ancorada
	// na Sacerdotisa e segue o passo dela dali em diante (host_sanctuary.go
	// reancora todo quadro). O deslocamento que existia aqui era do tempo em
	// que o Santuario era um lugar escolhido no chao.
	center := ctx.Position
	skill.SpawnSanctuary(ctx.Host.SkillManager(), ctx.PlayerID, center)
	ctx.Host.BroadcastSkill("sanctuary", ctx.PlayerID, center)
}

func (SanctuarySkill) Draw(m *skill.Manager) { m.DrawSanctuaries() }

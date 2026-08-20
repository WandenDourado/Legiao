package ability

// The four ultimate (supreme) skills. Each is a thin Strategy wrapper over
// its internal/skill implementation; they bind to slot 1 (see binds.go) and
// are cast with R on desktop / the golden button on Android.

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Mago: Chuva de Meteoros ---

// MeteorRainSkill rains meteors over the WHOLE map for 15s; damage only on
// ground impact. The host spawns each meteor at a random point (host_meteor).
type MeteorRainSkill struct{}

func NewMeteorRainSkill() *MeteorRainSkill { s := &MeteorRainSkill{}; RegisterSkill(s); return s }

func (MeteorRainSkill) ID() string        { return "meteor_rain" }
func (MeteorRainSkill) Cooldown() float32 { return skill.MeteorRainCooldown }

func (MeteorRainSkill) Cast(ctx *CastContext) {
	skill.StartMeteorRain(ctx.Host.SkillManager(), ctx.PlayerID, rl.Vector2{})
	// No activation broadcast needed: every meteor is replicated individually
	// as it spawns, which is what clients actually render.
}

func (MeteorRainSkill) Draw(m *skill.Manager) { m.DrawMeteors() }

// --- Sacerdotisa: Área Angelical ---

// AngelicAreaSkill surrounds the caster with a heavenly white/blue zone that
// resurrects the fallen and constantly heals allies inside. Follows her.
type AngelicAreaSkill struct{}

func NewAngelicAreaSkill() *AngelicAreaSkill { s := &AngelicAreaSkill{}; RegisterSkill(s); return s }

func (AngelicAreaSkill) ID() string        { return "angelic_area" }
func (AngelicAreaSkill) Cooldown() float32 { return skill.AngelicCooldown }

func (AngelicAreaSkill) Cast(ctx *CastContext) {
	skill.ActivateAngelic(ctx.Host.SkillManager(), ctx.PlayerID, ctx.Position)
	ctx.Host.BroadcastSkill("angelic_area", ctx.PlayerID, ctx.Position)
}

func (AngelicAreaSkill) Draw(m *skill.Manager) { m.DrawAngelics() }

// --- Arqueiro: Flechas Celestiais ---

// CelestialArrowsSkill grants two individually aimed celestial arrows that
// cross the whole map piercing enemies; the cooldown arms after the second
// shot (ability.Charged).
type CelestialArrowsSkill struct{}

func NewCelestialArrowsSkill() *CelestialArrowsSkill {
	s := &CelestialArrowsSkill{}
	RegisterSkill(s)
	return s
}

func (CelestialArrowsSkill) ID() string        { return "celestial_arrows" }
func (CelestialArrowsSkill) Cooldown() float32 { return skill.CelestialCooldown }
func (CelestialArrowsSkill) Charges() int      { return skill.CelestialCharges }

func (CelestialArrowsSkill) Cast(ctx *CastContext) {
	start := ctx.Position
	dir := rl.Vector2Subtract(ctx.Aim, start)
	if rl.Vector2Length(dir) < 1 {
		dir = rl.NewVector2(0, 1)
	}
	skill.SpawnCelestialArrow(ctx.Host.SkillManager(), ctx.PlayerID, start, dir)
	ctx.Host.BroadcastSkillDir("celestial_arrows", ctx.PlayerID, start, dir)
}

func (CelestialArrowsSkill) Draw(m *skill.Manager) { m.DrawCelestials() }

// --- Paladina: Avatar dos Deuses ---

// DivineAvatarSkill transfigures the paladin: 15s of total damage immunity
// while golden energy emanates from her.
type DivineAvatarSkill struct{}

func NewDivineAvatarSkill() *DivineAvatarSkill { s := &DivineAvatarSkill{}; RegisterSkill(s); return s }

func (DivineAvatarSkill) ID() string        { return "divine_avatar" }
func (DivineAvatarSkill) Cooldown() float32 { return skill.AvatarCooldown }

func (DivineAvatarSkill) Cast(ctx *CastContext) {
	skill.ActivateAvatar(ctx.Host.SkillManager(), ctx.PlayerID, ctx.Position)
	ctx.Host.BroadcastSkill("divine_avatar", ctx.PlayerID, ctx.Position)
}

func (DivineAvatarSkill) Draw(m *skill.Manager) { m.DrawAvatars() }

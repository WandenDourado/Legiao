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
	// O SELO E A UNICA COISA DESTA SUPREMA QUE PRECISA DE DONO NO FIO.
	//
	// Os meteoros continuam sem: cada um viaja como um alvo no mapa e ninguem
	// precisa saber de quem ele e para desenha-lo cair. O pentagrama e o
	// oposto — ele so quer dizer alguma coisa se estiver aos pes de UM
	// jogador —, e por isso ele viaja com o ownerID, uma vez, no lancamento.
	// Depois disso os dois lados o envelhecem pelo mesmo relogio.
	skill.ActivateMeteorSigil(ctx.Host.SkillManager(), ctx.PlayerID, ctx.Position)
	ctx.Host.BroadcastSkill("meteor_sigil", ctx.PlayerID, ctx.Position)
}

// So os meteoros. O SELO NAO E DESENHADO AQUI: `ability.DrawAll` roda depois
// dos corpos, e uma estrela aditiva por cima do mago lavava o rosto e a roupa
// dele (relato do Gui, 10/09/2026). Ele e decalque de chao, entao vai antes
// dos corpos, junto com o portal — ver `DrawMeteorSigils` em game/renderer.go.
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

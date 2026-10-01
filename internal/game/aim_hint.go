package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiAimIndicator is the ground chevron that says where the next shot goes.
// One instance, because it carries the fade between frames.
var uiAimIndicator = ui.NewAimIndicator()

// resolveAimHint picks the direction the ground chevron shows this frame.
//
// The three action buttons keep three INDEPENDENT saved directions
// (AimJoystick.AimDir, and one SkillDir per SkillButton), so "the aim" is not
// a single value the game can read — it is whichever button the thumb is on.
// Priority runs outside-in through the cluster (R, then Q, then attack). It
// only ever matters with two fingers down, which the aim lock now prevents
// (input.ActionLock) — it survives as the tie-break for the frame a claim and
// a release cross, and the ultimate wins because it is the deliberate one.
//
// With no finger anywhere it falls back to the ATTACK direction, since that is
// what a tap on the biggest button would fire.
func resolveAimHint(aj *input.AimJoystick) ui.AimHint {
	switch {
	case uiUltimateButton.IsActive:
		return ui.AimHint{Dir: uiUltimateButton.SkillDir, Aiming: true}
	case uiSkillButton.IsActive:
		return ui.AimHint{Dir: uiSkillButton.SkillDir, Aiming: true}
	case aj.IsActive:
		return ui.AimHint{Dir: aj.AimDir, Aiming: true}
	default:
		return ui.AimHint{Dir: aj.AimDir}
	}
}

// drawAimHint places the chevron on the ray the shot actually travels.
//
// The anchor is `p.Position` and not the soles because Position is where the
// projectile is BORN: `ProcessInput` aims at `p.Position + dir*100` and the
// host spawns from the same point. Anchored at the soles the mark was a
// parallel ray one GroundOffset (~103 px) below the fireball — same angle,
// wrong line, and on a phone that reads as the spell ignoring the indicator.
//
// Nothing is drawn for a player who cannot shoot: a corpse has no aim, and a
// body waiting inside a portal is not even on the map (renderer.go skips it),
// so a chevron there would be a mark floating over empty ground.
func drawAimHint(p *entity.Player, aj *input.AimJoystick) {
	if p.IsDead || network.GameOver || network.LocalPlayerInPortal {
		return
	}
	// Configuracoes > Interface > Mostrar indicador de mira.
	if !settings.Get().ShowAimHint {
		return
	}
	uiAimIndicator.Draw(resolveAimHint(aj), p.Position, rl.GetFrameTime())
}

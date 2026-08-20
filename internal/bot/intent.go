package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// Cast is a skill the bot wants to fire this tick, at Aim (a world-space
// point or, for aimed skills, a point that encodes a direction from Self).
type Cast struct {
	SkillID string
	Aim     rl.Vector2
}

// Intent is what a Brain decided to do this tick.
//
// Movement is expressed as WHERE, not HOW: Dest is a destination point (only
// meaningful when HasDest is true), and Push is the local separation from
// allies the brain already computed (steering.separation). It is the host's
// job (host_bot_tick.go), not the brain's, to turn Dest into an actual
// direction — by consulting the navigation mesh when the straight line to
// Dest is blocked — and blend Push on top of it. A brain that used to call
// seekAndSeparate directly was deciding BOTH where to go and how to get
// there; now it only decides where.
//
// Attack, when non-nil, is the world-space point to fire the basic attack
// at. Skill, when non-nil, is one skill cast to request.
type Intent struct {
	Dest    rl.Vector2
	HasDest bool
	Push    rl.Vector2
	Attack  *rl.Vector2
	Skill   *Cast
}

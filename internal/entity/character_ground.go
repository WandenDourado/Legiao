package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// GroundOffset is how far below Position a character's soles are drawn.
//
// A character sprite is drawn centered on Position, but the figure inside it
// stands: the frame is 192 px tall and the sole sits at 187, so at a render
// scale of 1.15 the feet land about 105 px below the point the game was
// colliding with. That gap is a bug the player can see — walking above a tree
// trunk, the collision box has already cleared it while the feet are still
// drawn on top of the roots.
//
// Placing the collision box here instead of at the sprite's centre makes the
// rule the obvious one: what blocks the character is what its feet touch.
func GroundOffset(def CharacterDef) float32 {
	footLine := float32(def.FootLine)
	if footLine <= 0 {
		footLine = float32(def.FrameHeight)
	}
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	return (footLine - float32(def.FrameHeight)/2) * scale
}

// GroundPoint is where a character's feet are in world space.
func GroundPoint(pos rl.Vector2, def CharacterDef) rl.Vector2 {
	return rl.NewVector2(pos.X, pos.Y+GroundOffset(def))
}

// PlayerGroundBox is the box movement resolution actually tests, centered on
// the point returned here.
//
// The box keeps the size it always had (Radius * 2), so every gap the player
// used to fit through still fits: only where it sits changed. Its bottom edge
// is the sole line, so the box covers the character from the ankles down.
func PlayerGroundBox(p *Player) (center rl.Vector2, width, height float32) {
	return GroundBoxAt(p.Position, p.CharType)
}

// GroundBoxAt is the same box for a player this machine does not own: a remote
// player is a position and a character name in a snapshot, never a *Player.
//
// It exists so the box can only be defined once. The portal has to test the
// whole party, and testing the local player by their feet while testing
// everyone else by their chest would mean the counter on the portal disagrees
// with itself depending on who is looking at it.
//
// The side is PlayerSize because every character shares it (see NewPlayer);
// the character only decides where the box sits, through its foot line.
func GroundBoxAt(pos rl.Vector2, charType CharacterType) (center rl.Vector2, width, height float32) {
	def := GetCharacter(charType)
	width, height = PlayerSize*2, PlayerSize*2
	center = GroundPoint(pos, def)
	center.Y -= height / 2
	return center, width, height
}

// MoveByGroundCorrection applies to Position whatever the foot box was
// actually allowed to do.
//
// The player is moved by the DIFFERENCE between where the box wanted to go and
// where it ended up, not by rebuilding Position out of the resolved centre.
// Rebuilding would mean adding and then subtracting a ~105 px offset in
// float32 every frame, and that round trip is not exact: an unobstructed step
// would nudge the player by a fraction of a pixel forever. Adding a correction
// that is exactly zero when nothing blocked is.
func MoveByGroundCorrection(p *Player, wanted, resolved rl.Vector2) {
	p.Position = rl.Vector2Add(p.Position, rl.Vector2Subtract(resolved, wanted))
}

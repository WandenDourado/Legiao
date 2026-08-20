package entity

// StepWalkAnimation advances the frame/timer pair the same way
// Player.updateAnimation does, for a bot that has no *Player and no
// texture. The caller sets CurrentRow separately from WalkRowFor — this
// only owns frame advancement, since a bot standing still still needs its
// frame reset to 0 exactly like a real player does.
func StepWalkAnimation(def CharacterDef, frame int, timer float32, moving, sprinting bool, dt float32) (nextFrame int, nextTimer float32) {
	if !moving {
		return 0, 0
	}
	frameTime := def.FrameTime
	if sprinting {
		frameTime = def.SprintTime
	}
	timer += dt
	if timer >= frameTime {
		timer -= frameTime
		frame++
		if frame >= def.Columns {
			frame = 0
		}
	}
	return frame, timer
}

package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiSkillButton is the on-screen fireball button used on Android.
var uiSkillButton = ui.NewSkillButton()

// ProcessInput handles all input for one frame and returns the movement direction vector.
// On Android this reads the touch joystick, aim joystick, and handles sprint detection.
// On desktop this reads WASD/arrow keys and mouse input.
func ProcessInput(cfg Config, p *entity.Player, ts *input.TouchState, aj *input.AimJoystick, sw, sh float32, cam Camera2DState) rl.Vector2 {
	var dir rl.Vector2

	if cfg.FullScreen {
		// Android: compute dynamic control positions proportional to screen height
		joystickCenterX := sw * 0.15
		joystickCenterY := sh * 0.80
		attackCenterX := sw * 0.85
		attackCenterY := sh * 0.80
		baseRadius := sh * 0.08

		joystickRect := rl.NewRectangle(
			joystickCenterX-baseRadius,
			joystickCenterY-baseRadius,
			baseRadius*2,
			baseRadius*2,
		)
		attackRect := rl.NewRectangle(
			attackCenterX-baseRadius,
			attackCenterY-baseRadius,
			baseRadius*2,
			baseRadius*2,
		)

		ts.SetMaxRadius(baseRadius * 0.75)
		ts.Update(joystickRect, attackRect)

		dir = ts.GetJoystickDelta()

		// Sprint detection: joystick displacement > 70% of max radius
		joyMag := rl.Vector2Length(dir)
		p.IsSprinting = joyMag > entity.SprintThreshold

		// Aim joystick update — returns true on release (fire)
		if aj.Update(attackRect, *ts) {
			aimDir := aj.AimDir
			if network.Role == "host" && network.CurrentHost != nil {
				targetX := p.Position.X + aimDir.X*100
				targetY := p.Position.Y + aimDir.Y*100
				network.CurrentHost.HandleAttack(network.LocalPlayerID, targetX, targetY)
			} else if network.Role == "client" {
				targetX := p.Position.X + aimDir.X*100
				targetY := p.Position.Y + aimDir.Y*100
				attackMsg := network.Message{
					Type: network.MsgAttack,
					Payload: network.MustMarshal(network.AttackPayload{
						PlayerID: network.LocalPlayerID,
						TargetX:  int(targetX),
						TargetY:  int(targetY),
					}),
				}
				network.SendMessage(attackMsg)
			}
		}

		// Skill button (fireball) — Android on-screen action above attack button.
		if uiSkillButton.Update() {
			aimDir := aj.AimDir
			if aimDir.X == 0 && aimDir.Y == 0 {
				aimDir = rl.NewVector2(0, -1)
			}
			targetX := p.Position.X + aimDir.X*100
			targetY := p.Position.Y + aimDir.Y*100
			activateFireball(p, targetX, targetY)
		}
	} else {
		// Desktop sprint detection: Left Shift key
		p.IsSprinting = rl.IsKeyDown(rl.KeyLeftShift)
	}

	// Keyboard input (WASD + arrow keys) — desktop primary input
	if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
		dir.Y -= 1
	}
	if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
		dir.Y += 1
	}
	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		dir.X -= 1
	}
	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		dir.X += 1
	}

	// Normalize keyboard input if diagonal
	if dir.X != 0 || dir.Y != 0 {
		length := float32(1.0)
		if dir.X != 0 && dir.Y != 0 {
			length = 0.7071 // 1/sqrt(2) for diagonal
		}
		dir.X *= length
		dir.Y *= length
	}

	// Handle mouse click attack on desktop — convert screen coords to world coords
	if !cfg.FullScreen && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		screenPos := rl.GetMousePosition()
		mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.HandleAttack(network.LocalPlayerID, mousePos.X, mousePos.Y)
		} else if network.Role == "client" {
			attackMsg := network.Message{
				Type: network.MsgAttack,
				Payload: network.MustMarshal(network.AttackPayload{
					PlayerID: network.LocalPlayerID,
					TargetX:  int(mousePos.X),
					TargetY:  int(mousePos.Y),
				}),
			}
			network.SendMessage(attackMsg)
		}
	}

	// Desktop fireball skill (Q key) — aim at the mouse world position.
	if rl.IsKeyPressed(rl.KeyQ) {
		screenPos := rl.GetMousePosition()
		mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
		activateFireball(p, mousePos.X, mousePos.Y)
	}

	return dir
}

// activateFireball sends a fireball skill aimed at world target (tx,ty).
func activateFireball(p *entity.Player, tx, ty float32) {
	if network.Role == "host" && network.CurrentHost != nil {
		network.CurrentHost.HandleSkill(network.LocalPlayerID, int(tx), int(ty))
		return
	}
	if network.Role == "client" {
		msg := network.Message{
			Type: network.MsgSkill,
			Payload: network.MustMarshal(network.SkillPayload{
				PlayerID: network.LocalPlayerID,
				Skill:    "fireball",
				TargetX:  int(tx),
				TargetY:  int(ty),
			}),
		}
		network.SendMessage(msg)
	}
}

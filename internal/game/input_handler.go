package game

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiSkillButton is the on-screen fireball button used on Android.
var uiSkillButton = ui.NewSkillButton()

// uiUltimateButton is the on-screen ultimate (supreme skill) button (Android).
var uiUltimateButton = ui.NewUltimateButton()

// ProcessInput handles all input for one frame and returns the movement direction vector.
// On Android this reads the touch joystick, aim joystick, and handles sprint detection.
// On desktop this reads WASD/arrow keys and mouse input.
func ProcessInput(cfg Config, p *entity.Player, w *World, ts *input.TouchState, aj *input.AimJoystick, sw, sh float32, cam Camera2DState) rl.Vector2 {
	var dir rl.Vector2

	// Meta controls first: the test-mode switch and the host's stage restart
	// answer even when the run is over and nothing else does.
	UpdateTestMode(cfg)
	UpdateStageReset(p, sw, sh)
	UpdateNavDebugToggle()
	if network.GameOver {
		// A finished run takes no gameplay input from anyone. Without this a
		// dead player could still fire into the Game Over screen.
		return dir
	}
	if network.LocalPlayerInPortal {
		// Vanished into the party's portal and frozen (host_portal_presence.go):
		// no movement, no attack, no skill — only the way out. ESC/SAIR
		// steps the player clear of the rectangle and clears the flag
		// locally; the host confirms it on its own the next tick it
		// recomputes presence.
		UpdatePortalCancel(p, w, cfg, sw, sh)
		return dir
	}

	if cfg.FullScreen {
		// Android: compute dynamic control positions proportional to screen height.
		// Use one shared geometry source so hit-areas match the visuals and the
		// fire and skill buttons never overlap.
		geom := input.ComputeAndroidControlGeom(sw, sh)
		joystickCenterX := sw * 0.15
		joystickCenterY := sh * 0.80
		baseRadius := sh * 0.08

		joystickRect := rl.NewRectangle(
			joystickCenterX-baseRadius,
			joystickCenterY-baseRadius,
			baseRadius*2,
			baseRadius*2,
		)
		attackRect := geom.AttackRect()

		ts.SetMaxRadius(baseRadius * 0.75)
		ts.Update(joystickRect, attackRect)

		dir = ts.GetJoystickDelta()

		// Sprint detection: joystick displacement > 70% of max radius
		joyMag := rl.Vector2Length(dir)
		p.IsSprinting = joyMag > entity.SprintThreshold

		// Aim joystick update — returns true on release (fire).
		// Pass the skill button geometry so the fire button does not claim a
		// touch that belongs to the skill button.
		if aj.Update(attackRect, *ts, rl.NewVector2(geom.SkillCenterX, geom.SkillCenterY), geom.SkillRadius, sw, sh) {
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

		// On-screen skill button — casts the player's primary ability.
		// Isolated to the Android path so desktop input is untouched.
		updateAndroidSkillButton(uiSkillButton, p, 0, sw, sh)

		// On-screen ULTIMATE button — casts the player's supreme ability.
		if abilityUsable(p.CharType, 1) {
			updateAndroidSkillButton(uiUltimateButton, p, 1, sw, sh)
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

	// Desktop Q key — cast the player's primary ability at the mouse aim.
	if rl.IsKeyPressed(rl.KeyQ) {
		screenPos := rl.GetMousePosition()
		mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
		castAbilityAt(p, 0, mousePos.X, mousePos.Y)
	}

	// Desktop R key — cast the player's ULTIMATE ability at the mouse aim.
	if rl.IsKeyPressed(rl.KeyR) {
		screenPos := rl.GetMousePosition()
		mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
		castAbilityAt(p, 1, mousePos.X, mousePos.Y)
	}

	return dir
}

// castAbilityAt casts the idx-th ability bound to the player's character
// (0 = primary, 1 = ultimate) at world target (tx,ty). Untargeted abilities
// simply ignore the aim point.
func castAbilityAt(p *entity.Player, idx int, tx, ty float32) {
	skillID := ability.AbilityAt(p.CharType, idx)
	if skillID == "" {
		return
	}
	// Suprema travada nao chega a virar pedido. O host recusaria de qualquer
	// forma; parar aqui evita a mensagem de rede e o log de tentativa a cada
	// toque de R de quem ainda nao ganhou a habilidade.
	if !abilityUsable(p.CharType, idx) {
		return
	}
	if network.Role == "host" && network.CurrentHost != nil {
		network.CurrentHost.HandleSkillMessage(network.LocalPlayerID, skillID, tx, ty)
		return
	}
	if network.Role == "client" {
		msg := network.Message{
			Type: network.MsgSkill,
			Payload: network.MustMarshal(network.SkillPayload{
				PlayerID: network.LocalPlayerID,
				Skill:    skillID,
				TargetX:  int(tx),
				TargetY:  int(ty),
			}),
		}
		network.SendMessage(msg)
	}
}

// updateAndroidSkillButton handles an on-screen ability button for Android
// (touch): abilityIdx 0 = primary skill, 1 = ultimate. Geometry is recomputed
// every frame via Layout() so the hit area always matches the drawn circle.
// Kept separate from desktop input so a change here can never affect the
// keyboard/mouse path.
func updateAndroidSkillButton(sb *ui.SkillButton, p *entity.Player, abilityIdx int, sw, sh float32) {
	sb.Layout(sw, sh)
	// Feed the button its recharge state so Draw can put the counter on top
	// of it — on a phone that is the only place the player is looking.
	sb.SetCooldown(SkillSlotCooldown(p.CharType, abilityIdx))
	if sb.Update() {
		// Each button uses its OWN aim direction (sb.SkillDir) only.
		// It must never fall back to the attack button's AimDir, otherwise
		// the cast would inherit the fire-button's direction. If the
		// button has never been aimed, sb.SkillDir defaults to (0,1).
		aimDir := sb.SkillDir
		targetX := p.Position.X + aimDir.X*100
		targetY := p.Position.Y + aimDir.Y*100
		castAbilityAt(p, abilityIdx, targetX, targetY)
	}
}

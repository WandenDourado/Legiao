package game

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/assets"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var remoteTextures = make(map[entity.CharacterType]rl.Texture2D)

// DrawFrame renders the entire frame in three depth passes:
// bottom layers → entities (player, enemies) → top layers.
func DrawFrame(
	cfg Config,
	p *entity.Player,
	mapRenderer *tilemap.MapRenderer,
	ts *input.TouchState,
	aj *input.AimJoystick,
	frameCount int,
	cam Camera2DState,
	bounds world.Bounds,
) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())

	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	// ========== World space (single BeginMode2D block) ==========

	mapRenderer.DrawWithCamera(cam.Camera, "objetcs", "objetcs", func() {
		// Map boundary
		rl.DrawRectangleLines(0, 0, int32(bounds.Width), int32(bounds.Height), rl.DarkGray)

		// Draw remote players
		allPlayers := network.GetAllPlayers()
		for id, state := range allPlayers {
			if id == network.LocalPlayerID {
				continue
			}

			charType := entity.CharacterType(state.Character)
			if charType == "" {
				charType = entity.CharWizard
			}
			
			tex, ok := remoteTextures[charType]
			if !ok {
				def := entity.GetCharacter(charType)
				tex = rl.LoadTexture(assets.Path(def.SpritePath))
				remoteTextures[charType] = tex
			}

			entity.DrawRemotePlayer(
				tex,
				true,
				entity.GetCharacter(charType),
				float32(state.X),
				float32(state.Y),
				state.CurrentFrame,
				state.CurrentRow,
				state.VelX,
				state.Color,
				20,
			)
		}

		// Draw local player (if alive)
		if !network.LocalPlayerDead {
			p.Draw()
		}

		// Draw enemies
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.EntityManager.DrawAll()
			network.CurrentHost.EntityManager.DrawFire()
		} else {
			network.RemoteEnemiesMutex.Lock()
			for _, e := range network.RemoteEnemies {
				entity.DrawEnemyAt(float32(e.X), float32(e.Y), e.Color, 15)
				entity.DrawEnemyHealthBarAt(float32(e.X), float32(e.Y), e.Health, e.MaxHealth, 15)
			}
			network.RemoteEnemiesMutex.Unlock()

			network.RemoteProjectilesMutex.Lock()
			for _, proj := range network.RemoteProjectiles {
				if proj.Active {
					entity.DrawProjectileAt(float32(proj.X), float32(proj.Y))
				}
			}
			network.RemoteProjectilesMutex.Unlock()

			if network.ClientFireEM != nil {
				network.ClientFireEM.DrawFire()
			}
		}
	})

	// ========== Screen space (unaffected by camera) ==========

	if cfg.FullScreen {
		joystickCenterX := sw * 0.15
		joystickCenterY := sh * 0.80
		attackCenterX := sw * 0.85
		attackCenterY := sh * 0.80
		baseRadius := sh * 0.08
		knobRadius := sh * 0.04
		attackRadius := sh * 0.06

		input.DrawJoystickVisual(*ts, joystickCenterX, joystickCenterY, baseRadius, knobRadius)
		input.DrawAimJoystick(*aj, attackCenterX, attackCenterY, attackRadius)
		uiSkillButton.Draw()
	}

	if !network.LocalPlayerDead {
		network.RemotePlayersMutex.Lock()
		if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
			p.Health = state.Health
			p.MaxHealth = state.MaxHealth
			ui.DrawHealthBar(p.Health, p.MaxHealth)
		} else {
			ui.DrawHealthBar(p.Health, p.MaxHealth)
		}
		network.RemotePlayersMutex.Unlock()
	}

	rl.DrawText(fmt.Sprintf("Players: %d", len(network.GetAllPlayers())), 10, 35, 20, rl.White)

	if network.ServerAddress != "" {
		addrText := fmt.Sprintf("Server: %s", network.ServerAddress)
		textWidth := rl.MeasureText(addrText, 20)
		xPos := int32((int(sw) - int(textWidth)) / 2)
		rl.DrawText(addrText, xPos, 10, 20, rl.White)
	}

	if network.LocalPlayerDead {
		timeLeft := entity.RespawnDelay - network.RespawnTimer
		if timeLeft < 0 {
			timeLeft = 0
		}
		respawnText := fmt.Sprintf("Respawning in: %.1fs", timeLeft)
		textWidth := rl.MeasureText(respawnText, 30)
		xPos := int32((int(sw) - int(textWidth)) / 2)
		yPos := int32(int(sh)/2 - 30)
		rl.DrawText(respawnText, xPos, yPos, 30, rl.Red)
	}

	if network.GameOver {
		goText := "GAME OVER"
		textWidth := rl.MeasureText(goText, 60)
		xPos := int32((int(sw) - int(textWidth)) / 2)
		yPos := int32(int(sh)/2 - 60)
		rl.DrawText(goText, xPos, yPos, 60, rl.Red)
	}
	rl.EndDrawing()
}

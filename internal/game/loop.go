package game

import (
	"encoding/json"
	"fmt"
	log "log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Config holds platform-specific configuration for the game.
type Config struct {
	Width      int
	Height     int
	Title      string
	FullScreen bool
}

// DefaultConfig returns a default configuration for desktop.
func DefaultConfig() Config {
	return Config{
		Width:  1280,
		Height: 720,
		Title:  "Legião - Survival Shooter",
	}
}

// AndroidConfig returns a configuration for Android.
func AndroidConfig() Config {
	return Config{
		Width:      0,
		Height:     0,
		Title:      "Legião",
		FullScreen: true,
	}
}

// Run starts the game with the given configuration.
// It handles the menu, networking, and main game loop.
func Run(cfg Config) {
	// Initialize window
	if cfg.FullScreen {
		rl.InitWindow(0, 0, cfg.Title)
	} else {
		rl.InitWindow(int32(cfg.Width), int32(cfg.Height), cfg.Title)
	}
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Show the start menu (host/join)
	log.Println("Exibir menu")
	ui.ShowMenu()

	// Create local player
	p := entity.NewPlayer()
	p.InitSprite()
	defer p.UnloadSprite()

	// Set initial position and color from network state if available
	network.RemotePlayersMutex.Lock()
	if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
		p.Position.X = float32(state.X)
		p.Position.Y = float32(state.Y)
		p.Color = state.Color
	}
	network.RemotePlayersMutex.Unlock()

	// Setup touch input state
	ts := input.NewTouchState()
	joystickRect := rl.NewRectangle(90, float32(entity.ScreenHeight)-230, 120, 120)
	attackRect := rl.NewRectangle(float32(entity.ScreenWidth)-140, float32(entity.ScreenHeight)-140, 70, 70)
	frameCount := 0

	// Main game loop
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// ========== PHASE 1: INPUT ==========
		// Update touch state
		ts.Update(joystickRect, attackRect)

		// Get joystick delta (normalized [-1, 1])
		dir := ts.GetJoystickDelta()

		// Add keyboard input (WASD) - for desktop
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

		// Handle attack
		if ts.IsAttacking {
			// Send attack to network
			if network.Role == "host" && network.CurrentHost != nil {
				// Host: handle attack directly
				mousePos := rl.GetMousePosition()
				network.CurrentHost.HandleAttack(network.LocalPlayerID, mousePos.X, mousePos.Y)
			} else if network.Role == "client" {
				// Client: send attack to host
				mousePos := rl.GetMousePosition()
				attackMsg := network.Message{
					Type: network.MsgAttack,
					Payload: mustMarshal(network.AttackPayload{
						PlayerID: network.LocalPlayerID,
						TargetX:  int(mousePos.X),
						TargetY:  int(mousePos.Y),
					}),
				}
				network.SendMessage(attackMsg)
			}
		}

		// ========== PHASE 2: LOCAL UPDATE ==========
		if !network.LocalPlayerDead {
			p.Update(dir, dt)

			// Send movement to network
			if network.Role != "" {
				network.UpdatePlayerState(network.PlayerState{
					PlayerID: network.LocalPlayerID,
					X:        int(p.Position.X),
					Y:        int(p.Position.Y),
					Color:    p.Color,
					Health:   p.Health,
					MaxHealth: p.MaxHealth,
					IsDead:   p.IsDead,
				})

				if network.Role == "host" && network.CurrentHost != nil {
					// Host: update authoritative state
					network.CurrentHost.UpdatePlayerPosition(network.LocalPlayerID, int(p.Position.X), int(p.Position.Y))
				} else if network.Role == "client" {
					// Client: send input to host
					inputMsg := network.Message{
						Type: network.MsgInput,
						Payload: mustMarshal(network.InputPayload{
							PlayerID: network.LocalPlayerID,
							X:        int(p.Position.X),
							Y:        int(p.Position.Y),
						}),
					}
					network.SendMessage(inputMsg)
				}
			}
		} else {
			// Player is dead - update respawn timer
			network.RespawnTimer += dt
			if network.RespawnTimer >= RespawnDelay {
				// Request respawn
				if network.Role == "host" && network.CurrentHost != nil {
					network.CurrentHost.RespawnPlayer(network.LocalPlayerID)
				}
				network.LocalPlayerDead = false
				network.RespawnTimer = 0
				p.Respawn(RespawnHealthPercent, p.Position.X, p.Position.Y)
			}
		}

		// ========== PHASE 3: HOST SIMULATION ==========
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.UpdateSimulation(dt)
		}

		// ========== PHASE 4: RENDER ==========
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		// Draw ALL players (local + remote)
		allPlayers := network.GetAllPlayers()

		// Debug: log player count every 300 frames
		if frameCount%300 == 0 {
			log.Printf("[Main] Frame %d: RemotePlayers has %d players", frameCount, len(allPlayers))
		}
		frameCount++

		// Draw remote players
		for id, state := range allPlayers {
			if id == network.LocalPlayerID {
				continue // Skip local player, drawn separately
			}
			entity.DrawPlayerAt(float32(state.X), float32(state.Y), state.Color, 20)
		}

		// Draw local player (if alive)
		if !network.LocalPlayerDead {
			p.Draw()
		}

		// Draw enemies (from RemoteEnemies or host's entity manager)
		if network.Role == "host" && network.CurrentHost != nil {
			// Host: draw from entity manager
			network.CurrentHost.EntityManager.DrawAll()
		} else {
			// Client: draw from RemoteEnemies
			network.RemoteEnemiesMutex.Lock()
			for _, e := range network.RemoteEnemies {
				entity.DrawEnemyAt(float32(e.X), float32(e.Y), e.Color, 15)
				// Draw enemy health bar
				entity.DrawEnemyHealthBarAt(float32(e.X), float32(e.Y), e.Health, e.MaxHealth, 15)
			}
			network.RemoteEnemiesMutex.Unlock()

			// Draw projectiles
			network.RemoteProjectilesMutex.Lock()
			for _, proj := range network.RemoteProjectiles {
				if proj.Active {
					entity.DrawProjectileAt(float32(proj.X), float32(proj.Y))
				}
			}
			network.RemoteProjectilesMutex.Unlock()
		}

		// Draw virtual joystick (visual only)
		drawJoystickVisual(ts)

		// Draw attack button
		if !network.LocalPlayerDead {
			drawAttackButton(attackRect)
		}

		// Draw HUD - sync health from RemotePlayers and draw
		if !network.LocalPlayerDead {
			// Sync local player health with network state
			network.RemotePlayersMutex.Lock()
			if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
				p.Health = state.Health // Update local health for drawing
				p.MaxHealth = state.MaxHealth
				ui.DrawHealthBar(p.Health, p.MaxHealth)
			} else {
				ui.DrawHealthBar(p.Health, p.MaxHealth)
			}
			network.RemotePlayersMutex.Unlock()
		}

		// Draw player count (top-left, below health bar)
		rl.DrawText(fmt.Sprintf("Players: %d", len(allPlayers)), 10, 35, 20, rl.White)

		// Draw server address (top-center)
		if network.ServerAddress != "" {
			addrText := fmt.Sprintf("Server: %s", network.ServerAddress)
			textWidth := rl.MeasureText(addrText, 20)
			screenW := rl.GetScreenWidth()
			xPos := int32((int(screenW) - int(textWidth)) / 2)
			rl.DrawText(addrText, xPos, 10, 20, rl.White)
		}

		// Draw respawn timer if dead
		if network.LocalPlayerDead {
			timeLeft := RespawnDelay - network.RespawnTimer
			if timeLeft < 0 {
				timeLeft = 0
			}
			respawnText := fmt.Sprintf("Respawning in: %.1fs", timeLeft)
			textWidth := rl.MeasureText(respawnText, 30)
			screenW := rl.GetScreenWidth()
			screenH := rl.GetScreenHeight()
			xPos := int32((int(screenW) - int(textWidth)) / 2)
			yPos := int32(int(screenH)/2 - 30)
			rl.DrawText(respawnText, xPos, yPos, 30, rl.Red)
		}

		// Draw game over screen
		if network.GameOver {
			goText := "GAME OVER"
			textWidth := rl.MeasureText(goText, 60)
			screenW := rl.GetScreenWidth()
			screenH := rl.GetScreenHeight()
			xPos := int32((int(screenW) - int(textWidth)) / 2)
			yPos := int32(int(screenH)/2 - 60)
			rl.DrawText(goText, xPos, yPos, 60, rl.Red)
		}

		rl.EndDrawing()
	}
}

// drawJoystickVisual draws the joystick base and knob based on touch state.
func drawJoystickVisual(ts input.TouchState) {
	// Draw base circle (left side of screen)
	baseCenter := rl.NewVector2(150, float32(entity.ScreenHeight)-150)
	rl.DrawCircleV(baseCenter, 80, rl.Fade(rl.Gray, 0.5))

	// Draw knob at current position if active
	if ts.JoystickTouchID != input.NoTouch {
		rl.DrawCircleV(ts.JoystickCurrent, 40, rl.Fade(rl.LightGray, 0.8))
	} else {
		rl.DrawCircleV(baseCenter, 40, rl.Fade(rl.LightGray, 0.8))
	}
}

// drawAttackButton draws the attack button (right side of screen).
func drawAttackButton(rect rl.Rectangle) {
	center := rl.NewVector2(rect.X+rect.Width/2, rect.Y+rect.Height/2)
	radius := rect.Width / 2
	color := rl.Fade(rl.Red, 0.7)
	rl.DrawCircleV(center, radius, color)
	rl.DrawCircleLinesV(center, radius, rl.White)

	// Draw "FIRE" text
	text := "FIRE"
	textWidth := rl.MeasureText(text, 14)
	rl.DrawText(text,
		int32(center.X)-textWidth/2,
		int32(center.Y)-7,
		14, rl.White)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Failed to marshal: %v", err)
		return nil
	}
	return data
}

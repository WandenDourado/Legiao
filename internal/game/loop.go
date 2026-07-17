package game

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Run starts the game with the given configuration.
// It handles the menu, networking, and main game loop.
func Run(cfg Config) {
	rl.InitWindow(0, 0, cfg.Title)
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	bounds := world.NewBounds(sw, sh, 5.0)
	cam := NewCamera(sw, sh)

	renderer, collisionRects, mapBounds := loadMap("assets/maps/world_01.json")
	defer renderer.Unload()

	bounds = mapBounds

	playerSpawn := entity.InitialPlayerSpawn(bounds)

	charType := ui.ShowMenu(playerSpawn)
	if network.CurrentHost != nil {
		network.CurrentHost.EntityManager.WorldBounds = bounds
		network.CurrentHost.SetCollisionRects(collisionRects)
	}

	p := entity.NewPlayer(playerSpawn, charType)
	p.InitSprite()
	defer p.UnloadSprite()

	network.RemotePlayersMutex.Lock()
	if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
		p.Position.X = float32(state.X)
		p.Position.Y = float32(state.Y)
		p.Color = state.Color
	}
	network.RemotePlayersMutex.Unlock()

	ts := input.NewTouchState()
	aj := input.NewAimJoystick()
	frameCount := 0

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()
		sw := float32(rl.GetScreenWidth())
		sh := float32(rl.GetScreenHeight())

		dir := ProcessInput(cfg, p, &ts, &aj, sw, sh, cam)

		// Local update
		if !network.LocalPlayerDead {
			p.Update(dir, dt, bounds)
			ResolveCollision(p, collisionRects)
			if network.Role != "" {
				network.UpdatePlayerState(network.PlayerState{
					PlayerID:     network.LocalPlayerID,
					X:            int(p.Position.X),
					Y:            int(p.Position.Y),
					Color:        p.Color,
					Character:    string(p.CharType),
					Health:       p.Health,
					MaxHealth:    p.MaxHealth,
					IsDead:       p.IsDead,
					CurrentFrame: p.CurrentFrame,
					CurrentRow:   p.CurrentRow,
					IsSprinting:  p.IsSprinting,
					VelX:         p.Velocity.X,
					VelY:         p.Velocity.Y,
				})
				if network.Role == "host" && network.CurrentHost != nil {
					network.CurrentHost.UpdatePlayerState(network.PlayerState{
						PlayerID:     network.LocalPlayerID,
						X:            int(p.Position.X),
						Y:            int(p.Position.Y),
						Color:        p.Color,
						Character:    string(p.CharType),
						Health:       p.Health,
						MaxHealth:    p.MaxHealth,
						IsDead:       p.IsDead,
						CurrentFrame: p.CurrentFrame,
						CurrentRow:   p.CurrentRow,
						IsSprinting:  p.IsSprinting,
						VelX:         p.Velocity.X,
						VelY:         p.Velocity.Y,
					})
				} else if network.Role == "client" {
					inputMsg := network.Message{
						Type: network.MsgInput,
						Payload: network.MustMarshal(network.InputPayload{
							PlayerID:     network.LocalPlayerID,
							X:            int(p.Position.X),
							Y:            int(p.Position.Y),
							CurrentFrame: p.CurrentFrame,
							CurrentRow:   p.CurrentRow,
							IsSprinting:  p.IsSprinting,
							VelX:         p.Velocity.X,
							VelY:         p.Velocity.Y,
						}),
					}
					network.SendMessage(inputMsg)
				}
			}
		} else {
			network.RespawnTimer += dt
			if network.RespawnTimer >= entity.RespawnDelay {
				if network.Role == "host" && network.CurrentHost != nil {
					network.CurrentHost.RespawnPlayer(network.LocalPlayerID)
				}
				network.LocalPlayerDead = false
				network.RespawnTimer = 0
				p.Respawn(entity.RespawnHealthPercent, playerSpawn.X, playerSpawn.Y)
			}
		}

		// Host simulation
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.UpdateSimulation(dt)
		}

		// Animate client-side fire effects (explosions, ground fire).
		if network.Role == "client" && network.ClientFireEM != nil {
			network.ClientFireEM.UpdateFire(dt)
		}

		// Update camera
		cam.Update(p.Position, sw, sh, bounds)

		// Render
		DrawFrame(cfg, p, renderer, &ts, &aj, frameCount, cam, bounds)
		frameCount++

		if frameCount%300 == 0 {
			log.Printf("[Main] Frame %d: RemotePlayers has %d players", frameCount, len(network.GetAllPlayers()))
		}
	}
}

// loadMap loads a Tiled JSON map and returns a ready-to-use renderer,
// collision rectangles, and world bounds derived from the map's pixel dimensions.
func loadMap(path string) (*tilemap.MapRenderer, []rl.Rectangle, world.Bounds) {
	tm, err := tilemap.LoadTiledMap(path)
	if err != nil {
		log.Fatalf("[Tilemap] Failed to load map: %v", err)
	}
	renderer := tilemap.NewMapRenderer(tm)
	renderer.Load()
	collisionRects := tilemap.GetCollisionRects(tm)
	bounds := world.NewBoundsFromMap(tm.Width, tm.Height, tm.TileWidth, tm.TileHeight)
	return renderer, collisionRects, bounds
}

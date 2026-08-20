package game

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Run starts the game with the given configuration. It owns the window for
// the whole process; playSession owns one menu-to-menu session inside it.
//
// A session ends normally when the window closes. It can also end early when
// a client exhausts the reconnect window (network.SessionEnded,
// network/reconnect.go): rather than leave the app stuck mid-match with no
// host in reach, the loop returns here and opens the menu again in a fresh
// session, exactly like a clean process start.
func Run(cfg Config) {
	rl.InitWindow(0, 0, cfg.Title)
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		playSession(cfg)
	}
}

// playSession runs the menu and one full match: world load through however
// it ends. It handles the menu, networking, and main game loop.
func playSession(cfg Config) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	cam := NewCamera(sw, sh)

	w := LoadWorld("assets/maps/world_01.json")
	defer func() { w.Unload() }()

	// bounds follows the loaded world and changes with it when a portal swaps
	// the map, so nothing is ever clamped against the previous map's size.
	bounds := w.Bounds

	charType := ui.ShowMenu(w.PlayerSpawn)
	// The host has to be pointed at the loaded world; WorldBounds was once
	// never assigned and stayed {0,0}, which silently broke every spawn
	// position clamped or generated against it.
	w.ApplyToHost()

	p := entity.NewPlayer(w.PlayerSpawn, charType)
	p.InitSprite()
	// Antes do laco, e nao dentro dele: carregada sob demanda, a folha do orc
	// subia para a GPU no quadro em que ele ficava visivel — com a camera
	// andando e o jogador olhando. Era o pico isolado de 43 ms.
	entity.PreloadEnemyTextures()
	defer p.UnloadSprite()
	defer entity.UnloadEnemyTextures()
	defer entity.UnloadSharedTextures()
	defer entity.UnloadDeathTint()

	network.RemotePlayersMutex.Lock()
	if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
		p.Position.X = float32(state.X)
		p.Position.Y = float32(state.Y)
		p.Color = state.Color
	}
	network.RemotePlayersMutex.Unlock()

	ts := input.NewTouchState()
	aj := input.NewAimJoystick()
	dlg := NewDialogueDirector()
	frameCount := 0

	for !rl.WindowShouldClose() && !network.SessionEnded() {
		dt := rl.GetFrameTime()
		sw := float32(rl.GetScreenWidth())
		sh := float32(rl.GetScreenHeight())

		// Drives the client-side reconnect state machine (network/reconnect.go)
		// every frame, win or lose the connection this frame or not — the same
		// single delegated call UpdateSimulation is for the host below.
		if network.Role == "client" {
			network.TickReconnect()
		}

		// A lost connection owns the frame like a dialogue does: local input
		// freezes and nothing simulates, but the mirrored world keeps being
		// drawn as last known so the screen is not simply blank while the
		// overlay counts down.
		if network.Role == "client" && network.Reconnecting() {
			cam.Update(p.Position, sw, sh, bounds)
			DrawFrame(cfg, p, w, &ts, &aj, frameCount, cam)
			frameCount++
			continue
		}

		// A running dialogue owns the frame: the world is frozen for everyone
		// while the narrative plays, so nothing below this runs.
		if dlg.Update(w) {
			cam.Update(p.Position, sw, sh, bounds)
			DrawFrame(cfg, p, w, &ts, &aj, frameCount, cam)
			frameCount++
			continue
		}

		dir := ProcessInput(cfg, p, w, &ts, &aj, sw, sh, cam)

		// Death, the revive countdown and stage resets are decided by the
		// host; this mirrors that verdict onto the local player before it is
		// moved. A dead player is skipped below: it stays where it fell.
		SyncLocalPlayer(p)

		// A viagem entre mapas fica FORA do bloco de vivo, e essa e a regra
		// inteira do jogador morto: o portal leva o grupo, entao quem caiu no
		// caminho atravessa junto e espera o ressurgimento do outro lado. Se
		// dependesse de estar vivo, a maquina dele ficaria sozinha no mapa que
		// todo mundo acabou de deixar.
		w = ApplyPendingTravel(w, p)
		bounds = w.Bounds

		// Local update
		if !p.IsDead {
			p.Update(dir, dt, bounds)
			ResolveCollision(p, w.Collision)
			w.UpdateArenaGate(p)
			UpdateStageSkip(cfg, w)
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
		}

		// A PORTA DO PORTAL le o grupo inteiro, entao vem DEPOIS de o jogador
		// local publicar a posicao deste quadro: lida antes, a propria posicao
		// dele seria a do quadro anterior e o ultimo passo para dentro do
		// portal so contaria 16 ms mais tarde. Fica fora do bloco de vivo pelo
		// mesmo motivo que a viagem: o contador tem de continuar certo na tela
		// de quem morreu esperando.
		UpdatePortal(w, p)

		// Host simulation
		if network.Role == "host" && network.CurrentHost != nil {
			// A PORTA DO CLIMAX vem antes da simulacao, e a ordem importa: ela
			// instala a corrida da emboscada, e `UpdateSimulation` e quem faz
			// a corrida andar. Depois, a primeira leva so nasceria no quadro
			// seguinte — invisivel, mas e um quadro de atraso entre o grupo
			// chegar e o mapa reagir.
			if stageClimax.update(w) {
				network.CurrentHost.StartClimax(w.Path, w.ClimaxSpawns)
			}
			network.CurrentHost.UpdateSimulation(dt)
		}

		// Animate client-side skill visuals (fireballs, explosions,
		// ground fire, sanctuaries — placement + dissipation only).
		if network.Role == "client" && network.ClientSkills != nil {
			network.ClientSkills.UpdateFire(dt)
			network.ClientSkills.AdvanceSanctuaries(dt)
			network.AdvanceClientSkills(dt, w.Collision.Rects())
		}

		// Update camera
		cam.Update(p.Position, sw, sh, bounds)

		// Render
		DrawFrame(cfg, p, w, &ts, &aj, frameCount, cam)
		frameCount++

		if frameCount%300 == 0 {
			log.Printf("[Main] Frame %d: RemotePlayers has %d players", frameCount, len(network.GetAllPlayers()))
		}
	}

	if network.SessionEnded() {
		// The reconnect window ran out with no host in reach. Clean client
		// state before playSession is called again, or the next menu pass
		// would inherit a dead Client and a stale roster instead of starting
		// clean like a fresh process would.
		log.Printf("[Main] sessao encerrada: janela de reconexao esgotada")
		network.ResetClientSession()
	}
}

package game

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/skill"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// alivePlayerPositions lists everyone a monster could be looking at.
//
// The client mirrors the host's Host.getAlivePlayerPositions from the snapshot
// map it already keeps, so both sides answer "who is the nearest player" the
// same way. Dead players are excluded for the same reason the host excludes
// them: a monster does not stare at a corpse it has stopped chasing.
func alivePlayerPositions() []rl.Vector2 {
	// Interpolado, e nao o snapshot cru: o monstro tem de olhar para onde o
	// jogador APARECE, senao a mira dele adianta o desenho em ate 100 ms e
	// fica visivelmente torta enquanto alguem corre.
	players := network.InterpolatedPlayers()
	out := make([]rl.Vector2, 0, len(players))
	for _, p := range players {
		if p.IsDead {
			continue
		}
		out = append(out, rl.NewVector2(float32(p.X), float32(p.Y)))
	}
	return out
}

// DrawFrame renders the entire frame in three depth passes:
// bottom layers -> entities (player, enemies) -> top layers.
func DrawFrame(
	cfg Config,
	p *entity.Player,
	w *World,
	ts *input.TouchState,
	aj *input.AimJoystick,
	frameCount int,
	cam Camera2DState,
) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	mapRenderer := w.Renderer
	bounds := w.Bounds
	// A janela visivel, em unidades de mundo. O mapa deriva a sua por dentro
	// (tilemap.MapRenderer.viewport); esta e a mesma conta para as ENTIDADES,
	// que nao vivem numa grade e por isso recebem o retangulo pronto.
	view := tilemap.NewViewport(cam.Camera, sw, sh,
		mapRenderer.Map.TileWidth, mapRenderer.Map.TileHeight).World
	// A MESMA janela que o mapa e os inimigos usam vai para as magias. Uma vez
	// por quadro, aqui, porque `Manager.Draw*` tem dezesseis pontos de entrada
	// e todos desenham dentro do bloco de camera abaixo — ver skill/view.go.
	skill.SetDrawView(view)

	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	// ========== World space (single BeginMode2D block) ==========

	mapRenderer.DrawWithCamera(cam.Camera, "foreground", "foreground", func() {
		// Map boundary
		rl.DrawRectangleLines(0, 0, int32(bounds.Width), int32(bounds.Height), rl.DarkGray)

		// The portal stands on the ground but is drawn before the entities, so
		// whoever walks up to it always passes in front of the light.
		w.DrawPortals()

		// Draw remote players. Interpolado no cliente (~100 ms atras do ultimo
		// snapshot, entre os dois ultimos); no host devolve o estado direto,
		// que ele ja tem a cada quadro.
		allPlayers := network.InterpolatedPlayers()
		for id, state := range allPlayers {
			if id == network.LocalPlayerID {
				continue
			}
			if state.InPortal {
				// Vanished waiting inside the party's open portal — the
				// counter drawn over the portal (w.DrawPortals, above)
				// already explains the disappearance; drawing the body
				// too would contradict it.
				continue
			}

			charType := entity.CharacterType(state.Character)
			if charType == "" {
				charType = entity.CharMago
			}

			tex := entity.SharedTexture(charType)

			entity.DrawRemotePlayer(
				tex,
				tex.ID != 0,
				entity.GetCharacter(charType),
				float32(state.X),
				float32(state.Y),
				state.CurrentFrame,
				state.CurrentRow,
				state.VelX,
				state.Color,
				20,
				state.IsDead,
			)
			// The ally bar reuses the character's own frame geometry, not the
			// monster's foot-anchored one: a player sprite is centered on
			// Position, a directional monster is anchored at its feet. Drawn
			// even when Absent (a connection state, not a health one) so a
			// dropped party member's last-known health stays visible; skipped
			// only for the dead, whose grey body already reads as "no HP".
			if !state.IsDead {
				entity.DrawAllyHealthBar(
					entity.GetCharacter(charType),
					float32(state.X),
					float32(state.Y),
					state.Health,
					state.MaxHealth,
				)
			}
			// Absent (mid-reconnect, host_absence.go) is a connection state,
			// not a death state: it is drawn on top of whatever the body
			// already looks like, dead or alive, so the party can tell "the
			// connection dropped" apart from "actually died".
			if state.Absent {
				ui.DrawAbsentMarker(float32(state.X), float32(state.Y))
			}
		}

		// O "heroi invocado" do ultimo suspiro era desenhado aqui. Ele deixou
		// de existir em 23/08/2026: toda classe vaga passou a ser preenchida
		// por um BOT (internal/network/host_bots.go), entao a cena sempre
		// encontra um corpo de verdade para reerguer e quem lanca a suprema e
		// um jogador — que ja e desenhado pelo laco de jogadores remotos
		// acima. Ver doc/combat_rules.md, "Nao ha mais NPC".

		// Draw the local player. A dead one is still drawn: it greys out and
		// stops animating instead of disappearing. Waiting inside a portal
		// is the one state that DOES disappear — see the panel drawn in
		// screen space, below.
		if !network.LocalPlayerInPortal {
			p.Draw()
		}

		// F3 overlay: the exact footprints movement resolution tests (player
		// and enemies), each turning orange whenever it overlaps a solid cell.
		DrawFootprintDebug(p, mapRenderer.Collision)

		// F4 overlay: the navigation mesh (free/blocked cells) and every
		// bot/monster's current mesh-planned route.
		DrawNavDebug(view)

		// Draw enemies
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.EntityManager.DrawAll(view)
			ability.DrawAll(network.CurrentHost.Skills)
			network.CurrentHost.Skills.DrawSwords()
			// A esfera da sentinela nao passa pelo ability.DrawAll: aquele
			// registro mapeia PERSONAGEM -> magia, e esta magia e de um
			// monstro. Desenha aqui, no mesmo espaco de mundo.
			skill.DrawSentryOrbs(true)
			// As bolas de canhao do corredor final, pela mesma razao.
			skill.DrawCannonBalls(true)
			// Espinhoes e nevoa da chefe do mapa 7: mesma razao das duas acima —
			// sao magias de MONSTRO, e o ability.DrawAll mapeia personagem.
			// A nevoa vem por ultimo porque ela cobre a arena inteira: desenhada
			// antes, os espinhoes ficariam por cima dela e o veu perderia o sentido.
			network.CurrentHost.Skills.DrawThorns()
			network.CurrentHost.Skills.DrawFog()
		} else {
			enemyDT := rl.GetFrameTime()
			// Onde os inimigos direcionais devem olhar. Levantado UMA vez por
			// quadro e nao por inimigo: com vinte orcs em campo, refazer a
			// lista dentro do laco seria vinte copias do mesmo mapa.
			facingCandidates := alivePlayerPositions()
			counts := entity.EnemyDrawCounts{}
			// Interpolado pelo mesmo motivo dos jogadores: a 20 Hz de
			// snapshot, desenhar a posicao crua faria cada monstro andar tres
			// quadros parado e um salto.
			for _, e := range network.InterpolatedEnemies() {
				enemyType := entity.EnemyType(e.Type)
				pos := rl.NewVector2(float32(e.X), float32(e.Y))
				counts.Alive++
				// Mesmo culling do host. Cullar aqui tambem congela o tracker
				// de animacao de quem esta fora da tela, e isso e aceitavel:
				// o tracker deduz "andando" da variacao de posicao entre
				// snapshots, entao um monstro que reentra na tela tendo se
				// movido volta ja andando.
				if !entity.EnemyVisible(entity.RemoteEnemyDrawBox(enemyType, pos.X, pos.Y), view) {
					continue
				}
				counts.Drawn++
				entity.DrawRemoteEnemy(e.EnemyID, enemyType,
					pos.X, pos.Y, e.Color, enemyDT,
					entity.NearestFacingTarget(pos, facingCandidates))
				entity.DrawRemoteEnemyHealthBar(enemyType,
					float32(e.X), float32(e.Y), e.Health, e.MaxHealth)
			}
			// Sem Unlock aqui: InterpolatedEnemies devolve uma COPIA e cuida
			// do proprio bloqueio por dentro. O laco antigo segurava
			// RemoteEnemiesMutex durante o desenho inteiro; ao troca-lo, o
			// Unlock ficou orfao e o cliente morria no primeiro quadro com
			// "sync: unlock of unlocked mutex".

			network.RemoteProjectilesMutex.Lock()
			for _, proj := range network.RemoteProjectiles {
				if proj.Active {
					counts.Projectiles++
					entity.DrawProjectileKindAt(proj.Kind,
						float32(proj.X), float32(proj.Y), proj.DirX, proj.DirY)
				}
			}
			network.RemoteProjectilesMutex.Unlock()
			entity.RecordEnemyDraw(counts)

			if network.ClientSkills != nil {
				ability.DrawAll(network.ClientSkills)
				network.ClientSkills.DrawSwords()
			}
			skill.DrawSentryOrbs(false)
			skill.DrawCannonBalls(false)
		}
	})

	// ========== Screen space (unaffected by camera) ==========

	if cfg.FullScreen {
		geom := input.ComputeAndroidControlGeom(sw, sh)
		joystickCenterX := sw * 0.15
		joystickCenterY := sh * 0.80
		baseRadius := sh * 0.08
		knobRadius := sh * 0.04

		input.DrawJoystickVisual(*ts, joystickCenterX, joystickCenterY, baseRadius, knobRadius)
		input.DrawAimJoystick(*aj, geom.AttackCenterX, geom.AttackCenterY, geom.AttackRadius)
		// The fire button dims while the character's cadence recharges, so a
		// tap that lands too soon reads as "too fast" rather than "ignored".
		// No figures here: at one to three hits per second they would be an
		// unreadable blur.
		ui.DrawCooldownRing(
			rl.NewVector2(geom.AttackCenterX, geom.AttackCenterY),
			geom.AttackRadius,
			network.AttackCooldown(network.LocalPlayerID),
			entity.GetCharacter(p.CharType).AttackInterval(),
		)
		uiSkillButton.Layout(sw, sh)
		uiSkillButton.Draw()
		if abilityUsable(p.CharType, 1) {
			uiUltimateButton.Layout(sw, sh)
			uiUltimateButton.Draw()
		}
	} else {
		// The keyboard has no ability buttons to draw the counters on, so the
		// desktop gets its own recharge bar.
		DrawSkillCooldownBar(p, sw, sh)
	}

	// Health is synced onto the player by SyncLocalPlayer; the bar stays up
	// while dead so the player can see they are at zero.
	ui.DrawHealthBar(p.Health, p.MaxHealth)

	rl.DrawText(fmt.Sprintf("Players: %d", len(network.GetAllPlayers())), 10, 35, 20, rl.White)

	ui.DrawWaveHUD(sw, sh)
	ui.DrawBossBar(sw, sh)

	// Revive countdown. Hidden during a Game Over: nobody is coming back
	// until the host restarts the stage, and the overlay says so.
	if p.IsDead && !network.GameOver {
		respawnText := fmt.Sprintf("Ressuscitando em: %.0fs", p.RespawnIn)
		textWidth := rl.MeasureText(respawnText, 30)
		xPos := int32((int(sw) - int(textWidth)) / 2)
		yPos := int32(int(sh)/2 - 30)
		rl.DrawText(respawnText, xPos, yPos, 30, rl.Red)
	}

	if network.TestMode {
		rl.DrawText("MODO TESTE - sem recarga", 10, int32(sh)-30, 20, rl.Lime)
	}

	// The local player's body is not drawn while waiting inside a portal
	// (above): this is the only thing on screen saying why, and the way out.
	if network.LocalPlayerInPortal {
		uiPortalWait.Draw(sw, sh, cfg.FullScreen)
	}

	// Last of the screen-space passes: the dialogue dims the scene and has to
	// sit above the HUD it is dimming.
	ui.DrawDialogueBox(sw, sh)

	if network.GameOver {
		uiGameOver.Draw(sw, sh, CanResetStage())
	}

	// Above the Game Over panel too: a lost connection can happen while a
	// prior run is already showing that overlay, and the countdown is the
	// more urgent thing to read.
	if network.Role == "client" && network.Reconnecting() {
		ui.DrawReconnectOverlay(sw, sh, network.ReconnectRemaining())
	}

	// Por ultimo de tudo: o medidor le os contadores que os passes acima
	// acabaram de encher, e o quadro que ele cronometra e este.
	DrawPerfHUD(sw, sh, w, p)

	rl.EndDrawing()
}

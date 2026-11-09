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

	// Durante o teste automatico de desempenho (Configuracoes > Graficos) o
	// mundo NAO e desenhado: o teste mede a propria cena, e o mapa atras dela
	// somaria um custo que muda com o lugar onde o jogador estava. A partida
	// continua simulando, como sempre que a tela de Configuracoes esta aberta.
	if ui.PerfBenchRunning() {
		UpdateAndDrawSettingsOverlay(sw, sh)
		rl.EndDrawing()
		return
	}

	// ========== World space (single BeginMode2D block) ==========

	mapRenderer.DrawWithCamera(cam.Camera, "foreground", "foreground", func() {
		// Map boundary
		rl.DrawRectangleLines(0, 0, int32(bounds.Width), int32(bounds.Height), rl.DarkGray)

		// The portal stands on the ground but is drawn before the entities, so
		// whoever walks up to it always passes in front of the light.
		w.DrawPortals()

		// O SELO DO MAGO PELA MESMA RAZAO DA LINHA ACIMA, e nao la embaixo com
		// as outras magias: ele e DECALQUE DE CHAO, e o bloco de magias roda
		// depois dos corpos. Desenhado junto com elas, a estrela aditiva
		// passava por cima do rosto e das vestes de quem estivesse dentro dela
		// (relato do Gui, 10/09/2026, com print). Aqui o mago PISA no selo.
		//
		// Os meteoros continuam no bloco de magias: aqueles estao no ar e
		// TEM de passar na frente de quem esta embaixo.
		// O ramo host/cliente aqui e o mesmo do bloco de magias mais abaixo;
		// ele so aparece duas vezes porque este desenho mudou de lugar no
		// quadro, nao de dono.
		//
		// O CHAO DO CEMITERIO DO NECROMANTE ENTRA AQUI PELA MESMA RAZAO, e
		// pelo mesmo relato: em 20/09/2026 os testadores disseram que a magia
		// "poluia o mapa passando por cima dos personagens". Ela e chao — o
		// personagem tem de PISAR nela. So o chao sobe para ca; as maos de
		// esqueleto e as almas continuam no bloco de magias, porque estao
		// acima do solo e e a mao agarrando a perna de quem esta dentro que
		// faz a magia comunicar o que ela faz. Ver skill/graveyard_draw.go.
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.Skills.DrawMeteorSigils()
			network.CurrentHost.Skills.DrawGraveyardGrounds()
		} else if network.ClientSkills != nil {
			network.ClientSkills.DrawMeteorSigils()
			network.ClientSkills.DrawGraveyardGrounds()
		}

		// OS CORPOS, EM ORDEM DE PROFUNDIDADE — e o jogador local no meio da
		// fila, no lugar que o Y dele mandar.
		//
		// Antes este laco percorria o mapa de jogadores direto, e a ordem de
		// um mapa em Go e sorteada a cada passagem: dois personagens em cima
		// um do outro trocavam de camada 60 vezes por segundo e piscavam. Ver
		// game/render_order.go.
		//
		// Interpolado no cliente (~100 ms atras do ultimo snapshot, entre os
		// dois ultimos); no host devolve o estado direto, que ele ja tem a
		// cada quadro.
		bodies := drawablePlayers(network.InterpolatedPlayers())
		// Esperar dentro de um portal e o unico estado que faz o corpo local
		// SUMIR — o painel em espaco de tela, mais abaixo, explica a ausencia.
		// Um morto continua sendo desenhado: ele acinzenta e para de animar.
		localDrawn := network.LocalPlayerInPortal
		for _, b := range bodies {
			// O jogador local entra aqui, e nao depois do laco: desenhado
			// sempre por ultimo, ele aparecia na frente de um aliado que
			// estava mais ao sul que ele.
			if !localDrawn && p.Position.Y < float32(b.State.Y) {
				p.Draw()
				localDrawn = true
			}
			state := b.State
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
				state.Invuln,
			)
		}
		// O "heroi invocado" do ultimo suspiro era desenhado aqui. Ele deixou
		// de existir em 23/08/2026: toda classe vaga passou a ser preenchida
		// por um BOT (internal/network/host_bots.go), entao a cena sempre
		// encontra um corpo de verdade para reerguer e quem lanca a suprema e
		// um jogador — que ja e desenhado pelo laco de corpos acima. Ver
		// doc/combat_rules.md, "Nao ha mais NPC".
		if !localDrawn {
			p.Draw()
		}

		// O CHEVRON DE MIRA, logo depois dos corpos e antes das barras.
		//
		// Depois dos corpos porque a mira aponta para os quatro lados: desenhado
		// antes, o proprio sprite cobriria a marca sempre que o jogador mirasse
		// para o norte — justo a direcao em que ele nao ve nada. Antes das
		// barras porque leitura de vida continua sendo a camada de cima.
		//
		// So no Android: no desktop a mira e o mouse, e o cursor ja diz para
		// onde ela vai.
		if cfg.FullScreen {
			drawAimHint(p, aj)
		}

		// AS BARRAS E OS MARCADORES VEM DEPOIS DE TODOS OS CORPOS, num passe
		// so. Desenhados dentro do laco acima, a barra de um aliado ao norte
		// ficava por baixo do corpo de quem estava ao sul dele — informacao
		// coberta por decoracao. Aqui elas sao sempre a camada de cima, e a
		// ordem entre elas nao importa porque nao se sobrepoem entre si.
		for _, b := range bodies {
			state := b.State
			charType := entity.CharacterType(state.Character)
			if charType == "" {
				charType = entity.CharMago
			}
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

		// F3 overlay: the exact footprints movement resolution tests (player
		// and enemies), each turning orange whenever it overlaps a solid cell.
		DrawFootprintDebug(p, mapRenderer.Collision)

		// F4 overlay: the navigation mesh (free/blocked cells) and every
		// bot/monster's current mesh-planned route.
		DrawNavDebug(view)

		// Draw enemies. Antes deles, a desintegracao de quem morreu desde o
		// quadro passado (game/death_fx.go); depois deles, o desenho dela.
		beginEnemyDeathFX(w)
		if network.Role == "host" && network.CurrentHost != nil {
			network.CurrentHost.EntityManager.DrawAll(view)
			drawEnemyDeathFX(view)
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
			// O anel de contato dela (item 5, doc/plan_world07_arena_tuning.md):
			// mesma familia das duas linhas acima, so que sem relogio — ele existe
			// o tempo inteiro que ela estiver viva, entao so precisa da posicao.
			// BossContactCenter() ja devolve HitCenter(), nao o pe. So com
			// Legiao em campo: desde 27/09/2026 o anel so fere espectro.
			if center, alive := network.CurrentHost.BossContactCenter(); alive && network.CurrentHost.Skills.AnyLegion() {
				skill.DrawBossContactRing(center)
			}
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
			// Ordem estavel (por tipo, depois por ID), a MESMA do host em
			// entity.visibleEnemies: percorrer o mapa direto fazia dois
			// monstros sobrepostos piscarem trocando quem fica na frente.
			// Ver game/render_order.go.
			for _, e := range drawableEnemies(network.InterpolatedEnemies()) {
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
			drawEnemyDeathFX(view)
			// Sem Unlock aqui: InterpolatedEnemies devolve uma COPIA e cuida
			// do proprio bloqueio por dentro. O laco antigo segurava
			// RemoteEnemiesMutex durante o desenho inteiro; ao troca-lo, o
			// Unlock ficou orfao e o cliente morria no primeiro quadro com
			// "sync: unlock of unlocked mutex".

			network.RemoteProjectilesMutex.Lock()
			// Ordem estavel pelo mesmo motivo dos corpos e das magias.
			for _, projID := range drawableProjectileIDs(network.RemoteProjectiles) {
				proj := network.RemoteProjectiles[projID]
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
			// Espinhoes e nevoa da chefe do mapa 7, replicados pelo host via
			// MsgBossThorn/MsgBossFog (host_boss.go) e envelhecidos aqui a
			// cada quadro por AdvanceClientSkills. Mesma ordem do ramo host
			// logo acima: a nevoa por ultimo, porque ela cobre a arena
			// inteira e os espinhoes ficariam por baixo dela se viessem antes.
			if network.ClientSkills != nil {
				network.ClientSkills.DrawThorns()
				network.ClientSkills.DrawFog()
			}
			// O anel de contato, do lado do cliente: sem mensagem de protocolo
			// nova (o raio e constante), so a posicao — e o cliente ja a tem em
			// InterpolatedEnemies(), o mesmo mapa que acabou de desenhar os
			// corpos acima. `e.X/e.Y` sao o PE (mesma ancora de `Position` no
			// host), entao o `HitOffsetY` do EnemyDef sobe o centro ate o corpo
			// visivel — a mesma conta que `Enemy.HitCenter()` faz do lado host.
			// So com Legiao em campo, pela mesma razao do ramo do host.
			for _, e := range network.InterpolatedEnemies() {
				if network.ClientSkills == nil || !network.ClientSkills.AnyLegion() {
					break
				}
				if e.Type == string(entity.EnemyTypeDarkLady) {
					offY := entity.GetEnemyDef(entity.EnemyTypeDarkLady).HitOffsetY
					center := rl.NewVector2(float32(e.X), float32(e.Y)+offY)
					skill.DrawBossContactRing(center)
					break
				}
			}
		}

		// A barra de vida dos canhoes, FORA do galho host/cliente: o canhao
		// nao e um `entity.Enemy` e nao entra nem no EntityManager do host nem
		// no RemoteEnemies do cliente — a vida dele vive em network/cannon_view.go
		// e as duas maquinas leem do mesmo lugar. Desenhar aqui e a unica forma
		// de as duas verem a mesma barra sem duplicar o laco.
		//
		// Por ultimo dentro do espaco de mundo porque e leitura, e leitura fica
		// por cima: uma bola de fogo passando na frente nao pode esconder a
		// unica informacao que diz se vale continuar batendo.
		for _, c := range network.CannonBars() {
			entity.DrawStructureHealthBar(c.Position.X, c.Position.Y, c.HP, c.Max,
				network.CannonBarAbove, network.CannonBarHalfWidth)
		}
	})

	// ========== Screen space (unaffected by camera) ==========

	// As setas das gargulas fora da tela (game/sentry_markers.go), ANTES dos
	// controles do celular: uma seta que cair sob o joystick fica por baixo
	// dele, e nao o contrario. Vale para PC e celular.
	//
	// Esta chamada ja existiu e sumiu numa edicao paralela deste arquivo em
	// 28/09/2026 — sem ela o linker descarta o indicador inteiro e nada
	// avisa. TestRendererDrawsSentryMarkers (sentry_markers_test.go) trava.
	drawSentryMarkers(cam, sw, sh)

	if cfg.FullScreen {
		geom := input.ComputeAndroidControlGeom(sw, sh)
		joystickCenterX := sw * 0.15
		joystickCenterY := sh * 0.80
		baseRadius := sh * 0.08
		knobRadius := sh * 0.04

		input.DrawJoystickVisual(*ts, joystickCenterX, joystickCenterY, baseRadius, knobRadius)

		// THE ACTION CLUSTER, from the outside in: the abilities first, the
		// basic attack last. The attack is the biggest circle and the one the
		// thumb rests on; drawn first, the two orbiting buttons would overlap
		// its ring and the middle would stop reading as the middle.
		//
		// All three go through ui.DrawActionButton so they cannot drift
		// apart. The fire button dims while the cadence recharges (RingOnly:
		// no figures — at one to three hits per second they would be an
		// unreadable blur), the abilities show the seconds.
		uiSkillButton.Layout(sw, sh)
		uiSkillButton.Draw(androidActionLock.LockedFor(input.SlotSkill))
		if abilityUsable(p.CharType, 1) {
			uiUltimateButton.Layout(sw, sh)
			uiUltimateButton.Draw(androidActionLock.LockedFor(input.SlotUltimate))
		}
		ui.DrawActionButton(ui.ActionButtonState{
			Center:        geom.AttackCenter(),
			Radius:        geom.AttackRadius,
			Icon:          ui.IconAttack,
			Label:         "FIRE",
			Aiming:        aj.IsActive,
			Origin:        aj.Origin,
			Current:       aj.Current,
			Locked:        androidActionLock.LockedFor(input.SlotAttack),
			Cooldown:      network.AttackCooldown(network.LocalPlayerID),
			CooldownTotal: entity.GetCharacter(p.CharType).AttackInterval(),
			RingOnly:      true,
		})
	} else {
		// The keyboard has no ability buttons to draw the counters on, so the
		// desktop gets its own recharge bar.
		DrawSkillCooldownBar(p, sw, sh)
	}

	// Health is synced onto the player by SyncLocalPlayer; the bar stays up
	// while dead so the player can see they are at zero.
	ui.DrawHealthBar(p.Health, p.MaxHealth)
	ui.DrawNetworkBadge(sw)

	// Configuracoes: o botao discreto (so celular; desktop usa ESC).
	DrawSettingsOverlayButton(cfg, sw, sh)

	// Tamanhos BASE: ui.TextSize aplica a escala de Configuracoes > Interface
	// e o piso. Ver internal/ui/text.go.
	// Posicoes em pixels da tela nativa (ui.Px): em 540p nada dobra.
	rl.DrawText(fmt.Sprintf("Players: %d", len(network.GetAllPlayers())), int32(ui.Px(10)), int32(ui.Px(35)),
		ui.TextSize(22), rl.White)

	ui.DrawWaveHUD(sw, sh)
	ui.DrawBossBar(sw, sh)
	ui.DrawBossVictoryFX(sw, sh)

	// Revive countdown. Hidden during a Game Over: nobody is coming back
	// until the host restarts the stage, and the overlay says so.
	if p.IsDead && !network.GameOver {
		respawnSize := ui.TextSize(38)
		respawnText := fmt.Sprintf("Ressuscitando em: %.0fs", p.RespawnIn)
		textWidth := rl.MeasureText(respawnText, respawnSize)
		xPos := int32((int(sw) - int(textWidth)) / 2)
		yPos := int32(sh/2 - ui.Px(30))
		rl.DrawText(respawnText, xPos, yPos, respawnSize, rl.Red)
	}

	if network.TestMode {
		testSize := ui.TextSize(22)
		rl.DrawText("MODO TESTE - sem recarga", int32(ui.Px(10)), int32(sh-ui.Px(10))-testSize, testSize, rl.Lime)
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
	//
	// Duas esperas, uma exclusiva da outra: a conexao que ainda nao abriu nao
	// tem contagem para mostrar (o prazo do dial resolve em segundos), a que
	// caiu tem. Ver network/client_connect.go.
	if network.Role == "client" {
		if network.Connecting() {
			ui.DrawConnectingOverlay(sw, sh)
		} else if network.Reconnecting() {
			ui.DrawReconnectOverlay(sw, sh, network.ReconnectRemaining())
		}
	}

	// Configuracoes, por cima de tudo o que e jogo: e alcancavel de dentro
	// da partida (Passo 6) e a partida continua simulando atras dela, como
	// a sobreposicao de reconexao acima.
	UpdateAndDrawSettingsOverlay(sw, sh)

	// Por ultimo de tudo: o medidor le os contadores que os passes acima
	// acabaram de encher, e o quadro que ele cronometra e este.
	DrawPerfHUD(sw, sh, w, p)

	rl.EndDrawing()
}

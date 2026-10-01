package game

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiSkillButton is the on-screen fireball button used on Android.
var uiSkillButton = ui.NewSkillButton()

// uiUltimateButton is the on-screen ultimate (supreme skill) button (Android).
var uiUltimateButton = ui.NewUltimateButton()

// androidActionLock is the aim cluster's single-gesture rule: the first of the
// three action buttons to take a touch owns it, and the other two stop
// answering until it lets go.
//
// It exists because aiming is a HELD drag that routinely crosses the other
// buttons — they are one thumb-width away, which is the point of a cluster —
// and each one was arming on the way past and firing on release. The player
// asked for an angle and got an ultimate. The movement joystick is outside the
// lock on purpose: walking while aiming is the game working as intended.
var androidActionLock input.ActionLock

// cancelAndroidAim drops any aim in flight and frees the cluster.
//
// For every path that stops updating the buttons: a finished run, a body
// frozen inside a portal, a session on desktop. Release is detected by
// scanning touches, so returning before that scan would leave a button armed
// and the other two locked out for the rest of the run.
func cancelAndroidAim(aj *input.AimJoystick) {
	aj.Cancel(&androidActionLock)
	uiSkillButton.Cancel(&androidActionLock)
	uiUltimateButton.Cancel(&androidActionLock)
	androidActionLock.Reset()
}

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
		cancelAndroidAim(aj)
		return dir
	}
	if network.LocalPlayerInPortal {
		// Vanished into the party's portal and frozen (host_portal_presence.go):
		// no movement, no attack, no skill — only the way out. ESC/SAIR
		// steps the player clear of the rectangle and clears the flag
		// locally; the host confirms it on its own the next tick it
		// recomputes presence.
		cancelAndroidAim(aj)
		UpdatePortalCancel(p, w, cfg, sw, sh)
		return dir
	}

	// Configuracoes DE DENTRO da partida (Passo 6). Fora do galho acima de
	// proposito: enquanto o jogador espera um portal, ESC so cancela o
	// portal — ver settings_overlay.go para o motivo por extenso.
	RequestOpenSettingsOverlay(cfg, aj, sw, sh)
	if SettingsOverlayOpen() {
		// A tela possui o toque/mouse deste quadro; nenhum movimento,
		// ataque ou habilidade e lido enquanto ela estiver aberta. A
		// PARTIDA CONTINUA simulando (loop.go nao para por causa disto) —
		// so a leitura de controle local congela.
		cancelAndroidAim(aj)
		return dir
	}

	if cfg.FullScreen {
		// Android: compute dynamic control positions proportional to screen height.
		// Use one shared geometry source so hit-areas match the visuals and the
		// fire and skill buttons never overlap.
		geom := input.ComputeAndroidControlGeom(sw, sh)
		joystickCenterX := sw * 0.15
		if input.LeftHanded() {
			// O joystick troca de lado com o cluster de acoes (Modo
			// canhoto, doc/settings.md): sem isto os dois ficariam do
			// MESMO lado depois de espelhar so o cluster.
			joystickCenterX = sw * (1 - 0.15)
		}
		joystickCenterY := sh * 0.80
		baseRadius := sh * 0.08

		joystickRect := rl.NewRectangle(
			joystickCenterX-baseRadius,
			joystickCenterY-baseRadius,
			baseRadius*2,
			baseRadius*2,
		)
		attackRect := geom.AttackRect()

		// JoystickRadiusScale (0.7..1.5) vem da tela de Configuracoes.
		ts.SetMaxRadius(baseRadius * 0.75 * input.JoystickRadiusScale())
		ts.Update(joystickRect, attackRect)

		// raw e o deslocamento do polegar (0..1 do raio maximo); dir passa
		// pela curva de resposta, que chega ao comprimento 1 do teclado
		// antes da borda — sem ela o celular andava mais devagar que o
		// desktop (input/joystick_curve.go).
		raw := ts.GetJoystickDelta()
		dir.X, dir.Y = input.JoystickMoveResponse(raw.X, raw.Y)

		// Sprint detection: joystick displacement > 70% of max radius.
		// Le o deslocamento CRU: a curva satura em 1 bem antes de 0,70.
		joyMag := rl.Vector2Length(raw)
		p.IsSprinting = joyMag > entity.SprintThreshold

		// Aim joystick update — returns true on release (fire).
		// The whole cluster geometry goes in: the fire button hit-tests as a
		// circle now, and the lock is what keeps it from claiming a touch
		// while an ability is being aimed.
		if aj.Update(geom, *ts, &androidActionLock, sw, sh) {
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
		} else {
			// Stopped being available — possibly mid-aim, and a button that
			// disappears while holding the lock locks the cluster forever.
			uiUltimateButton.Cancel(&androidActionLock)
		}
	} else {
		// Desktop sprint detection: Left Shift key
		p.IsSprinting = rl.IsKeyDown(rl.KeyLeftShift)
		cancelAndroidAim(aj)
		// "Mira segue o mouse continuamente" (Configuracoes > Controles):
		// so entao o chevron (ui/aim_indicator.go) segue o cursor a cada
		// quadro. No modo padrao ("so no clique") o chevron so muda no
		// instante do clique — ver o bloco de ataque do mouse mais abaixo.
		if settings.Get().ContinuousMouseAim {
			updateDesktopAimFromMouse(p, aj, cam)
		}
	}

	// Keyboard input (WASD + arrow keys) — desktop primary input.
	//
	// O TECLADO E NORMALIZADO NUM VETOR PROPRIO, e so depois somado. Ate
	// 30/09/2026 as teclas eram somadas direto em dir e a correcao de diagonal
	// (x 0,7071) rodava sobre dir inteiro — inclusive sobre o vetor que o
	// JOYSTICK tinha acabado de escrever no Android. Como um polegar quase
	// nunca empurra exatamente na horizontal ou na vertical, as duas
	// componentes eram sempre diferentes de zero e o jogador do celular andava
	// a ~71% da velocidade do PC em praticamente toda direcao. Era a causa do
	// "celular anda mais devagar".
	var kb rl.Vector2
	if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
		kb.Y -= 1
	}
	if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
		kb.Y += 1
	}
	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		kb.X -= 1
	}
	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		kb.X += 1
	}
	if kb.X != 0 && kb.Y != 0 {
		kb.X *= 0.7071 // 1/sqrt(2) for diagonal
		kb.Y *= 0.7071
	}
	dir.X += kb.X
	dir.Y += kb.Y
	// Joystick e teclado juntos (teclado fisico no celular) nao passam de 1.
	if l := rl.Vector2Length(dir); l > 1 {
		dir = rl.Vector2Scale(dir, 1/l)
	}

	// Handle mouse click attack on desktop — convert screen coords to world coords
	if !cfg.FullScreen && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		screenPos := rl.GetMousePosition()
		mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
		// "So no clique" (o padrao): o chevron muda AQUI, no instante do
		// tiro, mesmo quando ContinuousMouseAim esta desligado — antes
		// desta linha o indicador de mira nunca refletia o mouse no
		// desktop (ficava preso na direcao padrao "para baixo").
		aj.AimDir = aimDirTo(p.Position, mousePos)
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
	if sb.Update(&androidActionLock) {
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

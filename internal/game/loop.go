package game

import (
	"log"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/sound"
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
	// settings.Load() ANTES de InitWindow, e nao "logo depois de sound.Init()"
	// como a leitura mais direta do pedido sugere: o VSync escolhido em
	// Configuracoes so tem onde pegar se for lido A TEMPO de virar o hint de
	// criacao do contexto GL, e isso so existe ANTES da janela abrir. Carregar
	// aqui e reaplicar o audio (abaixo, apos sound.Init) cobre as duas partes
	// do pedido sem duas leituras do arquivo — ver doc/settings.md, "Ordem de
	// carregamento (decisao)".
	loaded := settings.Load()
	cfg.VSync = loaded.VSync
	cfg.TargetFPS = int32(loaded.TargetFPS)

	// ANTES de InitWindow: o vsync e um hint de criacao do contexto GL e nao
	// tem efeito depois que a janela existe. Ver Config.VSync para por que a
	// falta dele nao aparecia no medidor de quadro — `SetTargetFPS` cadencia,
	// nao sincroniza, e a imagem rasga do mesmo jeito.
	//
	// FlagWindowHighdpi tambem e so hint de criacao: sem ele, num monitor
	// com escala do Windows diferente de 100% (comum em notebook — e o "ponto
	// vermelho" fora do botao visto na sessao de teste de 04/09/2026 era
	// exatamente isso, GetMousePosition() num espaco de pixel diferente do
	// que GetScreenWidth/Height usam pra desenhar a tela de Configuracoes).
	// Com a flag os dois passam a falar o mesmo pixel.
	flags := uint32(rl.FlagWindowHighdpi)
	if cfg.VSync {
		flags |= uint32(rl.FlagVsyncHint)
	}
	rl.SetConfigFlags(flags)
	// (0,0) = resolucao nativa. No celular pode vir menor: Configuracoes >
	// Graficos > Resolucao, ampliada pelo scaler de hardware (render_scale.go).
	winW, winH := startupWindowSize(loaded)
	rl.InitWindow(winW, winH, cfg.Title)
	defer rl.CloseWindow()
	noteDisplaySize(winW, winH, loaded)
	// A tela de Configuracoes compara isto com a resolucao escolhida para
	// dizer se ela ja vale ou se pede REINICIAR (ui/render_restart.go).
	ui.SetStartupWindowSize(winW, winH)
	// O MESMO fator vai para a interface: todo pixel fixo dela e pixel da
	// tela nativa, e em 540p ela nao pode dobrar de tamanho (ui/text.go).
	renderScale := renderZoomScale(winW, winH, loaded)
	cfg.CameraZoom *= renderScale
	ui.SetRenderScale(renderScale)
	if loaded.Fullscreen {
		// ToggleBorderlessWindowed, nao ToggleFullscreen: a janela ja nasce
		// do tamanho do monitor (InitWindow(0,0,...) acima), entao entrar
		// em tela cheia so precisa tirar a borda — ToggleFullscreen troca de
		// modo de video de verdade, e foi isso que piscava preto ao ligar
		// "TELA CHEIA" em Configuracoes (sessao de teste de 04/09/2026). Ver
		// o mesmo comentario em ui/settings_apply_desktop.go.
		rl.ToggleBorderlessWindowed()
	}

	// A musica de fase vive pelo processo inteiro: um so InitAudioDevice, e as
	// trocas de trilha (playSession no mapa inicial, travelTo a cada portal)
	// so trocam o que esta tocando, nao o dispositivo.
	sound.Init()
	defer sound.Shutdown()
	// Aplica os volumes/geometria/HUD carregados AOS BARRAMENTOS antes do
	// primeiro quadro (Passo 1) — settings.Load() so troca o MODELO, quem
	// guarda a propria copia (sound, input, ui) precisa ser avisado.
	// applyTargetFPS (dentro de ApplyAllSettings) ja chama rl.SetTargetFPS
	// com o teto carregado — cfg.TargetFPS fica so como o registro do valor
	// efetivo. Zero = sem teto proprio, com o vsync cadenciando no refresh
	// do monitor; um teto MENOR que o refresh continua valendo (o 30 do
	// celular) — ver doc/performance.md, "Vsync e teto de quadro".
	ui.ApplyAllSettings(loaded)

	for !rl.WindowShouldClose() {
		playSession(cfg)
	}
	// Tira a sala online da lista na hora, sem esperar o relay notar o fim do
	// sinal de vida (network/host_online.go).
	network.ShutdownOnline()
}

// playSession runs the menu and one full match: world load through however
// it ends. It handles the menu, networking, and main game loop.
func playSession(cfg Config) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	cam := NewCamera(sw, sh, cfg.CameraZoom)

	w := LoadWorld("assets/maps/world_01.json")
	defer func() { w.Unload() }()
	sound.PlayForMap(w.Path)

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
	// Os retratos ja saem a cada troca de mapa (world_travel.go); isto cobre o
	// fim da sessao, quando nao houve troca depois da ultima cena.
	defer ui.UnloadPortraits()
	// Os glifos dos botoes, ao contrario dos retratos, vivem a sessao inteira
	// (sao tres, e os botoes existem em todo mapa) — entao este e o UNICO
	// lugar que os descarrega. Ver ui/action_icons.go.
	defer ui.UnloadActionIcons()

	network.RemotePlayersMutex.Lock()
	if state, ok := network.RemotePlayers[network.LocalPlayerID]; ok {
		p.Position.X = float32(state.X)
		p.Position.Y = float32(state.Y)
		p.Color = state.Color
	}
	network.RemotePlayersMutex.Unlock()

	// Quem volta de um REINICIAR como host (ui/resume_session.go) ja deixou a
	// fase enfileirada. Aplicada AQUI, antes do primeiro quadro: dentro do
	// laco o dialogo de abertura do mapa 1 tomaria o quadro antes da viagem.
	// Sem nada enfileirado (toda sessao comum) e um no-op.
	w = ApplyPendingTravel(w, p)
	bounds = w.Bounds

	ts := input.NewTouchState()
	aj := input.NewAimJoystick()
	dlg := NewDialogueDirector()
	frameCount := 0

	for !rl.WindowShouldClose() && !network.SessionEnded() {
		dt := rl.GetFrameTime()
		// Prova de vida do host para o relay: sem quadros (app em segundo
		// plano, travado), a sala online morre no servidor.
		network.MarkFrame()
		// O OUVINTE E O JOGADOR LOCAL. E a distancia ate ELE que decide se um
		// efeito de magia e ouvido e com que volume (internal/sound/sfx.go),
		// entao esta linha e o que faz a regra "so quem esta perto escuta"
		// existir nesta maquina. Fica junto de sound.Update pelo mesmo motivo
		// que ele: precisa valer tambem nos quadros congelados por dialogo ou
		// reconexao, onde magias lancadas antes ainda estao no ar.
		sound.SetListener(p.Position)
		// "Silenciar em segundo plano" (Configuracoes > Audio): tambem roda
		// todo quadro, pelo mesmo motivo de sound.Update logo abaixo — um
		// dialogo ou a reconexao nao podem fazer o jogo esquecer de checar o
		// foco da janela.
		updateBackgroundAudioMute()
		// Roda todo quadro, inclusive quando um dialogo ou a reconexao congelam
		// o resto do laco logo abaixo: a musica nao para com eles, e os
		// efeitos ja disparados precisam continuar sendo bombeados.
		sound.Update()
		// Depois de sound.Update e antes de qualquer `continue` do laco:
		// a virada da trilha de vitoria e um ESTADO observado todo quadro
		// (ver victory_music.go), entao nao pode ficar atras do freio do
		// dialogo nem do da reconexao.
		updateVictoryMusic()
		sw := float32(rl.GetScreenWidth())
		sh := float32(rl.GetScreenHeight())

		// Drives the client-side reconnect state machine (network/reconnect.go)
		// every frame, win or lose the connection this frame or not — the same
		// single delegated call UpdateSimulation is for the host below.
		if network.Role == "client" {
			network.TickReconnect()
		}

		// Estar sem ligacao com o host manda no quadro como um dialogo manda:
		// a entrada local congela, nada simula, e o mundo espelhado continua
		// sendo desenhado como visto por ultimo para a tela nao ficar em
		// branco atras da sobreposicao.
		//
		// AwaitingHost, e nao Reconnecting: os dois casos sao o mesmo quadro.
		// Enquanto a PRIMEIRA conexao nao abre, este bloco nao existia, e o
		// resultado era um cliente andando por um mapa vazio como se estivesse
		// jogando — ver network/client_connect.go.
		if network.Role == "client" && network.AwaitingHost() {
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
			ApplyMoveBuff(p)
			p.Update(dir, dt, bounds)
			ResolveCollision(p, w.Collision)
			w.UpdateArenaGate(p)
			UpdateStageSkip(cfg, w)
			if network.Role != "" {
				// UpdateLocalPlayerState, e nao UpdatePlayerState: este
				// PlayerState e o que ESTA maquina prediz, e ele nao carrega
				// os veredictos do host (InPortal, Absent). Ver globals.go.
				network.UpdateLocalPlayerState(network.PlayerState{
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
		// A saida se abre pelo mesmo motivo que o portal aparece: quando a
		// fase termina, nao quando alguem entra nela. Roda todo quadro, em
		// toda maquina, pelo mesmo motivo de UpdateArenaGate — ver
		// portal_exit_gate.go.
		w.UpdatePortalExitGate()

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
			simStart := time.Now()
			network.CurrentHost.UpdateSimulation(dt)
			recordSimTime(time.Since(simStart))
		}

		// Animate client-side skill visuals (fireballs, explosions,
		// ground fire, sanctuaries — placement + dissipation only).
		if network.Role == "client" && network.ClientSkills != nil {
			simStart := time.Now()
			network.ClientSkills.UpdateFire(dt)
			network.ClientSkills.AdvanceSanctuaries(dt)
			network.AdvanceClientSkills(dt, w.Collision)
			recordSimTime(time.Since(simStart))
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

package ui

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/settings"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// menuScreen e o estado do laco do menu inicial. Era tres bool soltos
// (selected, joinMode e agora settingsMode) — tres booleanos que se excluem
// e o comeco de um estado impossivel (os tres true ao mesmo tempo nunca
// deveria compilar, e com bool solto compila). Um enum fecha essa porta.
type menuScreen int

const (
	screenMain menuScreen = iota
	screenJoin
	screenSettings
)

// localIdentityOnce guards the ONE player ID this process ever uses.
//
// generatePlayerID used to run on every call to startHost/connectToHost, so
// a client that lost its connection and reconnected would show up as a
// brand-new player — the old body would stay parked in the field forever,
// and the reconnecting player would lose whatever position, health and
// cooldowns it had. The host recognizes a reconnect by PlayerID
// (host_rejoin.go), so the ID has to survive the reconnect to mean anything.
var (
	localIdentityOnce sync.Once
	localPlayerID     string
)

// ensureLocalPlayerID returns the same ID for the lifetime of this process,
// generating it once. Known, accepted limit: if the OS kills the app, this is
// a new process and a new session — there is no disk persistence, and none is
// wanted here.
func ensureLocalPlayerID() string {
	localIdentityOnce.Do(func() {
		localPlayerID = generatePlayerID()
	})
	return localPlayerID
}

// adoptLocalPlayerID faz este processo usar id em vez de gerar um. So vale
// antes do primeiro ensureLocalPlayerID — e o caso de quem volta de um
// REINICIAR (resume_session.go): o processo e novo, a identidade no host
// nao, e e ela que devolve o slot preservado.
func adoptLocalPlayerID(id string) {
	localIdentityOnce.Do(func() {
		localPlayerID = id
	})
}

// hitPad is the extra hit area (in pixels) added around each menu button so
// small drawn rectangles are still easy to tap on touch screens. It scales
// with screen height because a fixed pixel pad is tiny on high-DPI phones.
func hitPad() float32 {
	sh := float32(rl.GetScreenHeight())
	return max(sh*0.035, Px(18))
}

// hit returns true if the point is inside rect OR within hitPad pixels of it.
// The drawn rectangle is unchanged; only the clickable area is enlarged.
func hit(rect rl.Rectangle, point rl.Vector2) bool {
	pad := hitPad()
	expanded := rl.NewRectangle(
		rect.X-pad,
		rect.Y-pad,
		rect.Width+2*pad,
		rect.Height+2*pad,
	)
	return rl.CheckCollisionPointRec(point, expanded)
}

// ShowMenu renders the start menu where the player can host or join a game.
func ShowMenu(playerSpawn rl.Vector2) entity.CharacterType {
	// Voltando de um REINICIAR (Configuracoes > Graficos): direto para a
	// partida, sem menu. Ver resume_session.go.
	if char, ok := resumeSession(playerSpawn); ok {
		return char
	}
	selected := false
	screen := screenMain
	refreshTimer := 0
	scanning := false
	settingsScreen := NewSettingsScreen()

	// Read and consume ONCE: a client that never managed to connect, or that
	// exhausted the reconnect window (network/reconnect.go), lands back here
	// instead of being left in limbo, and the player deserves to know why they
	// are looking at the menu again instead of the match they were just in.
	notice := disconnectNotice(network.LastFailure())
	network.ClearGaveUp()

	var chosenChar entity.CharacterType

	for !selected && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		switch screen {
		case screenJoin:
			drawDiscoveryView(&selected, &screen, &refreshTimer, &scanning, &chosenChar)
		case screenSettings:
			sw := float32(rl.GetScreenWidth())
			sh := float32(rl.GetScreenHeight())
			if settingsScreen.Update(sw, sh) {
				// Gravar em disco so ao SAIR da tela (Passo 4) — nunca a
				// cada quadro de arraste.
				if err := settings.Save(); err != nil {
					log.Printf("[Configuracoes] falha ao gravar settings.json: %v", err)
				}
				screen = screenMain
			}
		default:
			drawMainMenu(&selected, &screen, playerSpawn, &chosenChar)
			if notice != "" {
				drawDisconnectedNotice(notice)
			}
		}

		rl.EndDrawing()
		refreshTimer++
	}

	// Stop discovery when leaving menu
	network.StopDiscovery()
	fmt.Printf("[Menu] Exiting menu, role=%s\n", network.Role)
	return chosenChar
}

// startHost initializes the host
func startHost(selected *bool, playerSpawn rl.Vector2, charType string) {
	log.Println("[Menu] Host selected")
	startHostAs(playerSpawn, charType, entity.PresetColors[rand.Intn(len(entity.PresetColors))])
	*selected = true
}

// startHostAs sobe o host com uma cor ja decidida — sorteada no menu, ou a
// mesma de antes quando o host volta de um REINICIAR (resume_session.go).
func startHostAs(playerSpawn rl.Vector2, charType, color string) {
	network.Role = "host"
	network.LocalPlayerID = ensureLocalPlayerID()
	network.LocalPlayerColor = color
	network.LocalPlayerCharacter = charType
	log.Printf("[Menu] Host ID: %s, Color: %s", network.LocalPlayerID, color)

	network.UpdatePlayerState(network.PlayerState{
		PlayerID:  network.LocalPlayerID,
		X:         int(playerSpawn.X),
		Y:         int(playerSpawn.Y),
		Color:     color,
		Character: charType,
	})

	h, err := network.StartHost(9000, network.LocalPlayerID, color, charType, playerSpawn)
	if err != nil {
		log.Fatalf("Failed to start host: %v", err)
	}
	network.CurrentHost = h
	network.ServerAddress = getDisplayAddress(9000)
	log.Printf("[Menu] Host started, broadcasting initial state...")

	// Start discovery broadcaster
	stopBroadcast := make(chan struct{})
	go network.StartDiscoveryBroadcaster(9000, stopBroadcast)

	// Start query responder for Android clients
	stopResponder := make(chan struct{})
	go network.StartQueryResponder(9000, stopResponder)

	network.CurrentHost.BroadcastRoster()
}

// connectToHost entrega a abertura da sessao a camada de rede
// (network.StartClientSession, network/client_connect.go) e volta na hora. A
// cor fica sendo escolhida aqui porque e identidade de apresentacao; o socket,
// o prazo do dial e o join nao sao assunto de menu, e era manter os tres aqui
// que deixava o estado "conectando" sem dono.
//
// Nao ha mais log.Fatalf: uma conexao que falha volta ao menu com o motivo na
// tela (drawDisconnectedNotice, abaixo). Fechar o aplicativo nunca foi
// resposta para "o anfitriao nao respondeu".
func connectToHost(addr string, charType string) {
	color := entity.PresetColors[rand.Intn(len(entity.PresetColors))]
	network.StartClientSession(addr, ensureLocalPlayerID(), color, charType)
}

// disconnectNotice traduz o motivo do fim da sessao anterior. O texto e da UI
// de proposito: a camada de rede diz o QUE aconteceu
// (network.SessionFailure), nao como contar.
func disconnectNotice(kind network.SessionFailure) string {
	// Sala online: a camada de rede sabe o motivo exato (sala cheia,
	// encerrada, servidor lotado, sem internet) - network/online_errors.go.
	if detail := network.LastFailureDetail(); detail != "" && kind != network.FailureNone {
		return detail
	}
	switch kind {
	case network.FailureConnect:
		if network.IsOnlineRoom(network.ServerAddress) {
			return "Nao foi possivel entrar na sala online."
		}
		// Aponta para a causa provavel em vez de so lamentar: o anfitriao
		// APARECE na lista pelo broadcast de saida dele, que nenhum firewall
		// barra, enquanto entrar depende da porta de jogo estar aberta na
		// maquina dele. Uma sala visivel e inalcancavel e exatamente este
		// caso, e sem a dica ele nao tem por onde comecar.
		return "Nao foi possivel conectar ao anfitriao (porta 9000 bloqueada?)."
	case network.FailureReconnect:
		return "Nao foi possivel reconectar ao host. A partida foi encerrada."
	}
	return ""
}

// drawDisconnectedNotice explains why the player is looking at the menu
// again instead of the match they were just in: reconnecting is a state the
// player understood ("Reconectando...", reconnect_overlay.go), but not
// hearing about giving up would just look like the app quietly kicked them.
func drawDisconnectedNotice(msg string) {
	sh := float32(rl.GetScreenHeight())
	sw := float32(rl.GetScreenWidth())
	size := int32(max(sh*0.03, Px(18)))
	width := rl.MeasureText(msg, size)
	rl.DrawText(msg, int32(sw/2)-width/2, int32(sh*0.28), size, rl.Maroon)
}

func generatePlayerID() string {
	return fmt.Sprintf("player_%d", time.Now().UnixNano())
}

// getDisplayAddress returns the LAN IP address for display purposes.
func getDisplayAddress(port int) string {
	ip := getOutboundIP()
	return fmt.Sprintf("%s:%d", ip, port)
}

// getOutboundIP gets the preferred outbound IP address.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

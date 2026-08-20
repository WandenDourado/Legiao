package ui

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
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

// hitPad is the extra hit area (in pixels) added around each menu button so
// small drawn rectangles are still easy to tap on touch screens. It scales
// with screen height because a fixed pixel pad is tiny on high-DPI phones.
func hitPad() float32 {
	sh := float32(rl.GetScreenHeight())
	return max(sh*0.035, 18)
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
	selected := false
	joinMode := false
	refreshTimer := 0
	scanning := false

	// Read and consume ONCE: a client that exhausted the reconnect window
	// (network/reconnect.go) lands back here instead of being left in limbo,
	// and the player deserves to know why they are looking at the menu again
	// instead of the match they were just in.
	disconnected := network.GaveUp()
	network.ClearGaveUp()

	var chosenChar entity.CharacterType

	for !selected && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		if joinMode {
			drawDiscoveryView(&selected, &joinMode, &refreshTimer, &scanning, &chosenChar)
		} else {
			drawMainMenu(&selected, &joinMode, playerSpawn, &chosenChar)
			if disconnected {
				drawDisconnectedNotice()
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

// drawMainMenu draws the host/join selection screen. Buttons are sized and
// positioned proportionally to the screen so the clickable area (hit) always
// matches the drawn rectangle, even on large / high-DPI displays.
func drawMainMenu(selected *bool, joinMode *bool, playerSpawn rl.Vector2, chosenChar *entity.CharacterType) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())

	btnW := sw * 0.5
	btnH := max(sh*0.09, 56)
	btnX := (sw - btnW) / 2
	hostRect := rl.NewRectangle(btnX, sh*0.4, btnW, btnH)
	joinRect := rl.NewRectangle(btnX, sh*0.4+btnH*1.4, btnW, btnH)

	fontSize := int32(max(sh*0.035, 22))

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if hit(hostRect, mouse) {
			// A cancelled character select drops back to this menu instead of
			// opening a host nobody asked for.
			charType, confirmed := ShowCharacterSelect()
			if confirmed {
				*chosenChar = charType
				startHost(selected, playerSpawn, string(charType))
			}
		} else if hit(joinRect, mouse) {
			*joinMode = true
		}
	}

	rl.DrawRectangleRec(hostRect, rl.LightGray)
	rl.DrawRectangleRec(joinRect, rl.LightGray)
	hostLabel := "Host Game (Wi-Fi)"
	joinLabel := "Join Game (Wi-Fi)"
	rl.DrawText(hostLabel, int32(hostRect.X+(hostRect.Width-float32(rl.MeasureText(hostLabel, fontSize)))/2),
		int32(hostRect.Y+(hostRect.Height-float32(fontSize))/2), fontSize, rl.Black)
	rl.DrawText(joinLabel, int32(joinRect.X+(joinRect.Width-float32(rl.MeasureText(joinLabel, fontSize)))/2),
		int32(joinRect.Y+(joinRect.Height-float32(fontSize))/2), fontSize, rl.Black)
}

// drawDiscoveryView lists the hosts found on the local network. Discovery is
// automatic (UDP listener, UDP query and TCP scan all run on entry), so the
// only control here is Back.
func drawDiscoveryView(selected *bool, joinMode *bool, refreshTimer *int, scanning *bool, chosenChar *entity.CharacterType) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())

	// Start TCP scan when entering discovery view (more reliable for Android/Desktop)
	if !*scanning {
		*scanning = true
		// Clear previous results
		network.ClearDiscoveredHosts()
		// Listen for host UDP broadcasts (LEGION_HOST) so the client can
		// find a host without relying on the query/response or TCP scan paths.
		go network.StartDiscoveryListener()
		// Start TCP scan of local network as fallback
		go network.StartTCPScan(9000)
		// Also start UDP query sender as backup
		go network.StartQuerySender(9000)
	}

	hosts := network.GetDiscoveredHosts()

	rl.DrawText("Discovered Hosts:", 80, 60, 20, rl.Black)
	rl.DrawText("Port 9000 is used automatically", 80, 82, 16, rl.DarkGray)

	if len(hosts) == 0 {
		rl.DrawText("Searching for hosts...", 80, 120, 20, rl.Gray)
		// Show scanning animation
		dots := strings.Repeat(".", 1+(*refreshTimer/30)%3)
		rl.DrawText(dots, int32(sw*0.45), int32(sh*0.13), int32(max(sh*0.025, 16)), rl.Gray)
	} else {
		rl.DrawText(fmt.Sprintf("Found %d host(s):", len(hosts)), int32(sw*0.1), int32(sh*0.11), int32(max(sh*0.025, 16)), rl.DarkGray)
		for i, host := range hosts {
			hostH := max(sh*0.07, 44)
			hostRect := rl.NewRectangle(sw*0.1, sh*0.16+float32(i)*hostH*1.25, sw*0.8, hostH)
			rl.DrawRectangleRec(hostRect, rl.LightGray)
			rl.DrawRectangleLinesEx(hostRect, 2, rl.DarkGray)
			hostFont := int32(max(sh*0.025, 16))
			rl.DrawText(host, int32(hostRect.X+10), int32(hostRect.Y+(hostRect.Height-float32(hostFont))/2), hostFont, rl.Black)
		}
	}

	// Back button (proportional to screen height so the hit area matches the
	// drawn rectangle and stays comfortable to tap).
	btnH := max(sh*0.06, 46)
	btnY := sh * 0.72
	backW := sw * 0.3
	backRect := rl.NewRectangle((sw-backW)/2, btnY, backW, btnH)

	rl.DrawRectangleRec(backRect, rl.LightGray)

	btnFont := int32(max(sh*0.028, 18))
	rl.DrawText("Back", int32(backRect.X+(backRect.Width-float32(rl.MeasureText("Back", btnFont)))/2),
		int32(backRect.Y+(backRect.Height-float32(btnFont))/2), btnFont, rl.Black)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()

		// Check if a host was clicked
		for i, host := range hosts {
			hostH := max(sh*0.07, 44)
			hostRect := rl.NewRectangle(sw*0.1, sh*0.16+float32(i)*hostH*1.25, sw*0.8, hostH)
			if hit(hostRect, mouse) {
				charType, confirmed := ShowCharacterSelect()
				if !confirmed {
					break
				}
				*chosenChar = charType
				connectToHost(host, string(charType))
				*selected = true
				break
			}
		}

		if !*selected && hit(backRect, mouse) {
			// Leaving the list must also tear down discovery, otherwise the
			// scanners keep running behind the main menu and a later Join
			// starts a second set of them.
			log.Println("[Menu] Leaving discovery view")
			network.StopDiscovery()
			network.ClearDiscoveredHosts()
			*scanning = false
			*joinMode = false
		}
	}
}

// startHost initializes the host
func startHost(selected *bool, playerSpawn rl.Vector2, charType string) {
	log.Println("[Menu] Host selected")
	network.Role = "host"
	network.LocalPlayerID = ensureLocalPlayerID()
	color := entity.PresetColors[rand.Intn(len(entity.PresetColors))]
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
	*selected = true
}

func connectToHost(addr string, charType string) {
	network.Role = "client"
	playerID := ensureLocalPlayerID()
	network.LocalPlayerID = playerID
	network.ServerAddress = addr
	go func() {
		log.Printf("[Menu] Connecting to host at %s", addr)
		c, err := network.ConnectClient(addr)
		if err != nil {
			log.Fatalf("Failed to connect client: %v", err)
		}
		network.CurrentClient = c
		color := entity.PresetColors[rand.Intn(len(entity.PresetColors))]
		// Cached so a later reconnect (network/reconnect.go) can resend the
		// exact same identity instead of asking the player to pick again.
		network.LocalPlayerColor = color
		network.LocalPlayerCharacter = charType
		joinMsg := network.Message{
			Type: network.MsgJoin,
			Payload: network.MustMarshal(network.JoinPayload{
				PlayerID:  playerID,
				Color:     color,
				Character: charType,
			}),
		}
		network.SendMessage(joinMsg)
	}()
}

// drawDisconnectedNotice explains why the player is looking at the menu
// again instead of the match they were just in: reconnecting is a state the
// player understood ("Reconectando...", reconnect_overlay.go), but not
// hearing about giving up would just look like the app quietly kicked them.
func drawDisconnectedNotice() {
	sh := float32(rl.GetScreenHeight())
	sw := float32(rl.GetScreenWidth())
	msg := "Nao foi possivel reconectar ao host. A partida foi encerrada."
	size := int32(max(sh*0.03, 18))
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

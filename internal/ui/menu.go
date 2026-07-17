package ui

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ShowMenu renders the start menu where the player can host or join a game.
func ShowMenu(playerSpawn rl.Vector2) entity.CharacterType {
	selected := false
	joinMode := false
	manualMode := false
	manualIP := ""
	refreshTimer := 0
	scanning := false

	var chosenChar entity.CharacterType

	for !selected && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		if joinMode {
			if manualMode {
				drawManualInput(&manualIP, &selected, &chosenChar)
			} else {
				drawDiscoveryView(&selected, &manualMode, &manualIP, &refreshTimer, &scanning, &chosenChar)
			}
		} else {
			drawMainMenu(&selected, &joinMode, playerSpawn, &chosenChar)
		}

		rl.EndDrawing()
		refreshTimer++
	}

	// Stop discovery when leaving menu
	network.StopDiscovery()
	fmt.Printf("[Menu] Exiting menu, role=%s\n", network.Role)
	return chosenChar
}

// drawMainMenu draws the host/join selection screen
func drawMainMenu(selected *bool, joinMode *bool, playerSpawn rl.Vector2, chosenChar *entity.CharacterType) {
	hostRect := rl.NewRectangle(200, 150, 200, 50)
	joinRect := rl.NewRectangle(200, 250, 200, 50)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mouse, hostRect) {
			charType := ShowCharacterSelect()
			*chosenChar = charType
			startHost(selected, playerSpawn, string(charType))
		} else if rl.CheckCollisionPointRec(mouse, joinRect) {
			*joinMode = true
		}
	}

	rl.DrawRectangleRec(hostRect, rl.LightGray)
	rl.DrawRectangleRec(joinRect, rl.LightGray)
	rl.DrawText("Host Game (Wi-Fi)", int32(hostRect.X+10), int32(hostRect.Y+15), 20, rl.Black)
	rl.DrawText("Join Game (Wi-Fi)", int32(joinRect.X+10), int32(joinRect.Y+15), 20, rl.Black)
}

// drawDiscoveryView shows discovered hosts with options to scan/enter manually
func drawDiscoveryView(selected *bool, manualMode *bool, manualIP *string, refreshTimer *int, scanning *bool, chosenChar *entity.CharacterType) {
	// Start TCP scan when entering discovery view (more reliable for Android/Desktop)
	if !*scanning {
		*scanning = true
		// Clear previous results
		network.ClearDiscoveredHosts()
		// Start TCP scan of local network
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
		dots := "."
		for i := 0; i < (*refreshTimer/30)%3; i++ {
			dots += "."
		}
		rl.DrawText(dots, 280, 120, 20, rl.Gray)
	} else {
		rl.DrawText(fmt.Sprintf("Found %d host(s):", len(hosts)), 80, 110, 18, rl.DarkGray)
		for i, host := range hosts {
			hostRect := rl.NewRectangle(80, float32(140+i*40), 340, 30)
			rl.DrawRectangleRec(hostRect, rl.LightGray)
			rl.DrawRectangleLinesEx(hostRect, 2, rl.DarkGray)
			rl.DrawText(host, int32(hostRect.X+5), int32(hostRect.Y+5), 18, rl.Black)
		}
	}

	// Buttons
	scanRect := rl.NewRectangle(80, 340, 120, 30)
	manualRect := rl.NewRectangle(210, 340, 120, 30)
	backRect := rl.NewRectangle(340, 340, 80, 30)

	rl.DrawRectangleRec(scanRect, rl.LightGray)
	rl.DrawRectangleRec(manualRect, rl.LightGray)
	rl.DrawRectangleRec(backRect, rl.LightGray)

	rl.DrawText("Scan TCP", int32(scanRect.X+10), int32(scanRect.Y+5), 20, rl.Black)
	rl.DrawText("Manual IP", int32(manualRect.X+10), int32(manualRect.Y+5), 20, rl.Black)
	rl.DrawText("Back", int32(backRect.X+10), int32(backRect.Y+5), 20, rl.Black)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()

		// Check if a host was clicked
		for i, host := range hosts {
			hostRect := rl.NewRectangle(80, float32(140+i*40), 340, 30)
			if rl.CheckCollisionPointRec(mouse, hostRect) {
				charType := ShowCharacterSelect()
				*chosenChar = charType
				connectToHost(host, string(charType))
				*selected = true
				break
			}
		}

		if !*selected {
			if rl.CheckCollisionPointRec(mouse, scanRect) {
				// Start TCP scan as fallback
				log.Println("[Menu] Starting TCP scan of local network...")
				network.ClearDiscoveredHosts()
				go network.StartTCPScan(9000)
				*scanning = true
			} else if rl.CheckCollisionPointRec(mouse, manualRect) {
				*manualMode = true
			} else if rl.CheckCollisionPointRec(mouse, backRect) {
				*manualMode = false
			}
		}
	}
}

// drawManualInput shows manual IP input field
func drawManualInput(manualIP *string, selected *bool, chosenChar *entity.CharacterType) {
	rl.DrawText("Enter Host IP Address:", 80, 80, 20, rl.Black)
	rl.DrawText("Port 9000 will be used automatically", 80, 105, 16, rl.DarkGray)

	// IP input field
	ipRect := rl.NewRectangle(80, 140, 300, 30)
	rl.DrawRectangleRec(ipRect, rl.White)
	rl.DrawRectangleLinesEx(ipRect, 2, rl.Black)

	// Show entered IP with blinking cursor
	displayText := *manualIP
	if (time.Now().UnixNano()/500000000)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, int32(ipRect.X+5), int32(ipRect.Y+5), 20, rl.Black)

	// Capture keyboard input
	key := rl.GetCharPressed()
	for key > 0 {
		if key >= 32 && key <= 126 {
			*manualIP += string(rune(key))
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(*manualIP) > 0 {
		*manualIP = (*manualIP)[:len(*manualIP)-1]
	}

	// Buttons
	connectRect := rl.NewRectangle(80, 190, 100, 30)
	backRect := rl.NewRectangle(190, 190, 100, 30)

	rl.DrawRectangleRec(connectRect, rl.LightGray)
	rl.DrawRectangleRec(backRect, rl.LightGray)

	rl.DrawText("Connect", int32(connectRect.X+10), int32(connectRect.Y+5), 20, rl.Black)
	rl.DrawText("Back", int32(backRect.X+10), int32(backRect.Y+5), 20, rl.Black)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mouse, connectRect) {
			ip := *manualIP
			if ip == "" {
				ip = "127.0.0.1"
			}
			charType := ShowCharacterSelect()
			*chosenChar = charType
			connectToHost(ip + ":9000", string(charType))
			*selected = true
		} else if rl.CheckCollisionPointRec(mouse, backRect) {
			*manualIP = ""
		}
	}
}

// startHost initializes the host
func startHost(selected *bool, playerSpawn rl.Vector2, charType string) {
	log.Println("[Menu] Host selected")
	network.Role = "host"
	network.LocalPlayerID = generatePlayerID()
	color := entity.PresetColors[rand.Intn(len(entity.PresetColors))]
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

	network.CurrentHost.BroadcastStateUpdate()
	*selected = true
}

func connectToHost(addr string, charType string) {
	network.Role = "client"
	playerID := generatePlayerID()
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

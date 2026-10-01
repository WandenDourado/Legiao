package ui

import (
	"log"
	"strings"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// drawDiscoveryView searches the LAN and presents every discovered host as a
// selectable card. Network work starts once on entry; drawing stays local.
func drawDiscoveryView(selected *bool, screen *menuScreen, refreshTimer *int, scanning *bool, chosenChar *entity.CharacterType) {
	if !*scanning {
		*scanning = true
		network.ClearDiscoveredHosts()
		go network.StartDiscoveryListener()
		go network.StartTCPScan(9000)
		go network.StartQuerySender(9000)
		network.StartOnlineRoomPoll()
	}

	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	drawMenuBackdrop(sw, sh)
	drawMenuCentered("ENCONTRAR PARTIDA", sw/2, sh*0.205, int32(max(sh*0.032, Px(19))), rl.NewColor(239, 230, 192, 255))
	panel := rl.NewRectangle((sw-min(sw*0.68, Px(760)))/2, sh*0.285, min(sw*0.68, Px(760)), min(sh*0.61, Px(520)))
	drawMenuPanel(panel)

	hosts := discoveryEntries()
	listTop := panel.Y + panel.Height*0.18
	drawMenuCentered(discoveryTitle(), sw/2, panel.Y+panel.Height*0.075, int32(max(panel.Height*0.055, Px(15))), rl.NewColor(245, 218, 145, 255))
	drawMenuCentered(discoverySubtitle(), sw/2, panel.Y+panel.Height*0.125, int32(max(panel.Height*0.032, Px(11))), rl.Fade(rl.White, 0.66))

	hostRects := make([]rl.Rectangle, len(hosts))
	if len(hosts) == 0 {
		dots := strings.Repeat(".", 1+(*refreshTimer/30)%3)
		drawDiscoverySearching(sw, panel, dots)
	} else {
		rowH := max(panel.Height*0.14, Px(52))
		for i, host := range hosts {
			rect := rl.NewRectangle(panel.X+panel.Width*0.09, listTop+float32(i)*rowH*1.18, panel.Width*0.82, rowH)
			hostRects[i] = rect
			drawDiscoveredHost(rect, host.label, hit(rect, rl.GetMousePosition()))
		}
	}

	back := rl.NewRectangle(panel.X+panel.Width*0.31, panel.Y+panel.Height*0.84, panel.Width*0.38, max(panel.Height*0.1, Px(46)))
	drawCharacterAction(back, "VOLTAR", false, hit(back, rl.GetMousePosition()))

	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return
	}
	mouse := rl.GetMousePosition()
	for i, host := range hosts {
		if !hit(hostRects[i], mouse) || host.full {
			continue
		}
		charType, confirmed := ShowCharacterSelect()
		if confirmed {
			*chosenChar = charType
			connectToHost(host.addr, string(charType))
			*selected = true
		}
		return
	}
	if hit(back, mouse) {
		log.Println("[Menu] Leaving discovery view")
		network.StopDiscovery() // tambem para a lista online
		network.ClearDiscoveredHosts()
		*scanning = false
		*screen = screenMain
	}
}

func drawDiscoverySearching(sw float32, panel rl.Rectangle, dots string) {
	center := rl.NewVector2(sw/2, panel.Y+panel.Height*0.45)
	rl.DrawCircleLinesV(center, panel.Height*0.1, rl.NewColor(143, 192, 178, 255))
	rl.DrawCircleV(center, panel.Height*0.022, rl.NewColor(234, 211, 144, 255))
	drawMenuCentered("PROCURANDO GRUPOS"+dots, sw/2, panel.Y+panel.Height*0.59, int32(max(panel.Height*0.052, Px(15))), rl.NewColor(219, 235, 225, 255))
	drawMenuCentered("Verifique se o anfitriao esta na mesma rede Wi-Fi.", sw/2, panel.Y+panel.Height*0.66, int32(max(panel.Height*0.031, Px(11))), rl.Fade(rl.White, 0.65))
}

func drawDiscoveredHost(rect rl.Rectangle, label string, hovered bool) {
	fill := rl.NewColor(39, 70, 69, 255)
	if hovered {
		fill = rl.NewColor(57, 91, 86, 255)
	}
	rl.DrawRectangleRounded(rect, 0.07, 8, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.07, 8, 2, rl.NewColor(135, 184, 168, 255))
	rl.DrawCircleV(rl.NewVector2(rect.X+rect.Height*0.5, rect.Y+rect.Height/2), rect.Height*0.12, rl.NewColor(144, 207, 157, 255))
	size := int32(max(rect.Height*0.3, Px(15)))
	rl.DrawText(label, int32(rect.X+rect.Height*0.9), int32(rect.Y+(rect.Height-float32(size))/2), size, rl.NewColor(228, 239, 228, 255))
}

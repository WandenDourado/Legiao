package ui

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// drawMainMenu presents the local multiplayer entry point. The layout uses
// only screen-relative measurements, keeping its generous tap targets intact
// on both a landscape phone and a desktop window.
func drawMainMenu(selected *bool, screen *menuScreen, playerSpawn rl.Vector2, chosenChar *entity.CharacterType) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	drawMenuBackdrop(sw, sh)
	drawMenuCentered("UNAM-SE PARA DEFENDER O REINO", sw/2, sh*0.205, int32(max(sh*0.021, Px(13))), rl.Fade(rl.NewColor(239, 235, 211, 255), 0.8))

	panelW := min(sw*0.7, Px(760))
	// panelH cresceu de 0.56/440 para 0.66/540: com o terceiro botao
	// (Configuracoes) as proporcoes antigas nao sobravam espaco numa tela de
	// celular deitado sem os botoes se sobreporem (Passo 3).
	panelH := min(sh*0.66, Px(540))
	panel := rl.NewRectangle((sw-panelW)/2, sh*0.24, panelW, panelH)
	drawMenuPanel(panel)

	// Com o servidor online configurado entra um quarto botao, e os quatro
	// ficam um pouco mais baixos para caber no mesmo painel.
	online := network.OnlineConfigured()
	buttonW := panel.Width * 0.82
	buttonH := max(panel.Height*0.17, Px(54))
	top, step := panel.Y+panel.Height*0.20, float32(1.22)
	if online {
		buttonH = max(panel.Height*0.145, Px(50))
		top, step = panel.Y+panel.Height*0.13, 1.18
	}
	buttonX := panel.X + (panel.Width-buttonW)/2
	row := 0
	next := func() rl.Rectangle {
		r := rl.NewRectangle(buttonX, top+float32(row)*buttonH*step, buttonW, buttonH)
		row++
		return r
	}
	hostRect := next()
	onlineRect := rl.Rectangle{}
	if online {
		onlineRect = next()
	}
	joinRect := next()
	settingsRect := next()

	mouse := rl.GetMousePosition()
	hostHovered := rl.CheckCollisionPointRec(mouse, hostRect)
	onlineHovered := online && rl.CheckCollisionPointRec(mouse, onlineRect)
	joinHovered := rl.CheckCollisionPointRec(mouse, joinRect)
	settingsHovered := rl.CheckCollisionPointRec(mouse, settingsRect)
	drawMenuButton(hostRect, "CRIAR PARTIDA", "ABRA UMA SALA NA REDE LOCAL", true, hostHovered)
	joinDetail := "ENCONTRE UM GRUPO NO WI-FI"
	if online {
		drawMenuButton(onlineRect, "CRIAR PARTIDA ONLINE", "JOGUE COM AMIGOS PELA INTERNET", true, onlineHovered)
		joinDetail = "SALAS NO WI-FI OU ONLINE"
	}
	drawMenuButton(joinRect, "ENTRAR EM PARTIDA", joinDetail, false, joinHovered)
	drawMenuButton(settingsRect, "CONFIGURACOES", "AUDIO, DESEMPENHO E CONTROLES", false, settingsHovered)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if hostHovered {
			charType, confirmed := ShowCharacterSelect()
			if confirmed {
				*chosenChar = charType
				startHost(selected, playerSpawn, string(charType))
			}
		} else if onlineHovered {
			// Personagem primeiro, sala por ultimo: a sala so e aberta no
			// servidor no instante em que a partida vai comecar.
			charType, confirmed := ShowCharacterSelect()
			if confirmed {
				if room, ok := ShowOnlineRoomSetup(); ok {
					*chosenChar = charType
					startHost(selected, playerSpawn, string(charType))
					network.CurrentHost.ServeOnline(room)
				}
			}
		} else if joinHovered {
			*screen = screenJoin
		} else if settingsHovered {
			*screen = screenSettings
		}
	}
}

func drawMenuBackdrop(sw, sh float32) {
	top := rl.NewColor(14, 30, 35, 255)
	bottom := rl.NewColor(35, 58, 50, 255)
	for y := float32(0); y < sh; y += 4 {
		t := y / sh
		rl.DrawRectangle(0, int32(y), int32(sw), 4, lerpMenuColor(top, bottom, t))
	}

	// Sparse points make the otherwise flat backdrop feel like a night sky
	// without competing with the menu's labels.
	for x := float32(36); x < sw; x += 97 {
		for y := float32(28); y < sh; y += 83 {
			rl.DrawCircle(int32(x+(y/83)*13), int32(y), 1.3, rl.Fade(rl.NewColor(222, 202, 138, 255), 0.18))
		}
	}

	titleSize := int32(max(sh*0.075, Px(34)))
	drawMenuCentered("LEGIAO", sw/2, sh*0.07, titleSize, rl.NewColor(235, 208, 135, 255))
	lineY := int32(sh * 0.175)
	rl.DrawLineEx(rl.NewVector2(sw*0.29, float32(lineY)), rl.NewVector2(sw*0.44, float32(lineY)), 1, rl.Fade(rl.Gold, 0.58))
	rl.DrawLineEx(rl.NewVector2(sw*0.56, float32(lineY)), rl.NewVector2(sw*0.71, float32(lineY)), 1, rl.Fade(rl.Gold, 0.58))
	rl.DrawCircle(int32(sw/2), lineY, 3, rl.Gold)
}

func drawMenuPanel(panel rl.Rectangle) {
	shadow := panel
	shadow.X += 6
	shadow.Y += 8
	rl.DrawRectangleRounded(shadow, 0.04, 8, rl.Fade(rl.Black, 0.35))
	rl.DrawRectangleRounded(panel, 0.04, 8, rl.NewColor(24, 43, 43, 242))
	rl.DrawRectangleRoundedLinesEx(panel, 0.04, 8, 2, rl.NewColor(202, 170, 92, 255))

	// The four corner marks make the panel read as a crafted game element,
	// rather than a generic operating-system dialog.
	for _, point := range []rl.Vector2{
		rl.NewVector2(panel.X+18, panel.Y+18), rl.NewVector2(panel.X+panel.Width-18, panel.Y+18),
		rl.NewVector2(panel.X+18, panel.Y+panel.Height-18), rl.NewVector2(panel.X+panel.Width-18, panel.Y+panel.Height-18),
	} {
		rl.DrawCircleV(point, 3, rl.NewColor(216, 187, 105, 255))
	}
}

func drawMenuButton(rect rl.Rectangle, title, detail string, primary, hovered bool) {
	border := rl.NewColor(126, 163, 154, 255)
	fill := rl.NewColor(39, 70, 69, 255)
	accent := rl.NewColor(137, 191, 174, 255)
	if primary {
		border = rl.NewColor(224, 187, 97, 255)
		fill = rl.NewColor(91, 70, 37, 255)
		accent = rl.NewColor(246, 216, 142, 255)
	}
	if hovered {
		fill = rl.NewColor(uint8(min(int(fill.R)+20, 255)), uint8(min(int(fill.G)+20, 255)), uint8(min(int(fill.B)+16, 255)), 255)
	}

	rl.DrawRectangleRounded(rect, 0.08, 8, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.08, 8, 2, border)
	iconCenter := rl.NewVector2(rect.X+rect.Height*0.58, rect.Y+rect.Height/2)
	drawMenuButtonIcon(iconCenter, rect.Height*0.23, accent, primary)

	titleSize := int32(max(rect.Height*0.29, Px(18)))
	detailSize := int32(max(rect.Height*0.15, Px(11)))
	textX := rect.X + rect.Height*1.08
	rl.DrawText(title, int32(textX), int32(rect.Y+rect.Height*0.2), titleSize, accent)
	rl.DrawText(detail, int32(textX), int32(rect.Y+rect.Height*0.61), detailSize, rl.Fade(rl.White, 0.72))

	arrowX := rect.X + rect.Width - rect.Height*0.42
	rl.DrawTriangle(rl.NewVector2(arrowX+rect.Height*0.12, rect.Y+rect.Height/2), rl.NewVector2(arrowX, rect.Y+rect.Height*0.36), rl.NewVector2(arrowX, rect.Y+rect.Height*0.64), accent)
}

func drawMenuButtonIcon(center rl.Vector2, radius float32, color rl.Color, primary bool) {
	rl.DrawCircleV(center, radius, rl.Fade(color, 0.15))
	rl.DrawCircleLinesV(center, radius, color)
	if primary {
		rl.DrawLineEx(rl.NewVector2(center.X-radius*0.48, center.Y), rl.NewVector2(center.X+radius*0.48, center.Y), 2, color)
		rl.DrawLineEx(rl.NewVector2(center.X, center.Y-radius*0.48), rl.NewVector2(center.X, center.Y+radius*0.48), 2, color)
		return
	}
	rl.DrawCircleV(center, radius*0.16, color)
	for _, offset := range []rl.Vector2{{X: -0.48, Y: -0.36}, {X: 0.5, Y: -0.12}, {X: -0.26, Y: 0.52}} {
		rl.DrawCircleV(rl.NewVector2(center.X+offset.X*radius, center.Y+offset.Y*radius), radius*0.11, color)
	}
}

func drawMenuCentered(text string, centerX, y float32, size int32, color rl.Color) {
	rl.DrawText(text, int32(centerX)-rl.MeasureText(text, size)/2, int32(y), size, color)
}

func lerpMenuColor(from, to rl.Color, t float32) rl.Color {
	return rl.NewColor(
		uint8(float32(from.R)+(float32(to.R)-float32(from.R))*t),
		uint8(float32(from.G)+(float32(to.G)-float32(from.G))*t),
		uint8(float32(from.B)+(float32(to.B)-float32(from.B))*t),
		255,
	)
}

package ui

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawNetworkBadge e uma linha discreta logo abaixo da barra de vida:
//
//   - HOST de partida online: o codigo da sala ("SALA ONLINE #K7M2Q") ou, se a
//     ligacao com o servidor caiu, que a sala saiu do ar e o jogo esta
//     tentando voltar. Sem isso o host so descobriria pelos amigos.
//   - CLIENTE: a latencia ate o host ("PING 45 ms"), verde, amarela ou
//     vermelha. Da ao jogador o motivo de um atraso antes que ele ache que o
//     jogo travou.
func DrawNetworkBadge(sw float32) {
	size := max(FitTextSize(sw*0.013*hudScale), int32(max(Px(12), 10)))
	// Logo abaixo da barra de vida (hud.go), nos mesmos pixels nativos.
	x := int32(Px(10))
	y := int32(Px(60 + 20*hudScale + 8))

	if network.Role == "client" {
		ping := network.Ping()
		if ping <= 0 {
			return
		}
		ms := ping.Milliseconds()
		color := rl.NewColor(144, 207, 157, 255) // bom
		switch {
		case ms >= 150:
			color = rl.NewColor(235, 110, 90, 255) // ruim
		case ms >= 80:
			color = rl.NewColor(240, 200, 100, 255) // aceitavel
		}
		rl.DrawText(fmt.Sprintf("PING %d ms", ms), x, y, size, rl.Fade(color, 0.9))
		return
	}

	code, connected, active := network.OnlineHostStatus()
	if !active {
		return
	}
	if connected {
		rl.DrawText("SALA ONLINE #"+code, x, y, size, rl.Fade(rl.NewColor(144, 207, 157, 255), 0.9))
		return
	}
	// Piscar devagar chama atencao sem cobrir o jogo.
	alpha := float32(0.55 + 0.4*float32(int(rl.GetTime()*2)%2))
	rl.DrawText("SERVIDOR ONLINE FORA - RECONECTANDO...", x, y, size, rl.Fade(rl.NewColor(240, 170, 90, 255), alpha))
}

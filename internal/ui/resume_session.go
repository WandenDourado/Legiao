package ui

// Voltar a partida depois de um REINICIAR (29/09/2026).
//
// Pedido do Gui: trocar a resolucao no meio da partida e REINICIAR nao pode
// jogar o jogador no menu. Antes de o processo morrer, captureResume anota
// de que lado da partida ele estava (internal/resume); o processo novo, no
// primeiro ShowMenu, pula o menu e volta:
//
//   - CLIENTE: reconecta ao mesmo endereco com o MESMO PlayerID. O host
//     guardou o slot (ReconnectGrace) e devolve posicao, vida, recargas e o
//     mapa do grupo — e a reconexao comum de network/host_rejoin.go, so que
//     atravessando um processo novo. Volta exatamente onde estava.
//   - HOST (inclui o solo com bots): a partida morreu com o processo. Volta
//     no COMECO DA FASE em que estava — mesmo mapa, mesmo personagem e cor,
//     supremas daquela fase; monstros e ondas do zero, como o reinicio de
//     fase. Os convidados reconectam sozinhos: na rede local ao mesmo IP;
//     online ao mesmo CODIGO, que o host pede de volta ao relay com o
//     segredo da sala (relay.HostAs). Voltam como quem acabou de entrar
//     (no comeco da fase, com vida cheia), porque a partida e nova.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/resume"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// captureResume monta o bilhete da partida em curso. Falso fora de partida
// (menu inicial): ai o reinicio volta ao menu, como sempre.
func captureResume() (resume.Ticket, bool) {
	t := resume.Ticket{
		PlayerID:  network.LocalPlayerID,
		Color:     network.LocalPlayerColor,
		Character: network.LocalPlayerCharacter,
	}
	switch {
	case network.Role == "client" && network.ServerAddress != "":
		t.Role = resume.RoleClient
		t.Address = network.ServerAddress
	case network.Role == "host" && network.CurrentHost != nil:
		t.Role = resume.RoleHost
		t.MapPath = network.CurrentHost.StageMap()
		if code := network.OnlineRoomCode(); code != "" {
			t.OnlineRoom = network.OnlineRoomName()
			t.OnlineCode = code
			t.OnlineSecret = network.OnlineRoomSecret()
		}
	default:
		return resume.Ticket{}, false
	}
	return t, t.PlayerID != "" && t.Character != ""
}

// resumeSession consome o bilhete e sobe a sessao. Falso = abrir o menu.
func resumeSession(playerSpawn rl.Vector2) (entity.CharacterType, bool) {
	t, ok := resume.Take()
	if !ok {
		return "", false
	}
	adoptLocalPlayerID(t.PlayerID)
	char := entity.CharacterType(t.Character)

	switch t.Role {
	case resume.RoleClient:
		log.Printf("[Retomar] voltando como cliente a %s", t.Address)
		// Assincrono: o laco do jogo mostra "conectando" e, se o host nao
		// responder, a sessao acaba com o motivo no menu (client_connect.go).
		network.StartClientSession(t.Address, ensureLocalPlayerID(), t.Color, t.Character)
		return char, true

	case resume.RoleHost:
		log.Printf("[Retomar] voltando como host em %s", t.MapPath)
		drawResumeFrame("Voltando a partida...")
		startHostAs(playerSpawn, t.Character, t.Color)
		if t.OnlineRoom != "" {
			drawResumeFrame("Reabrindo a sala online...")
			if room, err := network.OpenOnlineRoomAs(t.OnlineRoom, t.OnlineCode, t.OnlineSecret); err == nil {
				network.CurrentHost.ServeOnline(room)
			} else {
				// A partida segue na rede local: perder a sala nao e motivo
				// para perder a volta.
				log.Printf("[Retomar] sala online nao reabriu: %v", err)
			}
		}
		// O mapa inicial ja esta carregado pelo game; a fase e aplicada no
		// primeiro quadro, pelo mesmo caminho de um portal (portal.go).
		if t.MapPath != "" {
			network.StartTravel(t.MapPath, "")
		}
		return char, true
	}
	return "", false
}

// drawResumeFrame desenha um quadro de espera enquanto algo bloqueia (abrir
// a sala online leva ate alguns segundos).
func drawResumeFrame(msg string) {
	sw, sh := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	rl.BeginDrawing()
	drawMenuBackdrop(sw, sh)
	size := int32(max(sh*0.04, Px(18)))
	w := rl.MeasureText(msg, size)
	rl.DrawText(msg, int32(sw/2)-w/2, int32(sh/2)-size/2, size, rl.NewColor(246, 216, 142, 255))
	rl.EndDrawing()
}

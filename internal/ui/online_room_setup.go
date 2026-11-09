package ui

import (
	"log"
	"strings"
	"unicode"

	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/settings"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// maxRoomName cabe inteiro num cartao da lista de salas no celular deitado.
const maxRoomName = 16

type openResult struct {
	room *network.OnlineRoom
	err  error
}

// ShowOnlineRoomSetup pede o nome da sala e a registra no relay. So devolve
// ok com a sala JA aberta no servidor: se ele nao responde, o jogador le isso
// aqui, em vez de hospedar uma partida que ninguem de fora enxerga.
//
// O nome pode repetir: quem distingue as salas e o codigo que o relay gera
// (mostrado na lista como #K7M2Q).
func ShowOnlineRoomSetup() (*network.OnlineRoom, bool) {
	name := defaultRoomName()
	var status string
	var pending chan openResult

	for !rl.WindowShouldClose() {
		if pending != nil {
			select {
			case res := <-pending:
				pending = nil
				if res.err == nil {
					return res.room, true
				}
				log.Printf("[Menu] sala online nao abriu: %v", res.err)
				status = network.OnlineErrorText(res.err)
			default:
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		create, back, key := drawOnlineRoomSetup(name, status, pending != nil)
		rl.EndDrawing()

		if pending != nil {
			continue // registrando: nada muda ate o servidor responder
		}
		name = applyRoomKeys(name, key)
		if back || rl.IsKeyPressed(rl.KeyEscape) {
			return nil, false
		}
		if create || rl.IsKeyPressed(rl.KeyEnter) {
			if strings.TrimSpace(name) == "" {
				status = "Digite um nome para a sala."
				continue
			}
			status = ""
			pending = make(chan openResult, 1)
			go func(n string, out chan openResult) {
				room, err := network.OpenOnlineRoom(n)
				out <- openResult{room, err}
			}(strings.TrimSpace(name), pending)
		}
	}
	return nil, false
}

// applyRoomKeys aplica a tecla da tela e o que veio do teclado fisico.
func applyRoomKeys(name string, key rune) string {
	typed := []rune{}
	if key != keyNone {
		typed = append(typed, key)
	}
	for ch := rl.GetCharPressed(); ch != 0; ch = rl.GetCharPressed() {
		typed = append(typed, ch)
	}
	if rl.IsKeyPressed(rl.KeyBackspace) {
		typed = append(typed, keyBackspace)
	}
	runes := []rune(name)
	for _, ch := range typed {
		switch ch = unicode.ToUpper(ch); {
		case ch == keyBackspace:
			if len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
		case len(runes) >= maxRoomName:
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			runes = append(runes, ch)
		case ch == ' ' && len(runes) > 0 && runes[len(runes)-1] != ' ':
			runes = append(runes, ch)
		}
	}
	return string(runes)
}

// defaultRoomName parte do nome do jogador (Configuracoes > Jogador).
func defaultRoomName() string {
	var b strings.Builder
	for _, ch := range strings.ToUpper(settings.Get().PlayerName) {
		if ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == ' ' {
			b.WriteRune(ch)
		}
	}
	player := strings.TrimSpace(b.String())
	if player == "" {
		return "SALA DO LEGIAO"
	}
	name := "SALA DE " + player
	if len(name) > maxRoomName {
		name = strings.TrimSpace(name[:maxRoomName])
	}
	return name
}

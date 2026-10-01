package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/WandenDourado/Legiao/internal/network"
)

// discoveryEntry e um cartao da tela de entrar: o endereco que vai para
// network.StartClientSession e o texto que o jogador le.
type discoveryEntry struct {
	addr, label string
	full        bool // sala online cheia: aparece, mas nao responde ao toque
}

// discoveryEntries junta as salas online (relay) e as da rede local. As
// online vem primeiro: sao as que o jogador fora de casa procura, e a LAN
// continua logo abaixo quando todos estao no mesmo Wi-Fi.
//
// Um host em casa aparece nas DUAS listas para quem esta na mesma rede; os
// dois cartoes levam a mesma partida, e o da LAN e o caminho mais curto.
func discoveryEntries() []discoveryEntry {
	var out []discoveryEntry
	rooms, _ := network.GetOnlineRooms()
	// Salas com vaga primeiro, e entre elas as mais cheias (onde ja tem gente
	// para jogar); o servidor devolve em ordem aleatoria.
	sort.SliceStable(rooms, func(i, j int) bool {
		fi, fj := rooms[i].Players >= rooms[i].Max, rooms[j].Players >= rooms[j].Max
		if fi != fj {
			return !fi
		}
		if rooms[i].Players != rooms[j].Players {
			return rooms[i].Players > rooms[j].Players
		}
		return rooms[i].Code < rooms[j].Code // ordem estavel entre consultas
	})
	for _, r := range rooms {
		full := r.Players >= r.Max
		seats := fmt.Sprintf("%d/%d", r.Players+1, r.Max+1) // +1: o host
		if full {
			seats = "CHEIA"
		}
		out = append(out, discoveryEntry{
			addr:  network.OnlineRoomAddr(r.Code),
			label: fmt.Sprintf("ONLINE  %s  #%s  %s", strings.ToUpper(r.Name), r.Code, seats),
			full:  full,
		})
	}
	for _, h := range network.GetDiscoveredHosts() {
		out = append(out, discoveryEntry{addr: h, label: fmt.Sprintf("REDE LOCAL  %s", h)})
	}
	return out
}

// discoverySubtitle conta o estado do servidor online. Fora do ar, o jogador
// precisa saber que o problema e o servidor, e que a LAN continua valendo.
func discoverySubtitle() string {
	_, status := network.GetOnlineRooms()
	switch status {
	case network.OnlineChecking:
		return "Consultando o servidor online..."
	case network.OnlineDisabled:
		return "Jogo online desativado no momento - a rede local continua funcionando"
	case network.OnlineUnreachable:
		return "Servidor online indisponivel - a rede local continua funcionando"
	case network.OnlineReachable:
		return "Salas online e da sua rede Wi-Fi"
	}
	return "A porta 9000 e usada automaticamente"
}

func discoveryTitle() string {
	if network.OnlineConfigured() {
		return "PARTIDAS ONLINE E NA REDE LOCAL"
	}
	return "PARTIDAS NA REDE LOCAL"
}

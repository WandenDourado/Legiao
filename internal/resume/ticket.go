// Package resume guarda, atravessando um reinicio do app, a partida a que o
// jogador tem de voltar (29/09/2026).
//
// Quem grava e o REINICIAR AGORA de Configuracoes > Graficos
// (ui/render_restart.go), no instante antes de o processo morrer; quem le e
// o menu inicial do processo novo (ui/resume_session.go), que pula direto
// para a partida. O arquivo e de uso UNICO: Take apaga ao ler, entao uma
// volta que falhe nao vira um laco de voltas a cada abertura.
//
// Nao passa pelo settings.json de proposito: preferencia e para sempre, isto
// vale alguns minutos.
package resume

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/WandenDourado/Legiao/internal/settings"
)

// MaxAge e ate quando a volta vale. Abaixo dos 5 minutos em que o host
// guarda o slot de quem caiu (network.ReconnectGrace): depois disso um
// cliente ja nao teria o que retomar, e entrar direto numa partida que
// talvez nem exista mais seria pior que o menu.
const MaxAge = 4 * time.Minute

// Role diz de que lado da partida o jogador estava.
type Role string

const (
	RoleHost   Role = "host"
	RoleClient Role = "client"
)

// Ticket e tudo o que o processo novo precisa para voltar.
type Ticket struct {
	Role      Role   `json:"role"`
	PlayerID  string `json:"player_id"`
	Color     string `json:"color"`
	Character string `json:"character"`
	// Address: cliente — o endereco do host (IP:porta ou sala online), o
	// mesmo que a reconexao usaria.
	Address string `json:"address,omitempty"`
	// MapPath: host — a fase a reabrir, desde o comeco.
	MapPath string `json:"map_path,omitempty"`
	// OnlineRoom: host — o NOME da sala online a reabrir; "" = so rede local.
	// OnlineCode/OnlineSecret pedem ao relay o MESMO codigo de volta, e e por
	// ele que os amigos reconectam sozinhos.
	OnlineRoom   string    `json:"online_room,omitempty"`
	OnlineCode   string    `json:"online_code,omitempty"`
	OnlineSecret string    `json:"online_secret,omitempty"`
	SavedAt      time.Time `json:"saved_at"`
}

func path() string {
	dir := settings.DataDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "resume.json")
}

// Save grava t, carimbando a hora. No-op sem pasta gravavel.
func Save(t Ticket) error {
	p := path()
	if p == "" {
		return nil
	}
	t.SavedAt = time.Now()
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Take le e APAGA o bilhete. Falso quando nao ha, quando esta corrompido ou
// quando passou de MaxAge — nos tres casos o jogo abre no menu, como sempre.
func Take() (Ticket, bool) {
	return take(time.Now())
}

func take(now time.Time) (Ticket, bool) {
	p := path()
	if p == "" {
		return Ticket{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Ticket{}, false
	}
	_ = os.Remove(p)
	var t Ticket
	if json.Unmarshal(data, &t) != nil {
		return Ticket{}, false
	}
	return t, t.valid(now)
}

func (t Ticket) valid(now time.Time) bool {
	age := now.Sub(t.SavedAt)
	if age < 0 || age > MaxAge || t.PlayerID == "" || t.Character == "" {
		return false
	}
	switch t.Role {
	case RoleClient:
		return t.Address != ""
	case RoleHost:
		return true
	}
	return false
}

package resume

import (
	"testing"
	"time"
)

func TestTicketValid(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	base := Ticket{PlayerID: "player_1", Character: "mago", SavedAt: now.Add(-time.Minute)}

	cases := []struct {
		name string
		edit func(*Ticket)
		want bool
	}{
		{"host sem mapa ainda vale", func(t *Ticket) { t.Role = RoleHost }, true},
		{"cliente com endereco", func(t *Ticket) { t.Role = RoleClient; t.Address = "192.168.0.10:9000" }, true},
		{"cliente sem endereco", func(t *Ticket) { t.Role = RoleClient }, false},
		{"papel desconhecido", func(t *Ticket) { t.Role = "x" }, false},
		{"velho demais", func(t *Ticket) { t.Role = RoleHost; t.SavedAt = now.Add(-MaxAge - time.Second) }, false},
		{"do futuro", func(t *Ticket) { t.Role = RoleHost; t.SavedAt = now.Add(time.Minute) }, false},
		{"sem id", func(t *Ticket) { t.Role = RoleHost; t.PlayerID = "" }, false},
	}
	for _, c := range cases {
		tk := base
		c.edit(&tk)
		if got := tk.valid(now); got != c.want {
			t.Errorf("%s: valid = %v, want %v", c.name, got, c.want)
		}
	}
}

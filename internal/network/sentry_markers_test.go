package network

// A seta de borda de tela so aparece para gargula ACORDADA (28/09/2026): no
// mapa 4 as torres estao em campo desde a chegada, caladas ate o grupo chegar
// ao saguao ("As sentinelas despertaram").

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

func TestSentryMarkersWaitForTheWake(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		"p1": {PlayerID: "p1", X: 500, Y: 2500}, // vestibulo, degrau 1
	})
	h.Skills = skill.NewManager()
	h.EntityManager = entity.NewEntityManager()
	h.EntityManager.AddEnemy(entity.NewEnemy(entity.EnemyTypeCastleSentry, 800, 400))

	prevRole, prevHost := Role, CurrentHost
	Role, CurrentHost = "host", h
	t.Cleanup(func() {
		Role, CurrentHost = prevRole, prevHost
		setSentriesAwake(false)
		skill.ResetSentryOrbs()
	})

	h.handleSentryOrbTick(0, nil)
	if SentriesAwake() {
		t.Fatal("a porta foi publicada aberta com o grupo no vestibulo")
	}
	if got := LivingSentryCenters(nil); len(got) != 0 {
		t.Fatalf("gargula calada ganhou %d seta(s)", len(got))
	}

	h.players["p1"].X, h.players["p1"].Y = 500, 500 // saguao, degrau 3
	h.handleSentryOrbTick(0, nil)
	if !SentriesAwake() {
		t.Fatal("o grupo chegou ao saguao e a porta nao foi publicada aberta")
	}
	if got := LivingSentryCenters(nil); len(got) != 1 {
		t.Fatalf("gargula acordada deveria ganhar 1 seta, ganhou %d", len(got))
	}
}

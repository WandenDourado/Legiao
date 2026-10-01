package network

// Fase B2 (doc/plan_avanco_bots_e_gargula.md §B4): the sentry now enters
// buildBotView's Foes tagged IsSentry with the real hit point, and does not
// inflate EnemiesLeft (a fixed post is not the horde).

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

func TestBuildBotViewIncludesSentryTaggedAndExcludesItFromEnemiesLeft(t *testing.T) {
	em := entity.NewEntityManager()
	sentry := entity.NewEnemy(entity.EnemyTypeCastleSentry, 900, 900)
	em.AddEnemy(sentry)
	orc := entity.NewEnemy(entity.EnemyTypeBasic, 100, 100)
	em.AddEnemy(orc)

	h := &Host{
		EntityManager: em,
		players: map[string]*PlayerState{
			"bot_arqueiro": {PlayerID: "bot_arqueiro", X: 0, Y: 0, Character: "arqueiro"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
		},
	}

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}

	var foundSentry bool
	for _, f := range v.Foes {
		if f.ID == sentry.ID {
			foundSentry = true
			if !f.IsSentry {
				t.Error("expected the castle sentry foe to be tagged IsSentry")
			}
			if f.HitCentre != sentry.HitCenter() {
				t.Errorf("HitCentre = %+v, want %+v (entity.HitCenter())", f.HitCentre, sentry.HitCenter())
			}
		}
	}
	if !foundSentry {
		t.Fatal("expected the sentry to appear in Foes at all (it used to be filtered out entirely)")
	}

	if v.EnemiesLeft != 1 {
		t.Fatalf("EnemiesLeft = %d, want 1 (only the orc; the sentry is a fixed post, not the horde)", v.EnemiesLeft)
	}
}

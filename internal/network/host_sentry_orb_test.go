package network

// Fase B1 (doc/plan_avanco_bots_e_gargula.md §B2): global range only works if
// the orb's TTL scales with the actual flight distance, and if only one orb
// per sentry is ever in flight.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestSentryOrbTTLGrowsWithDistanceAndCaps(t *testing.T) {
	near := sentryOrbTTLFor(rl.NewVector2(0, 0), rl.NewVector2(300, 0))
	far := sentryOrbTTLFor(rl.NewVector2(0, 0), rl.NewVector2(6000, 0))
	if far <= near {
		t.Fatalf("expected a farther shot to get more TTL: near=%.2f far=%.2f", near, far)
	}
	// A shot from one corner of the biggest map to the other must not
	// exceed the cap.
	extreme := sentryOrbTTLFor(rl.NewVector2(0, 0), rl.NewVector2(16000, 16000))
	if extreme > skill.SentryOrbMaxTTL {
		t.Fatalf("expected TTL capped at %.1f, got %.2f", skill.SentryOrbMaxTTL, extreme)
	}
}

func TestSentryOrbTTLReaches6000PxFlight(t *testing.T) {
	// A shot 6000px away must live long enough to actually cover that
	// distance at SentryOrbSpeed, with slack for the turn-limited chase.
	ttl := sentryOrbTTLFor(rl.NewVector2(0, 0), rl.NewVector2(6000, 0))
	straightFlightTime := float32(6000) / skill.SentryOrbSpeed
	if ttl <= straightFlightTime {
		t.Fatalf("TTL %.2f is not enough to cover a straight 6000px flight (%.2fs)", ttl, straightFlightTime)
	}
}

func TestSentryHasLiveOrbGatesASecondCast(t *testing.T) {
	skill.ResetSentryOrbs()
	defer skill.ResetSentryOrbs()

	if skill.SentryHasLiveOrb(true, "sentry_1") {
		t.Fatal("expected no live orb before any cast")
	}
	skill.SpawnSentryOrb(true, "", "sentry_1", "target", rl.NewVector2(0, 0), rl.NewVector2(500, 0), 5)
	if !skill.SentryHasLiveOrb(true, "sentry_1") {
		t.Fatal("expected a live orb right after casting")
	}
	if skill.SentryHasLiveOrb(true, "sentry_2") {
		t.Fatal("a different sentry's orb must not count as this one's")
	}
}

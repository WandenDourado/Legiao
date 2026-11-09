package bot

// Fase A3 (doc/plan_avanco_bots_e_gargula.md §A4): retreat needs hysteresis
// — enter low, only rejoin well above — or a bot flickers combat/retreat
// every tick a single hit crosses one exact threshold.

import "testing"

func TestRetreatHysteresisEntersAndOnlyRejoinsAboveTheHigherBar(t *testing.T) {
	var retreating bool

	if retreatHysteresis(&retreating, 0.50, retreatUnder, rejoinAbove) {
		t.Fatal("expected no retreat at 50% health, above retreatUnder")
	}
	if !retreatHysteresis(&retreating, 0.30, retreatUnder, rejoinAbove) {
		t.Fatal("expected retreat to start below retreatUnder (0.35)")
	}
	// Healed back up to 50% — still below rejoinAbove (0.60), must still be
	// retreating, or a single heal tick would flip it straight back into
	// combat.
	if !retreatHysteresis(&retreating, 0.50, retreatUnder, rejoinAbove) {
		t.Fatal("expected retreat to persist below rejoinAbove even after climbing above retreatUnder")
	}
	if retreatHysteresis(&retreating, 0.65, retreatUnder, rejoinAbove) {
		t.Fatal("expected retreat to end once health climbs above rejoinAbove (0.60)")
	}
}

func TestPaladinaRetreatHysteresisRequiresShieldAlreadySpent(t *testing.T) {
	var retreating bool

	// Low health but Shield still available: she must try to mitigate
	// first, not retreat immediately.
	if paladinaRetreatHysteresis(&retreating, 0.10, true) {
		t.Fatal("expected no retreat while Shield is still ready to cast")
	}
	if !paladinaRetreatHysteresis(&retreating, 0.10, false) {
		t.Fatal("expected retreat once Shield is unavailable and health is below paladinaRetreatUnder")
	}
}

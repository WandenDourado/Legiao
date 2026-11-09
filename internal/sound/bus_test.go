package sound

import "testing"

// resetBusesToDefault restores the four buses to Default()'s numbers
// (settings.Default() mirrors these) so a test that changes them never
// leaks state into a test that runs after it.
func resetBusesToDefault(t *testing.T) {
	t.Helper()
	SetMasterVolume(1.0)
	SetMuteMaster(false)
	SetMusicBusVolume(0.5)
	SetMuteMusic(false)
	SetSkillVolume(0.85)
	SetMuteSkill(false)
	SetAttackVolume(0.85)
	SetMuteAttack(false)
}

// TestFinalShotVolumeIsDistanceTimesCategoryTimesMaster is the contract the
// whole bus system exists for: what a listener actually hears is distance
// attenuation times the effect's own category bus times the master bus.
func TestFinalShotVolumeIsDistanceTimesCategoryTimesMaster(t *testing.T) {
	resetBusesToDefault(t)
	defer resetBusesToDefault(t)

	SetMasterVolume(0.5)
	SetSkillVolume(0.4)

	dist := attenuation(SFXFullRadius) // full volume up to this radius
	if dist != 1 {
		t.Fatalf("attenuation at SFXFullRadius = %v, want 1", dist)
	}
	got := dist * busVolume(busSkill)
	want := float32(1) * (0.4 * 0.5)
	if got != want {
		t.Errorf("final volume = %v, want distance(1) x category(0.4) x master(0.5) = %v", got, want)
	}

	// Half-way through the falloff band the distance factor is neither 0
	// nor 1, and it still has to multiply the same category x master term.
	mid := SFXFullRadius + (SFXHearRadius-SFXFullRadius)/2
	distMid := attenuation(mid)
	if distMid <= 0 || distMid >= 1 {
		t.Fatalf("attenuation midway through the falloff band = %v, want strictly between 0 and 1", distMid)
	}
	gotMid := distMid * busVolume(busSkill)
	wantMid := distMid * (0.4 * 0.5)
	if gotMid != wantMid {
		t.Errorf("final volume midway = %v, want %v", gotMid, wantMid)
	}
}

// TestMasterZeroSilencesEveryBus is the explicit requirement: master at
// zero (or muted) has to reach the skill bus, the attack bus AND the music
// bus, whatever their own individual volumes are set to.
func TestMasterZeroSilencesEveryBus(t *testing.T) {
	resetBusesToDefault(t)
	defer resetBusesToDefault(t)

	SetSkillVolume(1)
	SetAttackVolume(1)
	SetMusicBusVolume(1)

	SetMasterVolume(0)
	if got := busVolume(busSkill); got != 0 {
		t.Errorf("busVolume(busSkill) with master=0 = %v, want 0", got)
	}
	if got := busVolume(busAttack); got != 0 {
		t.Errorf("busVolume(busAttack) with master=0 = %v, want 0", got)
	}
	if got := effectiveMusicVolume(); got != 0 {
		t.Errorf("effectiveMusicVolume() with master=0 = %v, want 0", got)
	}

	// Muting master (instead of setting the number to zero) has to reach
	// exactly as far — that is the whole point of a mute switch separate
	// from the slider.
	SetMasterVolume(1)
	SetMuteMaster(true)
	if got := busVolume(busSkill); got != 0 {
		t.Errorf("busVolume(busSkill) with MuteMaster=true = %v, want 0", got)
	}
	if got := effectiveMusicVolume(); got != 0 {
		t.Errorf("effectiveMusicVolume() with MuteMaster=true = %v, want 0", got)
	}
}

// TestMuteBusPreservesItsOwnVolumeNumber: muting a single bus silences it
// without discarding the slider's value, so unmuting must bring the exact
// same volume back.
func TestMuteBusPreservesItsOwnVolumeNumber(t *testing.T) {
	resetBusesToDefault(t)
	defer resetBusesToDefault(t)

	SetSkillVolume(0.73)
	before := busVolume(busSkill)
	if before <= 0 {
		t.Fatalf("busVolume(busSkill) before mute = %v, want > 0", before)
	}

	SetMuteSkill(true)
	if got := busVolume(busSkill); got != 0 {
		t.Errorf("busVolume(busSkill) muted = %v, want 0", got)
	}
	if got := SkillVolume(); got != 0.73 {
		t.Errorf("SkillVolume() while muted = %v, want the number unchanged at 0.73", got)
	}

	SetMuteSkill(false)
	if got := busVolume(busSkill); got != before {
		t.Errorf("busVolume(busSkill) after unmute = %v, want back to %v", got, before)
	}
}

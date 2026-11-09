package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIsAlreadyValid(t *testing.T) {
	d := Default()
	if c := d.clamped(); c != d {
		t.Fatalf("Default() is not stable under clamp: got %+v, want %+v", c, d)
	}
}

func TestClampOutOfRangeValues(t *testing.T) {
	s := Settings{
		MasterVolume: 4.0,
		MusicVolume:  -1.0,
		SkillVolume:  0.5,
		AttackVolume: 0.5,
		TargetFPS:    45, // not one of the three allowed values
		ActionButtonScale:   99,
		ControlsOpacity:     -5,
		JoystickRadiusScale: 0,
		HUDScale:            10,
		PlayerName:          "esteNomeAquiTemMaisDeVinteCaracteresDeProposito",
	}
	c := s.clamped()

	if c.MasterVolume != 1.0 {
		t.Errorf("MasterVolume = %v, want clamped to 1.0", c.MasterVolume)
	}
	if c.MusicVolume != 0.0 {
		t.Errorf("MusicVolume = %v, want clamped to 0.0", c.MusicVolume)
	}
	if c.TargetFPS != FrameCap60 {
		t.Errorf("TargetFPS = %v, want fallback to FrameCap60 for an invalid value", c.TargetFPS)
	}
	if c.ActionButtonScale != 1.3 {
		t.Errorf("ActionButtonScale = %v, want clamped to 1.3", c.ActionButtonScale)
	}
	if c.ControlsOpacity != 0 {
		t.Errorf("ControlsOpacity = %v, want clamped to 0", c.ControlsOpacity)
	}
	if c.JoystickRadiusScale != 0.7 {
		t.Errorf("JoystickRadiusScale = %v, want clamped to 0.7", c.JoystickRadiusScale)
	}
	if c.HUDScale != HUDScaleMax {
		t.Errorf("HUDScale = %v, want clamped to %v", c.HUDScale, HUDScaleMax)
	}
	if got := len([]rune(c.PlayerName)); got > maxPlayerNameLen {
		t.Errorf("PlayerName length = %d, want <= %d", got, maxPlayerNameLen)
	}
}

func TestSetAndGetRoundTrip(t *testing.T) {
	want := Default()
	want.MasterVolume = 0.3
	want.PlayerName = "Testador"
	got := Set(want)
	if got != want {
		t.Fatalf("Set() returned %+v, want %+v", got, want)
	}
	if Get() != want {
		t.Fatalf("Get() = %+v, want %+v", Get(), want)
	}
	// Leave the package state clean for other tests in this file.
	Set(Default())
}

func TestUpdateAppliesClampAfterMutation(t *testing.T) {
	Set(Default())
	got := Update(func(s *Settings) { s.MasterVolume = 5.0 })
	if got.MasterVolume != 1.0 {
		t.Fatalf("Update() left MasterVolume = %v, want clamped to 1.0", got.MasterVolume)
	}
	Set(Default())
}

// TestSaveLoadRoundTrip only exercises the desktop store: on Android
// configPath() is documented to return "" (session-only), so a round trip
// through disk has nothing to assert there. Build-tagged tests would be the
// cleaner split, but this file is the one place all three settings_test
// concerns (defaults, clamp, persistence) already live under 150 lines.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := configPath()
	if path == "" {
		t.Skip("no writable config path on this platform (see store_android.go)")
	}

	// Load() writes into the shared package state, so this test can't run
	// in parallel with the others, and it must restore whatever was on disk
	// before it (or remove the file if there was none) when it's done.
	var original []byte
	hadOriginal := false
	if data, err := os.ReadFile(path); err == nil {
		original, hadOriginal = data, true
	}
	t.Cleanup(func() {
		if hadOriginal {
			_ = os.WriteFile(path, original, 0o644)
		} else {
			_ = os.Remove(path)
		}
		Set(Default())
	})

	want := Default()
	want.MasterVolume = 0.42
	want.PlayerName = "Ida-e-volta"
	Set(want)
	if err := Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	Set(Default()) // simulate a fresh process before Load()
	got := Load()
	if got != want {
		t.Fatalf("Load() after Save() = %+v, want %+v", got, want)
	}
}

func TestLoadFallsBackToDefaultOnCorruptJSON(t *testing.T) {
	path := configPath()
	if path == "" {
		t.Skip("no writable config path on this platform (see store_android.go)")
	}

	var original []byte
	hadOriginal := false
	if data, err := os.ReadFile(path); err == nil {
		original, hadOriginal = data, true
	}
	t.Cleanup(func() {
		if hadOriginal {
			_ = os.WriteFile(path, original, 0o644)
		} else {
			_ = os.Remove(path)
		}
		Set(Default())
	})

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	Set(Settings{}) // poison the in-memory state so a silent no-op would be caught
	got := Load()
	if got != Default() {
		t.Fatalf("Load() with corrupt JSON = %+v, want Default()", got)
	}
}

func TestLoadFallsBackToDefaultOnMissingFile(t *testing.T) {
	path := configPath()
	if path == "" {
		t.Skip("no writable config path on this platform (see store_android.go)")
	}
	_ = os.Remove(path)
	Set(Settings{})
	got := Load()
	if got != Default() {
		t.Fatalf("Load() with no file on disk = %+v, want Default()", got)
	}
}

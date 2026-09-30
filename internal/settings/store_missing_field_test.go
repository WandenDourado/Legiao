package settings

import (
	"os"
	"path/filepath"
	"testing"
)

// Um settings.json gravado antes de um campo existir nao o traz. O campo tem
// de chegar com o valor de FABRICA, nao com o zero do tipo — o caso real foi
// DeathEffect (28/09/2026): o zero de um bool e "desligado", e quem ja tinha
// salvo qualquer preferencia abriria o jogo sem a desintegracao dos monstros.
func TestLoadKeepsDefaultsForFieldsMissingFromOldFile(t *testing.T) {
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
		t.Fatal(err)
	}
	old := []byte(`{"MasterVolume": 0.3, "TargetFPS": 30}`)
	if err := os.WriteFile(path, old, 0o644); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if got.MasterVolume != 0.3 || got.TargetFPS != FrameCap30 {
		t.Fatalf("campos presentes no arquivo nao foram lidos: %+v", got)
	}
	if !got.DeathEffect || !got.ShowAimHint || got.MusicVolume != Default().MusicVolume {
		t.Fatalf("campos ausentes nao ficaram no padrao: %+v", got)
	}
}

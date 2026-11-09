//go:build android

package settings

import (
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// configPath no Android: `settings.json` no armazenamento PRIVADO do app
// (`ANativeActivity.internalDataPath`, algo como
// /data/user/0/br.wandendourado.legiao/files). Nenhuma permissao e necessaria
// e o arquivo some junto com o app.
//
// Ate 28/09/2026 isto devolvia "" e a configuracao valia so a sessao: nenhuma
// chamada do raylib-go para um diretorio gravavel tinha sido confirmada. Ela
// existe — `rl.HomeDir()` devolve exatamente `internalDataPath` no build
// Android (raylib-go v0.55.1, utils_android.go). E pode ser chamada ANTES de
// InitWindow, que e quando settings.Load roda: o raylib guarda o
// `android_app` na primeira linha de `android_main`, antes de chamar o Go.
// Sem isto os presets de qualidade grafica voltariam ao padrao a cada
// abertura, justo no aparelho que precisa deles.
func configPath() string {
	dir := rl.HomeDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "settings.json")
}

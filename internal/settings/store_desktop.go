//go:build !android

package settings

import (
	"os"
	"path/filepath"
)

// configPath no desktop: os.UserConfigDir()/Legiao/settings.json — o
// candidato natural (AppData\Roaming no Windows, ~/.config no Linux,
// ~/Library/Application Support no macOS). Devolve "" só quando o SO recusa
// dizer onde fica o diretório de configuração do usuário, o que Load()/Save()
// tratam como "sem caminho gravável" e não como erro.
func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "Legiao", "settings.json")
}

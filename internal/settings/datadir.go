package settings

import "path/filepath"

// DataDir e a pasta gravavel do app (a mesma do settings.json), para outros
// pacotes guardarem um cache pequeno - ex.: o endereco do servidor online
// (network/online_config.go). "" quando a plataforma nao da uma.
func DataDir() string {
	p := configPath()
	if p == "" {
		return ""
	}
	return filepath.Dir(p)
}

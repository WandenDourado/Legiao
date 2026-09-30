package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// configPath devolve o arquivo onde este processo grava e lê as preferências,
// ou "" quando a plataforma não tem um caminho gravável confiável — ver
// store_desktop.go e store_android.go. Implementado por plataforma (build
// tags), não aqui.

// Load lê o arquivo de configuração e o aplica como o estado atual, já
// clampado. QUALQUER falha — arquivo ausente, JSON corrompido, plataforma
// sem caminho gravável — cai em silêncio no padrão (Default()) em vez de
// travar o jogo: um arquivo de configuração quebrado não pode impedir
// alguém de jogar. Chamar uma vez, no início de game.Run.
func Load() Settings {
	path := configPath()
	if path == "" {
		return Set(Default())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Set(Default())
	}
	// Parte do PADRÃO, não do zero: um settings.json gravado antes de um
	// campo existir não o traz, e o zero de um bool novo é "desligado". Foi o
	// que motivou isto — sem esta linha, quem já tinha salvo qualquer
	// preferência abriria o jogo com a desintegração dos monstros desligada
	// sem nunca ter mexido nela.
	s := Default()
	if err := json.Unmarshal(data, &s); err != nil {
		return Set(Default())
	}
	return Set(s)
}

// Save grava o estado atual em disco. No-op documentado quando a plataforma
// não tem caminho gravável (ver store_android.go) — a configuração vale só a
// sessão nesse caso, e isso é dito em doc/settings.md em vez de fingido.
func Save() error {
	path := configPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Get(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

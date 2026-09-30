package network

// O folego do renascer (playtest de 27/09/2026, plans/plan_playtest_27_09.md
// §4). Quem levanta ganha entity.RespawnGrace segundos de imunidade e de passo
// mais rapido, para sair de onde caiu — que nas fases de horda e o meio dela.
//
// As duas metades moram em lugares diferentes pelo mesmo motivo de sempre:
//   - a IMUNIDADE e do host, porque dano e do host: GrantInvulnerability, e
//     todo caminho de dano ja pergunta IsInvulnerable;
//   - a VELOCIDADE e de quem move o corpo. Um humano move o proprio corpo na
//     maquina dele (local_buffs.go, LocalRespawnBoost); um bot e movido aqui,
//     no host, e por isso o relogio do bot fica neste arquivo.

import (
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
)

var (
	graceMu sync.Mutex
	// graceLeft e o relogio de velocidade dos BOTS. Humanos nao entram aqui:
	// o passo deles e decidido na maquina deles (LocalRespawnBoost).
	graceLeft = map[string]float32{}
)

// grantRespawnGrace e o que tickRespawns da a cada ressuscitado.
func (h *Host) grantRespawnGrace(playerID string) {
	h.GrantInvulnerability(playerID, entity.RespawnGrace)
	graceMu.Lock()
	graceLeft[playerID] = entity.RespawnGrace
	graceMu.Unlock()
}

// tickRespawnGrace envelhece o relogio de velocidade dos bots.
func tickRespawnGrace(dt float32) {
	graceMu.Lock()
	defer graceMu.Unlock()
	for id, left := range graceLeft {
		if left -= dt; left <= 0 {
			delete(graceLeft, id)
			continue
		}
		graceLeft[id] = left
	}
}

// respawnGraceMultiplier e o fator de passo de um bot recem-levantado.
func respawnGraceMultiplier(playerID string) float32 {
	graceMu.Lock()
	defer graceMu.Unlock()
	if graceLeft[playerID] > 0 {
		return entity.RespawnGraceSpeed
	}
	return 1
}

// clearRespawnGrace zera tudo. Chamado pelo reinicio de fase, ao lado de
// clearInvulnerability.
func clearRespawnGrace() {
	graceMu.Lock()
	defer graceMu.Unlock()
	graceLeft = map[string]float32{}
}

package network

// Generic, data-driven gating for registry-cast skills: character binding and
// per-(player, skill) cooldowns. Keeps HandleSkillMessage switch-free.

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
)

// characterHasSkill reports whether skillID is bound to the character type in
// the ability registry.
func characterHasSkill(char entity.CharacterType, skillID string) bool {
	for _, id := range ability.AbilitiesOf(char) {
		if id == skillID {
			return true
		}
	}
	return false
}

// skillUnlocked reports whether the player may cast this skill at all, before
// any cooldown is considered.
//
// The party starts the campaign without its ultimates and earns them by
// clearing phases, so a locked ultimate is refused at the source: the host is
// the only place that decides what may be cast, and a client that draws the
// button anyway still gets a no.
//
// Test mode (F2) unlocks everything for that player alone, which is what makes
// a later phase testable from the first map.
func (h *Host) skillUnlocked(playerID, skillID string, char entity.CharacterType) bool {
	if ability.UltimateAbilityOf(char) != skillID {
		return true // primary skills are never gated by progression
	}
	return UltimateUnlockedFor(char) || h.TestModeEnabled(playerID)
}

// beginSkillCooldown returns false if (playerID, skillID) is still cooling
// down; otherwise it consumes a cast and returns true. For skills that
// implement ability.Charged, the cooldown is only armed once the last charge
// is spent (e.g., the Arqueiro fires both celestial arrows, then waits).
func (h *Host) beginSkillCooldown(playerID, skillID string) bool {
	s := ability.Get(skillID)
	if s == nil {
		return true // unknown here; CastByID will report it
	}
	// Test mode ignores recharges entirely, charges included. Checked before
	// cdMutex is taken: TestModeEnabled has its own lock.
	if h.TestModeEnabled(playerID) {
		return true
	}
	cd := s.Cooldown()
	if cd <= 0 {
		return true
	}
	key := playerID + "|" + skillID
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	if h.skillCooldowns[key] > 0 {
		return false
	}
	if c, ok := s.(ability.Charged); ok && c.Charges() > 1 {
		h.skillCharges[key]++
		if h.skillCharges[key] >= c.Charges() {
			h.skillCharges[key] = 0
			h.skillCooldowns[key] = cd
		}
		return true
	}
	h.skillCooldowns[key] = cd
	return true
}

// ClearSkillCooldown makes one player's skill castable again, right now.
//
// Existe para o ultimo suspiro. A cena reergue o heroi inteiro e anuncia que a
// ultimate vem — mas ela nao vinha: a Chuva de Meteoros recarrega em 60 s, e o
// Mago que ja tinha usado a dele durante as quatro hordas era devolvido de pe,
// curado, invulneravel e SEM PODER LANCAR NADA. A cena inteira dependia de o
// jogador nao ter usado a propria ultimate, o que ninguem garante.
//
// Zera a carga junto: uma ultimate de multiplas cargas (as duas flechas do
// Arqueiro) tem de voltar com as duas, senao o resgate entrega meia.
func (h *Host) ClearSkillCooldown(playerID, skillID string) {
	key := playerID + "|" + skillID
	h.cdMutex.Lock()
	delete(h.skillCooldowns, key)
	delete(h.skillCharges, key)
	h.cdMutex.Unlock()
}

// tickSkillCooldowns counts every armed cooldown down and drops expired ones.
func (h *Host) tickSkillCooldowns(dt float32) {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	for key, cd := range h.skillCooldowns {
		cd -= dt
		if cd <= 0 {
			delete(h.skillCooldowns, key)
			continue
		}
		h.skillCooldowns[key] = cd
	}
}

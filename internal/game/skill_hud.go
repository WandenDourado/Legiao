package game

// Cooldown readout for the local player's abilities. The numbers themselves
// come from the host (network.LocalSkillCooldown); this file only decides how
// each slot is labelled and hands the result to the ui package.

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// slotKeys and slotAccents index the ability slots: 0 is the primary skill and
// 1 the ultimate. They match the on-screen buttons on Android (orange Q, gold
// R) so the two platforms describe the same ability the same way.
var (
	slotKeys    = []string{"Q", "R"}
	slotAccents = []rl.Color{rl.Orange, rl.Gold}
)

// skillNames maps the wire IDs used by the ability registry to the names the
// players actually use. Text is unaccented: raylib's default font has no
// glyphs for the accented letters, exactly as in the wave announcements.
var skillNames = map[string]string{
	"fireball":         "Bola de Fogo",
	"sanctuary":        "Santuario",
	"arrow_volley":     "Rajada de Flechas",
	"shield":           "Escudo Sagrado",
	"graveyard":        "Cemiterio",
	"meteor_rain":      "Chuva de Meteoros",
	"angelic_area":     "Area Angelical",
	"celestial_arrows": "Flechas Celestiais",
	"divine_avatar":    "Avatar dos Deuses",
	"spectral_legion":  "Legiao Espectral",
}

// SkillSlotCooldown returns the remaining and total cooldown of the idx-th
// ability bound to the character, both zero when the slot is empty.
func SkillSlotCooldown(char entity.CharacterType, idx int) (remaining, total float32) {
	skillID := ability.AbilityAt(char, idx)
	if skillID == "" {
		return 0, 0
	}
	s := ability.Get(skillID)
	if s == nil {
		return 0, 0
	}
	return network.LocalSkillCooldown(skillID), s.Cooldown()
}

// DrawSkillCooldownBar draws the desktop cooldown pips for every ability bound
// to the local player's character.
func DrawSkillCooldownBar(p *entity.Player, sw, sh float32) {
	ids := ability.AbilitiesOf(p.CharType)
	entries := make([]ui.CooldownEntry, 0, len(ids))
	for i, id := range ids {
		s := ability.Get(id)
		if s == nil {
			continue
		}
		// Suprema ainda nao ganha nao entra na barra. Mostrar um contador de
		// algo que o host vai recusar so ensina o jogador errado.
		if !abilityUsable(p.CharType, i) {
			continue
		}
		entries = append(entries, ui.CooldownEntry{
			Key:       slotKey(i),
			Label:     skillNames[id],
			Remaining: network.LocalSkillCooldown(id),
			Total:     s.Cooldown(),
			Accent:    slotAccent(i),
		})
	}
	ui.DrawSkillBar(sw, sh, entries)
}

// slotKey names the key that casts slot idx, falling back to the slot number
// for any ability bound beyond the two the input layer currently reads.
func slotKey(idx int) string {
	if idx >= 0 && idx < len(slotKeys) {
		return slotKeys[idx]
	}
	return string(rune('1' + idx))
}

// slotAccent tints slot idx, reusing the ultimate's gold for anything beyond.
func slotAccent(idx int) rl.Color {
	if idx >= 0 && idx < len(slotAccents) {
		return slotAccents[idx]
	}
	return rl.Gold
}

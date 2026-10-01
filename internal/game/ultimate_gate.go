package game

// A ultimate so existe para o jogador depois que ele a ganha.
//
// O host ja recusa o lancamento de uma suprema travada (Host.skillUnlocked),
// mas recusa em silencio: o botao continuava na tela e a tecla continuava
// respondendo com nada. Botao que nao faz nada le como bug, nao como bloqueio.
//
// Entao a mesma pergunta e feita em tres lugares, e de proposito:
//
//	no HUD      para o botao e o contador nao aparecerem
//	na entrada  para a tecla e o toque nao pedirem o que sera negado
//	no host     porque so ele decide, e um cliente adulterado nao pede licenca
//
// As duas primeiras sao conforto; a terceira e a regra.

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// UltimateAvailableFor reports whether THIS character may see and use their
// supreme skill: either the campaign has granted it, or F2 is on.
//
// A pergunta e por personagem porque a campanha entrega as supremas uma por
// fase, e nao todas de uma vez (ver game/progression.go). Enquanto era uma
// pergunta sem sujeito, o botao da Sacerdotisa acendia no mapa 3 — onde so o
// Necromante tem a dele — e o host recusava o lancamento em silencio.
func UltimateAvailableFor(char entity.CharacterType) bool {
	return network.UltimateUnlockedFor(char) || network.TestMode
}

// abilityUsable reports whether the idx-th ability of a character is available
// to the local player right now. Only the ultimate is ever gated.
func abilityUsable(char entity.CharacterType, idx int) bool {
	id := ability.AbilityAt(char, idx)
	if id == "" {
		return false
	}
	if id != ability.UltimateAbilityOf(char) {
		return true
	}
	return UltimateAvailableFor(char)
}

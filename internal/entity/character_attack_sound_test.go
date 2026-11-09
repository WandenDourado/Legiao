package entity

import "testing"

// TestEveryCharacterDeclaresAnAttackSound pega a falha silenciosa do
// ataque basico: AttackSound vazio nao da erro em lugar nenhum —
// PlayAttackSound devolve sem tocar e a classe simplesmente ataca em
// silencio, o que so aparece jogando com ela.
func TestEveryCharacterDeclaresAnAttackSound(t *testing.T) {
	defs := AllCharacters()
	if len(defs) == 0 {
		t.Fatal("registro vazio: o init() dos personagens nao rodou")
	}
	seen := map[string]string{}
	for _, d := range defs {
		if d.AttackSound == "" {
			t.Errorf("%s sem AttackSound: o ataque basico dela seria mudo", d.Name)
			continue
		}
		if other, dup := seen[string(d.AttackSound)]; dup {
			t.Errorf("%s e %s compartilham o efeito %q", d.Name, other, d.AttackSound)
		}
		seen[string(d.AttackSound)] = d.Name
	}
}

// TestEveryCharacterHasAnAttackCadence guarda o prazo do som: PlayAttackSound
// corta o efeito em AttackInterval(), e cadencia zero significaria "sem
// limite" — o arquivo tocaria inteiro e os golpes se empilhariam (o do
// Necromante tem 8 s para uma cadencia de 0,77 s).
func TestEveryCharacterHasAnAttackCadence(t *testing.T) {
	for _, d := range AllCharacters() {
		if d.AttackInterval() <= 0 {
			t.Errorf("%s sem cadencia de ataque: o som do golpe nao teria onde terminar", d.Name)
		}
	}
}

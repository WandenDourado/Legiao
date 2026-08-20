package ability

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// TestEveryRegisteredSkillDeclaresACooldown guards the gate that makes the HUD
// counter meaningful: Cooldown() == 0 means "castable every frame", which the
// host reads as "no gate at all". The Mago's fireball shipped that way and had
// no recharge in game.
func TestEveryRegisteredSkillDeclaresACooldown(t *testing.T) {
	if len(registry) == 0 {
		t.Fatal("registro vazio: o init() das habilidades nao rodou")
	}
	for id, s := range registry {
		if s.Cooldown() <= 0 {
			t.Errorf("habilidade %q sem cooldown: seria lancavel a cada quadro", id)
		}
	}
}

// TestEveryCharacterHasBothSlots checks that every playable character has a
// primary and an ultimate bound. An unbound slot is silently ignored by the
// input layer, so the button simply does nothing.
func TestEveryCharacterHasBothSlots(t *testing.T) {
	for _, def := range entity.AllCharacters() {
		if PrimaryAbilityOf(def.Type) == "" {
			t.Errorf("%s sem habilidade primaria (slot 0)", def.Name)
		}
		if UltimateAbilityOf(def.Type) == "" {
			t.Errorf("%s sem ultimate (slot 1)", def.Name)
		}
	}
}

// TestBoundAbilitiesExistInTheRegistry catches a BindAbility typo: the host
// would refuse the cast as an unknown skill and nothing would happen.
func TestBoundAbilitiesExistInTheRegistry(t *testing.T) {
	for _, def := range entity.AllCharacters() {
		for _, id := range AbilitiesOf(def.Type) {
			if Get(id) == nil {
				t.Errorf("%s: habilidade %q ligada mas nao registrada", def.Name, id)
			}
		}
	}
}

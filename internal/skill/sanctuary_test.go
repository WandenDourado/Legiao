package skill

// O que o Santuario passou a prometer em 31/08/2026, travado por teste: a aura
// ANDA com a Sacerdotisa, ela anda mais rapido enquanto a aura vive, e
// relancar renova em vez de empilhar.

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A cura mede a partir da ancora, e a ancora e a Sacerdotisa. Um alvo que
// estava fora do circulo entra nele quando ela caminha ate perto dele — se
// este teste cair, a aura voltou a ser um lugar no chao.
func TestSanctuaryFollowsItsOwner(t *testing.T) {
	m := NewManager()
	m.AddSanctuary(NewSanctuary("priest", rl.NewVector2(0, 0)), nil)

	ally := rl.NewVector2(SanctuaryRadius+80, 0)
	if m.Sanctuaries["priest"].Contains(ally) {
		t.Fatalf("aliado a %.0f px deveria estar fora do raio (%.0f)",
			SanctuaryRadius+80, SanctuaryRadius)
	}

	m.SetSanctuaryAnchor("priest", rl.NewVector2(100, 0))
	if !m.Sanctuaries["priest"].Contains(ally) {
		t.Error("a aura nao acompanhou a dona: o aliado continua fora depois do passo")
	}
}

// O bonus vale na janela de cura e so nela: o fade final e desenho, a
// habilidade ja acabou quando ele comeca.
func TestSanctuarySpeedBonusLastsAsLongAsTheHealing(t *testing.T) {
	m := NewManager()
	m.AddSanctuary(NewSanctuary("priest", rl.NewVector2(0, 0)), nil)

	want := 1 + SanctuarySpeedBonus
	if got := m.SanctuaryMoveMultiplier("priest"); got != want {
		t.Errorf("com a aura no ar: %v, esperado %v", got, want)
	}
	if got := m.SanctuaryMoveMultiplier("outro"); got != 1 {
		t.Errorf("quem nao lancou nada anda a %v, esperado 1", got)
	}

	// Envelhece a aura ate depois da janela de cura (ainda dentro do fade).
	m.Sanctuaries["priest"].Age = SanctuaryDuration + SanctuaryFade*0.5
	if got := m.SanctuaryMoveMultiplier("priest"); got != 1 {
		t.Errorf("no fade a Sacerdotisa ainda corre a %v, esperado 1", got)
	}
}

// Uma Sacerdotisa, uma aura: relancar reinicia o relogio no lugar de somar uma
// segunda area (o mesmo contrato do Escudo Sagrado).
func TestRecastingSanctuaryRenewsInsteadOfStacking(t *testing.T) {
	m := NewManager()
	SpawnSanctuary(m, "priest", rl.NewVector2(0, 0))
	m.Sanctuaries["priest"].Age = SanctuaryDuration - 1

	SpawnSanctuary(m, "priest", rl.NewVector2(300, 0))

	if len(m.Sanctuaries) != 1 {
		t.Fatalf("relancar criou %d auras, esperado 1", len(m.Sanctuaries))
	}
	if age := m.Sanctuaries["priest"].Age; age != 0 {
		t.Errorf("o relogio da aura nao reiniciou: idade %.2f", age)
	}
}

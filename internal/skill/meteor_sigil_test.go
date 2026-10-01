package skill

// O que o selo promete, travado por teste: ele dura exatamente a chuva, ele
// anda com o mago, e relancar renova em vez de empilhar.

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// O selo e a assinatura da chuva: se ele apagar antes, o jogador ve pedras
// caindo sem origem; se sobrar depois, ve uma origem sem pedras. As duas
// constantes tem de andar juntas — este teste e o que impede uma de ser
// mexida sozinha.
func TestMeteorSigilBurnsForTheWholeRain(t *testing.T) {
	m := NewManager()
	ActivateMeteorSigil(m, "mago", rl.NewVector2(0, 0))

	// Um quadro antes do fim da chuva o selo ainda esta na janela cheia.
	m.UpdateMeteorSigils(MeteorRainDuration - 0.05)
	if !m.MeteorSigils["mago"].Active() {
		t.Fatalf("o selo apagou antes do fim da chuva (%.1fs)", MeteorRainDuration)
	}
	if got := m.MeteorSigils["mago"].fade(); got != 1 {
		t.Errorf("durante a chuva o selo esta em %.2f de brilho, esperado 1", got)
	}

	// Passado o fim, ele entra no apagamento e sai de campo depois dele.
	m.UpdateMeteorSigils(0.1)
	if m.MeteorSigils["mago"].Active() {
		t.Error("o selo continuou aceso depois do fim da chuva")
	}
	m.UpdateMeteorSigils(meteorSigilFade + 1)
	if _, ok := m.MeteorSigils["mago"]; ok {
		t.Error("o selo nao foi recolhido depois do apagamento")
	}
}

// A ancora e o mago, nao o lugar onde ele estava: a chuva dura 15 segundos e
// ninguem passa 15 segundos parado sob ela.
func TestMeteorSigilFollowsItsOwner(t *testing.T) {
	m := NewManager()
	ActivateMeteorSigil(m, "mago", rl.NewVector2(0, 0))

	m.SetMeteorSigilAnchor("mago", rl.NewVector2(900, 400))
	if got := m.MeteorSigils["mago"].Position; got.X != 900 || got.Y != 400 {
		t.Errorf("o selo ficou em %v em vez de acompanhar o dono", got)
	}
	// A figura toca o chao, e o chao fica ABAIXO do centro do corpo (a sola,
	// nao o umbigo) — ver entity.GroundOffset.
	if c := m.MeteorSigils["mago"].center(); c.Y <= 400 {
		t.Errorf("o selo desenhou em %v, na altura do corpo em vez do pe", c)
	}
}

// Um mago, um selo.
func TestRecastingMeteorSigilRenewsInsteadOfStacking(t *testing.T) {
	m := NewManager()
	ActivateMeteorSigil(m, "mago", rl.NewVector2(0, 0))
	m.UpdateMeteorSigils(MeteorRainDuration - 1)

	ActivateMeteorSigil(m, "mago", rl.NewVector2(300, 0))
	if len(m.MeteorSigils) != 1 {
		t.Fatalf("relancar criou %d selos, esperado 1", len(m.MeteorSigils))
	}
	if got := m.MeteorSigils["mago"].Remaining; got != MeteorRainDuration {
		t.Errorf("o relogio do selo nao reiniciou: restam %.2f", got)
	}
}

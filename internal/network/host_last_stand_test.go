package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// A cena so tem valor se acontecer ANTES do Game Over, e se acontecer uma vez.
// Estes testes cobrem a maquina de estado que garante as duas coisas; o
// resgate em si (reviver, invocar) depende do host e e conferido em jogo.

func TestLastStandHoldsGameOverUntilResolved(t *testing.T) {
	ResetLastStand()
	if LastStandPending() {
		t.Fatal("a cena nasce armada; deveria comecar desarmada")
	}

	ArmLastStand()
	if !LastStandPending() {
		t.Error("armada, mas o Game Over nao seria segurado")
	}

	// Resolver e o que solta o Game Over. Simulado aqui porque a resolucao
	// completa precisa de um Host; o que importa para o gate e o par
	// armed/done, e ele e desta camada.
	lastStand.mu.Lock()
	lastStand.done, lastStand.armed = true, false
	lastStand.mu.Unlock()

	if LastStandPending() {
		t.Error("resolvida, mas ainda segurando o Game Over")
	}
	if !LastStandDone() {
		t.Error("resolvida, mas nao marcada como gasta")
	}
}

func TestLastStandCannotArmTwiceInOneRun(t *testing.T) {
	ResetLastStand()
	lastStand.mu.Lock()
	lastStand.done = true
	lastStand.mu.Unlock()

	// Um grupo que cai DE NOVO na mesma corrida nao ganha um segundo resgate,
	// senao a fase vira inperdivel: cada queda devolveria a legiao.
	ArmLastStand()
	if LastStandPending() {
		t.Error("a cena rearmou depois de ja ter sido gasta")
	}
}

func TestResetLastStandGivesTheSceneBack(t *testing.T) {
	ResetLastStand()
	lastStand.mu.Lock()
	lastStand.done, lastStand.armed = true, true
	lastStand.mu.Unlock()

	// Reiniciar a fase e uma corrida NOVA: quem perdeu tem direito a cena de
	// novo.
	ResetLastStand()
	if LastStandPending() || LastStandDone() {
		t.Error("o reset nao devolveu a cena")
	}
}

func TestInvulnerabilityWindowExpires(t *testing.T) {
	clearInvulnerability()
	h := &Host{}
	if h.IsInvulnerable("p1") {
		t.Fatal("invulneravel sem ninguem ter concedido")
	}

	h.GrantInvulnerability("p1", 2.0)
	if !h.IsInvulnerable("p1") {
		t.Error("a janela concedida nao valeu")
	}
	// Uma janela que nao expira e imunidade permanente, que e exatamente o que
	// a cena NAO pode deixar para tras.
	h.tickInvulnerability(1.0)
	if !h.IsInvulnerable("p1") {
		t.Error("a janela caiu na metade do tempo")
	}
	h.tickInvulnerability(1.5)
	if h.IsInvulnerable("p1") {
		t.Error("a janela nao expirou")
	}
}

func TestGrantInvulnerabilityKeepsTheLongerWindow(t *testing.T) {
	clearInvulnerability()
	h := &Host{}
	h.GrantInvulnerability("p1", 3.0)
	h.GrantInvulnerability("p1", 1.0)
	// Uma concessao curta nao pode ENCURTAR uma longa que ainda esta correndo:
	// duas cenas sobrepostas deixariam o jogador exposto no meio da primeira.
	h.tickInvulnerability(1.5)
	if !h.IsInvulnerable("p1") {
		t.Error("a concessao curta sobrescreveu a longa")
	}
	clearInvulnerability()
}

// O heroi do ultimo suspiro e da FASE. Este teste existe porque a versao
// anterior tinha o Necromante escrito no codigo em oito lugares, e a unica
// forma de descobrir que o mapa 3 usa a Sacerdotisa era ler todos eles.
func TestLastStandHeroIsPerMap(t *testing.T) {
	cases := []struct {
		mapPath string
		want    entity.CharacterType
		skill   string
	}{
		{"assets/maps/world_02.json", entity.CharNecromante, "spectral_legion"},
		{"assets/maps/world_03.json", entity.CharSacerdotisa, "angelic_area"},
		// Mapa fora da tabela (sandbox, validacao de terreno) cai no
		// Necromante: uma fase de teste que arme a cena precisa ter saida.
		{"assets/maps/sandbox.json", entity.CharNecromante, "spectral_legion"},
	}
	for _, tc := range cases {
		hero := LastStandHeroFor(tc.mapPath)
		if hero.character != tc.want || hero.skillID != tc.skill {
			t.Errorf("%s: heroi %q com %q; esperado %q com %q",
				tc.mapPath, hero.character, hero.skillID, tc.want, tc.skill)
		}
	}
}

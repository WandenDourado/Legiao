package dialogue

import "testing"

// O que estes testes protegem ja quebrou em jogo: depois de perder a fase e
// reiniciar, a cena do climax nao abria de novo — ela seguia marcada como
// tocada — e como e ela que segura o Game Over, o grupo caia e perdia na hora,
// sem resgate nenhum.

func TestPerRunTriggers(t *testing.T) {
	cases := []struct {
		trigger Trigger
		perRun  bool
		reason  string
	}{
		// A abertura e do MAPA: reiniciar a fase nao devolve o grupo a
		// floresta, e repetir a conversa de chegada a cada tentativa cansaria.
		{TriggerMapStart, false, "a abertura nao repete a cada tentativa"},
		// As duas abaixo falam sobre UMA luta. Quem perdeu vai lutar de novo.
		{TriggerLastStand, true, "o climax volta quando a fase recomeca"},
		{TriggerWavesCleared, true, "o fim de fase volta quando a fase recomeca"},
	}
	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			if got := tc.trigger.PerRun(); got != tc.perRun {
				t.Errorf("%s.PerRun() = %v, esperado %v", tc.trigger, got, tc.perRun)
			}
		})
	}
}

func TestUnknownTriggerIsNotPerRun(t *testing.T) {
	// Um gatilho que o loader nem aceita nao pode ser tratado como cena de
	// corrida: esquecer o que nunca tocou seria trabalho a toa, e a resposta
	// honesta para "isto e da corrida?" sobre algo desconhecido e nao.
	if Trigger("on_qualquer_coisa").PerRun() {
		t.Error("gatilho desconhecido foi tratado como cena de corrida")
	}
}

func TestKnownTriggerAcceptsExactlyTheThree(t *testing.T) {
	for _, ok := range []Trigger{TriggerMapStart, TriggerWavesCleared, TriggerLastStand} {
		if !knownTrigger(ok) {
			t.Errorf("%s deveria ser aceito pelo loader", ok)
		}
	}
	// Gatilho que o diretor nao sabe responder vira roteiro que nunca toca, em
	// silencio — por isso o loader recusa em vez de aceitar e ignorar.
	if knownTrigger("on_inventado") {
		t.Error("gatilho inventado foi aceito pelo loader")
	}
}

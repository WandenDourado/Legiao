package sound

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// allSFX e a lista das constantes do catalogo. Escrita a mao de proposito: e
// ela que faz TestEverySFXHasADefinition valer alguma coisa — derivar a lista
// do proprio mapa faria o teste concordar consigo mesmo.
var allSFX = []SFX{
	SFXSanctuary, SFXAngelic,
	SFXFireLaunch, SFXFireImpact, SFXFireGround, SFXMeteor,
	SFXGraveyard, SFXLegion,
	SFXArrowVolley, SFXCelestial,
	SFXShield, SFXAvatar,
	SFXAttackMago, SFXAttackSacerdotisa, SFXAttackPaladina,
	SFXAttackArqueiro, SFXAttackNecromante,
}

// TestEverySFXHasADefinition pega a falha silenciosa deste pacote: um id sem
// linha em sfxDefs nao da erro em lugar nenhum — enqueue devolve sem tocar, e
// a magia simplesmente fica muda.
func TestEverySFXHasADefinition(t *testing.T) {
	if len(allSFX) != len(sfxDefs) {
		t.Errorf("catalogo com %d ids e %d definicoes: alguma sobra ou falta",
			len(allSFX), len(sfxDefs))
	}
	for _, id := range allSFX {
		def, ok := sfxDefs[id]
		if !ok {
			t.Errorf("%q sem definicao: a magia ficaria muda sem erro", id)
			continue
		}
		if def.file == "" {
			t.Errorf("%q sem arquivo", id)
		}
		if def.voices < 1 {
			t.Errorf("%q com %d vozes: nada tocaria", id, def.voices)
		}
	}
}

// TestArrowVolleyGapFitsBetweenWaves guarda o pedido explicito de que a rajada
// do Arqueiro soe UMA VEZ POR LEVA. As levas saem a ArrowVolleyWaveDelay uma
// da outra (0,18 s hoje, em skill/arrow.go); um intervalo minimo maior que
// isso engoliria a segunda e a terceira em silencio.
func TestArrowVolleyGapFitsBetweenWaves(t *testing.T) {
	const arrowVolleyWaveDelay float32 = 0.18 // skill.ArrowVolleyWaveDelay
	if gap := sfxDefs[SFXArrowVolley].minGap; gap >= arrowVolleyWaveDelay {
		t.Errorf("intervalo minimo %.2fs >= %.2fs entre levas: as levas 2 e 3 nao soariam",
			gap, arrowVolleyWaveDelay)
	}
}

// TestAttenuationFallsOffWithDistance verifica a regra inteira do recurso:
// perto ouve cheio, longe nao ouve, e no meio o volume so cai.
func TestAttenuationFallsOffWithDistance(t *testing.T) {
	if got := attenuation(0); got != 1 {
		t.Errorf("na origem: %.3f, esperado 1", got)
	}
	if got := attenuation(SFXFullRadius); got != 1 {
		t.Errorf("no raio de volume cheio: %.3f, esperado 1", got)
	}
	if got := attenuation(SFXHearRadius); got != 0 {
		t.Errorf("no limite da audicao: %.3f, esperado 0", got)
	}
	if got := attenuation(SFXHearRadius * 4); got != 0 {
		t.Errorf("muito alem do limite: %.3f, esperado 0", got)
	}

	prev := float32(1)
	for d := SFXFullRadius; d <= SFXHearRadius; d += 50 {
		got := attenuation(d)
		if got > prev {
			t.Fatalf("volume subiu de %.3f para %.3f indo ate %.0f unidades", prev, got, d)
		}
		if got < 0 || got > 1 {
			t.Fatalf("volume fora de [0,1] em %.0f: %.3f", d, got)
		}
		prev = got
	}
}

// TestEnqueueIsSilentBeforeInit garante que uma magia lancada com o audio
// fechado (todo teste de internal/skill, que nunca abre dispositivo) nao toca
// nem enfileira nada.
func TestEnqueueIsSilentBeforeInit(t *testing.T) {
	sfxMutex.Lock()
	pending = pending[:0]
	sfxMutex.Unlock()

	PlayAt(SFXFireLaunch, rl.Vector2{})
	PlayAtFor(SFXMeteor, rl.Vector2{}, 1)

	sfxMutex.Lock()
	defer sfxMutex.Unlock()
	if len(pending) != 0 {
		t.Errorf("%d disparos enfileirados com o audio fechado", len(pending))
	}
}

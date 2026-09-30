package skill

import (
	"fmt"
	"sync/atomic"

	"github.com/WandenDourado/Legiao/internal/sound"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Meteor Rain constants tune the Mago's ultimate (Chuva de Meteoros).
const (
	// MeteorRainDuration is how long the rain keeps spawning meteors.
	MeteorRainDuration float32 = 15.0
	// MeteorRainRefArea e a area do world_05 (8192 x 11520), o mapa contra o
	// qual a chuva foi calibrada em 27/08/2026.
	MeteorRainRefArea float32 = 8192 * 11520
	// MeteorRainRefInterval e o intervalo de nascimento NAQUELA area (o
	// world_05). Em qualquer outro mapa ele e reescalado por
	// MeteorRainIntervalFor para manter a DENSIDADE — ver o comentario
	// daquela funcao.
	//
	// 0,025 -> 0,015 (27/08/2026, a pedido do Gui): de ~40 para ~67
	// nascimentos por segundo NO WORLD_05. REGIME (doc/skill_performance.md):
	// 1/0,015 x 1,4 s de vida = ~93 meteoros no ar ao mesmo tempo naquele
	// mapa, contra os ~56 de antes, e ~1000 na chuva inteira contra 600.
	//
	// Este numero e o valor que `doc/combat_rules.md` ja documentava desde
	// 23/08 ("A Chuva de Meteoros mede-se contra o Orc") e que o codigo nunca
	// recebeu — junto com o dano abaixo. A tabela de la e este arquivo agora
	// concordam.
	//
	// 31/08/2026: virou uma REFERENCIA, nao mais um numero fixo em todo mapa.
	// O sorteio e sobre o mapa inteiro (StepMeteorRains recebe worldW/worldH),
	// entao um intervalo fixo espalhava o mesmo numero de pedras por areas que
	// variam onze vezes — o world_07, o menor mapa do jogo e um dos dois onde
	// a suprema de fato existe em campanha (a outra fase, world_06, e o
	// unico lugar antes dele), recebia a mesma densidade do world_05, so que
	// 11x mais concentrada. MeteorRainIntervalFor corrige isso mantendo a
	// densidade (nascimentos por segundo por unidade de area) constante. Ver
	// doc/combat_rules.md ("A chuva mede-se por densidade, nao por
	// contagem").
	//
	// O CUSTO ESTA MEDIDO E TEM ONDE SER PAGO. O desenho ja e cullado por
	// meteoro e por particula desde 22/08 (doc/performance.md §8), entao o
	// que sobe de verdade e a REDE: `tickMeteors` transmite um evento POR
	// meteoro. Se o cliente cair na chuva, o proximo passo ja esta escrito
	// naquele documento e nao e baixar esta constante — e lotear o broadcast
	// no tique de rede, ou semear a chuva para host e cliente sortearem a
	// mesma sequencia (600 mensagens viram uma). Mapas menores que o world_05
	// ja tem menos meteoros com este reescalonamento, entao a pressao de rede
	// so cresce se um mapa novo for maior que o world_05.
	MeteorRainRefInterval float32 = 0.015
	// MeteorRainMinInterval e o TETO DE DESEMPENHO, nao um numero de design:
	// 0,015 e o regime medido em doc/performance.md secao 8 (~93 meteoros
	// simultaneos, ~2.500 particulas no world_05). Um mapa maior que o
	// world_05 para aqui — MeteorRainIntervalFor nunca desce abaixo disto,
	// mesmo que a densidade constante pedisse mais.
	MeteorRainMinInterval float32 = 0.015
	// MeteorRainDensityBoost multiplica a DENSIDADE de referencia: 1,02 = 2% a
	// mais de pedras por area em todo mapa (27/09/2026, a pedido do Gui
	// depois do playtest: na arena a suprema "nao conseguiu ajudar o time", e
	// o ajuste pedido foi sutil de proposito). Entra como fator na densidade,
	// e nao como tabela por fase, pelo mesmo motivo de MeteorRainIntervalFor:
	// a chuva mede-se por densidade.
	//
	// O world_05 NAO muda: ele ja esta no teto de desempenho
	// (MeteorRainMinInterval), e o teto ganha do reforco. Na campanha isso nao
	// custa nada — o Mago so tem a Chuva a partir do world_06
	// (game.UltimatesGrantedOn), e 06 e 07 sao os dois menores mapas do jogo,
	// longe do teto. Na arena (world_07) sao ~90 pedras numa chuva de 15 s;
	// com 1,02, ~92.
	MeteorRainDensityBoost float32 = 1.02
	// MeteorRainCooldown is the caster's cooldown in seconds.
	MeteorRainCooldown float32 = 60.0
	// MeteorFallTime is how long a meteor streaks down before impact.
	MeteorFallTime float32 = 0.9
	// MeteorImpactRadius is the area-damage radius on ground impact.
	MeteorImpactRadius float32 = 160
	// MeteorImpactDamage e o dano de area de UM impacto.
	//
	// 100 -> 480 (27/08/2026, a pedido do Gui: "um meteoro deve tirar 80% da
	// vida de um orc"). E o valor que `doc/combat_rules.md` e
	// `TestOneMeteorTakesFourFifthsOfAnOrc` (network/balance_test.go) ja
	// exigiam desde 23/08 — o teste estava vermelho porque a constante ficou
	// para tras. O numero e derivado, nao escolhido: o orc
	// (EnemyTypeGarrison) tem 600 de vida em entity/enemy_sprite.go, e 80%
	// disso e 480. Se a vida do orc mudar, este numero muda junto.
	//
	// Os 100 vinham de quando o slime de 100 de vida era o elenco inteiro, e
	// a frase do comentario antigo — "mata qualquer inimigo pego no impacto" —
	// deixou de ser verdade no dia em que o orc entrou: a chuva varria o campo
	// de slime e lobo e deixava os orcs de pe, que sao justamente os que
	// decidem a fase.
	//
	// DELIBERADAMENTE ABAIXO DE 600: um meteoro nao mata o orc, dois matam. A
	// suprema arrasa a massa e ABRE o orc para o grupo terminar; ela nao
	// resolve a fase sozinha.
	MeteorImpactDamage float32 = 480
	// meteorImpactTime is the impact flash/shockwave duration.
	meteorImpactTime float32 = 0.5
	// meteorSoundLen e por quanto tempo o estrondo de UM impacto soa antes de
	// desvanecer. E mais curto que o arquivo (2,9 s) de proposito: a chuva
	// derruba pedras a cada 15 ms, e uma cauda longa por impacto empilharia
	// dezenas de estrondos ate virar um zumbido continuo. Com este prazo, e
	// com o intervalo minimo do catalogo (sound/sfx.go), o que se ouve e um
	// bombardeio em que cada queda ainda se distingue.
	meteorSoundLen float32 = 1.2
	// meteorFallDist is how far up-right the meteor spawns from its target.
	meteorFallDist float32 = 780
)

// MeteorRainIntervalFor devolve o intervalo de nascimento para um mapa deste
// tamanho, mantendo constante a DENSIDADE (nascimentos por segundo por
// unidade de area) calibrada no world_05, vezes MeteorRainDensityBoost.
//
// POR QUE DENSIDADE, E NAO CONTAGEM. A chuva e a unica suprema que nao
// escolhe alvo: ela sorteia pontos no mapa INTEIRO. Entao o que ela controla
// nao e quantas pedras caem, e sim quantas caem PERTO DE ALGUEM — e isso e
// area. Com intervalo fixo, o world_07 (8,5 Mpx2) recebia as mesmas 1.000
// pedras do world_05 (94,4 Mpx2): onze vezes a densidade, no menor mapa do
// jogo.
//
// E COM O ZOOM FIXO POR PLATAFORMA (game/camera.go: 1,0 no PC, 0,8 no
// celular desde 27/09/2026) a janela visivel tem a mesma area de mundo em toda
// fase, entao densidade constante e tambem CONTAGEM NA TELA constante — o
// custo de desenho por quadro deixa de depender do mapa. No celular essa
// contagem e ~1,56x a do PC (a area que 0,8 mostra a mais).
//
// Funcao pura de proposito: ela e o ponto de teste. Ver meteor_scale_test.go.
func MeteorRainIntervalFor(worldW, worldH float32) float32 {
	area := worldW * worldH
	if area <= 0 {
		return MeteorRainRefInterval // mundo degenerado: nao inventa densidade
	}
	iv := MeteorRainRefInterval * (MeteorRainRefArea / area) / MeteorRainDensityBoost
	if iv < MeteorRainMinInterval {
		return MeteorRainMinInterval
	}
	return iv
}

var meteorSeq uint64

func nextMeteorID() string {
	return fmt.Sprintf("mt%d", atomic.AddUint64(&meteorSeq, 1))
}

// Meteor is one falling rock of the meteor rain: a fiery streak dives toward
// a marked ground target, then explodes. Damage happens ONLY at impact (host).
type Meteor struct {
	ID     string
	Target rl.Vector2
	// Age runs 0..MeteorFallTime (falling), then continues through the
	// impact window. Impacted flips exactly once, at the impact frame.
	Age      float32
	Impacted bool
	Trail    *ParticleEmitter
}

// NewMeteor creates a meteor aimed at the given ground target.
func NewMeteor(target rl.Vector2) *Meteor {
	return &Meteor{ID: nextMeteorID(), Target: target, Trail: NewParticleEmitter()}
}

// fallDir is the fixed world-space dive direction (from up-right to target).
var meteorFallDir = rl.Vector2{X: -0.42, Y: 0.91} // normalized-ish

// HeadPos returns the meteor head's current world position while falling.
func (mt *Meteor) HeadPos() rl.Vector2 {
	p := clamp01(mt.Age / MeteorFallTime)
	remaining := (1 - p) * meteorFallDist
	return rl.NewVector2(
		mt.Target.X-meteorFallDir.X*remaining,
		mt.Target.Y-meteorFallDir.Y*remaining,
	)
}

// Advance ages the meteor and reports (impactedNow, finished). impactedNow is
// true only on the exact frame the meteor hits the ground, so the host can
// apply damage exactly once.
func (mt *Meteor) Advance(dt float32) (impactedNow, finished bool) {
	mt.Age += dt
	if !mt.Impacted && mt.Age >= MeteorFallTime {
		mt.Impacted = true
		impactedNow = true
		// Aqui dentro, e nao em StepMeteors: `Advance` e o caminho comum do
		// host e do cliente, e a queda e deterministica dos dois lados, entao
		// o estrondo nasce no mesmo instante em toda maquina sem custar um
		// evento de rede.
		sound.PlayAtFor(sound.SFXMeteor, mt.Target, meteorSoundLen)
		mt.Trail.Burst(mt.Target, 26, 220, 520, 0.45, 22, rl.Orange)
		mt.Trail.Burst(mt.Target, 14, 120, 300, 0.5, 14, rl.Yellow)
	}
	// While falling, feed the fiery tail (big rock = big burning wake).
	if !mt.Impacted {
		head := mt.HeadPos()
		mt.Trail.Emit(head, rl.Vector2Scale(meteorFallDir, -160), 0.34, 38, rl.Orange)
		mt.Trail.Emit(head, rl.Vector2Scale(meteorFallDir, -70), 0.28, 22, rl.Red)
		mt.Trail.Emit(head, rl.NewVector2(0, 0), 0.22, 15, rl.Yellow)
	}
	mt.Trail.Update(dt)
	finished = mt.Age >= MeteorFallTime+meteorImpactTime
	return impactedNow, finished
}

// Draw renders the meteor: ground target marker, falling streak with fiery
// head, and the impact flash + shockwave ring.
func (mt *Meteor) Draw() {
	// Culling primeiro, e ANTES do BeginBlendMode: a chuva cobre o mapa
	// inteiro e a tela mostra uma fracao dele. Duas pontas porque as duas sao
	// desenhadas longe uma da outra — a marca no chao e a rocha, que nasce a
	// 780 unidades dali. O raio da rocha cobre o rastro de 300 mais o brilho
	// de 88; o do alvo cobre o anel de impacto.
	if !visibleAny(mt.Target, MeteorImpactRadius*1.2, mt.HeadPos(), 400) {
		return
	}
	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	if !mt.Impacted {
		p := clamp01(mt.Age / MeteorFallTime)
		// Ground marker: a ring that closes in as the meteor approaches.
		markR := MeteorImpactRadius * (1.15 - 0.55*p)
		rl.DrawRing(mt.Target, markR-3, markR+3, 0, 360, 40,
			rl.Fade(rl.NewColor(255, 90, 40, 255), 0.18+0.38*p))
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*0.5, rl.Fade(rl.NewColor(255, 70, 30, 255), 0.10+0.20*p), rl.Blank)

		// Falling streak: a HUGE burning rock with an elongated fire tail.
		head := mt.HeadPos()
		tail := rl.NewVector2(
			head.X-meteorFallDir.X*300,
			head.Y-meteorFallDir.Y*300,
		)
		rl.DrawLineEx(tail, head, 22, rl.Fade(rl.Red, 0.5))
		rl.DrawLineEx(tail, head, 12, rl.Fade(rl.Orange, 0.8))
		rl.DrawLineEx(tail, head, 5, rl.Fade(rl.NewColor(255, 250, 200, 255), 0.9))
		// Rock body: dark core silhouette wrapped in fire (reads as mass, not
		// just light).
		rl.DrawCircleGradient(int32(head.X), int32(head.Y), 88, rl.Fade(rl.Red, 0.7), rl.Blank)
		rl.DrawCircleGradient(int32(head.X), int32(head.Y), 54, rl.Fade(rl.Orange, 0.95), rl.Blank)
		rl.DrawCircle(int32(head.X), int32(head.Y), 24, rl.Fade(rl.NewColor(255, 252, 220, 255), 1.0))
		rl.EndBlendMode()
		// Solid molten-rock core drawn in normal blending over the glow.
		rl.DrawCircle(int32(head.X), int32(head.Y), 20, rl.NewColor(74, 40, 28, 255))
		rl.DrawCircleV(rl.NewVector2(head.X-6, head.Y-6), 9, rl.NewColor(255, 140, 40, 255))
		rl.DrawCircleV(rl.NewVector2(head.X+8, head.Y+5), 5.5, rl.NewColor(255, 200, 90, 255))
		rl.BeginBlendMode(rl.BlendAdditive)
	} else {
		// Impact: expanding shock ring + hot flash fading out.
		prog := clamp01((mt.Age - MeteorFallTime) / meteorImpactTime)
		r := MeteorImpactRadius * (0.35 + prog*1.05)
		a := 1 - prog
		rl.DrawRing(mt.Target, r-5, r+5, 0, 360, 48, rl.Fade(rl.NewColor(255, 150, 50, 255), 0.85*a))
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*(0.9-0.4*prog), rl.Fade(rl.Orange, 0.55*a), rl.Blank)
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*0.35*(1-prog), rl.Fade(rl.NewColor(255, 252, 220, 255), 0.9*a), rl.Blank)
	}

	for _, pt := range mt.Trail.particles {
		drawParticle(pt)
	}
}

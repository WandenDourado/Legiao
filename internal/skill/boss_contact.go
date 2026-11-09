package skill

// O anel de contato da Senhora das Trevas (item 5, 31/08/2026,
// doc/plan_world07_arena_tuning.md).
//
// Nao e um ataque: nao usa AttackCooldown, nao dispara animacao, nao pede
// alvo. E uma propriedade do CORPO, como o fogo de chao e uma propriedade do
// LUGAR — por isso este arquivo so guarda as constantes e o desenho; quem
// cobra o dano e `network/host_boss_contact.go`, pela mesma divisao que
// `boss_thorn.go` (marca) e `host_boss.go` (cobranca) ja seguem.

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// BossContactDPS e o dano por segundo cobrado dos ESPECTROS dentro do
	// anel. Ate 27/09/2026 valia para jogador tambem; o playtest daquele dia
	// tirou o jogador (plans/plan_playtest_27_09.md §2) e a calibracao
	// abaixo, que sempre foi contra a Legiao, continua valendo inteira.
	//
	// Calibrado contra a Legiao Espectral (skill/legion_manager.go), que sem
	// isto a derretia sozinha: um espectro de 60 de vida colado no corpo dela
	// sobrevivia ~8 s so ao revide (`AttackDamage` 22 a cada `AttackCooldown`
	// 4,0 s, ~5,5 dano/s), e os 30 da Legiao (`LegionCount`) tiravam de uma
	// vez ~14.700 — quase metade dos 30.000 de vida antigos, com 60 s de
	// recarga: duas ativacoes e a arena acabava.
	//
	// A 18/s um espectro sobrevive ~2,5 s no anel: a Legiao inteira tira
	// ~4.700, ~10% dos 45.000 novos (entity/enemy_sprite.go). Ainda vale
	// usar — 10% da barra por ativacao — sem virar o botao de vitoria que
	// era antes. 30 (= o mesmo da nevoa) reduziria a Legiao a 7%, pouco
	// demais para compensar gastar a ultimate nela em vez de na horda que
	// entra pelos portoes, que e onde ela sempre foi excelente.
	BossContactDPS float32 = 18
	// BossContactTickEvery agrupa o dano em pedacos legiveis, como a nevoa:
	// dois eventos por segundo por alvo, e nao sessenta.
	BossContactTickEvery float32 = 0.5
)

// BossContactRadius e o raio do anel, e ele NAO e um numero solto: e o
// `Radius` fisico da propria Senhora das Trevas
// (`entity.GetEnemyDef(EnemyTypeDarkLady).Radius`), o "corpo de aranha" que
// `enemy_sprite.go` ja documenta como deliberadamente menor que a envergadura
// das patas. Lido do EnemyDef em vez de repetido como constante para nao
// dessincronizar se aquele numero mudar.
//
// A PRIMEIRA versao deste anel usava 150 centrado em `Enemy.Position` — o PE
// dela, por causa do `FootLine` da folha de animacao. Como o corpo visivel
// fica `HitOffsetY` (-180) ACIMA do pe, o anel nascia baixo demais e lia como
// uma esfera solta no chao, sem ligacao nenhuma com a chefe (foi relatado
// assim: "uma esfera perto da boss, nao sei o que e isso"). A correcao tem
// duas partes, e as duas sao necessarias: o CENTRO passou a ser
// `Enemy.HitCenter()` — a mesma correcao de altura que a caixa de acerto de
// espada ja usa — e o RAIO passou a ser o corpo fisico (130) em vez de um
// numero inflado, para que o dano comece perto de onde a sprite realmente
// esta, e nao antes.
var BossContactRadius = entity.GetEnemyDef(entity.EnemyTypeDarkLady).Radius

var (
	// Mesma familia de cor dos espinhos (thornWarnDeep/thornWarnHot,
	// boss_thorn.go): e o mesmo tipo de dano, chao que machuca por perto.
	bossContactPulseSpeed float32 = 1.6
)

// DrawBossContactRing desenha o halo do anel de contato, pulsando devagar,
// no raio EXATO de BossContactRadius — a mesma regra da marca do espinhao,
// sem margem escondida entre o que se ve e o que machuca: CENTRO e RAIO nao
// mudam, so o ESTILO.
//
// Por que halo, e nao um anel de tracos finos: o espinhao desenha no chao,
// onde um traco de 1 px contra grama/pedra le bem. A Senhora das Trevas nao
// tem chao livre ali — o corpo dela (610 px) e maior que o diametro do anel
// (260), entao o anel inteiro cai POR CIMA da barriga opaca da aranha, e
// `renderer.go` desenha o anel depois do corpo nos dois ramos (nao da para
// inverter: por baixo da sprite o anel ficaria 100% invisivel e o jogador
// perderia o aviso). Um traco fino de 1 px nessa condicao nao le como aviso
// de perigo, le como um risco desenhado na pele dela. Uma pilha de circulos
// concentricos com alfa caindo para fora nao tem essa borda dura: ela lê
// como um brilho/aura de energia sombria em volta do corpo, que e o mesmo
// numero (o raio de contato exato) so que espalhado em vez de contornado.
//
// `center` TEM DE SER `Enemy.HitCenter()`, nao `Enemy.Position()`: a posicao
// e o PE (por causa do `FootLine`), e o corpo visivel fica 180 px acima
// dela. Chamado com o MESMO center nas duas pontas — o host tem a chefe no
// EntityManager, o cliente em `network.InterpolatedEnemies()` mais o
// `HitOffsetY` do EnemyDef — e sem NENHUM estado replicado: o raio e
// constante e a chefe e fixa, entao a posicao sozinha basta. Ao contrario do
// espinhao, este anel nao tem relogio: ele existe o tempo inteiro que a
// chefe estiver viva, entao o pulso vem do tempo do jogo (`rl.GetTime()`) e
// nao de uma idade guardada em algum lugar.
//
// Custo: 6 `DrawCircle` cheios por quadro enquanto a chefe estiver viva —
// um punhado, nao dezenas (`doc/skill_performance.md`, Regra 5): nao ha
// gradiente por particula nem crescimento com fase, e o numero de circulos e
// fixo, nao escala com nada.
func DrawBossContactRing(center rl.Vector2) {
	cx, cy := int32(center.X), int32(center.Y)
	pulse := 0.5 + 0.5*float32(math.Sin(float64(rl.GetTime())*float64(bossContactPulseSpeed)))

	// Base ampla e fraca: da o corpo do halo, quase nao muda com o pulso,
	// so o suficiente para nao desaparecer contra o roxo-azulado do abdomen.
	rl.DrawCircle(cx, cy, BossContactRadius, rl.Fade(thornWarnDeep, 0.14+0.08*pulse))

	// Aneis concentricos, do maior (mais fraco) ao menor (mais forte): cada
	// um desenhado por cima acumula opacidade em direcao ao centro, o mesmo
	// truque de "brilho radial" com poucos circulos cheios em vez de um
	// gradiente por triangulo. O maior para em BossContactRadius — nunca
	// passa dele, para o halo nao prometer alcance que o dano nao tem.
	const haloRings = 5
	for i := haloRings; i >= 1; i-- {
		t := float32(i) / float32(haloRings) // 1 (borda) -> 1/haloRings (centro)
		radius := BossContactRadius * t
		alpha := (0.10 + 0.08*pulse) * (1 - t*0.55)
		rl.DrawCircle(cx, cy, radius, rl.Fade(thornWarnHot, alpha))
	}
}

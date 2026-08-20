package entity

import (
	"math"
	"sync"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// EnemySpriteMode selects how an enemy sheet encodes facing.
type EnemySpriteMode string

const (
	// EnemyModeRadial holds a single true top-down view that the renderer
	// rotates toward the velocity vector. Only valid for amorphous or radially
	// symmetric creatures (slimes, swarms, floating orbs): a rigid body would
	// expose the baked-in highlight spinning with it.
	EnemyModeRadial EnemySpriteMode = "radial"

	// EnemyModeDirectional holds a 3/4 sheet with one row per direction and
	// one column per frame. The renderer picks the row from the heading and
	// never rotates the art, which is what a rigid body needs: rotating an
	// orc's pauldrons would spin the highlight baked into them.
	//
	// Rows store half the facings plus the mirror axis (see
	// enemy_sprite_direction.go); the other half is drawn flipped.
	EnemyModeDirectional EnemySpriteMode = "directional"

	// EnemyModeFixed holds a single 3/4 view that is NEVER rotated and never
	// picks a row. It is the mode for a creature that does not travel: the
	// gargoyle sentry beats its wings on a pedestal and never faces anywhere
	// else.
	//
	// It is not radial with the rotation switched off. Radial art is drawn from
	// directly above and is only legible because nothing in it says "up"; this
	// sheet is a face seen head-on with an upper-left key light baked in, which
	// is the exact case the radial mode is documented to fail. And it is not
	// directional either: a still creature has no heading to pick a row from,
	// so five rows would be four wasted generations.
	//
	// Practical consequence for the pipeline: with no rotation there is no
	// inscribed-circle containment to respect, so the frame does NOT have to be
	// square. The gargoyle's widest pose spans 572 px against a ~340 px body,
	// and forcing that into a square cell shrank the creature to fit its own
	// wing.
	EnemyModeFixed EnemySpriteMode = "fixed"
)

// EnemyAnim names one animation of an enemy.
type EnemyAnim string

const (
	// AnimIdle is what an enemy plays when it is not going anywhere. It is the
	// fallback for every enemy, including the ones that have no other.
	AnimIdle EnemyAnim = "idle"
	// AnimWalk is the pursuit cycle.
	AnimWalk EnemyAnim = "walk"

	// Os cinco abaixo sao do chefe fixo. Idle e walk descrevem LOCOMOCAO, e a
	// Senhora das Trevas nao se locomove: o que a move entre estados e tempo e
	// intencao. Ver enemy_boss_anim.go.

	// AnimIdleScan e a varredura: ela da uns passos, vira o torso procurando
	// alvo e volta. Toca de tempos em tempos por cima do idle.
	AnimIdleScan EnemyAnim = "idle_scan"
	// AnimCastLoop e a danca que roda enquanto ela conjura.
	AnimCastLoop EnemyAnim = "cast_loop"
	// AnimCastRelease e o feitico saindo, no fim da danca.
	AnimCastRelease EnemyAnim = "cast_release"
	// AnimAttackWindup sao os bracos cruzados tremendo. E a JANELA DE DESVIO:
	// enquanto isto toca, o jogador ainda tem tempo de sair.
	AnimAttackWindup EnemyAnim = "attack_windup"
	// AnimAttackStrike e o golpe - ela se abaixa abrindo os bracos.
	AnimAttackStrike EnemyAnim = "attack_strike"
)

// EnemyAnimDef is the geometry of ONE animation's sheet.
//
// It exists because a directional pack does not crop to a single size: the
// idle frame only has to hold a hunched orc (154x134) while a walk frame holds
// a stride (100x127) and an attack frame will have to hold the whole arc of a
// two-handed sword. Forcing every animation into the largest box would spend
// most of the texture on transparent pixels.
//
// The numbers are copied from the sheet's own manifest. Copying is the right
// call - reading JSON at draw time to learn a frame height would be both slow
// and fragile - but a copy without a check goes stale, which is what
// enemy_manifest_test.go is for.
type EnemyAnimDef struct {
	SpritePath  string // Relative path to the sheet (used with assets.Path).
	FrameWidth  int
	FrameHeight int
	Columns     int     // Frames in the cycle.
	Rows        int     // Directional mode only: stored facings. Zero for radial.
	FrameTime   float32 // Seconds per frame.

	// FootLine is where the creature's soles sit inside a frame, in frame
	// pixels from the top. Same role as CharacterDef.FootLine: it turns the
	// sprite into a figure standing on the ground, so Position is the point the
	// feet touch instead of the middle of the drawing.
	//
	// Per animation, not per enemy: each animation is cropped to its own
	// window, so the sole lands at a different height in each one. A single
	// number shared across them would make the orc bob as it changed state.
	//
	// Only meaningful for a sheet drawn upright. A radial sheet has no soles -
	// it is seen from directly above - so it leaves this at zero and keeps
	// being drawn centred on Position, exactly as before.
	FootLine int

	// PlayOrder e a ordem em que os quadros da folha sao tocados, quando ela nao
	// e 0,1,2,... O idle_scan da chefe usa (0,1,2,3,2,1): a volta e a ida tocada
	// ao contrario, entao os quadros do retorno nao precisam existir na arte.
	//
	// Vazio quer dizer "na ordem", que e o caso de todo inimigo anterior. Ver
	// Steps e Column em enemy_anim_playback.go.
	PlayOrder []int

	// OneShot marca a animacao que toca uma vez e trava no ultimo quadro.
	//
	// E OneShot, e nao Loop, por causa do zero value: `false` tem que significar
	// "em laco", que e o comportamento de todos os inimigos que existiam antes
	// deste campo.
	OneShot bool
}

// EnemyDef describes an enemy kind: its art, its hitbox and its combat stats.
// It mirrors CharacterDef on the visual side. Hitbox and stats live here too so
// that art and gameplay are tuned in one place - sizing a sprite without
// resizing its hitbox makes the player swing at empty space around it.
type EnemyDef struct {
	Type        EnemyType
	Name        string
	SpritePath  string          // Relative path to the sheet (used with assets.Path).
	Mode        EnemySpriteMode // How facing is encoded.
	FrameWidth  int
	FrameHeight int
	Columns     int     // Frames in the animation cycle.
	Rows        int     // Directional mode only: stored facings. Zero for radial.
	RenderScale float32 // Visual scale; does not affect Radius.
	FrameTime   float32 // Seconds per animation frame.
	TurnRate    float32 // Degrees per second for radial facing; 0 snaps instantly.

	// Anims holds one entry per animation for enemies that have more than one.
	// Empty for the slime and the wolf, which have a single looping sheet
	// described by the flat fields above - AnimDef falls back to those, so
	// nothing about the radial enemies changed when this field appeared.
	//
	// RenderScale deliberately stays OUT of here: every animation of the same
	// creature has to be drawn at the same scale, or the orc would change size
	// when it started walking.
	Anims map[EnemyAnim]EnemyAnimDef

	// Combat stats.
	Radius float32 // Hitbox radius, matched to the visible silhouette.
	// HitOffsetY e HitRadius descrevem a CAIXA DE ACERTO, que nao e a mesma
	// coisa que `Radius`.
	//
	// `Radius` e o corpo fisico: separacao entre monstros, colisao com o mapa e
	// o alcance do proprio golpe (`AttackRange + Radius`) saem dele. Engorda-lo
	// para o jogador acertar melhor mudaria as tres coisas de lado - o orc
	// passaria a bater de mais longe e a empurrar mais os vizinhos.
	//
	// A caixa de acerto e outra pergunta. Slime e lobo sao desenhados centrados
	// na propria posicao, entao o circulo fisico ja cobre o bicho e estes dois
	// campos ficam zerados. O orc tem `FootLine`: a posicao dele e o PE, e um
	// circulo de 45 px ali cobria as canelas de uma figura de ~176 px - em jogo
	// era preciso mirar no chao para acertar.
	//
	// Zero em qualquer um dos dois quer dizer "usa o corpo fisico".
	// HitOffsetY negativo sobe.
	HitOffsetY     float32
	HitRadius      float32
	Health         float32
	Speed          float32 // World units per second. The player moves at 200.
	AttackDamage   float32
	AttackRange    float32
	AttackCooldown float32 // Seconds between attacks.
	Color          string  // Fallback debug colour when the sheet is missing.
	// Vision e o piso do campo de visao deste tipo quando ele guarda um posto,
	// em unidades de mundo. Zero quer dizer "o que o territorio mandar".
	//
	// EXISTE PORQUE VISAO E VELOCIDADE SE PAGAM. O raio vinha so do mapa, igual
	// para todo mundo, e com isso o orc — o mais lento do elenco, a 130 contra
	// os 200 do jogador — notava o invasor no mesmo instante que o lobo a 240 e
	// nunca mais o alcancava. Ele precisa ver ANTES para ter alguma chance, e
	// isso e uma propriedade da criatura, nao do retangulo em que ela mora.
	Vision float32
}

// AnimDef returns the geometry of one animation.
//
// An enemy that declares no Anims map answers with its flat fields for every
// animation asked of it, which is what keeps the single-sheet radial enemies
// working through the same code path as the multi-sheet directional ones. An
// enemy that HAS the map but is asked for an animation it does not own falls
// back to idle rather than to nothing: a missing walk sheet should make the orc
// slide, not vanish.
func (d EnemyDef) AnimDef(anim EnemyAnim) EnemyAnimDef {
	if len(d.Anims) > 0 {
		if ad, ok := d.Anims[anim]; ok {
			return ad
		}
		if ad, ok := d.Anims[AnimIdle]; ok {
			return ad
		}
	}
	return EnemyAnimDef{
		SpritePath:  d.SpritePath,
		FrameWidth:  d.FrameWidth,
		FrameHeight: d.FrameHeight,
		Columns:     d.Columns,
		Rows:        d.Rows,
		FrameTime:   d.FrameTime,
	}
}

// HasAnim reports whether the enemy owns a dedicated sheet for an animation.
// The state machine uses it so it never selects a state the art cannot show.
func (d EnemyDef) HasAnim(anim EnemyAnim) bool {
	_, ok := d.Anims[anim]
	return ok
}

var enemyRegistry = map[EnemyType]EnemyDef{}

// RegisterEnemy adds an enemy definition to the registry.
func RegisterEnemy(def EnemyDef) {
	enemyRegistry[def.Type] = def
}

// GetEnemyDef returns the definition for an enemy type, falling back to the
// basic enemy when the type is unknown.
func GetEnemyDef(t EnemyType) EnemyDef {
	if def, ok := enemyRegistry[t]; ok {
		return def
	}
	return enemyRegistry[EnemyTypeBasic]
}

func init() {
	// The slime is the reference implementation of the radial mode. Its sheet
	// is 6 frames of an in-place squash-and-stretch pulse; the front of the
	// creature points to the top of the frame in every frame.
	//
	// Geometry (fixed center, per-frame scale, inscribed-circle containment)
	// was normalized offline by work/enemy-sprites/radial_normalize.py.
	// Measured max radius is 51 px against the 64 px inscribed limit, so no
	// rotation angle clips the art.
	RegisterEnemy(EnemyDef{
		Type:        EnemyTypeBasic,
		Name:        "Slime",
		SpritePath:  "assets/sprites/enemies/slime/slime.png",
		Mode:        EnemyModeRadial,
		FrameWidth:  256,
		FrameHeight: 256,
		Columns:     6,
		// 256 * 0.575 = 147 px on screen, the same size as before but reached
		// by DOWNscaling. Frames used to be 128 px and were magnified 1.15x at
		// draw time, which is what made the sprites look blocky.
		RenderScale: 0.575,
		FrameTime:   0.11,
		TurnRate:    540,

		Radius:         EnemySlimeRadius,
		Health:         100,
		Speed:          100,
		AttackDamage:   10,
		AttackRange:    25,
		AttackCooldown: 1.0,
		Color:          "#059A4F",
	})

	// A sentinela e a gargula do mapa 4: ela desperta no climax, bate as asas
	// sobre o pedestal que o mapa desenha, e nunca sai dali porque Speed e zero.
	//
	// Ate aqui ela nao tinha folha nenhuma, e o `Draw` caia no circulo de
	// depuracao - era ELE a "esfera roxa" que aparecia por cima da estatua, e
	// nao um efeito. Registrar a folha faz o circulo sumir sozinho.
	//
	// Modo FIXO, e nao radial: a arte e um rosto de frente com luz de cima-
	// esquerda pintada, exatamente o caso que a documentacao do modo radial diz
	// que quebra quando a sprite gira. Com Speed 0 o Angle nunca sairia de zero
	// de qualquer jeito, mas depender disso e deixar uma armadilha para quem um
	// dia der velocidade a ela.
	//
	// A geometria abaixo e COPIA de gargula_manifest.json, que o
	// work/enemy-sprites/gargula/build_gargula.py emite;
	// enemy_manifest_test.go existe para provar que a copia nao envelheceu.
	//
	// Quadro de 448x256 (nao quadrado): na folha a envergadura maxima e 321 px
	// contra um corpo de 226 px de altura, e num quadro quadrado a criatura
	// encolhia para caber a propria asa.
	//
	// RenderScale 0.95 (reducao, como manda a secao "Resolution and filtering"
	// da skill) poe a gargula em 215 px de altura e 305 de envergadura. Medido
	// contra a regua: o heroi tem 186 px de altura e a porta 150. Ela e mais
	// alta que o heroi e o dobro da porta de ponta a ponta de asa, que e o
	// tamanho de uma guarda de chefe. Um valor menor a deixava MAIS BAIXA que
	// o jogador, o que le como bicho comum.
	//
	// FootLine 253 ancora a criatura pelas garras: Position e o ponto do chao
	// onde o pedestal do mapa esta, e nao o meio do desenho.
	//
	// HitOffsetY -67 e o centro do corpo SOLIDO medido na folha (as linhas
	// 133-232, que sao as unicas cheias nos oito quadros - o resto varia porque
	// e asa). Sem ele o circulo de acerto ficaria na altura das garras e as
	// flechas celestes do Arqueiro, o unico projetil que pode toca-la, passariam
	// por cima do bicho inteiro.
	RegisterEnemy(EnemyDef{
		Type: EnemyTypeCastleSentry, Name: "Sentinela do Corrego",
		Mode:        EnemyModeFixed,
		RenderScale: 0.95,
		Anims: map[EnemyAnim]EnemyAnimDef{
			// Uma animacao so, e de proposito: a gargula nunca anda, entao
			// nao existe ciclo de perseguicao para desenhar. Ela vive no
			// Anims em vez dos campos planos porque FootLine so existe aqui.
			AnimIdle: {
				SpritePath:  "assets/sprites/enemies/gargula/gargula.png",
				FrameWidth:  448,
				FrameHeight: 256,
				Columns:     8,
				FrameTime:   0.20,
				FootLine:    253,
			},
		},

		Radius: 58, HitOffsetY: -67, HitRadius: 78,
		Health: 40, Speed: 0, AttackDamage: 14, AttackRange: 1900,
		AttackCooldown: 1.35, Color: "#6B287A",
	})

	// A Senhora das Trevas e o chefe final do world_05. Ela e o primeiro inimigo
	// do jogo com mais de duas animacoes, e o primeiro que nao decide o estado
	// por velocidade - ver enemy_boss_anim.go.
	//
	// RenderScale 1.0, e isso e uma escolha, nao preguica. As folhas foram
	// montadas no tamanho de tela (~558 px de vao, ~2,9x a altura do heroi), de
	// modo que o runtime nao reamostra nada: a regra da skill e que o quadro
	// chegue por reducao e nunca por ampliacao, e 1:1 e o unico ponto onde
	// nenhuma das duas acontece.
	//
	// Cada animacao tem quadro proprio porque cada uma foi recortada na propria
	// janela: o attack_strike e 100 px mais BAIXO que o idle (ela se agacha) e
	// mais largo (as patas abrem). Uma caixa unica seria quase toda transparente.
	//
	// FootLine e sempre altura-12: a montagem alinhou todos os quadros pela linha
	// do chao com 12 px de margem no rodape. E por isso que o agachamento do
	// golpe baixa o corpo sem tirar os pes do lugar.
	//
	// Radius 130 e o corpo de aranha, nao a envergadura: as patas passam de 270
	// px do centro e um circulo daquele tamanho empurraria o jogador para fora
	// da sala. HitOffsetY -180 sobe a caixa de acerto do chao para o meio do
	// corpo solido; sem isso o jogador teria que mirar no piso entre as patas.
	RegisterEnemy(EnemyDef{
		Type: EnemyTypeDarkLady, Name: "Senhora das Trevas",
		Mode:        EnemyModeFixed,
		RenderScale: 1.0,
		Anims: map[EnemyAnim]EnemyAnimDef{
			AnimIdle: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/idle.png",
				FrameWidth:  610, FrameHeight: 590, Columns: 8,
				FrameTime:   0.16, FootLine: 578,
			},
			// PlayOrder em vai-e-vem: ela vira o torso ate o extremo e volta
			// desfazendo o caminho. Os quadros da volta sao os da ida tocados ao
			// contrario, entao a arte tem 4 e a animacao tem 6.
			AnimIdleScan: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/idle_scan.png",
				FrameWidth:  596, FrameHeight: 568, Columns: 4,
				FrameTime:   0.16, FootLine: 556,
				OneShot:     true,
				PlayOrder:   []int{0, 0, 1, 2, 3, 3, 2, 1},
				// segura as pontas do giro
			},
			AnimCastLoop: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/cast_loop.png",
				FrameWidth:  594, FrameHeight: 632, Columns: 4,
				FrameTime:   0.22, FootLine: 620,
			},
			AnimCastRelease: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/cast_release.png",
				FrameWidth:  592, FrameHeight: 626, Columns: 4,
				FrameTime:   0.11, FootLine: 614,
				OneShot:     true,
				PlayOrder:   []int{0, 1, 1, 2, 2, 2, 3, 3},
				// segura o disparo (indice 2)
			},
			AnimAttackWindup: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/attack_windup.png",
				FrameWidth:  586, FrameHeight: 578, Columns: 4,
				FrameTime:   0.10, FootLine: 566,
			},
			AnimAttackStrike: {
				SpritePath:  "assets/sprites/enemies/senhora_das_trevas/attack_strike.png",
				FrameWidth:  584, FrameHeight: 488, Columns: 4,
				FrameTime:   0.10, FootLine: 476,
				OneShot:     true,
				PlayOrder:   []int{0, 0, 1, 2, 2, 2, 2, 3},
				// segura o AGACHAMENTO (indice 2)
			},
		},

		Radius: 130, HitOffsetY: -180, HitRadius: 200,
		Health: 400, Speed: 0, AttackDamage: 22, AttackRange: 1400,
		AttackCooldown: 4.0, Color: "#2A2540", Vision: 2600,
	})

	// The wolf is the fast enemy. Its sheet is 8 frames of an in-place gallop.
	//
	// RenderScale is 1.8 rather than the slime's 1.15 on purpose. Seen from
	// above a wolf is about 0.30 as wide as it is long - that is anatomically
	// correct, and an earlier attempt to widen it for readability produced a
	// splayed X pose instead of a run. So legibility comes from size: at 1.8
	// the animal is ~180 px nose to tail, slightly longer than a house door is
	// tall, which reads clearly on screen.
	//
	// Radius is a compromise a circle cannot win on an elongated body: 50 px
	// covers the torso generously while leaving the muzzle and tail outside the
	// hitbox. Worth revisiting if enemies ever get capsule collision.
	RegisterEnemy(EnemyDef{
		Type:        EnemyTypeFast,
		Name:        "Lobo",
		SpritePath:  "assets/sprites/enemies/wolf/wolf.png",
		Mode:        EnemyModeRadial,
		FrameWidth:  256,
		FrameHeight: 256,
		Columns:     8,
		// 256 * 0.9 = 230 px on screen. Same size as the old 128 px frame at
		// 1.8x, but downscaled instead of magnified.
		RenderScale: 0.9,
		FrameTime:   0.07,
		TurnRate:    720, // Turns harder than the slime; it is a hunter.
		Radius:      EnemyWolfRadius,

		// O lobo e a presa certa da Legiao Espectral e a ameaca errada para o
		// jogador: pouca vida, muita velocidade, muito dano.
		//
		// O duelo que estes numeros decidem, e que e o coracao do mapa 2:
		// um espectro morde 7 a cada 0.25s (28 dano/s), entao derruba 40 de
		// vida em ~1.4s; o lobo bate 18 a cada 0.7s nos 35 de vida do espectro,
		// entao precisa de ~1.4s + o primeiro intervalo. O espectro ganha, mas
		// por pouco - e essa margem estreita E a identidade da skill.
		//
		// Health e AttackDamage se afinam JUNTOS: baixar a vida do lobo ajuda o
		// espectro, subir o dano dele atrapalha. Mexer num sem o outro desfaz o
		// duelo.
		//
		// Velocidade 240 passa a do jogador (200): o lobo alcanca. Era 180,
		// "escapavel, por pouco" - o oposto do que a matilha precisa ser.
		Health:         40,
		Speed:          240,
		AttackDamage:   18,
		AttackRange:    30,
		AttackCooldown: 0.7,
		Color:          "#2B3138",
	})

	// O orc de guarnicao e a primeira arte DIRECIONAL do jogo, e a primeira que
	// nao foi gerada aqui: veio de um pacote de terceiro com 21 animacoes x 16
	// direcoes em celulas de 320px.
	//
	// Ele nao pode ser radial, e isso nao e preferencia. A propria
	// skills/create-enemy-sprites diz que o modo radial falha em couraca,
	// escudo e cranio, porque o brilho pintado na armadura gira junto com a
	// sprite e denuncia o truque. O slime e o lobo sao amorfos ou quadrupedes
	// vistos de cima; este e um humanoide rigido de pe.
	//
	// A folha vem de work/orc-guarnicao/build_orc.py. Os numeros de geometria
	// abaixo sao COPIA do orc_manifest.json que aquele script emite, e
	// enemy_manifest_test.go existe para provar que a copia nao envelheceu.
	//
	// RenderScale 1.6 e ampliacao, ao contrario do slime e do lobo, que
	// reduzem. Nao ha como evitar: 320p e o teto do fornecedor e o corpo ocupa
	// so ~110px da celula (a celula e grande para caber o arco do espadao, nao
	// porque o corpo seja pequeno). Se a moleza da ampliacao bilinear aparecer
	// na tela, a correcao NAO e mexer aqui - e rodar o build_orc.py com
	// --scale 2.0, que reamostra com Lanczos offline e deixa este numero cair
	// abaixo de 1. Ver doc/plan_orc_guarnicao.md.
	//
	// OS STATS SAO A CONTRAPARTIDA DA LEGIAO ESPECTRAL, e sairam de simulacao,
	// nao de gosto. O alvo do Gui: CINCO orcs derrotam a ultimate do
	// Necromante - os trinta espectros morrem, os orcs nao precisam sobreviver.
	//
	// A primeira coisa que a conta mostra e que a VIDA NAO E A ALAVANCA.
	// Trinta espectros somam 30 x 11 / 0.18 = 1833 de dano por segundo; com os
	// stats provisorios (220 de vida, 30 de dano, 1.8 s) os cinco orcs caiam em
	// 0.7 s sem matar UM espectro. Engordar so a vida cumpre o alvo em 1400,
	// mas por um penhasco: em 1200 eles perdem inteiros e em 1400 ganham sem
	// perder nenhum, e cada orc passa a levar 10 s de tiro do grupo todo.
	//
	// O que decide o duelo e o GOLPE:
	//
	//   - AttackDamage 30 e AttackCooldown 1.5. O DANO NAO PODE CHEGAR A 60.
	//     Em StepLegions o inimigo revida contra CADA espectro engajado, cada
	//     um no proprio `hurtTimer`, e esse timer nasce ZERADO: o primeiro
	//     revide sai no quadro em que o espectro encosta. Com 60 de dano - a
	//     vida cheia de um espectro - todo espectro que engaja morre na hora, e
	//     UM orc sozinho limpa a legiao inteira em quatro quadros. Isso nao
	//     cumpre o alvo, destroi a ultimate.
	//   - Health 600 e o que sobra. Trinta espectros sobre cinco orcs sao ~366
	//     de dano por segundo em cada um; 600 dura 1.6 s, e os cinco precisam
	//     de 1.5 s (dois golpes de 30 numa vida de 60). A margem e de decimos,
	//     e e ela que faz cinco ser o numero.
	//
	// Medido, com o modelo fiel ao StepLegions:
	//
	//     1 orc  -> morre, 30 espectros de pe
	//     4 orcs -> morrem, 30 espectros de pe
	//     5 orcs -> legiao limpa em 1.5 s
	//
	// O degrau em CINCO e o alvo, e nao ha meio-termo: quatro nao arranham.
	//
	// Speed 130 fica ABAIXO dos 200 do jogador de proposito, porque atravessar
	// o vao da barricada precisa continuar sendo uma opcao melhor do que
	// brigar. Um perseguidor que alcanca transformaria defesa de territorio em
	// horda.
	//
	// A prova viva desta conta esta em orc_legion_test.go: se alguem mexer num
	// destes numeros e o duelo deixar de fechar, o teste reprova.
	RegisterEnemy(EnemyDef{
		Type:        EnemyTypeGarrison,
		Name:        "Orc de Guarnicao",
		Mode:        EnemyModeDirectional,
		RenderScale: 1.6,
		// TurnRate fica em zero de proposito: modo direcional nao gira arte.

		Anims: map[EnemyAnim]EnemyAnimDef{
			AnimIdle: {
				SpritePath:  "assets/sprites/enemies/orc/idle.png",
				FrameWidth:  154,
				FrameHeight: 134,
				Columns:     8,
				Rows:        EnemyDirectionRows,
				FootLine:    131,
				// 8 quadros guardados de 16, entao o dobro do tempo por quadro
				// preserva a duracao que o fornecedor autorou.
				FrameTime: 0.16,
			},
			AnimWalk: {
				SpritePath:  "assets/sprites/enemies/orc/walk.png",
				FrameWidth:  100,
				FrameHeight: 127,
				Columns:     10,
				Rows:        EnemyDirectionRows,
				FootLine:    124,
				// 0.156 nao e gosto: e o unico valor que faz o pe NAO deslizar,
				// e sai de uma medida. No perfil 090 o pe varre 63,5 px em
				// relacao ao torso entre o passo mais a frente e o mais atras;
				// a 1.6x isso e uma passada de 102 px na tela, e o ciclo de 20
				// quadros da o pacote sao dois passos, ou 204 px de chao. A 130
				// px/s o ciclo tem de durar 204/130 = 1,56 s, divididos pelos 10
				// quadros guardados.
				//
				// Por isso este numero e AMARRADO ao Speed e ao RenderScale. Se
				// algum dos dois mudar e este nao, o orc volta a patinar:
				// FrameTime = 2 * 63,5 * RenderScale / (Speed * 10).
				FrameTime: 0.156,
			},
		},

		Radius: EnemyOrcRadius,
		// Medido na arte: o corpo ocupa ~110 px da celula de origem e e
		// desenhado a 1.6x, ou ~176 px de altura na tela. O tronco fica em
		// torno de 55% dessa altura acima do pe, entao o centro sobe 90 px e o
		// raio abre para 70 - o circulo passa a cobrir de ~20 a ~160 px acima
		// do chao, que e o corpo. 75 de raio contra os 112 px de largura do
		// tronco: cobre o corpo sem engolir o espadao, que adiciona ~90 px de
		// lamina e fica FORA de proposito - um circulo que a engolisse deixaria
		// o jogador acertar ar ao lado do bicho.
		HitOffsetY:     -90,
		HitRadius:      75,
		Health:         600,
		Speed:          130,
		AttackDamage:   30,
		AttackRange:    70,
		AttackCooldown: 1.5,
		// O ORC ENXERGA MAIS LONGE QUE TODO MUNDO, e a razao e a mesma que fez
		// a Speed dele ficar em 130: ele e o mais lento do elenco, contra um
		// jogador que anda a 200. Notar junto com o lobo e nunca alcancar
		// ninguem — em jogo ele lia como um monstro distraido, nao como um
		// monstro pesado. Vendo a 3400 ele comeca a caminhar enquanto o grupo
		// ainda esta longe, e chega junto com a briga em vez de depois dela.
		Vision: 3400,
		Color:          "#6B4A3A",
	})
}

// ---------------------------------------------------------------------------
// Texture cache
// ---------------------------------------------------------------------------

// enemySheetKey identifies one loaded sheet. It is a pair and not just a type
// because a directional enemy owns one texture per animation.
type enemySheetKey struct {
	Type EnemyType
	Anim EnemyAnim
}

var (
	enemyTexMu    sync.Mutex
	enemyTextures = map[enemySheetKey]rl.Texture2D{}
)

// enemyTexture lazily loads and caches an enemy sheet. It must only be called
// from the render thread, which is where every Draw path already runs.
func enemyTexture(def EnemyDef, anim EnemyAnim) (rl.Texture2D, EnemyAnimDef, bool) {
	ad := def.AnimDef(anim)
	if ad.SpritePath == "" {
		return rl.Texture2D{}, ad, false
	}
	enemyTexMu.Lock()
	defer enemyTexMu.Unlock()

	// Keyed on the ANIMATION the def resolved to, not on the one that was
	// asked for: when a missing walk sheet falls back to idle, both lookups
	// must land on the same cache entry instead of loading idle twice.
	key := enemySheetKey{def.Type, anim}
	if !def.HasAnim(anim) && len(def.Anims) > 0 {
		key.Anim = AnimIdle
	}

	if tex, ok := enemyTextures[key]; ok {
		return tex, ad, tex.ID != 0
	}
	tex := rl.LoadTexture(assets.Path(ad.SpritePath))
	if tex.ID != 0 {
		ApplySpriteFilter(tex)
	}
	// Cached even on failure so a missing file does not retry every frame;
	// the caller falls back to the debug circle.
	enemyTextures[key] = tex
	return tex, ad, tex.ID != 0
}

// ApplySpriteFilter sets bilinear sampling on a sprite sheet.
//
// raylib defaults to point sampling, which is right for pixel art and wrong for
// this game: the sprites are painterly digital paintings, so nearest-neighbour
// resampling produces hard blocky edges. Frames are authored larger than they
// are drawn so that sampling is a reduction, and bilinear makes that reduction
// smooth instead of aliased.
func ApplySpriteFilter(tex rl.Texture2D) {
	rl.SetTextureFilter(tex, rl.FilterBilinear)
}

// UnloadEnemyTextures releases every cached enemy sheet. Call once on shutdown.
func UnloadEnemyTextures() {
	enemyTexMu.Lock()
	defer enemyTexMu.Unlock()
	for key, tex := range enemyTextures {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
		delete(enemyTextures, key)
	}
}

// ---------------------------------------------------------------------------
// Facing math
// ---------------------------------------------------------------------------

// radialAngleFor converts a velocity vector into the sprite rotation in
// degrees. The art's front points to the top of the frame (screen -Y), which
// is -90 degrees in atan2 terms, hence the +90 offset. Returns ok=false when
// the enemy is effectively still, so the caller keeps the previous angle
// instead of snapping to zero.
func radialAngleFor(vx, vy float32) (float32, bool) {
	if vx*vx+vy*vy < 1e-4 {
		return 0, false
	}
	// atan2 yields (-180, 180]; the +90 offset pushes it to (-90, 270], so wrap
	// it back into the canonical range the easing below assumes.
	return wrapDegrees(float32(math.Atan2(float64(vy), float64(vx))*180/math.Pi) + 90), true
}

// wrapDegrees normalizes an angle to (-180, 180].
func wrapDegrees(a float32) float32 {
	for a <= -180 {
		a += 360
	}
	for a > 180 {
		a -= 360
	}
	return a
}

// approachAngle moves current toward target by at most maxStep degrees, along
// the shortest arc. A slime that reverses direction should pivot, not snap:
// snapping reads as a one-frame teleport of the whole silhouette.
func approachAngle(current, target, maxStep float32) float32 {
	delta := wrapDegrees(target - current)
	if maxStep <= 0 || float32(math.Abs(float64(delta))) <= maxStep {
		return wrapDegrees(target)
	}
	if delta > 0 {
		return wrapDegrees(current + maxStep)
	}
	return wrapDegrees(current - maxStep)
}

// advanceEnemyAnim steps a looping frame cycle and returns the new frame and
// timer. Kept free of Enemy so the client-side tracker can reuse it.
func advanceEnemyAnim(frame int, timer, dt, frameTime float32, columns int) (int, float32) {
	if columns <= 0 {
		return 0, 0
	}
	if frameTime <= 0 {
		return frame % columns, 0
	}
	timer += dt
	for timer >= frameTime {
		timer -= frameTime
		frame++
	}
	return frame % columns, timer
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

// drawEnemySprite renders one frame of an enemy sheet centered on (x, y) and
// rotated by angle degrees about that point.
func drawEnemySprite(tex rl.Texture2D, ad EnemyAnimDef, renderScale, x, y float32, frame int, angle float32) {
	frame = ad.Column(frame)

	fw := float32(ad.FrameWidth)
	fh := float32(ad.FrameHeight)
	scale := renderScale
	if scale <= 0 {
		scale = 1
	}

	source := rl.NewRectangle(float32(frame)*fw, 0, fw, fh)
	// With a non-zero origin, DrawTexturePro treats dest x/y as the pivot, so
	// the destination rectangle is positioned at the world point directly and
	// the origin is half the SCALED size.
	dest := rl.NewRectangle(x, y, fw*scale, fh*scale)
	origin := rl.NewVector2(fw*scale/2, fh*scale/2)

	rl.DrawTexturePro(tex, source, dest, origin, angle, rl.White)
}

// enemyGroundOffset is how far below Position the enemy's soles are drawn.
//
// Same rule as GroundOffset for characters, and it exists for the same reason:
// a sprite drawn centred on Position puts a standing figure's feet a hundred
// pixels below the point the game is colliding with, so the monster walks with
// its shins inside a tree while the collision box has already cleared it.
//
// Returns zero for radial enemies, which have no soles and stay centred.
func enemyGroundOffset(ad EnemyAnimDef, renderScale float32) float32 {
	if ad.FootLine <= 0 || ad.FrameHeight <= 0 {
		return 0
	}
	scale := renderScale
	if scale <= 0 {
		scale = 1
	}
	return (float32(ad.FootLine) - float32(ad.FrameHeight)/2) * scale
}

// drawEnemyFixed renders one frame of a fixed-view sheet with the creature
// standing on (x, y).
//
// It is drawEnemyDirectional without the row and the mirror, and drawEnemySprite
// without the rotation. Keeping it as its own function is the point: the two
// things this mode must never do - spin, or pick a facing - are absent here
// rather than passed in as zero, so they cannot come back by accident.
func drawEnemyFixed(tex rl.Texture2D, ad EnemyAnimDef, renderScale, x, y float32, frame int) {
	frame = ad.Column(frame)

	fw := float32(ad.FrameWidth)
	fh := float32(ad.FrameHeight)
	scale := renderScale
	if scale <= 0 {
		scale = 1
	}

	source := rl.NewRectangle(float32(frame)*fw, 0, fw, fh)
	// Lift the frame so its FootLine lands on y, instead of its centre: the
	// gargoyle's claws have to meet the pedestal the map draws underneath.
	dest := rl.NewRectangle(x, y-enemyGroundOffset(ad, renderScale), fw*scale, fh*scale)
	origin := rl.NewVector2(fw*scale/2, fh*scale/2)

	rl.DrawTexturePro(tex, source, dest, origin, 0, rl.White)
}

// drawEnemyDirectional renders one frame of a directional sheet with the
// figure standing on (x, y): the row comes from the heading, the art is never
// rotated, and mirror flips the half of the facings the sheet does not store.
func drawEnemyDirectional(tex rl.Texture2D, ad EnemyAnimDef, renderScale, x, y float32, frame, row int, mirror bool) {
	frame = ad.Column(frame)
	row = validEnemyRow(row, ad.Rows)

	fw := float32(ad.FrameWidth)
	fh := float32(ad.FrameHeight)
	scale := renderScale
	if scale <= 0 {
		scale = 1
	}

	source := rl.NewRectangle(float32(frame)*fw, float32(row)*fh, fw, fh)
	if mirror {
		// A negative source width is how raylib flips a region. It mirrors
		// about the middle of the frame, which is the pivot only because
		// build_orc.py forces the crop window to be symmetric about it.
		source.Width = -fw
	}

	// Lift the frame so its FootLine lands on y, instead of its centre.
	dest := rl.NewRectangle(x, y-enemyGroundOffset(ad, renderScale), fw*scale, fh*scale)
	origin := rl.NewVector2(fw*scale/2, fh*scale/2)

	// Rotation is hard-zero here, not a parameter. Turning a directional sheet
	// is the bug the mode exists to prevent, so there is nothing to pass in.
	rl.DrawTexturePro(tex, source, dest, origin, 0, rl.White)
}

// ---------------------------------------------------------------------------
// Client-side animation tracker
// ---------------------------------------------------------------------------

// Enemies arriving over the network carry position but no velocity, so the
// client derives facing from the position delta and runs the pulse cycle on a
// local clock. This keeps the wire format unchanged.
type remoteEnemyAnim struct {
	lastX, lastY float32
	targetAngle  float32
	angle        float32
	haveAngle    bool
	frame        int
	timer        float32
	lastSeen     float64
	row          int
	mirror       bool
	anim         EnemyAnim
	// stillFor is how long the enemy has been sitting at the same position, in
	// seconds. It exists because snapshots arrive slower than frames: on most
	// frames the delta is exactly zero even while the monster is walking, so
	// "did not move this frame" cannot mean "stopped". Only a gap longer than
	// the update interval does.
	stillFor float32
}

// remoteStillTimeout is how long a network enemy has to hold still before the
// client believes it stopped. Comfortably longer than the gap between
// snapshots, so a walking monster never flickers into its idle pose.
const remoteStillTimeout = 0.35

var (
	remoteAnimMu sync.Mutex
	remoteAnims  = map[string]*remoteEnemyAnim{}
)

// remoteAnimTTL drops trackers for enemies that stopped being drawn (died or
// went out of sync) so the map does not grow without bound.
const remoteAnimTTL = 5.0

func trackRemoteEnemy(id string, x, y float32, def EnemyDef, dt float32, facing FacingTarget) remoteEnemyAnim {
	remoteAnimMu.Lock()
	defer remoteAnimMu.Unlock()

	now := rl.GetTime()
	st, ok := remoteAnims[id]
	if !ok {
		// stillFor starts already expired so a monster seen for the first time
		// is drawn standing. Starting at zero would make every enemy that
		// scrolls into view play half a second of walking before settling.
		st = &remoteEnemyAnim{lastX: x, lastY: y, stillFor: remoteStillTimeout}
		remoteAnims[id] = st
		// lastSeen must be stamped before pruning, otherwise the entry just
		// inserted looks stale (lastSeen == 0) and is dropped on the same call,
		// which would rebuild the tracker every frame and freeze the cycle.
		st.lastSeen = now
		pruneRemoteAnimsLocked(now)
	}
	st.lastSeen = now

	// Network updates arrive slower than frames, so the delta is zero on most
	// frames. Only refresh the target when the enemy actually moved.
	dx, dy := x-st.lastX, y-st.lastY
	if dx*dx+dy*dy > 0.25 {
		if a, ok := radialAngleFor(dx, dy); ok {
			st.targetAngle = a
			if !st.haveAngle {
				st.angle = a
				st.haveAngle = true
			}
		}
		st.lastX, st.lastY = x, y
		st.stillFor = 0
	} else {
		st.stillFor += dt
	}

	// Facing follows the target, not the delta - the same rule the host uses,
	// and for the same reason: the delta between snapshots already carries the
	// separation push and the wall slide, so reading the row from it would put
	// the orc's back to the player exactly when the host has it facing forward.
	if facing.OK {
		if r, m, ok := enemyRowForHeading(facing.Position.X-x, facing.Position.Y-y); ok {
			st.row, st.mirror = r, m
		}
	}

	// Walking is inferred from the snapshots, not carried in them: EnemyState
	// has no animation field yet, and adding one for a state the client can
	// derive would cost bandwidth per enemy per snapshot. It stops being enough
	// the moment an attack exists - a client cannot guess a swing - and that is
	// when the protocol grows (doc/plan_orc_guarnicao.md, Fase 4).
	next := AnimIdle
	if def.HasAnim(AnimWalk) && st.stillFor < remoteStillTimeout {
		next = AnimWalk
	}
	if next != st.anim {
		st.anim = next
		st.frame, st.timer = 0, 0
	}

	if st.haveAngle {
		st.angle = approachAngle(st.angle, st.targetAngle, def.TurnRate*dt)
	}
	ad := def.AnimDef(st.anim)
	st.frame, st.timer = advanceEnemyAnim(st.frame, st.timer, dt, ad.FrameTime, ad.Columns)
	return *st
}

func pruneRemoteAnimsLocked(now float64) {
	for id, st := range remoteAnims {
		if now-st.lastSeen > remoteAnimTTL {
			delete(remoteAnims, id)
		}
	}
}

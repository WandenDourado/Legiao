package entity

import (
	"crypto/rand"
	"encoding/base64"
	"math"
	"time"

	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/nav"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// MoveEnv is what an enemy's movement needs to know about the map: Solid
// resolves a single step (collision.Resolve/ResolveDetour); Nav is the
// walkability mesh a stuck enemy plans a route across when the direct
// approach and local detouring both keep failing (moveTowardTarget's
// non-progress gate). Nav may be nil — "no mesh built yet" falls back to
// direct pursuit only, same as a nil Solid means "no map".
type MoveEnv struct {
	Solid collision.Solid
	Nav   *nav.Grid
}

// EnemyType represents different types of enemies for future extensibility.
type EnemyType string

const (
	// EnemyTypeBasic is the slime: slow, tough, amorphous.
	EnemyTypeBasic EnemyType = "basic"
	// EnemyTypeFast is the wolf: quick and fragile.
	EnemyTypeFast EnemyType = "fast"
	// EnemyTypeGarrison is the orc: slow, heavy, and the first enemy drawn
	// from a directional sheet instead of a rotated top-down one.
	EnemyTypeGarrison EnemyType = "garrison"
	// EnemyTypeCastleSentry is the stationary ranged creature anchored on the
	// inaccessible stream islands of map 4. Its damage is resolved by the same
	// host combat path as every enemy, but mundane projectiles cannot hurt it.
	EnemyTypeCastleSentry EnemyType = "castle_sentry"
	// EnemyTypeDarkLady e a Senhora das Trevas, chefe final do world_05: fixa
	// num canto, sem locomocao, com danca de conjuracao e golpe telegrafado.
	EnemyTypeDarkLady EnemyType = "dark_lady"
)

// PlayerState is a minimal version for enemy AI - defined here to avoid import cycle.
type PlayerState struct {
	PlayerID  string
	X         int
	Y         int
	Color     string
	Health    float32
	MaxHealth float32
	IsDead    bool
}

// Enemy represents an enemy entity in the game.
type Enemy struct {
	ID             string
	Type           EnemyType
	Position       rl.Vector2
	Velocity       rl.Vector2
	Health         float32
	MaxHealth      float32
	Speed          float32
	Radius         float32
	Color          string // Blood red #8B0000 for differentiation from players
	AttackDamage   float32
	AttackRange    float32
	AttackCooldown float32
	attackTimer    float32
	IsActive       bool
	// slowFactor multiplies Speed while slowTTL > 0 (e.g., 0.45 = 55% slower).
	slowFactor float32
	slowTTL    float32

	// detourDir e a DIRECAO NO MUNDO que este monstro escolheu para contornar o
	// obstaculo a frente; o vetor nulo quer dizer "caminho livre". Sem ela um
	// monstro encostado numa cerca reescolhe o lado a cada quadro e vibra
	// parado em vez de dar a volta.
	//
	// Isto era um SINAL (-1/+1) aplicado a uma rotacao de 90 graus do rumo, e
	// era esse o defeito: o rumo gira enquanto o monstro caminha, entao o mesmo
	// sinal apontava para lados diferentes do mundo ao longo do contorno. Ver
	// collision.ResolveDetour.
	detourDir rl.Vector2

	// follower drives the mesh-planning escalation
	// (doc/plan_navegacao_bots_monstros.md §4): the monster invests straight
	// ahead (with the local detour above) as normal, and only once that
	// keeps failing to close real distance on the target for
	// FoeStuckBefore seconds does it consult the navigation mesh.
	//
	// headwayWindow tracks that over a ROLLING WINDOW of straight-line
	// distance, not collision.Progressed's per-step dot product: the local
	// detour's own "still counts as progress" threshold (15% of the
	// intended step) is lenient enough that an agent sliding along an
	// obstacle's face — making real headway on NEITHER axis toward a gap
	// hundreds of pixels away — can keep reporting per-step progress
	// forever. A window over actual distance-to-target cannot be fooled by
	// that: it only cares whether the agent is any closer than it was
	// FoeStuckBefore seconds ago.
	follower        nav.Follower
	trackingHeadway bool
	windowStartDist float32
	windowElapsed   float32

	// Sprite animation state. AnimFrame indexes the looping cycle; Angle is
	// the current sprite rotation in degrees for radial-mode enemies, eased
	// toward the velocity heading instead of snapping to it.
	AnimFrame   int
	Angle       float32
	animTimer   float32
	haveAngle   bool
	targetAngle float32

	// Facing for directional-mode enemies. Row indexes the sheet's stored
	// facings and Mirror says whether that row is drawn flipped. They are kept
	// across frames on purpose: an orc that stops moving must keep looking
	// where it was looking, not snap back to row zero.
	Row     int
	Mirror  bool
	haveRow bool

	// Facing is the direction the enemy wants to LOOK, which is deliberately
	// NOT the direction it ends up moving.
	//
	// The two used to be the same thing, and for the slime and the wolf they
	// still are. For an upright humanoid they cannot be, because three separate
	// forces bend the actual velocity away from the target:
	//
	//   - separation steering, which with twenty orcs converging on one player
	//     can point a squeezed monster sideways or straight backwards;
	//   - the collision resolver, which slides the step along a barricade;
	//   - the detour logic, which commits to walking around a tree.
	//
	// All three are right about MOVEMENT and wrong about attention. A wolf that
	// turns its whole body to run along a fence reads correctly - a quadruped
	// does that. An orc that turns its back on the player to squeeze past
	// another orc reads as broken.
	//
	// So a directional enemy keeps its eyes on the target and side-steps. Cost
	// of the choice: while separation or a wall dominates, the walk cycle plays
	// while the body drifts sideways. That is a far cheaper artifact than a
	// monster walking away from the person it is hunting.
	Facing rl.Vector2

	// Anim is which sheet is playing. The zero value is the empty string, so
	// it is normalised to AnimIdle on first use rather than in a constructor:
	// enemies arrive from NewEnemy and from network snapshots, and only one of
	// those runs a constructor.
	Anim EnemyAnim

	// boss e o estado da maquina de animacao de chefe. Zero para todo mundo que
	// nao e chefe, e nunca lido nesse caso - ver EnemyDef.IsBoss.
	boss bossAnimState

	// Guard is the post and sector this monster defends. The zero value means
	// it defends nothing and hunts the nearest player anywhere, which is what
	// maps 1 and 2 want. See enemy_territory.go.
	Guard Guard
	// chasing diz que este monstro ja levantou os olhos e foi atras de alguem.
	//
	// Uma vez verdadeiro, o setor e o raio de visao param de valer: eles sao
	// perguntas de AQUISICAO. So volta a falso quando nao sobra ninguem vivo
	// para perseguir.
	//
	// Aqui moravam `chaseTTL` e `chaseSpent`, o prazo da perseguicao e a marca
	// de prazo vencido. Os dois sairam porque o teto que eles impunham era, na
	// pratica, uma saida gratuita: o jogador anda a 200 e o orc a 130, entao
	// recuar alguns passos vencia qualquer prazo ou coleira.
	chasing bool
	// patrolForward e patrolWait sao o vai-e-vem: para que ponta ele esta indo
	// e quanto ainda espera na ponta em que chegou.
	patrolForward bool
	patrolWait    float32
}

// enemyWalkThreshold is the speed below which an enemy is treated as standing
// still, in world units per second.
//
// It is not zero because Velocity keeps its last value while the monster is
// pinned against a fence - the resolver refuses the step but the intent is
// still there. Pressing into an obstacle SHOULD keep playing the walk cycle;
// what this number filters out is the residue of separation steering, which is
// a fraction of a unit.
const enemyWalkThreshold = 4.0

// NewEnemy creates a new enemy with the given type at position (x, y).
// Every stat comes from the type's EnemyDef, so balancing an enemy means
// editing its registration and nothing else.
func NewEnemy(enemyType EnemyType, x, y float32) *Enemy {
	def := GetEnemyDef(enemyType)
	return &Enemy{
		ID:             generateID(),
		Type:           enemyType,
		Position:       rl.NewVector2(x, y),
		Health:         def.Health,
		MaxHealth:      def.Health,
		Speed:          def.Speed,
		Radius:         def.Radius,
		Color:          def.Color,
		AttackDamage:   def.AttackDamage,
		AttackRange:    def.AttackRange,
		AttackCooldown: def.AttackCooldown,
		attackTimer:    0,
		IsActive:       true,
	}
}

// Update updates the enemy's AI and movement based on the nearest player.
// neighbors are the other active enemies used for separation steering; pass
// nil to disable it. env is the map's blocked space and navigation mesh,
// which movement is resolved against exactly like the player's - pass a
// zero-value MoveEnv for no map.
// Returns true if the enemy is in range and can attack.
func (e *Enemy) Update(dt float32, players []PlayerState, neighbors []*Enemy, env MoveEnv) bool {
	if !e.IsActive {
		return false
	}

	if e.slowTTL > 0 {
		e.slowTTL -= dt
	}

	// O ALVO PASSA PELO TERRITORIO. Sem guarda declarado isto e exatamente
	// FindNearestPlayer, entao mapa 1 e mapa 2 nao mudam de comportamento.
	nearest := e.guardTarget(players)
	if nearest == nil && e.Guard.Active() {
		// Ninguem invadindo: PATRULHAR. Sem curar - recuar e voltar e desgaste
		// legitimo neste mapa, e o dano que o monstro levou fica com ele.
		if to, walking := e.patrolStep(dt); walking {
			e.moveTowardTarget(to, dt, e.separationFrom(neighbors), env)
		} else {
			// Parado na ponta do trecho. Velocidade zerada de proposito: e o
			// que devolve a animacao parada em vez de um passo residual.
			e.Velocity = rl.Vector2{}
		}
		e.updateAnimation(dt)
		return false
	}
	if nearest == nil {
		// Sem alvo nao ha rumo. Zerar a velocidade e o que devolve o orc para
		// a animacao parada: sem isto ele guardaria a velocidade do ultimo
		// quadro em que perseguia alguem e continuaria caminhando parado para
		// sempre. O slime e o lobo nao mudam de aparencia com isso - velocidade
		// zero faz radialAngleFor recusar, e eles mantem o angulo que tinham.
		//
		// Facing NAO e zerado: quem para de perseguir continua olhando para
		// onde olhava, em vez de virar de costas para a camera.
		e.Velocity = rl.Vector2{}
		// A pulsacao parada corre mesmo sem ninguem para perseguir, entao a
		// animacao avanca antes deste retorno.
		e.updateAnimation(dt)
		return false
	}

	targetPos := rl.NewVector2(float32(nearest.X), float32(nearest.Y))
	e.moveTowardTarget(targetPos, dt, e.separationFrom(neighbors), env)

	// DEPOIS de mover, nao antes. A animacao le o que o movimento acabou de
	// decidir - direcao, velocidade e quadro pertencem todos a este quadro. Com
	// a ordem invertida, cada mudanca de rumo aparecia um quadro atrasada e um
	// monstro recem-nascido passava o primeiro quadro virado de costas, porque
	// Facing ainda era o vetor nulo.
	e.updateAnimation(dt)

	// Check if in attack range and update attack timer
	if e.IsInAttackRange(nearest) {
		e.attackTimer -= dt
		if e.attackTimer <= 0 {
			e.attackTimer = e.AttackCooldown
			// O ataque renova a perseguicao: quem esta acertando nao desiste.
			e.NoteAttack()
			return true // Enemy attacks this frame
		}
	}
	return false
}

// HitCenter e o centro da caixa de acerto, que NAO e a posicao do monstro.
//
// A posicao e a ancora - o pe, para quem tem `FootLine` - porque e ela que
// resolve movimento, colisao com o mapa e ordem de desenho. Acerto e outra
// pergunta: ela mira o corpo. Misturar as duas foi o que fez o orc so ser
// atingivel nos pes.
//
// Todo teste de acerto usa isto; movimento e IA continuam usando Position.
func (e *Enemy) HitCenter() rl.Vector2 {
	off := GetEnemyDef(e.Type).HitOffsetY
	if off == 0 {
		return e.Position
	}
	return rl.NewVector2(e.Position.X, e.Position.Y+off)
}

// HitRadius e o raio da caixa de acerto, que pode ser maior que o corpo.
//
// Zero no def quer dizer "usa o corpo": e o caso do slime e do lobo, desenhados
// centrados na propria posicao.
func (e *Enemy) HitRadius() float32 {
	if r := GetEnemyDef(e.Type).HitRadius; r > 0 {
		return r
	}
	return e.Radius
}

// FindNearestPlayer returns the closest living player, or nil if none.
func (e *Enemy) FindNearestPlayer(players []PlayerState) *PlayerState {
	var nearest *PlayerState
	minDist := float32(math.MaxFloat32)

	for _, p := range players {
		if p.IsDead {
			continue
		}
		playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
		dist := rl.Vector2Distance(e.Position, playerPos)
		if dist < minDist {
			minDist = dist
			nearest = &p
		}
	}
	return nearest
}

// Separation tuning. Enemies that pile up on the same pixel read as a single
// smeared blob, so they steer away from each other while chasing and any
// residual overlap is resolved positionally afterwards.
const (
	// enemySeparationRange scales the sum of two radii into the distance at
	// which enemies start pushing each other away. Slightly above 1 so they
	// begin spreading just before the sprites touch.
	enemySeparationRange = 1.15
	// enemySeparationWeight is how hard separation bends the chase direction.
	// Below ~2 the enemy still closes on the player instead of orbiting.
	enemySeparationWeight = 1.6
	// enemyMaxPushPerFrame caps positional correction so a tight pile
	// unstacks over a few frames rather than teleporting apart.
	enemyMaxPushPerFrame = 6.0
	// enemyOverlapIterations relaxes the separation constraints repeatedly.
	// Resolving a pair can push one of them into a third, so a single sweep
	// leaves residual overlap when many enemies converge at once. Measured on
	// 12 enemies closing on one player: 1 sweep leaves 7.1 px of overlap,
	// 2 leaves 2.6 px, 3 leaves 1.5 px, and 4 leaves 1.0 px. Three is the knee
	// of that curve, at 570 pair checks per frame for the 20-enemy cap.
	enemyOverlapIterations = 3
)

// separationFrom returns an unnormalized push away from every neighbor closer
// than the separation range, weighted by how deep the overlap is.
func (e *Enemy) separationFrom(neighbors []*Enemy) rl.Vector2 {
	push := rl.Vector2{}
	for _, other := range neighbors {
		if other == nil || other == e || !other.IsActive {
			continue
		}
		minDist := (e.Radius + other.Radius) * enemySeparationRange
		if minDist <= 0 {
			continue
		}
		delta := rl.Vector2Subtract(e.Position, other.Position)
		dist := rl.Vector2Length(delta)
		if dist >= minDist {
			continue
		}
		if dist < 0.001 {
			// Exactly coincident: without a tiebreak both would compute a zero
			// push and stay locked together forever. Split them along X using
			// their IDs so the two enemies pick opposite directions.
			sign := float32(1)
			if e.ID < other.ID {
				sign = -1
			}
			push.X += sign * enemySeparationWeight
			continue
		}
		// Closer neighbors push harder; divide by dist to normalize delta.
		weight := (minDist - dist) / minDist
		push = rl.Vector2Add(push, rl.Vector2Scale(delta, weight/dist))
	}
	return push
}

// MoveTowardTarget moves the enemy toward the target position, respecting the
// map's solid obstacles (and, once it has spent FoeStuckBefore seconds not
// getting anywhere, routing around them via env.Nav).
func (e *Enemy) MoveTowardTarget(target rl.Vector2, dt float32, env MoveEnv) {
	e.moveTowardTarget(target, dt, rl.Vector2{}, env)
}

// moveTowardTarget steers toward the target, blending in a separation push.
// The blend happens before normalization so the enemy always travels at full
// speed; separation changes its heading, never how fast it closes in.
//
// The resulting step goes through the shared collision resolver instead of
// being added straight to Position, which is what stops monsters from walking
// through trees and fences. When the map deflects the step, Velocity is
// realigned onto the distance actually covered, so a RADIAL sprite turns to
// face where the enemy is really going rather than where it wanted to go.
//
// Facing is captured separately and earlier, because a directional sprite
// wants the opposite answer. See the Facing field.
//
// The heading itself comes from the mesh instead of straight at target only
// once the direct approach (plus the local per-step detour in e.step) has
// spent FoeStuckBefore seconds not closing real distance on the target, or
// a route is already committed to — a monster invests directly by default,
// exactly like before this existed, and only "looks for a way around" once
// hitting something proves the direct approach is not working
// (doc/plan_navegacao_bots_monstros.md §4).
func (e *Enemy) moveTowardTarget(target rl.Vector2, dt float32, separation rl.Vector2, env MoveEnv) {
	toTarget := rl.Vector2Subtract(target, e.Position)
	length := rl.Vector2Length(toTarget)
	if length <= 0 {
		return
	}
	direct := rl.Vector2Scale(toTarget, 1.0/length)

	// Taken HERE, and the position in this function is the whole point: after
	// normalising toward the target, but before separation blends in and before
	// the resolver deflects anything.
	e.Facing = direct

	heading := direct
	if env.Nav != nil && (e.follower.Active() || e.notMakingHeadway(length, dt)) {
		heading, _ = e.follower.Desired(env.Nav, e.Position, target, dt, nav.FoeReplanEvery)
	}

	if separation.X != 0 || separation.Y != 0 {
		heading = rl.Vector2Add(heading, rl.Vector2Scale(separation, enemySeparationWeight))
		if blended := rl.Vector2Length(heading); blended > 0 {
			heading = rl.Vector2Scale(heading, 1.0/blended)
		}
	}

	speed := e.Speed
	if e.slowTTL > 0 && e.slowFactor > 0 {
		speed *= e.slowFactor
	}
	e.Velocity = rl.Vector2Scale(heading, speed)
	delta := rl.Vector2Scale(e.Velocity, dt)
	moved := e.step(delta, env.Solid)

	if travelled := rl.Vector2Length(moved); travelled > 0.001 {
		e.Velocity = rl.Vector2Scale(moved, speed/travelled)
	}
}

// Route returns this enemy's current mesh-planned waypoints, for the F4
// debug overlay only — nothing on any gameplay path reads this.
func (e *Enemy) Route() []rl.Vector2 {
	return e.follower.Path()
}

// notMakingHeadway reports whether the straight-line distance to the
// target has failed to close by a meaningful amount over the last
// FoeStuckBefore seconds — see the follower field's doc comment for why
// this is a window over real distance rather than a single step's
// dot-product test. distNow is the distance measured THIS call; the first
// call ever just starts the window instead of judging anything (there is
// nothing to compare against yet).
func (e *Enemy) notMakingHeadway(distNow, dt float32) bool {
	if !e.trackingHeadway {
		e.trackingHeadway = true
		e.windowStartDist = distNow
		e.windowElapsed = 0
		return false
	}
	e.windowElapsed += dt
	if e.windowElapsed < nav.FoeStuckBefore {
		return false
	}
	// Must close at least a quarter of the ground a clear run would cover
	// in one window — generous enough that ordinary slow-and-steady combat
	// approach (already slowed, sidestepping a neighbour) never trips it,
	// tight enough that grinding along a wall's face does.
	closed := e.windowStartDist - distNow
	stuck := closed < e.Speed*nav.FoeStuckBefore*0.25
	e.windowStartDist = distNow
	e.windowElapsed = 0
	return stuck
}

// ResolveEnemyOverlap pushes still-overlapping pairs apart after everyone has
// moved. Steering alone cannot guarantee separation - enemies converging on
// the same player from opposite sides will still meet - so this runs as a
// final pass. It moves Position only, leaving Velocity (and therefore the
// sprite's facing) untouched, so the correction is invisible in the animation.
// The pushes run through the map resolver too: unstacking a pile must not be
// able to shove an enemy inside a tree that movement just kept it out of.
func ResolveEnemyOverlap(enemies []*Enemy, solid collision.Solid) {
	for iteration := 0; iteration < enemyOverlapIterations; iteration++ {
		resolved := true
		for i := 0; i < len(enemies); i++ {
			a := enemies[i]
			if a == nil || !a.IsActive {
				continue
			}
			for j := i + 1; j < len(enemies); j++ {
				b := enemies[j]
				if b == nil || !b.IsActive {
					continue
				}
				minDist := a.Radius + b.Radius
				if minDist <= 0 {
					continue
				}
				delta := rl.Vector2Subtract(b.Position, a.Position)
				dist := rl.Vector2Length(delta)
				if dist >= minDist {
					continue
				}
				resolved = false

				var dir rl.Vector2
				if dist < 0.001 {
					// Exactly coincident: pick opposite directions from the IDs
					// so the pair separates instead of staying locked.
					dir = rl.NewVector2(1, 0)
					if a.ID < b.ID {
						dir = rl.NewVector2(-1, 0)
					}
				} else {
					dir = rl.Vector2Scale(delta, 1.0/dist)
				}

				overlap := minDist - dist
				if overlap > enemyMaxPushPerFrame {
					overlap = enemyMaxPushPerFrame
				}
				// QUEM NAO ANDA NAO E EMPURRADO. Dividir a correcao meio a
				// meio pressupoe dois corpos que se movem, e o jogo tem dois
				// que nao: a gargula e a chefe. Com o meio a meio, trinta
				// inimigos caminhando contra a Senhora das Trevas a arrastavam
				// para fora da ancora — de longe parecia que ela estava
				// PERSEGUINDO o jogador, quando na verdade estava sendo
				// empurrada. Uma torre de `Speed 0` e cenario com vida: o
				// corpo movel absorve a correcao inteira.
				aFixed, bFixed := a.Speed <= 0, b.Speed <= 0
				switch {
				case aFixed && bFixed:
					// Dois imoveis sobrepostos so acontece por postos mal
					// declarados no mapa. Ninguem cede: mexer num deles
					// mudaria de lugar algo que o mapa fixou de proposito.
				case aFixed:
					b.push(rl.Vector2Scale(dir, overlap), solid)
				case bFixed:
					a.push(rl.Vector2Scale(dir, -overlap), solid)
				default:
					// Each side takes half the correction, so neither enemy is
					// privileged and a symmetric pile expands evenly.
					shift := rl.Vector2Scale(dir, overlap/2)
					a.push(rl.Vector2Scale(shift, -1), solid)
					b.push(shift, solid)
				}
			}
		}
		if resolved {
			return
		}
	}
}

// IsInAttackRange returns true if the enemy is close enough to attack.
func (e *Enemy) IsInAttackRange(player *PlayerState) bool {
	playerPos := rl.NewVector2(float32(player.X), float32(player.Y))
	dist := rl.Vector2Distance(e.Position, playerPos)
	return dist <= e.AttackRange+e.Radius
}

// ApplySlow reduces the enemy's movement speed to factor (0..1 multiplier)
// for duration seconds. A stronger (lower) factor overrides a weaker one;
// re-applying refreshes the remaining time.
func (e *Enemy) ApplySlow(factor, duration float32) {
	if e.slowTTL <= 0 || factor < e.slowFactor {
		e.slowFactor = factor
	}
	if duration > e.slowTTL {
		e.slowTTL = duration
	}
}

// TakeDamage applies damage to the enemy. Returns true if the enemy died.
func (e *Enemy) TakeDamage(damage float32) bool {
	// LEVAR DANO E NOTAR. A porta de ameaca (enemy_territory.go) acorda o posto
	// quando alguem chega ao alcance de acerta-lo; esta e a garantia de que
	// nenhum caminho de dano escapa dela — magia de area, flecha celestial,
	// espectro, um tiro de um angulo que a geometria nao previu. Se o golpe
	// chegou, o guarda foi encontrado, e um guarda encontrado vai atras.
	//
	// Aqui, e nao em cada origem de dano, porque este e o funil por onde todo
	// dano a inimigo passa — e porque um caminho novo de dano nao pode depender
	// de alguem lembrar de avisar a IA.
	if e.Guard.Active() {
		e.chasing = true
	}
	e.Health -= damage
	if e.Health <= 0 {
		e.Health = 0
		e.IsActive = false
		return true
	}
	return false
}

// updateAnimation advances the looping pulse cycle and eases the sprite
// rotation toward the current heading. The angle is read from the velocity of
// the previous frame (movement is resolved later in Update); at 60 fps that
// single-frame lag is not visible, and it keeps the facing stable when the
// enemy has no target and stops moving.
func (e *Enemy) updateAnimation(dt float32) {
	def := GetEnemyDef(e.Type)

	// Chefe fixo tem maquina propria: enemyAnimFor decide por VELOCIDADE, e a
	// Senhora das Trevas tem Speed 0 - ela ficaria em idle para sempre.
	if def.IsBoss() {
		e.updateBossAnimation(def, dt)
		return
	}

	// Which sheet plays comes first, because the frame counter below belongs to
	// that sheet: idle has 8 frames and walk has 10, so advancing the counter
	// against the wrong one would land on a frame that does not exist.
	if next := enemyAnimFor(def, e.Velocity); next != e.Anim {
		e.Anim = next
		// Restart the cycle on a real state change. A walk that resumed at
		// frame 7 would begin mid-stride, with the wrong foot forward.
		e.AnimFrame, e.animTimer = 0, 0
	}

	ad := def.AnimDef(e.Anim)
	e.AnimFrame, e.animTimer = advanceEnemyAnim(e.AnimFrame, e.animTimer, dt, ad.FrameTime, ad.Columns)

	// Directional sheets pick a row and are never rotated. Falling through to
	// the easing below would spin an upright, armoured body around its own
	// centre, which is the exact failure the two modes exist to keep apart.
	if def.Mode == EnemyModeDirectional {
		// Facing, not Velocity: the orc looks at its target even while
		// separation or a barricade is pushing it somewhere else. Facing is
		// deliberately NOT cleared when the enemy loses its target, so a
		// monster that stops keeps staring where it last looked.
		if row, mirror, ok := enemyRowForHeading(e.Facing.X, e.Facing.Y); ok {
			e.Row, e.Mirror, e.haveRow = row, mirror, true
		}
		return
	}

	// O modo fixo nao tem para onde olhar: uma sentinela parada nao escolhe
	// linha nem angulo. Sair aqui deixa Angle em zero para sempre, que e o que
	// drawEnemyFixed espera.
	if def.Mode != EnemyModeRadial {
		return
	}
	if angle, ok := radialAngleFor(e.Velocity.X, e.Velocity.Y); ok {
		e.targetAngle = angle
		if !e.haveAngle {
			e.Angle = angle
			e.haveAngle = true
		}
	}
	if e.haveAngle {
		e.Angle = approachAngle(e.Angle, e.targetAngle, def.TurnRate*dt)
	}
}

// Draw renders the enemy from its sprite sheet, rotated toward its heading.
// Falls back to the debug circle when the sheet is missing.
func (e *Enemy) Draw() {
	if !e.IsActive {
		return
	}
	def := GetEnemyDef(e.Type)
	if tex, ad, ok := enemyTexture(def, e.Anim); ok {
		switch def.Mode {
		case EnemyModeDirectional:
			drawEnemyDirectional(tex, ad, def.RenderScale, e.Position.X, e.Position.Y, e.AnimFrame, e.Row, e.Mirror)
		case EnemyModeFixed:
			// bossVisualAdvanceY nao mexe em Position: so o desenho anda.
			drawEnemyFixed(tex, ad, def.RenderScale, e.Position.X, e.Position.Y+e.bossVisualAdvanceY(def), e.AnimFrame)
		default:
			drawEnemySprite(tex, ad, def.RenderScale, e.Position.X, e.Position.Y, e.AnimFrame, e.Angle)
		}
		return
	}
	col := hexToColor(e.Color)
	rl.DrawCircleV(e.Position, e.Radius, col)
	rl.DrawCircleLinesV(e.Position, e.Radius, rl.Fade(rl.Black, 0.5))
}

// DrawHealthBar draws a small health bar above the enemy.
func (e *Enemy) DrawHealthBar() {
	if !e.IsActive {
		return
	}
	// O chefe tem barra propria no HUD (ui/boss_bar.go). Duas barras para a
	// mesma criatura e ruido, e a flutuante seria a menos legivel das duas: com
	// trinta inimigos em campo ela e uma entre trinta.
	if GetEnemyDef(e.Type).IsBoss() {
		return
	}
	above, halfWidth := enemyBarLayout(GetEnemyDef(e.Type), e.Anim, e.Radius)
	drawEnemyHealthBar(e.Position.X, e.Position.Y, e.Health, e.MaxHealth, above, halfWidth)
}

// enemyBarLayout returns how far above Position the health bar sits and how
// wide half of it is.
//
// The two used to be the same number, which worked while every enemy was a
// blob drawn centred on Position. A directional enemy breaks that twice over:
// Position is at its FEET, so the bar has to clear a whole standing figure
// instead of half a sprite; and the figure is much taller than it is wide, so
// deriving the width from the height would draw a health bar wider than the
// orc is tall.
func enemyBarLayout(def EnemyDef, anim EnemyAnim, radius float32) (above, halfWidth float32) {
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	ad := def.AnimDef(anim)

	if (def.Mode == EnemyModeDirectional || def.Mode == EnemyModeFixed) && ad.FootLine > 0 {
		// Everything drawn sits between the soles and FootLine pixels above
		// them, so clearing FootLine clears the sprite including a raised
		// weapon - or, for the gargoyle, a wing at the top of its stroke.
		// Width follows the hitbox, which follows the torso.
		return float32(ad.FootLine) * scale, radius * 1.1
	}

	// Radial enemies keep exactly the geometry they had: the bar clears ~68% of
	// the scaled frame, and its width is twice that same number.
	offset := radius
	if ad.SpritePath != "" && ad.FrameHeight > 0 {
		if visible := float32(ad.FrameHeight) * scale * 0.34; visible > offset {
			offset = visible
		}
	}
	return offset, offset
}

// drawEnemyHealthBar renders a health bar centered horizontally on (x, y),
// `above` pixels above it and `halfWidth` pixels to each side.
func drawEnemyHealthBar(x, y, health, maxHealth, above, halfWidth float32) {
	if maxHealth <= 0 {
		return
	}
	barWidth := halfWidth * 2
	barHeight := 3.0
	barX := x - halfWidth
	barY := y - above - 8

	rl.DrawRectangle(int32(barX), int32(barY), int32(barWidth), int32(barHeight), rl.Fade(rl.Black, 0.5))

	healthPercent := health / maxHealth
	fillWidth := barWidth * healthPercent
	healthColor := rl.Red
	if healthPercent > 0.5 {
		healthColor = rl.Green
	} else if healthPercent > 0.25 {
		healthColor = rl.Orange
	}
	rl.DrawRectangle(int32(barX), int32(barY), int32(fillWidth), int32(barHeight), healthColor)
}

// generateID creates a unique ID for enemies and projectiles.
func generateID() string {
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	return base64.URLEncoding.EncodeToString(randBytes) + "-" + time.Now().Format("150405.000000")
}

// DrawRemoteEnemy renders a network-replicated enemy. EnemyState carries no
// velocity, so movement is derived from the position delta by a local tracker
// keyed on the enemy ID and the pulse cycle runs on the client's own clock.
// dt comes from the caller so the render path stays testable.
//
// facing is the world point a directional enemy should be looking at - in
// practice the nearest player, which the caller already knows. It is a
// parameter and not a derived value because the delta between snapshots
// carries the same separation and wall-slide that the host's velocity does,
// and deriving facing from it would reproduce on the client exactly the
// backwards-walking this argument exists to prevent. The client is applying
// the host's RULE, not guessing its result.
//
// Pass an ok of false when there is no player to look at; the tracker then
// keeps whatever facing it had. Radial enemies ignore both arguments.
func DrawRemoteEnemy(id string, enemyType EnemyType, x, y float32, color string, dt float32, facing FacingTarget) {
	def := GetEnemyDef(enemyType)
	st := trackRemoteEnemy(id, x, y, def, dt, facing)
	tex, ad, ok := enemyTexture(def, st.anim)
	if !ok {
		col := hexToColor(color)
		rl.DrawCircleV(rl.NewVector2(x, y), def.Radius, col)
		rl.DrawCircleLinesV(rl.NewVector2(x, y), def.Radius, rl.Fade(rl.Black, 0.5))
		return
	}
	switch def.Mode {
	case EnemyModeDirectional:
		drawEnemyDirectional(tex, ad, def.RenderScale, x, y, st.frame, st.row, st.mirror)
	case EnemyModeFixed:
		drawEnemyFixed(tex, ad, def.RenderScale, x, y, st.frame)
	default:
		drawEnemySprite(tex, ad, def.RenderScale, x, y, st.frame, st.angle)
	}
}

// DrawRemoteEnemyHealthBar draws the health bar for a network-replicated enemy,
// clearing the sprite silhouette rather than the hitbox.
func DrawRemoteEnemyHealthBar(enemyType EnemyType, x, y, health, maxHealth float32) {
	def := GetEnemyDef(enemyType)
	// Sempre a geometria do idle: a barra so precisa passar por cima da
	// cabeca, e entre idle e walk a linha do pe difere por poucos pixels. Ler
	// o estado do tracker aqui obrigaria a barra a saber de animacao para
	// ganhar nada visivel.
	above, halfWidth := enemyBarLayout(def, AnimIdle, def.Radius)
	drawEnemyHealthBar(x, y, health, maxHealth, above, halfWidth)
}

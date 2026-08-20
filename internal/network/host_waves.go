package network

import (
	"log"
	"math"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// WavePhase is what the run is currently doing.
type WavePhase string

const (
	// WavePhaseFighting means enemies are spawning and/or still alive.
	WavePhaseFighting WavePhase = "fighting"
	// WavePhaseBreak is the pause between waves.
	WavePhaseBreak WavePhase = "break"
	// WavePhaseCleared means every wave is done; nothing spawns any more and
	// the players are expected to move on to the next map.
	WavePhaseCleared WavePhase = "cleared"
)

// WaveBreakDuration is how long the pause between waves lasts, and therefore
// also how long the centred announcement stays on screen.
const WaveBreakDuration float32 = 5.0

// WaveDef describes one wave.
type WaveDef struct {
	Name string
	// Composition is how many of each enemy type the wave contains in total.
	Composition map[entity.EnemyType]int
	// MaxConcurrent caps how many of the wave's enemies are alive at once.
	// The rest are held back and released as those die, so a big wave keeps
	// pressure on instead of arriving as one unmanageable mob.
	MaxConcurrent int
	// SpawnInterval is the delay between spawn batches, in seconds.
	SpawnInterval float32
	// BatchSize is how many enemies are released per interval.
	BatchSize int
	// Announcement is shown centred on screen during the break that precedes
	// this wave.
	Announcement string
	// Endless faz a horda se REPOR em vez de acabar.
	//
	// Ela existe para um caso so, e o caso decide a forma: o climax do mapa 3
	// tem de ser impossivel de atravessar sem a ultimate da Sacerdotisa. Uma
	// horda finita, por maior que fosse, seria so uma questao de aguentar —
	// bastaria matar o ultimo. Repondo-se, ela nao tem ultimo: a unica saida e
	// o resgate.
	//
	// Quem a encerra e `LastStandDone()`. A partir do quadro em que a
	// Sacerdotisa ergue o altar, a horda para de repor e o que sobrou em campo
	// passa a ser finito — e ai sim matar o ultimo acaba a fase. A ultimate
	// nao mata ninguem; ela vira o jogo, que e o que uma ultimate deve fazer.
	Endless bool
	// EndsWithBoss diz que a reposicao de uma horda `Endless` para quando o CHEFE
	// da fase cai.
	//
	// Sem isto, `Endless` so conhece uma saida: `LastStandDone()`, o resgate da
	// Sacerdotisa no mapa 3. A arena do mapa 7 tem outra — matar quem comanda o
	// cerco —, e sem declara-la a fase nunca terminaria: o grupo derrubaria a
	// chefe e continuaria recebendo levas para sempre.
	//
	// Repara que ela para a REPOSICAO, nao a horda. O que ja esta em campo
	// continua vivo e precisa ser limpo. Matar os trinta no instante em que a
	// chefe cai seria anticlimatico: some a limpeza final, que e o momento em
	// que a vitoria assenta.
	EndsWithBoss bool
	// Ambush faz a horda nascer NOS MARCADORES DECLARADOS, sem o filtro de
	// anel que as hordas normais usam.
	//
	// O anel (1000 a 3200 px do jogador) existe para uma horda nao aparecer na
	// cara de quem esta lutando: ela vem de fora e caminha para dentro. Uma
	// EMBOSCADA e o contrario — ela e roteirizada, e o susto e justamente ela
	// surgir de cima da muralha e por tras, onde o mapa disse que ela surgiria.
	//
	// Sem isto o climax do mapa 3 nasceu em lugar nenhum: com o grupo parado no
	// meio da esplanada, marcador perto demais era descartado e marcador nenhum
	// sobrava, entao `batchPositions` devolvia vazio, `spawnEnemyAt` caia no
	// fallback de borda do mundo e os monstros apareciam do outro lado do mapa.
	// O anuncio na tela tocava e nada chegava.
	Ambush bool
	// Sentries e quantas GARGULAS o mapa tem em campo a partir desta horda.
	//
	// E um total acumulado, nao um acrescimo: a horda 3 do mapa 5 declara 2, a
	// 4 declara 4 e a 5 declara 6, e quem arma preenche a diferenca (ver
	// `Host.armSentries`). Escrito como acrescimo, uma gargula abatida na horda
	// 3 seria reposta na 4 sem ninguem pedir, e o Arqueiro estaria gastando a
	// suprema num alvo que renasce.
	//
	// Elas NAO entram em `Composition`, e portanto nao entram em
	// `waveTypeOrder`: a gargula tem Speed 0 e vive num posto declarado pelo
	// mapa, entao ela nao pode ser sorteada para o anel de spawn como o resto da
	// horda. Mas ela CONTA para o fim da horda — `aliveCount == 0` so e verdade
	// com as gargulas abatidas —, e isso e o objetivo e nao um efeito colateral:
	// uma torre que atira a 1900 px nao pode ser ignorada ate a fase acabar
	// sozinha.
	Sentries int
}

// Total returns how many enemies the wave contains.
func (w WaveDef) Total() int {
	n := 0
	for _, count := range w.Composition {
		n += count
	}
	return n
}

// A corrida de cada mapa mora em wave_runs.go; o runner so executa a que
// recebeu. Enquanto ela era uma variavel de pacote, todo mapa carregado
// herdava as hordas do world_01.

// Spawn point eligibility. Points closer than the minimum would pop into view;
// points beyond the maximum are so far that the walk is dead time. Both are in
// world units, against a 7680x5760 map.
const (
	spawnPointMinDistance float32 = 1000
	spawnPointMaxDistance float32 = 3200
	// spawnPointFallbackCount is how many nearest valid points to use when the
	// players stand somewhere with too few eligible ones (a map corner).
	spawnPointFallbackCount = 3
)

// WaveRunner owns the wave progression. It lives on the host, which is
// authoritative; clients only receive the resulting WaveState for display.
type WaveRunner struct {
	points []tilemap.SpawnPoint
	// defs is this map's run. The runner owns it instead of reading a package
	// variable, which is what let one map's waves play on another.
	defs []WaveDef

	index     int       // current wave index into defs
	phase     WavePhase
	breakTime float32   // seconds left in the current break
	// pending is what is left to spawn for the current wave, by type.
	pending map[entity.EnemyType]int
	// spawnTimer counts down to the next batch.
	spawnTimer float32
	// announced guards the log line so it prints once per wave.
	announced bool
	// sentriesOrdered marca que a horda atual ja pediu as gargulas dela.
	//
	// Um pedido POR HORDA, e nao uma reconciliacao por quadro: reconciliar todo
	// quadro faria a gargula abatida voltar no quadro seguinte, e a fase nunca
	// avancaria de horda — `aliveCount == 0` jamais seria verdade.
	sentriesOrdered bool
	// sectorCursor rotates through the compass so consecutive spawns come from
	// different sides of the player instead of clustering wherever rand landed.
	sectorCursor int
}

// spawnSectors is how many compass directions the ring around the player is
// divided into: N, NE, E, SE, S, SW, W, NW.
const spawnSectors = 8

// NewWaveRunner builds a runner from the map's enemy_spawn_* markers and the
// run that map declares. The run opens on a break so the first wave gets an
// announcement like the others.
//
// defs nil is a quiet map: nothing spawns and State reports Total 0, which is
// how the portal gate knows to stay open.
func NewWaveRunner(points []tilemap.SpawnPoint, defs []WaveDef) *WaveRunner {
	wr := &WaveRunner{
		points:    points,
		defs:      defs,
		index:     0,
		phase:     WavePhaseBreak,
		breakTime: WaveBreakDuration,
	}
	if len(points) == 0 {
		log.Printf("[Waves] mapa sem marcadores enemy_spawn_*; inimigos vao usar as bordas do mundo")
	} else {
		log.Printf("[Waves] %d pontos de spawn carregados", len(points))
		for _, p := range points {
			log.Printf("[Waves]   %s em (%.0f, %.0f)", p.Name, p.Position.X, p.Position.Y)
		}
	}
	return wr
}

// Update advances the wave state machine and spawns enemies. aliveCount is how
// many enemies are currently active; players are the live player positions used
// to pick spawn points.
func (wr *WaveRunner) Update(dt float32, aliveCount int, players []rl.Vector2, spawn func(entity.EnemyType, rl.Vector2)) {
	switch wr.phase {
	case WavePhaseCleared:
		return

	case WavePhaseBreak:
		wr.breakTime -= dt
		if wr.breakTime <= 0 {
			wr.startWave()
		}
		return

	case WavePhaseFighting:
		def := wr.current()
		if def == nil {
			wr.phase = WavePhaseCleared
			return
		}

		// HORDA QUE SE REPOE. Enquanto o resgate nao acontece, a composicao
		// volta para a fila toda vez que ela esvazia. Repor so quando zera, em
		// vez de manter um estoque, e o que deixa `MaxConcurrent` continuar
		// sendo o teto de pressao: o ritmo nao muda, o fim e que nao chega.
		if def.Endless && !LastStandDone() && !(def.EndsWithBoss && BossDown()) &&
			wr.pendingTotal() == 0 {
			for t, n := range def.Composition {
				wr.pending[t] += n
			}
		}

		remaining := wr.pendingTotal()
		// The wave is only over when nothing is left to spawn AND the field is
		// clear. Checking aliveCount alone would advance during the gap between
		// batches, when the player has simply killed the current batch.
		if remaining == 0 && aliveCount == 0 {
			wr.finishWave()
			return
		}

		if remaining == 0 {
			return
		}

		wr.spawnTimer -= dt
		if wr.spawnTimer > 0 {
			return
		}
		wr.spawnTimer = def.SpawnInterval

		room := def.MaxConcurrent - aliveCount
		if room <= 0 {
			return
		}
		batch := def.BatchSize
		if batch > room {
			batch = room
		}
		if batch > remaining {
			batch = remaining
		}

		positions := wr.batchPositions(players, batch)
		for i := 0; i < batch; i++ {
			enemyType, ok := wr.takePending()
			if !ok {
				break
			}
			var pos rl.Vector2
			if i < len(positions) {
				pos = positions[i]
			}
			spawn(enemyType, pos)
		}
	}
}

func (wr *WaveRunner) current() *WaveDef {
	if wr.index < 0 || wr.index >= len(wr.defs) {
		return nil
	}
	return &wr.defs[wr.index]
}

func (wr *WaveRunner) startWave() {
	def := wr.current()
	if def == nil {
		wr.phase = WavePhaseCleared
		return
	}
	wr.pending = make(map[entity.EnemyType]int, len(def.Composition))
	for t, n := range def.Composition {
		wr.pending[t] = n
	}
	wr.spawnTimer = 0 // first batch goes out immediately
	wr.phase = WavePhaseFighting
	wr.announced = false
	wr.sentriesOrdered = false
	if def.Endless {
		saida := "so o ultimo suspiro encerra"
		if def.EndsWithBoss {
			saida = "so a morte do chefe encerra"
		}
		log.Printf("[Waves] %s iniciada: SEM FIM (%d por leva de reposicao), "+
			"ate %d simultaneos; %s",
			def.Name, def.Total(), def.MaxConcurrent, saida)
	} else {
		log.Printf("[Waves] %s iniciada: %d inimigos, ate %d simultaneos",
			def.Name, def.Total(), def.MaxConcurrent)
	}
}

func (wr *WaveRunner) finishWave() {
	log.Printf("[Waves] %s concluida", wr.defs[wr.index].Name)
	wr.index++
	if wr.index >= len(wr.defs) {
		wr.phase = WavePhaseCleared
		log.Printf("[Waves] mapa limpo; nenhum inimigo novo vai surgir")
		return
	}
	wr.phase = WavePhaseBreak
	wr.breakTime = WaveBreakDuration
}

func (wr *WaveRunner) pendingTotal() int {
	n := 0
	for _, count := range wr.pending {
		n += count
	}
	return n
}

// takePending removes one enemy from the pending pool. Types are drawn in
// proportion to what is left, so a mixed wave stays mixed throughout instead of
// front-loading one type.
func (wr *WaveRunner) takePending() (entity.EnemyType, bool) {
	total := wr.pendingTotal()
	if total == 0 {
		return "", false
	}
	roll := rand.Intn(total)
	// Walk a fixed type order rather than ranging over the pending map: Go
	// randomises map iteration, which would make the draw non-reproducible even
	// with a seeded rand.
	for _, t := range waveTypeOrder {
		count := wr.pending[t]
		if count == 0 {
			continue
		}
		if roll < count {
			wr.pending[t] = count - 1
			return t, true
		}
		roll -= count
	}
	return "", false
}

// waveTypeOrder fixes the iteration order used by takePending.
//
// TODO TIPO NOVO PRECISA ENTRAR AQUI, e esquecer disso falha em silencio de um
// jeito especialmente ruim. O orc ficou de fora quando foi criado, e o efeito
// em jogo foi duplo: a emboscada da fortaleza nasceu so com slime e lobo, e
// como os 8 orcs continuavam contados em `pending`, `pendingTotal()` nunca
// chegava a zero — a horda NUNCA terminava. O jogador matava tudo o que via e
// nada acontecia.
//
// `TestWaveTypeOrderCoversEveryComposition` existe para que a proxima vez
// reprove em vez de virar um relatorio de bug.
var waveTypeOrder = []entity.EnemyType{
	entity.EnemyTypeBasic, entity.EnemyTypeFast, entity.EnemyTypeGarrison,
}

// eligiblePoints returns the spawn markers that sit in the usable ring around
// the players: far enough to be off screen, close enough that the walk is not
// dead time. Moving across the map therefore changes which points feed the
// fight, which is the point of having them scattered.
func (wr *WaveRunner) eligiblePoints(players []rl.Vector2) []rl.Vector2 {
	if len(wr.points) == 0 || len(players) == 0 {
		return nil
	}

	type scored struct {
		pos  rl.Vector2
		dist float32
	}
	var all []scored
	for _, p := range wr.points {
		nearest := float32(math.MaxFloat32)
		for _, player := range players {
			if d := rl.Vector2Distance(p.Position, player); d < nearest {
				nearest = d
			}
		}
		all = append(all, scored{p.Position, nearest})
	}

	var eligible []rl.Vector2
	for _, s := range all {
		if s.dist >= spawnPointMinDistance && s.dist <= spawnPointMaxDistance {
			eligible = append(eligible, s.pos)
		}
	}
	if len(eligible) >= 2 {
		return eligible
	}

	// Standing in a corner, or on a map whose markers are sparse: fall back to
	// the nearest points that are still off screen, rather than spawning on top
	// of the player or refusing to spawn at all.
	var valid []scored
	for _, s := range all {
		if s.dist >= spawnPointMinDistance {
			valid = append(valid, s)
		}
	}
	for i := 0; i < len(valid); i++ {
		for j := i + 1; j < len(valid); j++ {
			if valid[j].dist < valid[i].dist {
				valid[i], valid[j] = valid[j], valid[i]
			}
		}
	}
	limit := spawnPointFallbackCount
	if limit > len(valid) {
		limit = len(valid)
	}
	out := make([]rl.Vector2, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, valid[i].pos)
	}
	return out
}

// spawnScatter is how far an enemy is offset from its marker, so a batch does
// not stack on one pixel and immediately fight the separation solver.
const spawnScatter float32 = 140

// sectorOf returns which compass sector a point falls into as seen from origin.
// Sector 0 is east and they advance clockwise on screen, where Y grows downward.
func sectorOf(origin, p rl.Vector2) int {
	angle := math.Atan2(float64(p.Y-origin.Y), float64(p.X-origin.X)) * 180 / math.Pi
	step := 360.0 / spawnSectors
	// Shift by half a sector so each sector is centred on its cardinal.
	normalized := math.Mod(angle+360+step/2, 360)
	return int(normalized/step) % spawnSectors
}

// batchPositions returns one spawn position per enemy in the batch, drawing
// each from a different compass sector around the players.
//
// Picking eligible points at random clusters spawns: with six points and three
// of them to the south, most batches arrive from the south. Walking the compass
// instead guarantees that a batch fans out, and the cursor persists across
// batches so successive batches keep rotating around the player rather than
// restarting from the same side.
func (wr *WaveRunner) batchPositions(players []rl.Vector2, count int) []rl.Vector2 {
	if count <= 0 {
		return nil
	}
	// EMBOSCADA: os marcadores sao a resposta, na ordem em que o mapa os
	// declarou. Sem filtro de distancia e sem bussola — quem escolheu onde ela
	// surge foi quem desenhou o mapa.
	if def := wr.current(); def != nil && def.Ambush {
		out := make([]rl.Vector2, 0, count)
		for i := 0; i < count && len(wr.points) > 0; i++ {
			base := wr.points[wr.sectorCursor%len(wr.points)].Position
			wr.sectorCursor++
			out = append(out, rl.NewVector2(
				base.X+(rand.Float32()*2-1)*spawnScatter,
				base.Y+(rand.Float32()*2-1)*spawnScatter,
			))
		}
		return out
	}

	points := wr.eligiblePoints(players)
	if len(points) == 0 {
		return nil
	}

	origin := centroid(players)
	buckets := make([][]rl.Vector2, spawnSectors)
	for _, p := range points {
		s := sectorOf(origin, p)
		buckets[s] = append(buckets[s], p)
	}

	out := make([]rl.Vector2, 0, count)
	for i := 0; i < count; i++ {
		// Advance to the next sector that actually has a point. Bounded by one
		// full turn, so an empty compass cannot spin forever.
		var chosen []rl.Vector2
		for turn := 0; turn < spawnSectors; turn++ {
			candidate := buckets[wr.sectorCursor%spawnSectors]
			wr.sectorCursor = (wr.sectorCursor + 1) % spawnSectors
			if len(candidate) > 0 {
				chosen = candidate
				break
			}
		}
		if chosen == nil {
			break
		}
		base := chosen[rand.Intn(len(chosen))]
		out = append(out, rl.NewVector2(
			base.X+(rand.Float32()*2-1)*spawnScatter,
			base.Y+(rand.Float32()*2-1)*spawnScatter,
		))
	}
	return out
}

// centroid averages the player positions, so with several players the compass
// is measured from the middle of the group rather than from one of them.
func centroid(players []rl.Vector2) rl.Vector2 {
	if len(players) == 0 {
		return rl.Vector2{}
	}
	var sum rl.Vector2
	for _, p := range players {
		sum.X += p.X
		sum.Y += p.Y
	}
	n := float32(len(players))
	return rl.NewVector2(sum.X/n, sum.Y/n)
}

// State exports what clients need to render the HUD.
func (wr *WaveRunner) State(aliveCount int) WaveState {
	def := wr.current()
	name := ""
	announcement := ""
	if def != nil {
		name = def.Name
		announcement = def.Announcement
	}
	// Once the run is over, index has walked past the last wave; clamp it so a
	// HUD that ignores the phase never reads "Horda 4/3".
	shown := wr.index + 1
	if shown > len(wr.defs) {
		shown = len(wr.defs)
	}
	return WaveState{
		Index:        shown,
		Total:        len(wr.defs),
		Name:         name,
		Phase:        string(wr.phase),
		Remaining:    wr.pendingTotal() + aliveCount,
		BreakTime:    wr.breakTime,
		Announcement: announcement,
	}
}

// HasPoints reports whether the map supplied any enemy_spawn_* markers.
func (wr *WaveRunner) HasPoints() bool { return len(wr.points) > 0 }

// TakeSentryOrder devolve, UMA VEZ POR HORDA, quantos postos de sentinela a
// horda atual quer ocupados.
//
// O total e acumulado (ver `WaveDef.Sentries`), entao quem recebe o pedido
// preenche a diferenca. Devolve 0 quando a horda nao pede gargula nenhuma ou
// quando ela ja pediu — e por isso que o host pode chamar isto todo quadro sem
// pensar.
func (wr *WaveRunner) TakeSentryOrder() int {
	if wr.phase != WavePhaseFighting || wr.sentriesOrdered {
		return 0
	}
	def := wr.current()
	if def == nil || def.Sentries <= 0 {
		return 0
	}
	wr.sentriesOrdered = true
	return def.Sentries
}

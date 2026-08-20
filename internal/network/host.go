package network

// Host handles authoritative game simulation for multiplayer.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Host represents the server that manages the authoritative game state.
type Host struct {
	listener   net.Listener
	peers      map[string]*ClientConn // key: remote address
	peersMutex sync.Mutex
	// Authoritative game state: all connected players
	players      map[string]*PlayerState // key: player ID
	playersMutex sync.RWMutex
	// Entity management (enemies and projectiles)
	EntityManager *entity.EntityManager
	// Waves owns the horde progression for the current map. Nil until the map
	// is loaded and StartWaveRun is called, in which case no enemies
	// spawn at all.
	Waves *WaveRunner
	// stageMap e o caminho do mapa em jogo. Existe porque coisas que eram
	// globais passaram a ser da FASE - a corrida de hordas ja era, e agora o
	// heroi do ultimo suspiro tambem. StartWaveRun e o unico ponto por onde o
	// mapa e anunciado ao host, entao e la que ele e guardado.
	stageMap string
	// stagePosts e stageZones sao a geometria da guarnicao do mapa em jogo,
	// guardada para ser reposta no reinicio da fase.
	stagePosts []tilemap.SpawnPoint
	stageZones []tilemap.Zone
	// stageSentries sao os postos `enemy_sentry_*` do mapa em jogo. Guardados
	// pelo mesmo motivo que os de guarnicao — o reinicio de fase tem de repor as
	// gargulas — e tambem porque o WaveRunner do mapa 5 arma sentinelas a cada
	// horda e nao tem, ele proprio, acesso ao mapa.
	// O chefe da fase: tipo, ancora e se ela tem um. A ancora fica guardada
	// porque o reinicio de fase precisa repor a criatura — sem isso a segunda
	// tentativa do mapa 7 seria uma arena sem chefe, e a corrida infinita nunca
	// terminaria.
	bossType    entity.EnemyType
	bossAnchor  rl.Vector2
	bossPresent bool
	// boss sao os relogios da luta: espinhao, nevoa e o aviso da conjuracao.
	// Ver host_boss.go.
	boss bossClocks

	stageSentries []tilemap.SpawnPoint
	// sentriesArmed e quantos postos de sentinela ja foram OCUPADOS nesta
	// corrida. Conta postos e nao gargulas vivas de proposito: o que o jogador
	// derruba fica derrubado (ver sentries.go). Zerado a cada carregamento de
	// mapa e a cada reinicio de fase.
	sentriesArmed int
	// stageCannonPosts sao os postos `enemy_cannon_*` do mapa em jogo (mapa
	// 6), guardados para RestoreCannons repor no reinicio de fase.
	stageCannonPosts []tilemap.SpawnPoint
	// liveCannons e o estado em campo dos canhoes: posicao e se ja foi
	// destruido pelo julgamento do ultimo suspiro. Ver cannons.go.
	liveCannons []*cannonPost
	// World bounds for spawn positions and projectile validation
	WorldBounds world.Bounds
	PlayerSpawn rl.Vector2
	// Collision rectangles (walls/obstacles) for skill projectile checks.
	collisionRects []rl.Rectangle
	// Per-player cooldown remaining for the Sanctuary skill.
	sanctuaryCooldowns map[string]float32
	// Per-(player|skill) cooldowns for registry-cast skills, keyed
	// "playerID|skillID". Guarded by cdMutex (cast requests arrive from
	// client goroutines while the sim goroutine ticks them down).
	skillCooldowns map[string]float32
	// skillCharges counts casts already spent for ability.Charged skills
	// (cooldown arms only after the last charge). Guarded by cdMutex.
	skillCharges map[string]int
	// attackCooldowns is the per-player basic-attack cadence gate, derived
	// from the character's AttackSpeed. Guarded by cdMutex.
	attackCooldowns map[string]float32
	cdMutex         sync.Mutex
	// testPlayers holds the players running with test mode on (no cooldowns
	// at all). It has its own lock because the cooldown gates consult it
	// before taking cdMutex.
	testPlayers map[string]bool
	testMutex   sync.RWMutex
	// bots holds the AI runtime for every class currently filled by a bot
	// (host_bots.go). Guarded by playersMutex, like the h.players slot it
	// belongs to.
	bots map[string]*botRuntime
	// advanceDir is the whole party's smoothed heading, recomputed once per
	// tickBots call from the living humans' velocity and read by every bot
	// that same tick (host_bot_tick.go, doc/plan_avanco_bots_e_gargula.md
	// §A3, R3). Touched only from the simulation goroutine inside tickBots
	// — no mutex, same as WorldBounds and PlayerSpawn above.
	advanceDir rl.Vector2
	// vacatedBodies is where tickAbsence stashes a removed human's last
	// position/health, by class, so ReconcileBots can have the replacement
	// bot inherit the body instead of popping up fresh at the map spawn.
	// Guarded by playersMutex; consumed (and cleared) the moment a bot is
	// created for that class.
	vacatedBodies map[entity.CharacterType]PlayerState
	// Skills owns the fireball/sanctuary collections (separate from entity
	// so entity stays focused on players/enemies/projectiles).
	Skills *skill.Manager
	// announce lembra de quem ja foi anunciada a identidade (tipo, cor, vida
	// maxima), para nao repeti-la a cada tique. Ver wire.go.
	announce *announcer
	// snapshotTimer acumula ate o proximo snapshot. A simulacao roda a cada
	// quadro; a PUBLICACAO dela roda a SnapshotHz. Ver broadcast_rate.go.
	snapshotTimer float32
}

type ClientConn struct {
	conn     net.Conn
	writer   *bufio.Writer
	reader   *bufio.Reader
	playerID string // associated player ID for this connection
	// writeMu serializa TODA escrita nesta conexao.
	//
	// Duas goroutines escrevem no mesmo bufio.Writer: a da simulacao, por
	// broadcast, e a que atende este cliente, ao responder o join. bufio.Writer
	// nao e seguro para uso concorrente - duas escritas simultaneas intercalam
	// bytes, e como o cliente decodifica um FLUXO de JSON, um byte fora de
	// lugar corrompe tudo dali para a frente sem chance de ressincronizar.
	//
	// Nao usar peersMutex para isto: ele protege o MAPA de peers, e segurar um
	// lock global durante a escrita em todas as conexoes faz um cliente lento
	// travar os outros.
	writeMu sync.Mutex
	// Done channel signals that the client has disconnected.
	done chan struct{}
	// superseded is set by supersedeConnection (host_rejoin.go) on the OLD
	// connection when a reconnect with the same playerID replaces it. The
	// old connection's decode loop is about to fail (its socket just got
	// closed out from under it); handleClient's defer checks this so that
	// failure does not mark-absent or delete the slot the NEW connection now
	// owns. Atomic because supersedeConnection sets it from the new
	// connection's goroutine while the old connection's own goroutine may be
	// reading it in its defer at the same time.
	superseded atomic.Bool
}

// send escreve uma mensagem ja serializada nesta conexao, com o terminador de
// linha e o flush. Todo caminho de envio passa por aqui.
func (c *ClientConn) send(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	if err := c.writer.WriteByte('\n'); err != nil {
		return err
	}
	return c.writer.Flush()
}

// StartHost starts a TCP server listening on the given port.
// playerID and color are used to register the host as a player in the authoritative state.
func StartHost(port int, playerID string, color string, charType string, playerSpawn rl.Vector2) (*Host, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	h := &Host{
		listener:           ln,
		peers:              make(map[string]*ClientConn),
		players:            make(map[string]*PlayerState),
		EntityManager:      entity.NewEntityManager(),
		PlayerSpawn:        playerSpawn,
		sanctuaryCooldowns: make(map[string]float32),
		skillCooldowns:     make(map[string]float32),
		skillCharges:       make(map[string]int),
		attackCooldowns:    make(map[string]float32),
		testPlayers:        make(map[string]bool),
		Skills:             skill.NewManager(),
		announce:           newAnnouncer(),
	}
	// Register host as a player in the authoritative state
	h.players[playerID] = &PlayerState{
		PlayerID:  playerID,
		X:         int(h.PlayerSpawn.X),
		Y:         int(h.PlayerSpawn.Y),
		Color:     color,
		Character: charType,
		Health:    100,
		MaxHealth: 100,
		IsDead:    false,
	}
	log.Printf("[Host] StartHost: registered host %s, players map now has %d entries", playerID, len(h.players))
	go h.acceptLoop()
	return h, nil
}

func (h *Host) acceptLoop() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			log.Printf("host accept error: %v", err)
			continue
		}
		setKeepAlive(conn)
		client := &ClientConn{
			conn:   conn,
			writer: bufio.NewWriter(conn),
			reader: bufio.NewReader(conn),
			done:   make(chan struct{}),
		}
		h.peersMutex.Lock()
		h.peers[conn.RemoteAddr().String()] = client
		log.Printf("[Host] New client connected: %s", conn.RemoteAddr().String())
		h.peersMutex.Unlock()
		go h.handleClient(client)
	}
}

func (h *Host) handleClient(c *ClientConn) {
	defer func() {
		// A dropped connection MARKS the player absent instead of deleting
		// it (host_absence.go) — that is what makes a phone whose screen
		// locked able to come back at all. The superseded check matters
		// because this same defer also runs for the OLD connection when a
		// reconnect replaces it (supersedeConnection, host_rejoin.go): that
		// old connection's decode loop fails right after its socket gets
		// closed out from under it, and without the check this defer would
		// mark-absent (or worse, in the pre-absence world, delete) the slot
		// the NEW connection now owns.
		if c.playerID != "" && !c.superseded.Load() {
			h.playersMutex.Lock()
			h.markAbsent(c.playerID)
			h.playersMutex.Unlock()
			h.BroadcastRoster()
		}
		c.conn.Close()
		close(c.done)
		h.peersMutex.Lock()
		delete(h.peers, c.conn.RemoteAddr().String())
		h.peersMutex.Unlock()
	}()
	decoder := json.NewDecoder(c.reader)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("decode error from %s: %v", c.conn.RemoteAddr(), err)
			return
		}

		// While a dialogue is on screen the world is frozen for everyone. A
		// client that has not received the pause yet may still send a shot,
		// so the host drops combat requests rather than trusting every client
		// to have stopped at the same instant.
		if DialogueActive() && (msg.Type == MsgAttack || msg.Type == MsgSkill) {
			continue
		}

		switch msg.Type {
		case MsgJoin:
			// A fresh spawn and a reconnect with a KNOWN playerID both come
			// through here; handleJoin (host_rejoin.go) tells them apart and
			// does the roster/state/dialogue/map fan-out either way.
			h.handleJoin(c, msg.Payload)
			// Any ultimate the rescue already unlocked THIS RUN: this same
			// path handles a fresh join and a reconnect, and neither one saw
			// the original broadcast.
			h.sendUltimateGrantsTo(c)

		case MsgInput:
			var input InputPayload
			if err := json.Unmarshal(msg.Payload, &input); err != nil {
				log.Printf("failed to unmarshal input payload: %v", err)
				continue
			}
			// Update player position in authoritative state (absolute position)
			// Also update animation state if provided
			h.playersMutex.Lock()
			if p, ok := h.players[input.PlayerID]; ok {
				p.X = input.X
				p.Y = input.Y
				p.CurrentFrame = input.CurrentFrame
				p.CurrentRow = input.CurrentRow
				p.IsSprinting = input.IsSprinting
				p.VelX = input.VelX
				p.VelY = input.VelY
			}
			h.playersMutex.Unlock()
			// NAO transmite daqui. O input so atualiza o estado autoritativo;
			// quem publica e o tique da simulacao (BroadcastFullState).
			//
			// Transmitir aqui multiplicava o trabalho pelo numero de clientes:
			// cada um manda input a 60 Hz, entao com 4 jogadores saiam 240
			// broadcasts por segundo de uma lista que mudou em UM jogador -
			// quando 60 bastavam. E cada broadcast e serializacao mais uma
			// escrita por peer.
			//
			// A latencia nao piora: o tique roda no mesmo quadro, entao o
			// atraso maximo introduzido e um intervalo de snapshot.

		case MsgAttack:
			var attack AttackPayload
			if err := json.Unmarshal(msg.Payload, &attack); err != nil {
				log.Printf("failed to unmarshal attack payload: %v", err)
				continue
			}
			// Resolve the character-specific basic attack (same path the
			// host's own attacks use: fireball/holy/arrow or sword sweep).
			h.HandleAttack(attack.PlayerID, float32(attack.TargetX), float32(attack.TargetY))

		case MsgSkill:
			var skill SkillPayload
			if err := json.Unmarshal(msg.Payload, &skill); err != nil {
				log.Printf("failed to unmarshal skill payload: %v", err)
				continue
			}
			h.HandleSkillMessage(skill.PlayerID, skill.Skill, float32(skill.TargetX), float32(skill.TargetY))

		case MsgTestMode:
			var tm TestModePayload
			if err := json.Unmarshal(msg.Payload, &tm); err != nil {
				log.Printf("failed to unmarshal test mode payload: %v", err)
				continue
			}
			h.SetTestMode(tm.PlayerID, tm.Enabled)

		default:
			log.Printf("[Host] Unknown message type: %s", msg.Type)
		}
	}
}

// UpdateSimulation runs the game simulation (enemies, projectiles, combat).
// Should be called at fixed timestep (e.g., 60 FPS).
func (h *Host) UpdateSimulation(dt float32) {
	// A finished run freezes: no waves, no enemy movement, no skill ticks.
	// The only way forward is the host restarting the stage, and freezing here
	// means the world the players see is the one they lost in.
	if GameOver {
		return
	}

	h.updateWaves(dt)

	// Who is standing inside an open portal, before bots decide anything —
	// a bot that just vanished must not be given an Intent this same tick.
	h.tickPortalPresence()

	// Bots decide and move BEFORE the rest of the tick, same as a human's
	// input: it already arrived before this frame's simulation began, so a
	// bot's movement this tick counts for which player an enemy targets in
	// the SAME tick (plan §13.4).
	h.tickBots(dt)

	// Get all players for enemy AI
	players := h.getAllPlayersForAI()

	// Update all entities
	attackedEnemies := h.EntityManager.UpdateAll(dt, players)

	// Check projectile-enemy collisions
	h.checkProjectileCollisions()

	// Holy bolts (Sacerdotisa basic attack) heal allies they pass through.
	h.checkHolyProjectileHeals()

	// Advance Paladina sword sweeps (anchor follows the owner).
	h.handleSwordTick(dt)

	// Advance fireball skill effects (projectiles, explosions, ground fire).
	h.Skills.UpdateFire(dt)
	h.handleFireballTick(dt)

	// Advance sanctuary skill effects (healing zones, cooldowns).
	h.handleSanctuaryTick(dt)

	// Advance arrow volleys (Arqueiro) and shields (Paladina).
	h.handleArrowTick(dt)
	h.handleShieldTick(dt)

	// Advance ultimate skills (meteor rain, angelic area, celestial arrows,
	// divine avatar).
	h.handleUltimateTick(dt)

	// Advance Necromante effects (graveyards, spectral legions).
	h.handleNecroTick(dt)

	// Tick generic per-skill cooldowns and the basic-attack cadence gates.
	// Os contadores andam a cada quadro; o ESPELHO deles para os clientes vai
	// junto do snapshot, mais abaixo - um numero que so muda para desenhar nao
	// precisa de 60 atualizacoes por segundo.
	h.tickSkillCooldowns(dt)
	h.tickAttackCooldowns(dt)

	// Check enemy-player collisions (only for enemies that attacked this frame)
	h.checkEnemyPlayerCollisions(attackedEnemies)

	// A esfera sombria da Gargula Sentinela. Depende de attackedEnemies, entao
	// anda logo depois de quem o produziu.
	h.handleSentryOrbTick(dt, attackedEnemies)

	// Os canhoes do corredor final (mapa 6). Nao depende de attackedEnemies —
	// um canhao nao e um entity.Enemy — mas anda no mesmo lugar por ser a
	// mesma familia de ameaca a distancia.
	h.handleCannonTick(dt)

	// Revive whoever finished their countdown (frozen during a Game Over).
	h.tickRespawns(dt)
	h.tickInvulnerability(dt)
	h.tickLastStand()

	// Free any slot whose ReconnectGrace ran out. Before the game-over check
	// below, so a slot that just expired stops holding the party hostage in
	// the very frame it expires instead of one frame later.
	h.tickAbsence()

	// Check game over
	h.playersMutex.Lock()
	allDead := checkGameOver(h.players)
	h.playersMutex.Unlock()
	if allDead {
		// O ultimo suspiro segura o fim: a cena so vale se acontecer ANTES do
		// Game Over ser anunciado, senao o resgate chega depois do jogo ter
		// acabado - o mesmo que nao chegar.
		if LastStandPending() {
			return
		}
		// Announce it once: the check is true on every frame until the host
		// restarts the stage, and re-broadcasting it 60 times a second would
		// only flood the peers.
		if !GameOver {
			h.broadcastGameOver()
		}
		return
	}

	// Publicar e mais raro que simular: tudo acima roda a cada quadro, isto
	// roda a SnapshotHz. Ver broadcast_rate.go e interpolation.go.
	if h.dueForSnapshot(dt) {
		h.BroadcastFullState()
		h.broadcastCooldowns()
	}
}

// getAllPlayersForAI returns player states for enemy AI.
func (h *Host) getAllPlayersForAI() []entity.PlayerState {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()

	players := make([]entity.PlayerState, 0, len(h.players))
	for _, p := range h.players {
		players = append(players, entity.PlayerState{
			PlayerID:  p.PlayerID,
			X:         p.X,
			Y:         p.Y,
			Color:     p.Color,
			Health:    p.Health,
			MaxHealth: p.MaxHealth,
			IsDead:    p.IsDead,
		})
	}
	return players
}

// checkProjectileCollisions checks if projectiles hit enemies.
func (h *Host) checkProjectileCollisions() {
	projectiles := h.EntityManager.GetAllProjectiles()
	enemies := h.EntityManager.GetAllEnemies()

	for _, proj := range projectiles {
		for _, e := range enemies {
			// The stream sentries are a scripted objective: ordinary arrows,
			// fireballs and other basic projectiles pass below their inaccessible
			// islands. The Arqueiro's celestial arrows use their own piercing
			// resolver and are deliberately not filtered here.
			if e.Type == entity.EnemyTypeCastleSentry {
				continue
			}
			// HitCenter, e nao Position: no orc a posicao e o pe.
			dist := rl.Vector2Distance(proj.Position, e.HitCenter())
			if dist <= proj.Radius+e.HitRadius() {
				// Hit! Apply damage
				if e.TakeDamage(proj.Damage) {
					// Enemy died
					log.Printf("[Host] Enemy %s died from projectile", e.ID)
					h.EntityManager.RemoveEnemy(e.ID)
					h.broadcastCombatEvent("death", e.ID, "enemy", 0, "")
				} else {
					h.broadcastCombatEvent("damage", e.ID, "enemy", proj.Damage, "")
				}
				// Necromante shadow skulls steal life for their owner.
				h.applyProjectileLifesteal(proj)
				// Remove projectile on hit
				h.EntityManager.RemoveProjectile(proj.ID)
				break // Projectile can only hit one enemy
			}
		}
	}
}

// checkHolyProjectileHeals lets Sacerdotisa holy bolts heal living allies they
// pass through. The bolt is NOT consumed by allies (it pierces them) and heals
// each ally at most once per bolt. Healing is authoritative here and synced to
// clients via "heal" combat events (value = new absolute health).
func (h *Host) checkHolyProjectileHeals() {
	projectiles := h.EntityManager.GetAllProjectiles()

	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()

	for _, proj := range projectiles {
		if proj.Kind != entity.KindHoly || proj.Heal <= 0 {
			continue
		}
		for playerID, p := range h.players {
			if p.IsDead || playerID == proj.OwnerID {
				continue
			}
			if proj.HealedAllies[playerID] {
				continue
			}
			playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
			if rl.Vector2Distance(proj.Position, playerPos) > proj.Radius+entity.PlayerSize {
				continue
			}
			proj.HealedAllies[playerID] = true
			if p.Health >= p.MaxHealth {
				continue
			}
			p.Health += proj.Heal
			if p.Health > p.MaxHealth {
				p.Health = p.MaxHealth
			}
			h.broadcastCombatEvent("heal", playerID, "player", p.Health, proj.OwnerID)
			log.Printf("[Host] Holy bolt from %s healed %s to %.0f", proj.OwnerID, playerID, p.Health)
		}
	}
}

// checkEnemyPlayerCollisions checks if enemies that attacked this frame hit any players.
func (h *Host) checkEnemyPlayerCollisions(attackedEnemies map[string]bool) {
	enemies := h.EntityManager.GetAllEnemies()

	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()

	for _, e := range enemies {
		if !e.IsActive {
			continue
		}
		// Only apply damage if enemy attacked this frame
		if !attackedEnemies[e.ID] {
			continue
		}
		// A sentinela nao machuca por proximidade: o golpe dela viaja.
		//
		// Este caminho tira vida no instante em que o inimigo ataca, e o
		// alcance dela e 1900 - o jogador levaria dano do outro lado do saguao
		// sem nada atravessar a tela. handleSentryOrbTick lanca a esfera com o
		// mesmo aviso de ataque, e o dano chega quando ela encosta.
		if e.Type == entity.EnemyTypeCastleSentry {
			continue
		}
		for playerID, p := range h.players {
			if p.IsDead {
				continue
			}
			playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
			dist := rl.Vector2Distance(e.Position, playerPos)
			if dist <= e.AttackRange+e.Radius {
				// Divine Avatar (Paladina ultimate): total immunity.
				if h.Skills.HasAvatar(playerID) {
					break // attack landed on an invincible avatar
				}
				// Janela concedida por roteiro (o ultimo suspiro devolve o
				// Necromante com alguns segundos de imunidade). Fica ao lado
				// da do Avatar de proposito: sao a mesma pergunta, feita por
				// dois motivos diferentes.
				if h.IsInvulnerable(playerID) {
					break
				}
				// Shield (Paladina) absorbs damage before health is touched.
				dmg := e.AttackDamage
				if leftover, hpAfter, had := h.Skills.AbsorbShieldDamage(playerID, dmg); had {
					h.broadcastShieldEvent(playerID, hpAfter)
					if leftover <= 0 {
						break // fully absorbed: no health damage this attack
					}
					dmg = leftover
				}
				// Apply damage to player
				p.Health -= dmg
				if p.Health <= 0 {
					h.markPlayerDead(p)
					h.broadcastCombatEvent("death", playerID, "player", 0, "")
				} else {
					h.broadcastCombatEvent("damage", playerID, "player", dmg, "")
				}
				break // Only damage one player per enemy per attack
			}
		}
	}
}

// checkGameOver returns true if every PRESENT player is dead. Absent players
// (host_absence.go) are skipped both ways: one parked mid-field, alive,
// cannot hold the Game Over off forever, and one parked dead cannot cause it
// alone either — with nobody present at all, this is a party waiting for
// someone to reconnect, not a loss.
func checkGameOver(players map[string]*PlayerState) bool {
	present := 0
	for _, p := range players {
		if p.Absent {
			continue
		}
		present++
		if !p.IsDead {
			return false
		}
	}
	return present > 0
}

// sendStateToClient sends the current state only to a specific client
func (h *Host) sendStateToClient(c *ClientConn) {
	h.playersMutex.Lock()
	players := make([]PlayerState, 0, len(h.players))
	for _, p := range h.players {
		players = append(players, *p)
	}
	playerCount := len(h.players)
	h.playersMutex.Unlock()

	log.Printf("[Host] sendStateToClient: %d jogadores para o cliente novo", playerCount)

	// COMPLETO, com identidade. Este e o unico snapshot que o cliente novo
	// recebe sabendo de nada, e e dele que o cache de identidade nasce.
	state := StateUpdatePayload{Players: players}
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("failed to marshal state update: %v", err)
		return
	}

	msg := Message{Type: MsgStateUpdate, Payload: data}
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}

	if err := c.send(msgData); err != nil {
		log.Printf("failed to send state to client: %v", err)
		return
	}

	// E os inimigos que ja estao em campo, tambem COMPLETOS. Sem isto o
	// cliente novo receberia snapshots magros de monstros cuja identidade ele
	// nunca viu - o anunciante e um so para todos os peers, entao ele ja os
	// considera anunciados. Um mapa de guarnicao (world_03 nasce com 83) faria
	// o cliente desenhar 83 monstros sem tipo.
	h.sendEnemyIdentityTo(c)
	log.Printf("[Host] sendStateToClient: estado enviado")
}

// sendEnemyIdentityTo manda a lista de inimigos COM identidade para um cliente
// so. Usado quando ele entra, pelo motivo explicado em sendStateToClient.
func (h *Host) sendEnemyIdentityTo(c *ClientConn) {
	enemies := h.EntityManager.GetAllEnemies()
	states := make([]EnemyState, 0, len(enemies))
	for _, e := range enemies {
		if !e.IsActive {
			continue
		}
		states = append(states, EnemyState{
			EnemyID:   e.ID,
			Type:      string(e.Type),
			X:         int(e.Position.X),
			Y:         int(e.Position.Y),
			Health:    e.Health,
			MaxHealth: e.MaxHealth,
			Color:     e.Color,
		})
	}
	if len(states) == 0 {
		return
	}
	payload, err := json.Marshal(EnemyUpdatePayload{Enemies: states, Wave: GetWaveState()})
	if err != nil {
		log.Printf("[Host] falha ao serializar identidade de inimigos: %v", err)
		return
	}
	msgData, err := json.Marshal(Message{Type: MsgEnemyUpdate, Payload: payload})
	if err != nil {
		log.Printf("[Host] falha ao serializar mensagem de inimigos: %v", err)
		return
	}
	if err := c.send(msgData); err != nil {
		log.Printf("[Host] falha ao enviar inimigos ao cliente novo: %v", err)
	}
}

// BroadcastStateUpdate sends the full player list to all connected peers
// and updates the host's own RemotePlayers for rendering.
// BroadcastStateUpdate publica o estado autoritativo dos jogadores.
//
// NAO registre log por jogador aqui. Isto roda a cada tique de simulacao (60
// por segundo) e imprimia 2+2N linhas por chamada, com fmt e I/O sincrono no
// caminho quente do host: com 4 jogadores eram ~2.400 linhas por segundo. O
// custo aparecia duas vezes - como CPU do host em multiplayer, e como lixo
// entregue ao GC, que e o tipo de coisa que vira um quadro isolado de 40 ms no
// meio de uma caminhada. Log de EVENTO (join, disconnect, morte) continua
// certo; log de tique nao.
func (h *Host) BroadcastStateUpdate() {
	h.playersMutex.Lock()
	playerCount := len(h.players)
	players := make([]PlayerState, 0, playerCount)
	for _, p := range h.players {
		players = append(players, *p)
	}
	h.playersMutex.Unlock()

	// Update host's own RemotePlayers so it can render all players
	RemotePlayersMutex.Lock()
	if RemotePlayers == nil {
		RemotePlayers = make(map[string]PlayerState)
	}
	for _, p := range players {
		RemotePlayers[p.PlayerID] = p
	}
	pruneRemotePlayersLocked(players)
	RemotePlayersMutex.Unlock()

	// A identidade (cor, personagem, vida maxima) NAO vai no tique: ela e
	// anunciada no join e no snapshot completo que o cliente novo recebe. O
	// receptor recompoe pelo cache de identidade (wire.go), entao RemotePlayers
	// do outro lado continua com todos os campos.
	//
	// A copia local acima e a COMPLETA de proposito: o host renderiza a partir
	// dela e precisa da cor e do personagem de todo mundo.
	wire := make([]PlayerState, 0, len(players))
	for _, p := range players {
		wire = append(wire, slimPlayer(p))
	}

	state := StateUpdatePayload{Players: wire}
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("failed to marshal state update: %v", err)
		return
	}

	msg := Message{Type: MsgStateUpdate, Payload: data}
	h.broadcast(msg)
}

// BroadcastRoster publica o estado dos jogadores COM identidade.
//
// Existe porque BroadcastStateUpdate deixou de mandar cor, personagem e vida
// maxima: quem entra depois de um peer nunca teria visto a identidade dele e
// desenharia o Mago padrao sem cor. Este e o anuncio, e ele acontece nos
// momentos em que a identidade pode ter mudado - alguem entrou, alguem saiu, a
// fase reiniciou - e nao a cada tique.
func (h *Host) BroadcastRoster() {
	h.playersMutex.Lock()
	players := make([]PlayerState, 0, len(h.players))
	for _, p := range h.players {
		players = append(players, *p)
	}
	h.playersMutex.Unlock()

	RemotePlayersMutex.Lock()
	if RemotePlayers == nil {
		RemotePlayers = make(map[string]PlayerState)
	}
	for _, p := range players {
		RemotePlayers[p.PlayerID] = p
	}
	pruneRemotePlayersLocked(players)
	RemotePlayersMutex.Unlock()

	data, err := json.Marshal(StateUpdatePayload{Players: players})
	if err != nil {
		log.Printf("failed to marshal roster: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgStateUpdate, Payload: data})
}

// BroadcastFullState broadcasts players, enemies, and projectiles.
func (h *Host) BroadcastFullState() {
	// Broadcast player state
	h.BroadcastStateUpdate()

	// Broadcast enemy state
	enemies := h.EntityManager.GetAllEnemies()
	enemyStates := make([]EnemyState, 0, len(enemies))
	for _, e := range enemies {
		if e.IsActive {
			enemyStates = append(enemyStates, EnemyState{
				EnemyID:   e.ID,
				Type:      string(e.Type),
				X:         int(e.Position.X),
				Y:         int(e.Position.Y),
				Health:    e.Health,
				MaxHealth: e.MaxHealth,
				Color:     e.Color,
			})
		}
	}
	// Tipo, cor e vida maxima so vao no PRIMEIRO snapshot de cada inimigo; do
	// segundo em diante o receptor completa pelo cache. Com 83 inimigos no
	// world_03 isso e ~45 bytes por monstro por tique que deixam de existir.
	enemyPayload := EnemyUpdatePayload{
		Enemies: h.announce.wireEnemies(enemyStates),
		Wave:    GetWaveState(),
	}
	data, err := json.Marshal(enemyPayload)
	if err != nil {
		log.Printf("[Host] failed to marshal enemy update: %v", err)
	} else {
		msg := Message{Type: MsgEnemyUpdate, Payload: data}
		h.broadcast(msg)
	}

	// Broadcast projectile state
	projectiles := h.EntityManager.GetAllProjectiles()
	projStates := make([]ProjectileState, 0, len(projectiles))
	for _, p := range projectiles {
		if p.IsActive {
			dir := p.Dir()
			projStates = append(projStates, ProjectileState{
				ProjectileID: p.ID,
				OwnerID:      p.OwnerID,
				X:            int(p.Position.X),
				Y:            int(p.Position.Y),
				Active:       p.IsActive,
				Kind:         p.Kind,
				DirX:         dir.X,
				DirY:         dir.Y,
			})
		}
	}
	projPayload := ProjectileUpdatePayload{Projectiles: projStates}
	data, err = json.Marshal(projPayload)
	if err != nil {
		log.Printf("[Host] failed to marshal projectile update: %v", err)
	} else {
		msg := Message{Type: MsgProjectileUpdate, Payload: data}
		h.broadcast(msg)
	}
}

// broadcastCombatEvent broadcasts a combat event to all peers.
func (h *Host) broadcastCombatEvent(eventType, entityID, entityType string, value float32, killerID string) {
	payload := CombatEventPayload{
		EventType:  eventType,
		EntityID:   entityID,
		EntityType: entityType,
		Value:      value,
		KillerID:   killerID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] failed to marshal combat event: %v", err)
		return
	}
	msg := Message{Type: MsgCombatEvent, Payload: data}
	h.broadcast(msg)
}

// broadcastGameOver broadcasts game over to all peers.
func (h *Host) broadcastGameOver() {
	GameOver = true
	payload := GameOverPayload{Message: "All players are dead! Game Over!"}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] failed to marshal game over: %v", err)
		return
	}
	msg := Message{Type: MsgGameOver, Payload: data}
	h.broadcast(msg)
}

// RegisterPlayer adds a player to the authoritative state (used for host self-registration).
func (h *Host) RegisterPlayer(id string, state PlayerState) {
	h.playersMutex.Lock()
	h.players[id] = &PlayerState{
		PlayerID: state.PlayerID,
		X:        state.X,
		Y:        state.Y,
		Color:    state.Color,
	}
	h.playersMutex.Unlock()
}

// UpdatePlayerPosition updates a player's position in the authoritative state and broadcasts.
func (h *Host) UpdatePlayerPosition(id string, x, y int) {
	h.playersMutex.Lock()
	if p, ok := h.players[id]; ok {
		p.X = x
		p.Y = y
	}
	h.playersMutex.Unlock()
	// Publicada no proximo tique, como todo estado de movimento.
}

func (h *Host) broadcast(msg Message) {
	h.peersMutex.Lock()
	defer h.peersMutex.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}
	for addr, peer := range h.peers {
		if err := peer.send(data); err != nil {
			log.Printf("write error to %s: %v", addr, err)
		}
	}
}

// sendTo writes one message to a single connection. Used when the message is
// meant for one peer only, such as catching a client that joined mid-scene up
// with the dialogue everyone else is already reading.
func (h *Host) sendTo(c *ClientConn, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}
	if err := c.send(data); err != nil {
		log.Printf("write error to %s: %v", c.conn.RemoteAddr(), err)
	}
}

// Close shuts down the host and all client connections.
func (h *Host) Close() error {
	h.listener.Close()
	h.peersMutex.Lock()
	for _, p := range h.peers {
		p.conn.Close()
	}
	h.peersMutex.Unlock()
	return nil
}

// HandleAttack resolves a player's basic attack. Each character has its own
// attack: Mago fires a fireball, Sacerdotisa a holy bolt, Arqueiro an arrow,
// and Paladina performs a short-range 120-degree sword sweep (melee, no projectile).
func (h *Host) HandleAttack(playerID string, targetX, targetY float32) {
	h.playersMutex.Lock()
	if p, ok := h.players[playerID]; ok && !p.IsDead {
		// Attack speed gate: a held fire button or a fast tapper cannot beat
		// the character's cadence.
		if !h.beginAttackCooldown(playerID, entity.CharacterType(p.Character)) {
			h.playersMutex.Unlock()
			return
		}
		dir := rl.NewVector2(targetX-float32(p.X), targetY-float32(p.Y))
		startPos := rl.NewVector2(float32(p.X), float32(p.Y))
		charType := entity.CharacterType(p.Character)
		if charType == entity.CharPaladina {
			h.performSwordAttack(playerID, startPos, dir)
		} else if proj := entity.NewAttackProjectile(playerID, charType, startPos, dir); proj != nil {
			h.EntityManager.AddProjectile(proj)
			log.Printf("[Host] Player %s fired %s projectile %s", playerID, proj.Kind, proj.ID)
		}
	}
	h.playersMutex.Unlock()
}

// RespawnPlayer revives a dead player immediately, skipping the countdown.
// The normal path is tickRespawns; this exists for forced revivals. Like that
// path, the player gets up where it fell instead of at the map spawn.
func (h *Host) RespawnPlayer(playerID string) {
	h.playersMutex.Lock()
	if p, ok := h.players[playerID]; ok && p.IsDead {
		p.Health = entity.RespawnHealthPercent * p.MaxHealth
		p.IsDead = false
		p.RespawnIn = 0
		log.Printf("[Host] Player %s respawned with health %.0f", playerID, p.Health)
	}
	h.playersMutex.Unlock()
	// Evento raro e visivel: vale publicar na hora em vez de esperar o tique.
	h.BroadcastStateUpdate()
}

// ---- ability.HostLike implementation ----

// SkillManager returns the authoritative skill manager. (Named to avoid
// clashing with the Host.Skills field.)
func (h *Host) SkillManager() *skill.Manager { return h.Skills }

// PlayerState returns a read-only snapshot of a player, or nil.
func (h *Host) PlayerState(playerID string) *ability.PlayerStateView {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	p, ok := h.players[playerID]
	if !ok {
		return nil
	}
	return &ability.PlayerStateView{
		Character: p.Character,
		X:         float32(p.X),
		Y:         float32(p.Y),
		IsDead:    p.IsDead,
	}
}

// BroadcastSkill visualizes a skill cast for every connected client by
// delegating to the existing per-skill broadcast routines.
func (h *Host) BroadcastSkill(skillID string, ownerID string, center rl.Vector2) {
	switch skillID {
	case "sanctuary":
		h.broadcastSanctuary(ownerID, center)
	case "shield":
		h.broadcastShieldEvent(ownerID, skill.ShieldMaxHP)
	case "angelic_area", "divine_avatar", "spectral_legion":
		h.broadcastUltimate(skillID, ownerID, center, rl.Vector2{})
	}
}

// BroadcastSkillDir visualizes an aimed skill cast (origin + direction) for
// every connected client.
func (h *Host) BroadcastSkillDir(skillID string, ownerID string, origin, dir rl.Vector2) {
	switch skillID {
	case "fireball":
		h.broadcastFireCast(ownerID, origin, dir)
	case "arrow_volley":
		h.broadcastArrowVolley(ownerID, origin, dir)
	case "celestial_arrows", "graveyard":
		h.broadcastUltimate(skillID, ownerID, origin, dir)
	}
}

// HandleSkillMessage resolves a skill cast request from a client using the
// data-driven ability registry. Gating is data-driven and switch-free:
// character binding comes from the registry (characterHasSkill) and cooldowns
// from each skill's Cooldown() (beginSkillCooldown).
func (h *Host) HandleSkillMessage(playerID, skillID string, targetX, targetY float32) {
	ps := h.PlayerState(playerID)
	if ps == nil || ps.IsDead {
		return
	}
	// Character gating: only skills bound to the caster's character may cast.
	if !characterHasSkill(entity.CharacterType(ps.Character), skillID) {
		log.Printf("[Host] %s (%s) tried unbound skill %q", playerID, ps.Character, skillID)
		return
	}
	// Progression gating: the ultimate is earned, not owned from the start.
	// Checked BEFORE the cooldown, so a refused cast does not burn a charge.
	if !h.skillUnlocked(playerID, skillID, entity.CharacterType(ps.Character)) {
		log.Printf("[Host] %s (%s) tentou a ultimate %q ainda travada", playerID, ps.Character, skillID)
		return
	}
	// Cooldown gating (data-driven from the skill registry).
	if !h.beginSkillCooldown(playerID, skillID) {
		return // still on cooldown
	}
	ctx := &ability.CastContext{
		Host:      h,
		PlayerID:  playerID,
		Character: entity.CharacterType(ps.Character),
		Position:  rl.NewVector2(ps.X, ps.Y),
		Aim:       rl.NewVector2(targetX, targetY),
	}
	if !ability.CastByID(skillID, ctx) {
		log.Printf("[Host] unknown skill %q from %s", skillID, playerID)
	}
}

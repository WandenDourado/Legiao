package network

// Host handles authoritative game simulation for multiplayer.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
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
	spawnTimer    float32
	// World bounds for spawn positions and projectile validation
	WorldBounds world.Bounds
	PlayerSpawn rl.Vector2
}

type ClientConn struct {
	conn     net.Conn
	writer   *bufio.Writer
	reader   *bufio.Reader
	playerID string // associated player ID for this connection
	// Done channel signals that the client has disconnected.
	done chan struct{}
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
		listener:      ln,
		peers:         make(map[string]*ClientConn),
		players:       make(map[string]*PlayerState),
		EntityManager: entity.NewEntityManager(),
		PlayerSpawn:   playerSpawn,
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
		// Remove player from state when disconnecting
		if c.playerID != "" {
			h.playersMutex.Lock()
			delete(h.players, c.playerID)
			h.playersMutex.Unlock()
			log.Printf("[Host] Player %s disconnected, broadcasting updated state", c.playerID)
			h.BroadcastStateUpdate()
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

		switch msg.Type {
		case MsgJoin:
			var join JoinPayload
			if err := json.Unmarshal(msg.Payload, &join); err != nil {
				log.Printf("failed to unmarshal join payload: %v", err)
				continue
			}
			c.playerID = join.PlayerID
			// Register player in authoritative state
			h.playersMutex.Lock()
			h.players[join.PlayerID] = &PlayerState{
				PlayerID:  join.PlayerID,
				X:         int(h.PlayerSpawn.X),
				Y:         int(h.PlayerSpawn.Y),
				Color:     join.Color,
				Character: join.Character,
				Health:    100,
				MaxHealth: 100,
				IsDead:    false,
			}
			h.playersMutex.Unlock()
			log.Printf("[Host] Player %s joined with color %s", join.PlayerID, join.Color)
			// Broadcast updated state to all peers
			log.Printf("[Host] Calling BroadcastStateUpdate after join...")
			h.BroadcastStateUpdate()
			// Send current state to the new client specifically
			h.sendStateToClient(c)

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
			// Broadcast updated state to all peers
			h.BroadcastStateUpdate()

		case MsgAttack:
			var attack AttackPayload
			if err := json.Unmarshal(msg.Payload, &attack); err != nil {
				log.Printf("failed to unmarshal attack payload: %v", err)
				continue
			}
			// Create projectile at player position
			h.playersMutex.Lock()
			if p, ok := h.players[attack.PlayerID]; ok && !p.IsDead {
				dir := rl.NewVector2(float32(attack.TargetX)-float32(p.X), float32(attack.TargetY)-float32(p.Y))
				startPos := rl.NewVector2(float32(p.X), float32(p.Y))
				proj := entity.NewProjectile(attack.PlayerID, startPos, dir)
				h.EntityManager.AddProjectile(proj)
				log.Printf("[Host] Player %s fired projectile %s", attack.PlayerID, proj.ID)
			}
			h.playersMutex.Unlock()

		default:
			log.Printf("[Host] Unknown message type: %s", msg.Type)
		}
	}
}

// UpdateSimulation runs the game simulation (enemies, projectiles, combat).
// Should be called at fixed timestep (e.g., 60 FPS).
func (h *Host) UpdateSimulation(dt float32) {
	// Update spawn timer
	h.spawnTimer += dt
	if h.spawnTimer >= 3.0 { // Spawn every 3 seconds
		h.spawnTimer = 0
		h.spawnEnemies()
	}

	// Get all players for enemy AI
	players := h.getAllPlayersForAI()

	// Update all entities
	attackedEnemies := h.EntityManager.UpdateAll(dt, players)

	// Check projectile-enemy collisions
	h.checkProjectileCollisions()

	// Check enemy-player collisions (only for enemies that attacked this frame)
	h.checkEnemyPlayerCollisions(attackedEnemies)

	// Check game over
	h.playersMutex.Lock()
	if checkGameOver(h.players) {
		h.playersMutex.Unlock()
		h.broadcastGameOver()
		return
	}
	h.playersMutex.Unlock()

	// Broadcast updated state
	h.BroadcastFullState()
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
			dist := rl.Vector2Distance(proj.Position, e.Position)
			if dist <= proj.Radius+e.Radius {
				// Hit! Apply damage
				if e.TakeDamage(proj.Damage) {
					// Enemy died
					log.Printf("[Host] Enemy %s died from projectile", e.ID)
					h.EntityManager.RemoveEnemy(e.ID)
					h.broadcastCombatEvent("death", e.ID, "enemy", 0, "")
				} else {
					h.broadcastCombatEvent("damage", e.ID, "enemy", proj.Damage, "")
				}
				// Remove projectile on hit
				h.EntityManager.RemoveProjectile(proj.ID)
				break // Projectile can only hit one enemy
			}
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
		for playerID, p := range h.players {
			if p.IsDead {
				continue
			}
			playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
			dist := rl.Vector2Distance(e.Position, playerPos)
			if dist <= e.AttackRange+e.Radius {
				// Apply damage to player
				p.Health -= e.AttackDamage
				if p.Health <= 0 {
					p.Health = 0
					p.IsDead = true
					log.Printf("[Host] Player %s died", playerID)
					h.broadcastCombatEvent("death", playerID, "player", 0, "")
				} else {
					h.broadcastCombatEvent("damage", playerID, "player", e.AttackDamage, "")
				}
				break // Only damage one player per enemy per attack
			}
		}
	}
}

// checkGameOver returns true if all players are dead.
func checkGameOver(players map[string]*PlayerState) bool {
	if len(players) == 0 {
		return true
	}

	for _, p := range players {
		if !p.IsDead {
			return false // At least one player alive
		}
	}
	return true // All players dead
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

	log.Printf("[Host] sendStateToClient: sending %d players to new client:", playerCount)
	for _, p := range players {
		log.Printf("[Host]   -> %s at (%d,%d) color %s", p.PlayerID, p.X, p.Y, p.Color)
	}

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

	_, err = c.writer.Write(msgData)
	if err != nil {
		log.Printf("failed to send state to client: %v", err)
		return
	}
	c.writer.WriteByte('\n')
	c.writer.Flush()
	log.Printf("[Host] sendStateToClient: state sent successfully")
}

// BroadcastStateUpdate sends the full player list to all connected peers
// and updates the host's own RemotePlayers for rendering.
func (h *Host) BroadcastStateUpdate() {
	h.playersMutex.Lock()
	playerCount := len(h.players)
	log.Printf("[Host] BroadcastStateUpdate: map has %d players", playerCount)
	for id, p := range h.players {
		log.Printf("[Host]   - %s in map at (%d,%d) color %s", id, p.X, p.Y, p.Color)
	}
	players := make([]PlayerState, 0, playerCount)
	for _, p := range h.players {
		players = append(players, *p)
	}
	h.playersMutex.Unlock()

	log.Printf("[Host] BroadcastStateUpdate: %d players in map", playerCount)
	for _, p := range players {
		log.Printf("[Host]   - %s at (%d,%d) color %s", p.PlayerID, p.X, p.Y, p.Color)
	}

	// Update host's own RemotePlayers so it can render all players
	RemotePlayersMutex.Lock()
	if RemotePlayers == nil {
		RemotePlayers = make(map[string]PlayerState)
	}
	for _, p := range players {
		RemotePlayers[p.PlayerID] = p
	}
	RemotePlayersMutex.Unlock()

	state := StateUpdatePayload{Players: players}
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("failed to marshal state update: %v", err)
		return
	}

	msg := Message{Type: MsgStateUpdate, Payload: data}
	h.broadcast(msg)
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
	enemyPayload := EnemyUpdatePayload{Enemies: enemyStates}
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
			projStates = append(projStates, ProjectileState{
				ProjectileID: p.ID,
				OwnerID:      p.OwnerID,
				X:            int(p.Position.X),
				Y:            int(p.Position.Y),
				Active:       p.IsActive,
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
	// Broadcast updated state to all peers
	h.BroadcastStateUpdate()
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
		_, err := peer.writer.Write(data)
		if err != nil {
			log.Printf("write error to %s: %v", addr, err)
			continue
		}
		peer.writer.WriteByte('\n')
		peer.writer.Flush()
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

// HandleAttack creates a projectile when a player attacks.
func (h *Host) HandleAttack(playerID string, targetX, targetY float32) {
	h.playersMutex.Lock()
	if p, ok := h.players[playerID]; ok && !p.IsDead {
		dir := rl.NewVector2(targetX-float32(p.X), targetY-float32(p.Y))
		startPos := rl.NewVector2(float32(p.X), float32(p.Y))
		proj := entity.NewProjectile(playerID, startPos, dir)
		h.EntityManager.AddProjectile(proj)
		log.Printf("[Host] Player %s fired projectile %s", playerID, proj.ID)
	}
	h.playersMutex.Unlock()
}

// RespawnPlayer respawns a dead player with 15% health.
func (h *Host) RespawnPlayer(playerID string) {
	h.playersMutex.Lock()
	if p, ok := h.players[playerID]; ok && p.IsDead {
		p.Health = 0.15 * p.MaxHealth // 15% health on respawn
		p.IsDead = false
		p.X = int(h.PlayerSpawn.X)
		p.Y = int(h.PlayerSpawn.Y)
		log.Printf("[Host] Player %s respawned with health %.0f", playerID, p.Health)
	}
	h.playersMutex.Unlock()
	h.BroadcastStateUpdate()
}

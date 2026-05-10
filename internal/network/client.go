package network

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
)

type Client struct {
	conn   net.Conn
	writer *bufio.Writer
	reader *bufio.Reader
	done   chan struct{}
}

// ConnectClient connects to a host at the given address (ip:port).
func ConnectClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, writer: bufio.NewWriter(conn), reader: bufio.NewReader(conn), done: make(chan struct{})}
	go c.readLoop()
	return c, nil
}

func (c *Client) readLoop() {
	decoder := json.NewDecoder(c.reader)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("client decode error: %v", err)
			close(c.done)
			return
		}
		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MsgStateUpdate:
		var state StateUpdatePayload
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			log.Printf("client failed to unmarshal state update: %v", err)
			return
		}
		log.Printf("[Client] Received state update with %d players:", len(state.Players))
		for _, p := range state.Players {
			log.Printf("[Client]   - %s at (%d,%d) color %s", p.PlayerID, p.X, p.Y, p.Color)
		}
		// Update the shared remote players state
		RemotePlayersMutex.Lock()
		RemotePlayers = make(map[string]PlayerState)
		for _, p := range state.Players {
			RemotePlayers[p.PlayerID] = p
		}
		RemotePlayersMutex.Unlock()
		log.Printf("[Client] Updated RemotePlayers: %d players stored", len(state.Players))

	case MsgEnemyUpdate:
		var enemyUpdate EnemyUpdatePayload
		if err := json.Unmarshal(msg.Payload, &enemyUpdate); err != nil {
			log.Printf("client failed to unmarshal enemy update: %v", err)
			return
		}
		// Update remote enemies
		RemoteEnemiesMutex.Lock()
		RemoteEnemies = make(map[string]EnemyState)
		for _, e := range enemyUpdate.Enemies {
			RemoteEnemies[e.EnemyID] = e
		}
		RemoteEnemiesMutex.Unlock()
		log.Printf("[Client] Updated RemoteEnemies: %d enemies", len(enemyUpdate.Enemies))

	case MsgCombatEvent:
		var combat CombatEventPayload
		if err := json.Unmarshal(msg.Payload, &combat); err != nil {
			log.Printf("client failed to unmarshal combat event: %v", err)
			return
		}
		log.Printf("[Client] Combat event: %s on %s (type: %s, value: %.0f)", combat.EventType, combat.EntityID, combat.EntityType, combat.Value)

		// Handle player death/respawn
		if combat.EntityType == "player" {
			if combat.EventType == "death" {
				// Mark player as dead
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.IsDead = true
					p.Health = 0
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
				if combat.EntityID == LocalPlayerID {
					LocalPlayerDead = true
					RespawnTimer = 0
				}
			} else if combat.EventType == "damage" {
				// Apply damage to player locally
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.Health -= combat.Value
					if p.Health < 0 {
						p.Health = 0
					}
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
				log.Printf("[Client] Player %s took %.0f damage, health now %.0f", combat.EntityID, combat.Value, combat.Value)
			} else if combat.EventType == "respawn" {
				// Handle respawn
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.IsDead = false
					p.Health = combat.Value
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
				if combat.EntityID == LocalPlayerID {
					LocalPlayerDead = false
				}
			}
		}

	case MsgGameOver:
		var gameOver GameOverPayload
		if err := json.Unmarshal(msg.Payload, &gameOver); err != nil {
			log.Printf("client failed to unmarshal game over: %v", err)
			return
		}
		log.Printf("[Client] GAME OVER: %s", gameOver.Message)
		GameOver = true

	case MsgRespawn:
		var respawn RespawnPayload
		if err := json.Unmarshal(msg.Payload, &respawn); err != nil {
			log.Printf("client failed to unmarshal respawn: %v", err)
			return
		}
		log.Printf("[Client] Respawn: player %s at (%d,%d) with health %.0f", respawn.PlayerID, respawn.X, respawn.Y, respawn.Health)
		// Update player state
		RemotePlayersMutex.Lock()
		RemotePlayers[respawn.PlayerID] = PlayerState{
			PlayerID: respawn.PlayerID,
			X:        respawn.X,
			Y:        respawn.Y,
			Health:   respawn.Health,
			IsDead:   false,
		}
		RemotePlayersMutex.Unlock()
		if respawn.PlayerID == LocalPlayerID {
			LocalPlayerDead = false
			RespawnTimer = 0
		}

	default:
		log.Printf("[Client] Received message type: %s", msg.Type)
	}
}

func (c *Client) Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	log.Printf("[Client] Sending message: %s", string(data))
	_, err = c.writer.Write(data)
	if err != nil {
		return err
	}
	c.writer.WriteByte('\n')
	return c.writer.Flush()
}

func (c *Client) Close() error {
	err := c.conn.Close()
	<-c.done
	return err
}

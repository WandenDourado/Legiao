# Network Implementation (Wi-Fi)

The game supports multiplayer over Wi-Fi using a TCP client-server model with authoritative state management.

## Overview

* **Host** – Runs a TCP server (`network.StartHost`) on port 9000. It maintains the authoritative game state, registers players on join, updates positions on input, and broadcasts state updates to all connected peers.
* **Client** – Connects to a host (`network.ConnectClient`) and sends its absolute position (`MsgInput`). It receives `MsgStateUpdate` messages and updates the shared `RemotePlayers` map for rendering.

## Architecture

```
┌─────────────────────────────────────────┐
│                 HOST (Authoritative)             │
│  players map (PlayerID → PlayerState)          │
│  peers map (addr → ClientConn)                │
│  RemotePlayers (for host rendering)            │
└─────────────────────────────────────────┘
         │ MsgStateUpdate (broadcast)
         ▼
┌─────────────────────────────────────────┐
│                CLIENT (Receiver)                │
│  RemotePlayers map (for rendering)             │
│  Updates on MsgStateUpdate                    │
└─────────────────────────────────────────┘
```

## Message Protocol (`internal/network/protocol.go`)

### Message Types

```go
type MessageType string

const (
    MsgJoin        MessageType = "join"
    MsgInput       MessageType = "input"
    MsgStateUpdate MessageType = "state_update"
    MsgDisconnect  MessageType = "disconnect"
)
```

### Message Structure

```go
type Message struct {
    Type    MessageType   `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

### Payload Structures

**Join (sent by client on connect):**
```json
{
    "player_id": "player_1234567890",
    "color": "#FF5733"
}
```

**Input (sent by client/host on movement - absolute position):**
```json
{
    "player_id": "player_1234567890",
    "x": 150,
    "y": 200
}
```

**State Update (broadcast by host after each change):**
```json
{
    "players": [
        {"player_id": "player_123", "x": 100, "y": 100, "color": "#FF5733"},
        {"player_id": "player_456", "x": 150, "y": 200, "color": "#33FF57"}
    ]
}
```

## State Management

### Host (Authoritative State)

The host maintains a `players` map (`map[string]*PlayerState`) that stores the canonical state of all connected players. This map gets updated on:
- **Join:** New player registered with initial position (100, 100)
- **Input:** Player position updated to absolute coordinates received
- **Disconnect:** Player removed from map

After each modification, `BroadcastStateUpdate()` is called, which:
1. Collects all player states from `players` map
2. Updates host's own `RemotePlayers` for rendering
3. Marshals the state into `MsgStateUpdate`
4. Broadcasts to all connected peers

### Client (Shared State)

Clients maintain a `RemotePlayers` map (`map[string]PlayerState`) that gets updated when receiving `MsgStateUpdate`. The `readLoop()` in `client.go` calls `handleMessage()` which replaces the entire map with the latest state from the host.

## Connection Flow

1. **Host starts** – Calls `network.StartHost(9000, playerID, color)`, which registers host as a player in the authoritative `players` map with initial position (100, 100).

2. **Client connects** – The client automatically discovers hosts via UDP broadcast on port 9001. The user selects a discovered host from the menu, then `network.ConnectClient("IP:9000")` is called. The client generates player ID and sends `MsgJoin` with ID and color from `entity.PresetColors`.

3. **Host receives join** – Registers player in `players` map at initial position (100, 100), calls `BroadcastStateUpdate()` to notify all peers, and calls `sendStateToClient()` to send current state (including host) to the new client specifically.

4. **All peers receive state update** – Each peer (including the new client) updates its `RemotePlayers` map with all players and renders them with their assigned colors.

## Movement Synchronization

1. **Local player moves** – `main.go` game loop calls `p.Update(dir, dt)`, updates local player state in `RemotePlayers`, sends `MsgInput` with absolute position (X, Y).

2. **Host receives input** – Updates player position in `players` map to the received absolute coordinates, then broadcasts updated state via `BroadcastStateUpdate()`.

3. **Client receives input** – Host processes the input and broadcasts to all peers.

4. **All peers receive state update** – `handleMessage()` replaces `RemotePlayers` map with latest state, then re-render with new positions for all players.

## Rendering

The `main.go` game loop renders all players:

```go
// Draw ALL players (local + remote)
allPlayers := network.GetAllPlayers()

// Draw remote players
for id, state := range allPlayers {
    if id == network.LocalPlayerID {
        continue
    }
    entity.DrawPlayerAt(float32(state.X), float32(state.Y), state.Color, 20)
}

// Draw local player
p.Draw()
```

Each player has a unique color assigned from `entity.PresetColors` (hex format). The `entity.DrawPlayerAt()` function converts hex to `rl.Color` using `hexToColor()`.

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/network/host.go` | Authoritative server, state management, broadcasting |
| `internal/network/client.go` | Client connection, message handling, state updates |
| `internal/network/protocol.go` | Message types and payload structures |
| `internal/network/globals.go` | Shared state (`RemotePlayers`, `LocalPlayerID`) |
| `internal/ui/menu.go` | Host/client selection, join flow |
| `internal/entity/player.go` | Player rendering with colors |
| `cmd/desktop/main.go` | Game loop, input handling, rendering |

## Extensibility

The `Peer` interface (implemented by both `Host` and `Client`) allows future Bluetooth implementations to reuse the same message handling logic. Only the transport layer (TCP vs Bluetooth) differs.

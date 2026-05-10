# Multiplayer Networking Fix - Implementation Documentation

## Date
2026-05-01

## Problem Description
Each client was seeing only its own instance of the world. Players were not being synchronized correctly on the same shared map. The root cause was that there was no authoritative game state - each client maintained its own isolated game world.

## Root Cause Analysis

### 1. Host Did Not Maintain Authoritative State
**File:** `internal/network/host.go` (original)

The Host was only broadcasting raw messages (like `MsgInput`) to all peers without:
- Maintaining a centralized `PlayerState` map
- Tracking player positions, colors, and IDs
- Sending proper state updates to synchronize all clients

### 2. Client Did Not Process State Updates
**File:** `internal/network/client.go` (original)

The `readLoop()` only logged received messages but never:
- Processed `MsgStateUpdate` messages
- Updated local game state with remote player data
- Stored player states for rendering

### 3. No Join Flow Implementation
- New clients connecting didn't register with the host
- No player ID or color assignment on join
- Host didn't track new players in its authoritative state

### 4. Rendering Only Showed Local Player
**File:** `cmd/desktop/main.go` (original)

Only the local player was being rendered. There was no code to:
- Render remote players from network state
- Use player colors for visual distinction
- Display the shared game world

## Solution Implemented

### 1. Authoritative Host State (`internal/network/host.go`)

Added centralized game state management:

```go
type Host struct {
    listener   net.Listener
    peers      map[string]*ClientConn
    peersMutex sync.Mutex
    players      map[string]*PlayerState // NEW: Authoritative player state
    playersMutex sync.Mutex
}
```

**Key Changes:**
- `players` map stores all connected players with their states (ID, X, Y, Color)
- `handleClient()` now processes `MsgJoin` to register new players
- `handleClient()` processes `MsgInput` to update player positions
- `broadcastStateUpdate()` sends full state to all peers after each change
- Host's own `RemotePlayers` is updated for rendering

### 2. Client State Processing (`internal/network/client.go`)

Added message handling for state updates:

```go
func (c *Client) handleMessage(msg Message) {
    switch msg.Type {
    case MsgStateUpdate:
        var state StateUpdatePayload
        json.Unmarshal(msg.Payload, &state)
        RemotePlayersMutex.Lock()
        RemotePlayers = make(map[string]PlayerState)
        for _, p := range state.Players {
            RemotePlayers[p.PlayerID] = p
        }
        RemotePlayersMutex.Unlock()
    }
}
```

### 3. Shared State Management (`internal/network/globals.go`)

Added shared state and helper functions:

```go
var (
    CurrentHost   *Host
    CurrentClient *Client
    Role          string
    LocalPlayerID string
    RemotePlayers      map[string]PlayerState
    RemotePlayersMutex sync.Mutex
)

func GetAllPlayers() map[string]PlayerState { ... }
func UpdatePlayerState(p PlayerState) { ... }
func RemovePlayerState(playerID string) { ... }
```

### 4. Menu Integration (`internal/ui/menu.go`)

Added join flow:
- Host registers itself as a player on start
- Client sends `MsgJoin` with generated PlayerID and random color
- Uses `entity.PresetColors` for distinct player colors

### 5. Entity Rendering (`internal/entity/player.go`)

Added color support:
- `DrawPlayerAt(x, y, color, radius)` - renders a player at specific position
- `hexToColor(hex)` - converts hex color strings to raylib.Color

### 6. Main Game Loop (`cmd/desktop/main.go`)

Updated to:
- Render ALL players from `network.GetAllPlayers()`
- Send `MsgInput` to network when local player moves
- Update local player state in network
- Display player count on screen

## Files Modified

| File | Changes |
|------|---------|
| `internal/network/host.go` | Added authoritative state, join/input handling, state broadcasting |
| `internal/network/client.go` | Added state update processing, removed duplicate globals |
| `internal/network/globals.go` | Added shared state management functions |
| `internal/ui/menu.go` | Added join flow, player registration |
| `internal/entity/player.go` | Added hex color support, DrawPlayerAt function |
| `cmd/desktop/main.go` | Added multiplayer rendering, movement sync |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        HOST (Authoritative)                 │
│  ┌──────────────┐    ┌──────────────┐    ┌────────────┐  │
│  │   players    │    │    peers     │    │ RemotePlayers│  │
│  │  (state)     │    │  (clients)  │    │  (render)   │  │
│  └──────────────┘    └──────────────┘    └────────────┘  │
│         │                      │                  │         │
│         └──────────────────────┴──────────────────┘         │
│                        │                                    │
│              broadcastStateUpdate()                         │
└─────────────────────────────────────────────────────────────┘
                        │ MsgStateUpdate
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                      CLIENT (Receiver)                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              RemotePlayers (render)                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                        │                                    │
│                 handleMessage(MsgStateUpdate)               │
└─────────────────────────────────────────────────────────────┘
```

## Message Flow

1. **Client connects** → Sends `MsgJoin{PlayerID, Color}`
2. **Host receives join** → Registers player in `players` map → `broadcastStateUpdate()`
3. **All peers receive** `MsgStateUpdate{Players: [...]}` → Update `RemotePlayers`
4. **Client moves** → Sends `MsgInput{PlayerID, DX, DY}`
5. **Host receives input** → Updates player position in `players` → `broadcastStateUpdate()`
6. **All peers receive** updated state → Re-render with new positions

## Testing Checklist

- [x] Start host game
- [x] Connect multiple clients
- [x] Verify all players appear with different colors
- [x] Move local player, verify others see movement
- [x] Move remote players, verify host sees movement
- [x] Disconnect a client, verify it's removed from all peers
- [x] New client joins, verify existing players (including host) are propagated

## Cross-Platform Networking Fix (2026-05-03)

### Problem
Clients could not connect to hosts on different devices. The client hardcoded `127.0.0.1:9000` (localhost), which only works when both host and client are on the same machine.

### Solution
Modified `internal/ui/menu.go` to show an IP input dialog when the user clicks "Join Game". The user enters the host's LAN IP address, and the client connects to `IP:9000`.

### Changes
- Added IP input field in join menu with blinking cursor
- Port 9000 is automatically appended (user only enters the IP)
- Back button to return to main menu
- Updated documentation in `doc/running_android.md` and `doc/network.md`

## Automatic Host Discovery via UDP Broadcast (2026-05-03)

### Problem
Typing the host IP manually is a poor user experience. Users shouldn't need to know network details.

### Solution
Implemented automatic host discovery using UDP broadcast on port 9001:
- **Host**: Broadcasts `LEGION_HOST:0.0.0.0:9000` every 3 seconds via UDP broadcast
- **Client**: Listens on UDP port 9001, discovers hosts automatically, shows a list in the UI
- User just clicks on a discovered host to connect

### Files Added/Modified
- **`internal/network/discovery.go`** (new) - Discovery broadcaster and listener using UDP
- **`internal/ui/menu.go`** - Shows discovered hosts list instead of IP input field
- **`internal/network/globals.go`** - Added `ServerAddress` to display connection info
- **`internal/game/loop.go`** - Shows server address (IP:port) at top-center of screen
- **Documentation updated** in `doc/running_android.md`, `doc/network.md`, `doc/changelog.md`

## Notes

- The host runs its own game loop AND acts as server
- State updates are sent as full snapshots (not deltas) for simplicity
- Player colors are randomly assigned from `entity.PresetColors`
- Screen bounds checking is still applied to local player only
- Host must be accessible on the LAN (firewall may need inbound rule for port 9000)

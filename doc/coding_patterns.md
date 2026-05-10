# Coding Patterns - Legião Project

This document describes the coding patterns and conventions adopted in the Legião project.

## Project Structure

The project follows a standard Go project layout with clear separation of concerns:

```
internal/
├── entity/      # Pure data structures and basic logic (Player, Enemy, Projectile)
├── system/      # Game logic systems (Combat, Spawn, Movement)
├── network/     # Multiplayer networking (host/client, protocol, messages)
├── game/        # Game loop, configuration, and platform-specific setup
└── ui/          # User interface elements (HUD, menus, virtual joystick)
```

## Key Patterns

### 1. Entity-Component Pattern (Simplified)

Entities are pure data structures with basic methods. They are managed by systems.

**Example**: `internal/entity/player.go`
```go
type Player struct {
    Color     string
    Position  rl.Vector2
    Health    float32
    MaxHealth float32
    IsDead    bool
}

func (p *Player) Update(dir rl.Vector2, dt float32) { ... }
func (p *Player) TakeDamage(damage float32) bool { ... }
func (p *Player) Respawn(healthPercent float32, x, y float32) { ... }
```

### 2. System Pattern

Systems contain game logic that operates on entities. They are separate from entity definitions.

**Example**: `internal/system/combat.go`
- `CheckProjectileCollisions()` - checks if projectiles hit enemies
- `CheckEnemyPlayerCollisions()` - checks if enemies damage players
- `CheckGameOver()` - checks if all players are dead

### 3. Thread-Safe Entity Management

The `EntityManager` uses `sync.RWMutex` to protect shared state (enemies and projectiles maps).

**Example**: `internal/entity/manager.go`
```go
type EntityManager struct {
    Enemies     map[string]*Enemy
    Projectiles map[string]*Projectile
    enemiesMutex sync.RWMutex
    projMutex    sync.RWMutex
}
```

### 4. Authoritative Host Model (Networking)

The host maintains the authoritative game state. Clients send input and receive state updates.

**Host responsibilities** (`internal/network/host.go`):
- Maintain authoritative state for players, enemies, projectiles
- Run game simulation (enemy AI, combat, spawning)
- Broadcast state updates to all clients

**Client responsibilities** (`internal/network/client.go`):
- Send input (movement, attacks) to host
- Receive and render state updates (players, enemies, projectiles)
- Handle combat events and game state changes locally

### 5. Network Protocol

Messages use JSON over TCP with a simple type-payload structure.

**Message structure** (`internal/network/protocol.go`):
```go
type Message struct {
    Type    MessageType   `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

**Message types**:
- `join` - Player joins the game
- `input` - Player movement input
- `state_update` - Full player state broadcast
- `enemy_update` - Enemy positions/health broadcast
- `attack` - Player attack action
- `combat_event` - Damage/death events
- `game_over` - All players dead
- `respawn` - Player respawned

### 6. Game Loop Phases

The main game loop (`internal/game/loop.go`) is split into phases:

1. **Input Phase**: Read joystick/attack button input
2. **Local Update Phase**: Update local player position, respawn timer
3. **Host Simulation Phase** (host only): Update enemies, projectiles, check combat, spawn waves
4. **Render Phase**: Draw all entities, HUD, and UI elements

### 7. HUD Elements

HUD elements are drawn using Raylib primitives with simple positioning.

**Health Bar** (`internal/ui/hud.go`):
- Position: Top-left corner
- Color: Green (>50%), Orange (>25%), Red (≤25%)
- Shows text: "current/max"

**Attack Button** (`internal/ui/hud.go`):
- Position: Bottom-right corner
- Red circle with "FIRE" text
- Responds to mouse/touch input

**Virtual Joystick** (`internal/ui/hud.go`):
- Position: Bottom-left corner
- Base circle with draggable knob
- Returns normalized direction vector

### 8. Configuration Constants

Game constants are centralized in `internal/game/config.go`:

```go
const (
    ScreenWidth  = 1280
    ScreenHeight = 720
    PlayerSpeed = 200.0
    EnemySpeed  = 100.0
    RespawnDelay = 15.0
    // ... more constants
)
```

### 9. Color Handling

Colors are stored as hex strings (e.g., `"#8B0000"` for blood red) and converted to `rl.Color` at runtime.

**Helper function** (`internal/entity/player.go`):
```go
func hexToColor(hex string) rl.Color { ... }
```

### 10. Unique ID Generation

Entities (enemies, projectiles) get unique IDs using crypto/rand + timestamp.

**Helper function** (`internal/entity/enemy.go`):
```go
func generateID() string {
    randBytes := make([]byte, 8)
    rand.Read(randBytes)
    return base64.URLEncoding.EncodeToString(randBytes) + "-" + time.Now().Format("150405.000000")
}
```

## Import Cycle Prevention

The project avoids import cycles by:
- Defining `PlayerState` struct in both `entity` and `network` packages (minimal version in entity for AI, full version in network for protocol)
- Not importing `system` package from `network` (combat logic is handled inline in host.go)

## Error Handling

- Network errors are logged but don't crash the game
- Failed JSON marshal/unmarshal is logged with context
- Missing entities in maps return nil (caller checks)
- Mutexes protect all shared state access

## Testing Strategy

- Desktop build: `go run cmd/desktop/main.go`
- Android build: Use `cmd/android/build/` with Raylib's Android toolchain
- Multiplayer testing: Run host + multiple clients on same/different machines
- Verify: enemy spawning, combat, health bars, game over, respawn

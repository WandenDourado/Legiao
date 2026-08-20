# Coding Patterns

Keep this file short. Put package maps in `project_structure.md` and feature behavior in the relevant area doc.

## Package Ownership

| Package | Owns |
|---|---|
| `entity` | Data structs and basic entity methods. |
| `system` | Game rules operating on entities. |
| `network` | Multiplayer protocol, host/client, discovery, shared snapshots. |
| `game` | Orchestration, platform config, camera/update/render flow. |
| `input` | Touch, keyboard, mouse, joystick state. |
| `ui` | Menus and HUD drawing. |
| `assets` | Platform-aware asset path resolution. |
| `tilemap` | Tiled map parsing, tileset loading, map rendering, collision extraction. |
| `collision` | The single movement-vs-obstacle rule (footprint, sliding, detour) shared by every walking entity. |
| `world` | World bounds derived from map data. |
| `nav` | The route layer between deciding WHERE an agent goes and what a single step lets it do: a walkability mesh built from `collision.Solid`, A* + string-pulled paths, and a per-agent `Follower`. Pure — imports only `collision`, `world`, `rl.Vector2`; never `network`, `game` or `entity`. |

## Hard Rules

- `internal/game/loop.go` stays orchestration-only.
- Do not mix unrelated responsibilities in one `.go` file.
- Split files before they exceed roughly 150 lines unless they are pure orchestration.
- Use `assets.Path()` for every `rl.Load*` path.
- Use build tags for platform-specific behavior.
- Use `world.Bounds` for world logic; never screen constants for map, camera, projectile, spawn, or collision bounds.

## Entity/System Pattern

Entities hold state and simple methods:

```go
type Player struct {
    Position rl.Vector2
    Health float32
}
```

Systems or managers apply rules:

- `system/combat.go`: projectile/enemy/player hit checks.
- `system/movement.go`: enemy movement.
- `internal/network/host.go`: authoritative multiplayer simulation glue.
- `internal/network/host_spawn.go`: host enemy wave spawning.

## Networking Pattern

The host is authoritative. Clients send input/actions and render snapshots from the host.

Messages use:

```go
type Message struct {
    Type    MessageType     `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

Current message families: join/input/state, enemy/projectile snapshots, attack, combat events, game over, respawn.

## Error Handling

- Log network and JSON errors with context; do not crash transient networking paths.
- Protect shared maps with their existing mutexes.
- Return copies from shared-state getters.
- Keep generated build artifacts out of source edits unless the task is explicitly about artifacts.

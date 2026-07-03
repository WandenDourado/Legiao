# Network

The game uses TCP for gameplay state and UDP/TCP helpers for LAN discovery.

## Model

- Host is authoritative.
- Clients send input and actions.
- Host simulates enemies, projectiles, combat, deaths, respawn, and broadcasts snapshots.
- Clients render snapshots from the host.

## Ports

| Port | Protocol | Use |
|---|---|---|
| 9000 | TCP | Gameplay messages |
| 9001 | UDP | Host discovery query/response and broadcast compatibility |

## Main Files

| File | Responsibility |
|---|---|
| `internal/network/protocol.go` | Message types and payload structs. |
| `internal/network/host.go` | TCP server, authoritative state, combat/projectile simulation. |
| `internal/network/host_spawn.go` | Enemy wave spawning and safe edge spawn selection. |
| `internal/network/host_player_state.go` | Host player movement/animation snapshot update. |
| `internal/network/client.go` | TCP client and received message handling. |
| `internal/network/client_projectiles.go` | Applies projectile snapshots on clients. |
| `internal/network/discovery.go` | UDP discovery, query responder, TCP subnet scan fallback. |
| `internal/network/globals.go` | Process-wide network role and snapshot maps. |
| `internal/ui/menu.go` | Host/join UI and discovery/manual connection flow. |

## Message Families

| Type | Direction | Purpose |
|---|---|---|
| `join` | client -> host | Register player ID/color. |
| `input` | client -> host | Position, sprite frame/row, sprint flag, velocity. |
| `state_update` | host -> peers | Full player snapshot list. |
| `enemy_update` | host -> peers | Full enemy snapshot list. Empty lists clear clients. |
| `projectile_update` | host -> peers | Full projectile snapshot list. Empty lists clear clients. |
| `attack` | client -> host | Request projectile fire toward a world target. |
| `combat_event` | host -> peers | Damage/death/respawn events. |
| `game_over` | host -> peers | All players are dead. |
| `respawn` | host -> peers | Respawn update. |

## Discovery Flow

Current join flow starts two discovery paths:

1. `StartQuerySender(9000)` sends `LEGION_QUERY:<reply_port>` to UDP broadcast port 9001 and receives direct `LEGION_RESPONSE:host:9000`.
2. `StartTCPScan(9000)` scans the local subnet as fallback.

Host starts:

1. TCP server on port 9000.
2. UDP broadcast announcements for desktop compatibility.
3. UDP query responder on port 9001 for Android-friendly direct responses.

Manual IP remains available as a fallback in the menu.

## Snapshot Rules

- `RemotePlayers`, `RemoteEnemies`, and `RemoteProjectiles` are shared render snapshots.
- Snapshot getters return copies.
- Clients replace snapshot maps from host messages, they do not merge partial deltas.
- The host also updates its own `RemotePlayers` for rendering.
- Empty enemy/projectile snapshots are meaningful and must be sent to clear stale remote entities.

## Animation And Actions

`MsgInput` includes:

```json
{
  "player_id": "player_...",
  "x": 100,
  "y": 100,
  "current_frame": 2,
  "current_row": 1,
  "is_sprinting": false,
  "vel_x": 0,
  "vel_y": 200
}
```

Remote players render with the wizard sprite sheet via `entity.DrawWizardStateAt()`. Projectiles are created only by the host and broadcast through `projectile_update`.

## Operational Notes

- LAN play requires host and clients on the same network.
- Windows firewall must allow inbound TCP 9000 for the host.
- Discovery is LAN-only, not internet matchmaking.
- Do not reintroduce Java/MulticastLock code unless there is a verified Android requirement; the current query-response path avoids passive broadcast receive dependence.

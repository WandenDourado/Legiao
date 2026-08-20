# Camera And World Bounds

Use this doc for camera behavior, world-space rendering, bounds, spawn distance, and coordinate conversion.

## World Bounds

World size comes from the loaded Tiled map:

```go
bounds := world.NewBoundsFromMap(tm.Width, tm.Height, tm.TileWidth, tm.TileHeight)
```

For the current village map (60x45 tiles at 128px), bounds are `7680x5760`.

Do not use screen constants for world logic.

## Camera

`internal/game/camera.go` owns `Camera2DState`.

- Offset: screen center.
- Target: player position clamped to world bounds.
- Zoom: `1.0`.
- Rotation: `0`.
- No smoothing/lerp; camera follows directly.

Clamp rule:

```go
camera.Target.X = clamp(target.X, halfW, bounds.Width-halfW)
camera.Target.Y = clamp(target.Y, halfH, bounds.Height-halfH)
```

## Render Space Rule

World-space drawing must happen inside the map renderer camera callback in `DrawFrame`.

World-space examples:

- map layers
- map boundary
- players
- enemies and health bars
- projectiles

Screen-space examples:

- health bar
- player count
- server address
- respawn/game-over text
- Android joystick/action controls

Screen-space UI must stay outside camera transforms.

## Mouse Coordinate Conversion

Raylib mouse position is screen-space. Convert to world-space before attack targeting:

```go
screenPos := rl.GetMousePosition()
worldPos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
```

Android aim uses a direction vector relative to the player and does not need this conversion.

## Bounds Initialization Order

Set host entity bounds after the host exists:

1. Initialize window.
2. Load map and compute bounds.
3. Show menu; host may be created here.
4. If host exists, assign `network.CurrentHost.EntityManager.WorldBounds = bounds`.

Assigning bounds before menu creation leaves host bounds at zero and projectiles expire immediately.

## Player Spawn

`player_spawn` in the loaded Tiled map is the shared spawn for local and
authoritative network player state.

- Current spawn: the map object, with the center of map-derived
  `world.Bounds` retained only as a malformed-map fallback.
- Host registration, joining clients, and respawn use this same position.

## Enemy Spawn Distance

Authoritative enemy spawn is in `internal/network/host_spawn.go`.

- Spawn candidates are on map edges, `180px` outside the playable rectangle.
- The host avoids candidates within `520px` of alive players.
- If no candidate passes after the retry budget, the farthest sampled candidate is used.

The auxiliary `internal/system/spawn.go` uses the same edge offset for consistency.

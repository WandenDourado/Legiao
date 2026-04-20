# Legião - Project Overview

Legião is a 2D cooperative multiplayer local survival shooter inspired by Vampire Survivors.
The game is built in Go using Raylib for rendering and targets Android (with desktop for development).

## Project Structure

```
/cmd
  /android        → Entry point for Android (gomobile)
  /desktop        → Entry point for desktop (development and testing)

/internal
  /game
    game.go       → Main game loop
    config.go     → Global constants and configurations
  /entity
    player.go     → Player entity definition and behavior
    enemy.go      → Enemy entities
    projectile.go → Projectiles (bullets, etc.)
    manager.go    → Manages collections of entities (players, enemies, projectiles)
  /system
    movement.go   → Handles movement logic
    combat.go     → Handles combat (hit detection, damage)
    spawn.go      → Handles spawning of enemies and items
    upgrade.go    → Handles power-up/upgrade system
  /network
    protocol.go   → Message definitions and JSON serialization
    host.go       → Authoritative host simulation logic
    client.go     → Client logic (sends input, receives state)
    discovery.go  → Local network discovery (Wi-Fi)
    transport.go  → Common interface for Wi-Fi and Bluetooth transport
  /transport
    wifi.go       → Wi-Fi specific transport implementation
    bluetooth.go  → Bluetooth specific transport implementation
  /screen
    menu.go       → Main menu screen
    lobby.go      → Lobby screen (waiting for players)
    game.go       → In-game screen
    upgrade.go    → Upgrade selection screen
    gameover.go   → Game over screen
  /ui
    hud.go        → Heads-up display (including virtual joystick for touch)
    camera.go     → Camera system (follows players, zoom, etc.)

/assets
  /sprites        → 2D sprites for entities, UI, etc.
  /sounds         → Sound effects and music
  /maps           → Tilemaps or level data (if used)

/doc
  architecture.md → Detailed architecture (to be created)
  overview.md     → This file
```

## Key Components

### Game Loop
- Runs at a fixed tick rate (20 ticks per second for state updates, inputs processed in real-time).
- The host runs the authoritative simulation; clients are dumb terminals that send input and render state.

### Networking
- Uses TCP sockets for Wi-Fi LAN and Bluetooth for direct connections.
- Messages are JSON-encoded (defined in `protocol.go`).
- Communication channels:
  - `stateChan`: Network → Game (receives state from host)
  - `inputChan`: Game → Network (sends input to host)

### Entities
- Player, Enemy, Projectile are defined in `/internal/entity`.
- Managed by entity managers (in `/internal/entity/manager.go`).

### Systems
- Movement, Combat, Spawn, Upgrade systems are in `/internal/system`.
- These systems operate on entities each tick.

### Screens
- Different game states (menu, lobby, game, upgrade, gameover) are handled in `/internal/screen`.
- Each screen has its own update and draw logic.

### UI
- HUD includes health, score, and virtual joystick for touch input (`/internal/ui/hud.go`).
- Camera follows the player(s) and handles zoom (`/internal/ui/camera.go`).

## Multiplayer Architecture
- The host is authoritative: it runs the full game simulation.
- Clients only send their input (joystick, buttons) and receive the full game state to render.
- Input is sent in real-time; state is synchronized at 20 ticks per second.
- Discovery: Players can find each other via local network (Wi-Fi) or Bluetooth pairing.

## Development Notes
- Desktop (`/cmd/desktop`) is used for development and testing.
- Android builds are done via `gomobile` (`/cmd/android`).
- The game design is cooperative: two players on the same screen, surviving waves of enemies.

## Future Work
- Week 1: Basic game loop, player movement, virtual joystick, camera.
- Week 2: Enemy types, wave system, upgrades, HUD, screens.
- Week 3: Full multiplayer (Wi-Fi + Bluetooth), lobby, synchronization.
- Week 4: Android build, touch adjustments, assets, Google Play release.

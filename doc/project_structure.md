# Project Structure

## Entry Points

| Path | Responsibility |
|---|---|
| `AGENTS.md` | Single agent entrypoint, hard rules, doc routing, skill discovery. |
| `cmd/desktop/main.go` | Desktop launcher, calls `game.Run(game.DefaultConfig())`. |
| `cmd/android/build/main.go` | Android launcher used by the Gradle/raylib build. |

## Main Packages

| Path | Responsibility |
|---|---|
| `internal/assets/` | `assets.Path()` abstraction for desktop vs Android asset paths. |
| `internal/entity/` | Player, enemy, projectile data and drawing helpers. |
| `internal/game/` | Main run loop orchestration, input processing, camera, rendering, collision. |
| `internal/input/` | Touch state, aim joystick, platform-specific control drawing. |
| `internal/network/` | TCP host/client, UDP/TCP discovery, protocol, shared snapshots. |
| `internal/system/` | Reusable game systems: combat, movement, spawn, upgrade. |
| `internal/tilemap/` | Tiled JSON/TMJ/TSX loading, tileset rendering, collision extraction. |
| `internal/ui/` | Menu and HUD. |
| `internal/world/` | Map-derived bounds. |

## Assets

| Path | Responsibility |
|---|---|
| `assets/maps/` | Runtime maps bundled into Android APK. |
| `assets/tilesets/` | Tileset image and TSX files. |
| `assets/sprites/` | Sprite sheets, including the wizard player. |
| `maps/` | Source copies of maps; runtime should prefer `assets/maps/`. |

## Android Build Tree

| Path | Responsibility |
|---|---|
| `cmd/android/build/androidcompile.bat` | Compiles Go native libraries and copies assets. |
| `cmd/android/build/android/AndroidManifest.xml` | Android app manifest and permissions. |
| `cmd/android/build/android/build.gradle` | Android module build config. |
| `cmd/android/build/android/assets/` | Generated/copied APK asset tree. |
| `cmd/android/build/android/libs/` | Generated native libraries. |

## Agent Skills

| Path | Responsibility |
|---|---|
| `skills/legiao-android-build/` | Android APK/AAB build workflow for agents. |
| `skills/create-character-sprites/` | Directional character sprite generation, validation, sheet assembly, and metadata workflow for agents. |
| `mcp/legiao-android-build/` | MCP compatibility notes for the Android build skill. |

## Where To Add Code

| Need | Location |
|---|---|
| New entity or draw helper | `internal/entity/` |
| New game rule | `internal/system/` |
| New network message | `internal/network/protocol.go` plus host/client handlers |
| New multiplayer host behavior | Small focused file in `internal/network/` |
| New menu/HUD behavior | `internal/ui/` |
| New input behavior | `internal/input/` and `internal/game/input_handler.go` |
| New map/tileset behavior | `internal/tilemap/` |
| Platform-specific behavior | Build-tagged files |

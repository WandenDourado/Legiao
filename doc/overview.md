# Project Overview

Legiao is a 2D cooperative LAN survival shooter built in Go with raylib-go. Desktop is used for fast development and Android is a target platform.

## Current Gameplay

- One host runs the authoritative simulation.
- Clients join over local Wi-Fi.
- Players move, sprint, aim, fire projectiles, fight enemies, die, and respawn.
- The basic attack has a per-character cadence and every skill has a cooldown with an on-screen counter.
- A dead player stays on the field, greyed out and immobile, and is revived by the host after 30s at 30% health; if everyone dies the run is over until the host restarts the stage (see `combat_rules.md`).
- The map is a Tiled tilemap loaded from `assets/maps`.
- Camera follows the local player within map-derived world bounds.

## Current Architecture

| Area | Source |
|---|---|
| Game loop orchestration | `internal/game/loop.go` |
| Input processing | `internal/game/input_handler.go`, `internal/input/` |
| Rendering | `internal/game/renderer.go`, `internal/tilemap/renderer.go` |
| Entities | `internal/entity/` |
| Combat/movement/spawn systems | `internal/system/` |
| Multiplayer | `internal/network/` |
| Menu/HUD | `internal/ui/` |
| Map data | `assets/maps/`, `assets/tilesets/` |

See `project_structure.md` for the detailed package map.

## Design Direction

- Keep the host authoritative.
- Keep Android and desktop sharing the same game logic.
- Prefer small focused files over large mixed-responsibility files.
- Keep docs as current source-of-truth references; put historical details in `changelog.md`.

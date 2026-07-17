---
name: install-character-sprites
description: Add a validated `create-character-sprites` export to Legiao as a selectable playable character. Use when asked to install, register, import, or make playable a generated character sprite sheet, including adding it to the existing character-selection screen.
---

# Install Character Sprites

Install a production-ready character without changing the shared animation renderer. The character becomes available through the existing registry and character-selection UI.

## Inputs and Contract

1. Require a passing sheet, metadata, character bible, and per-direction review reports from `$create-character-sprites`. Metadata must declare `mirror_safe: true`, `E→W`, `SE→SW`, `NE→NW`, anchor `{x:64,y:186}`, and the alternating left/right walk cycle. Do not use raw AI grids as game assets.
2. Choose a unique lowercase kebab-case `<character-id>` and a player-facing display name. Reject collisions with `CharacterType` values or existing `assets/sprites/<character-id>/` directories unless the request explicitly replaces that character.
3. Preserve the current shared contract: RGBA PNG, `128x192` frames, 8 columns, 5 rows ordered `S,SW,W,N,NW`. The renderer mirrors the three east-facing directions. If the export does not meet this contract, return it to `$create-character-sprites`; do not introduce a character-specific rendering path.

## Install

1. Read `AGENTS.md`, `doc/coding_patterns.md`, `internal/entity/character.go`, `internal/entity/player_sprite.go`, `internal/ui/character_select.go`, and the generated metadata. Inspect the active network path as well if a character identifier travels in multiplayer messages.
2. Re-run the source skill's sheet validation:

```powershell
python skills/create-character-sprites/scripts/validate_frames.py <sheet.png> --sheet --columns 8 --rows 5 --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta --magenta-threshold 140
```

3. Copy the passing sheet and metadata to `assets/sprites/<character-id>/<character-id>.png` and `.json`. Copy the approved creation reference to `assets/sprites/<character-id>/reference.png`; this runtime asset is the full-character preview used by character selection. Keep the bible and review reports beside them or in the established source-art workspace.
4. Add a `CharacterType` constant and one `CharacterDef` registration in `internal/entity/character.go`. Use the display name, relative sprite path, relative `ReferenceImagePath`, `128`, `192`, `8`, `5`, `RenderScale: 1.15` for the current detailed 128x192 art (or a validated character-specific value), and the established walk/sprint timing. Preserve registration order because `AllCharacters()` supplies the selection UI.
5. Run preflight against the new registry entry:

```powershell
python skills/create-character-sprites/scripts/preflight_renderer.py --character-id <character-id> --asset assets/sprites/<character-id>/<character-id>.png --metadata assets/sprites/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --columns 8 --directions S,SW,W,N,NW
```

6. Do not edit `internal/ui/character_select.go` merely to add a normal character: it already lists the registry and renders `ReferenceImagePath`, falling back to the first south-facing sprite frame for legacy records. Do not add logic to `internal/game/loop.go`. Use `assets.Path()` for any new texture load.
7. Format changed Go files and run `go test ./...`; compile or run the desktop target when practical. Inspect the character-select screen and verify that the new card displays a non-clipped south-facing first frame, selection returns its type, and local plus remote players load the registered sheet.
8. Update affected documentation and append one concise line to `doc/changelog.md` describing the installed character and its identifier.

## Handoff

Report the identifier, asset paths, registry file, validation results, and any character-specific limitation. Never commit, push, or build Android artifacts unless requested.

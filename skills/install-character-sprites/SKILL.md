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

1. Read `AGENTS.md`, `doc/coding_patterns.md`, `internal/entity/character.go`, `internal/entity/player_sprite.go`, `internal/ui/character_select.go`, `internal/ui/dialogue_box.go`, and the generated metadata. Inspect the active network path as well if a character identifier travels in multiplayer messages.
2. Re-run the source skill's sheet validation:

```powershell
python skills/create-character-sprites/scripts/validate_frames.py <sheet.png> --sheet --columns 8 --rows 5 --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta --magenta-threshold 140
```

3. Copy the passing sheet and metadata to `assets/sprites/<character-id>/<character-id>.png` and `.json`. Copy the approved creation reference to `assets/sprites/<character-id>/reference.png`; this runtime asset is the full-character preview used by BOTH character selection and the dialogue box, so it must be the character's own art. Keep the bible and review reports beside them or in the established source-art workspace.
4. Add a `CharacterType` constant and one `CharacterDef` registration in `internal/entity/character.go`. Use the display name, relative sprite path, relative `ReferenceImagePath`, `128`, `192`, `8`, `5`, `RenderScale: 1.15` for the current detailed 128x192 art (or a validated character-specific value), and the established walk/sprint timing. Preserve registration order because `AllCharacters()` supplies the selection UI.

   **`FootLine` is not optional.** Copy the sheet metadata's `anchor.y` into it (`186` for the current rig). The sprite is drawn centred on `Position` while the figure inside it stands, so the collision box is placed at the sole — `FootLine` is the only thing that says where the sole is. Omit it and the field falls back to `FrameHeight`, putting the box 7 px below the feet: little enough to go unnoticed, enough for the character to plant a foot inside every tree trunk. Do not re-measure the PNG; the metadata already declares the number, and two sources would drift. `internal/entity/character_ground_test.go` fails on a 192-tall character whose `FootLine` is not 186.
5. **Clear the dialogue portrait block.** `ReferenceImagePath` feeds TWO screens, not one: character selection (`internal/ui/character_select_preview.go`) and the dialogue box (`internal/ui/dialogue_box.go`). The dialogue box additionally consults `placeholderPortraits`, a map of characters whose art is borrowed from somebody else; a character listed there speaks with a blank portrait no matter what `ReferenceImagePath` points at. So:
   - If this install REPLACES borrowed art with the character's own, DELETE its entry from `placeholderPortraits`. Forgetting this is silent — selection shows the new art, dialogue stays blank, and no test or preflight complains. It happened to the Necromante on 2026-08-03.
   - If this install deliberately ships a character reusing another's art, ADD the entry, with a comment naming whose art it borrows.
   - Then grep the repository for the character's `CharacterType` constant and for `placeholderPortraits`, including mirrored copies under `work/*-verify/`, so a stale assertion elsewhere does not encode the old state.

6. Run preflight against the new registry entry:

```powershell
python skills/create-character-sprites/scripts/preflight_renderer.py --character-id <character-id> --asset assets/sprites/<character-id>/<character-id>.png --metadata assets/sprites/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --columns 8 --directions S,SW,W,N,NW
```

7. Do not edit `internal/ui/character_select.go` merely to add a normal character: it already lists the registry and renders `ReferenceImagePath`, falling back to the first south-facing sprite frame for legacy records. Do not add logic to `internal/game/loop.go`. Use `assets.Path()` for any new texture load.
8. Format changed Go files and run `go test ./...`; compile or run the desktop target when practical. Then check BOTH surfaces that consume the art, because passing tests say nothing about either:
   - Character select: the new card displays the reference art (not the first sprite frame), non-clipped, and selection returns its type.
   - Dialogue box: trigger a line spoken by the character and confirm the portrait renders. This is the check that catches a stale `placeholderPortraits` entry.
   - In game: local plus remote players load the registered sheet, and the N row reads as a square back view rather than a duplicate of NW.
9. Update affected documentation and append one concise line to `doc/changelog.md` describing the installed character and its identifier.

## Handoff

Report the identifier, asset paths, registry file, validation results, and any character-specific limitation. State explicitly whether `placeholderPortraits` was touched and whether the dialogue portrait was verified on screen — "installed" without that check has already shipped a character with a blank portrait. If the Go toolchain was unavailable, say so instead of implying the build passed. Never commit, push, or build Android artifacts unless requested.

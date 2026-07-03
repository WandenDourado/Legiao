---
name: create-character-sprites
description: Create and validate 2D character sprites from reference images for isometric/top-down RPGs. Use when the user provides a character reference or concept and asks for directional sprites, animation frames, walk/idle/combat sprite sheets, RPG character model sheets, frame validation, transparent PNG exports, or metadata for game integration.
---

# Create Character Sprites

## Overview

Use this skill to turn a character reference into consistent directional sprite frames and sprite sheets. Favor an iterative workflow: define the character once, generate one direction at a time, validate, then assemble the final sheet and metadata.

## Defaults

- Target style: readable isometric/three-quarter fantasy RPG character art, with compact proportions and clear feet placement. Use the user's references for mood and angle, but do not copy a named game's exact style.
- Recommended production sizes: start with `128x192` px for painterly high-readability characters, or `96x128` px for lighter/mobile-friendly characters. Use `64x96` only for simplified sprites. Keep the sheet internally consistent even if the engine later scales it.
- Recommended animation counts: `6` frames for walk/run, `4` frames for idle, `6-8` frames for attack/cast, and `4-6` frames for hit/death placeholders. Prefer fewer clean frames over many inconsistent frames.
- Output defaults: transparent PNG, origin at foot-center, one row per direction, one frame size for the full character set.
- Legiao compatibility note: the current wizard asset/code uses `165x246`, `6` walk frames, 5 rows (`N`, `S`, `W`, `SW`, `NW`) and mirrors `E`, `SE`, `NE` from `W`, `SW`, `NW` respectively. Treat this as legacy compatibility, not the project standard.
- Preferred full direction order: `S`, `N`, `E`, `W`, `SE`, `SW`, `NE`, `NW`.
- Direction aliases: accept Portuguese aliases (`L` = `E`, `O` = `W`, `SO` = `SW`, `NO` = `NW`) but export canonical `N/S/E/W/NE/SE/SW/NW`.

## Workflow

1. Gather inputs: reference image, target frame size, frame count, animation names, output folder, and whether asymmetrical details must stay on the same body side.
2. If details are missing, use market-oriented production defaults above and state the assumptions.
3. Create a concise character bible before generating frames. Include silhouette, clothing, palette, props, face/hair, asymmetric details, scale, and forbidden drift.
4. Generate a small model sheet first when the reference only shows one angle: `S`, `N`, `E`, and one diagonal. Validate it before animation frames.
5. Generate only one final animation direction per image request. Do not ask an image model for a final 8-direction sheet in one pass unless the user explicitly wants a rough concept sheet.
6. Validate each direction before moving to the next one. Read `references/validation-checklist.md` before accepting frames.
7. Assemble the final sheet with `scripts/build_sheet.py` and validate it with `scripts/validate_frames.py`.
8. Deliver the sprite sheet, metadata, assumptions, and any directions that still need manual review.

## References

- Read `references/art-direction.md` when adapting a visual reference into a character bible or writing image-generation prompts.
- Read `references/generation-workflow.md` when planning a full sprite production run.
- Read `references/validation-checklist.md` before approving any direction or final sheet.
- Read `references/metadata-format.md` when exporting or explaining animation metadata.

## Scripts

- `scripts/validate_frames.py`: validate PNG dimensions, frame grid dimensions, transparency mode when Pillow is available, and basic file consistency.
- `scripts/build_sheet.py`: assemble direction folders into a single PNG sheet and JSON metadata.

If Pillow is missing, install it in the active environment before using `build_sheet.py`. `validate_frames.py` can still read PNG dimensions without Pillow, but alpha/transparency checks are limited.

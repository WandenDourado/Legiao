---
name: create-character-sprites
description: Create and validate reference-driven 2D RPG character sprites. Use when asked for directional walk, idle, combat, or model-sheet sprites; to turn AI grids into transparent animation frames; or to preflight, review, assemble, and install a game-ready sprite sheet.
---

# Create Character Sprites

Generate and validate each animation direction in isolation. Preserve the supplied character's identity; describe observable visual traits instead of copying a named game's exact style. Do not include any visual effects (VFX) in the sprites (e.g., fire on a staff, glowing hands, mystical auras); generate only the physical character and their props.

## Export Contract

- Ship lossless RGBA PNGs with no visible magenta, a fully transparent one-pixel outer border, and one `128x192` frame size. Magenta is generation matte only.
- Use eight walk frames per direction. Generate each source cell at least 2x target size, key the matte before the single downsample, and target a visible silhouette height of `80%–92%` of the target frame (`84%` preferred for front/back views).
- Use fixed source anchors: torso center `x=64`, foot baseline `y=186`. Normalization may recenter an intact frame by at most 24 px and may never clip or rescale art; a frame that cannot move within its transparent margin must be regenerated.
- Characters must be **mirror-safe**. Make left/right clothing, weapons, hair accessories, and silhouettes visually symmetric in the approved model sheet; adapt a one-sided reference detail by duplicating, centering, or removing it. Never generate `E`, `SE`, or `NE` source rows.
- The Legiao sheet is exactly `S,SW,W,N,NW`, eight frames per row. The renderer mirrors `W→E`, `SW→SE`, and `NW→NE`; metadata records those mappings, fixed anchor, and the alternating left/right contact plan.

## Workflow

1. Read `references/character_creation_guide.md`. Choose a lowercase kebab-case `<character-id>` and create a concise bible: silhouette, palette, symmetric props, target scale, fixed anchors, and forbidden drift. With one reference view, first approve a five-view model sheet.
2. Freeze the approved bible/model sheet before animation. This gate is serial: a changed identity, asymmetric detail, or scale invalidates all direction work.
3. Plan phases. After the model gate, use an orchestrator to spawn independent workers for `S`, `SW`, `W`, `N`, and `NW` when capacity and image-generation limits permit. A worker owns one direction and an isolated `work/<character-id>/attempts/<direction>/` directory; it may regenerate only its own direction. Parallelize generation, slicing, validation, and review across ready directions. Keep model approval, final selection, sheet assembly, and final validation serial. If the image service cannot run concurrently, serialize generation but parallelize the local processing stages.
4. Each worker generates one unguttered 2x4 grid at `1024x768` or larger. Require transparent safe margin around the body, a fixed `x=64` torso pivot, `y=186` foot baseline, and `80%–92%` visible height after export (`84%` preferred for front/back). Follow the guide's eight-pose plan. Never request a multi-direction final sheet.
5. Slice, normalize, validate, and review every candidate before accepting it. Regenerate a rejected direction; do not force large shifts, crop, scale, or substitute another direction.

```powershell
python skills/create-character-sprites/scripts/slice_and_stitch.py work/<character-id>/attempts/S/attempt-1/grid.png --direction S --output-root work/<character-id>/attempts/S/attempt-1/sliced --frame-width 128 --frame-height 192 --minimum-source-scale 2
python skills/create-character-sprites/scripts/normalize_frames.py --input-root work/<character-id>/attempts/S/attempt-1/sliced --output-root work/<character-id>/attempts/S/attempt-1/frames --directions S --frame-width 128 --frame-height 192 --max-shift 24 --anchor-x 64 --baseline 186
python skills/create-character-sprites/scripts/validate_frames.py work/<character-id>/attempts/S/attempt-1/frames/S/*.png --frame-width 128 --frame-height 192 --require-alpha --require-transparent --require-clear-border --reject-magenta --magenta-threshold 140 --check-baseline --baseline-tolerance 1 --expected-baseline 186 --check-center --center-tolerance 1 --expected-center 64 --min-foreground-height-ratio 0.80 --max-foreground-height-ratio 0.92
python skills/create-character-sprites/scripts/review_animation.py work/<character-id>/attempts/S/attempt-1/frames/S --gif work/<character-id>/review/S.gif --contact-sheet work/<character-id>/review/S.png --report work/<character-id>/review/S.json --frame-width 128 --frame-height 192
```

6. The orchestrator copies only accepted frame sets into `work/<character-id>/frames/`, then assembles and validates the single sheet. For a registered character, run renderer preflight; otherwise leave the passing package for `$install-character-sprites`.

```powershell
python skills/create-character-sprites/scripts/build_sheet.py --input-root work/<character-id>/frames --output work/<character-id>/<character-id>.png --metadata-output work/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --frames-per-direction 8 --directions S,SW,W,N,NW --anchor-x 64 --baseline 186
python skills/create-character-sprites/scripts/validate_frames.py work/<character-id>/<character-id>.png --sheet --columns 8 --rows 5 --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta --magenta-threshold 140
python skills/create-character-sprites/scripts/preflight_renderer.py --character-id <character-id> --asset assets/sprites/<character-id>/<character-id>.png --metadata work/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --columns 8 --directions S,SW,W,N,NW
```

7. Deliver the sheet, metadata, bible, approved source reference, model sheet, per-direction GIF/contact sheet/report, and an incremental attempt manifest. Record every accepted/rejected attempt with its reason and metrics. `$install-character-sprites` copies the approved reference to `assets/sprites/<character-id>/reference.png`.

## Tools

- `preflight_renderer.py`: compare the requested export with the live Legiao renderer constants and asset path, then open the PNG and cross-check its alpha, dimensions, and metadata contract.
- `slice_and_stitch.py`: split a 2x4 grid, key and despill magenta, and downsample production frames once.
- `normalize_frames.py`: correct small torso/foot-anchor drift to a fixed pivot without clipping or resizing frames.
- `validate_frames.py`: validate geometry, alpha, matte leakage, transparent borders, silhouette scale, torso center, and foot baseline; require every frame to meet the specified anchor targets.
- `review_animation.py`: create a looped GIF, a baseline/torso contact sheet, and an audit report.
- `build_sheet.py`: assemble direction folders and metadata.

Install Pillow in the active environment before using the image scripts.

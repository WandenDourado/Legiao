---
name: create-character-sprites
description: Create and validate reference-driven 2D RPG character sprites. Use when asked for directional walk, idle, combat, or model-sheet sprites; to turn AI grids into transparent animation frames; or to preflight, review, assemble, and install a game-ready sprite sheet.
---

# Create Character Sprites

Generate and accept one animation direction at a time. Preserve the supplied character's identity; describe observable visual traits instead of copying a named game's exact style.

## Export Contract

- Ship lossless RGBA PNGs with one frame size, stable foot anchor, and no visible magenta. Magenta is generation matte only.
- Default to eight `128x192` walk/run frames. Generate each source cell at least 2x target size, then downsample once.
- Keep asymmetric props on the same body side. Do not mirror a source direction until that remains true.
- For Legiao's wizard, export `S,SW,W,N,NW`, eight frames per row, at `128x192`. The renderer mirrors `W/SW/NW` for `E/SE/NE` and loads `assets/sprites/wizard/wizard.png`.
- Metadata does not configure the current renderer. Run preflight before generation and do not install until staging validation passes.

## Workflow

1. Read `references/character_creation_guide.md`; write a concise bible (silhouette, palette, props, body-side asymmetries, forbidden drift). Approve a five-view model sheet before animation when only one view exists.
2. Verify the active renderer contract:

```powershell
python skills/create-character-sprites/scripts/preflight_renderer.py --asset assets/sprites/wizard/wizard.png --frame-width 128 --frame-height 192 --columns 8 --directions S,SW,W,N,NW
```

3. Generate one direction as an unguttered 2 rows x 4 columns grid at `1024x768` or larger for the default export. Follow the guide's eight-pose plan; do not request a multi-direction final sheet.
4. Slice every approved grid into staging frames, then normalize only small anchor drift. If normalization rejects a direction, regenerate it; never force clipping or compensate by scaling.

```powershell
python skills/create-character-sprites/scripts/slice_and_stitch.py work/raw/S.png --direction S --output-root work/sliced --frame-width 128 --frame-height 192 --minimum-source-scale 2
python skills/create-character-sprites/scripts/normalize_frames.py --input-root work/sliced --output-root work/frames --directions S,SW,W,N,NW --frame-width 128 --frame-height 192 --max-shift 4
```

5. Validate and review each direction before proceeding. Replace `S` below for every direction; use the GIF for motion and the contact sheet to inspect the red baseline.

```powershell
python skills/create-character-sprites/scripts/validate_frames.py work/frames/S/*.png --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta --check-baseline --baseline-tolerance 2
python skills/create-character-sprites/scripts/review_animation.py work/frames/S --gif work/review/S.gif --contact-sheet work/review/S.png --report work/review/S.json --frame-width 128 --frame-height 192
```

6. Assemble and validate staging. Copy only a passing export to the path verified in step 2:

```powershell
python skills/create-character-sprites/scripts/build_sheet.py --input-root work/frames --output work/wizard.png --metadata-output work/wizard.json --frame-width 128 --frame-height 192 --frames-per-direction 8 --directions S,SW,W,N,NW
python skills/create-character-sprites/scripts/validate_frames.py work/wizard.png --sheet --columns 8 --rows 5 --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta
Copy-Item work/wizard.png assets/sprites/wizard/wizard.png
Copy-Item work/wizard.json assets/sprites/wizard/wizard.json
```

7. Deliver the sheet, metadata, bible, and per-direction review reports. Record an accepted/rejected decision and reason for every direction.

## Tools

- `preflight_renderer.py`: compare the requested export with the live Legiao renderer constants and asset path.
- `slice_and_stitch.py`: split a 2x4 grid, key and despill magenta, and downsample production frames once.
- `normalize_frames.py`: correct small foot-anchor drift without clipping or resizing frames.
- `validate_frames.py`: validate geometry, alpha, matte leakage, and body baseline.
- `review_animation.py`: create a looped GIF, a baseline contact sheet, and an audit report.
- `build_sheet.py`: assemble direction folders and metadata.

Install Pillow in the active environment before using the image scripts.

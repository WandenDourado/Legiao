# Generation Workflow

## Suggested File Layout

```text
sprites/<character_slug>/
  source/reference.png
  bible.md
  frames/
    S/000.png
    S/001.png
    ...
    N/000.png
  sheets/<character_slug>_walk.png
  sheets/<character_slug>_walk.json
```

## Direction Order

Generate in this order:

1. `S` to establish face, robe front, staff, and palette.
2. `N` to establish hood/back silhouette.
3. `E` and `W` to validate side readability.
4. `SE`, `SW`, `NE`, `NW` after the cardinal directions are stable.

Use mirroring only after checking asymmetry. A staff, shield, pouch, scar, hair part, or emblem may make a mirrored direction wrong.

## Per-Direction Loop

1. Write or update the character bible.
2. Create a direction prompt from `art-direction.md`.
3. Generate only that direction.
4. Save frames as zero-padded PNG files in the direction folder.
5. Run `scripts/validate_frames.py` on the direction.
6. Visually inspect the row/contact sheet.
7. Accept, regenerate, or manually edit before moving to the next direction.

## Common Failure Corrections

- Character changes age or face: strengthen the bible and reuse the approved `S` direction as an additional reference.
- Feet slide vertically: require same floor baseline and reject frames with robe/boots jumping.
- Staff swaps hands: specify body side instead of screen side; mention whether mirroring is allowed.
- Diagonal is too frontal: request visible shoulder/foot overlap and a clear diagonal torso angle.
- Ornament drift: simplify tiny embroidery if needed, but keep trim placement and color rhythm consistent.

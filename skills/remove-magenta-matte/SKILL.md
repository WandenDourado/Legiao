---
name: remove-magenta-matte
description: Remove residual pink/magenta left by the #FF00FF generation matte from Legiao art (portraits, reference art, illustrations) without eating the character's outline or legitimately violet paint. Use when the user reports pink edges, a pink halo, or pink spots inside hair/gaps, or after keying any new generated art.
---

# Remove Magenta Matte

The theory, the measured numbers, and what went wrong before are in
`doc/matte_magenta.md`. Read it before changing any threshold. This file is the
procedure.

## Scripts

| Script | What it does |
|---|---|
| `scripts/despill_matte.py in.png out.png` | Two passes. **Edge**: 8 px band, re-colors matte-mixed pixels with the neighboring interior color at their own luminance; alpha drops only in the outer 2 px ring. **Interior** (`interior()`): pixels much pinker than their 15 px neighborhood (matte trapped in hair gaps, between arrows). Protects context that is already violet (Necromante) and switches to strict mode (G < 25) for violet-lit characters. |
| `scripts/audit_matte.py a.png b.png --overlay-dir DIR` | Prints edge/interior/violet counts per file and writes an overlay on GREEN with the interior spots circled. Use it before and after. |

Both need Pillow, numpy, scipy (`pip install scipy` if the machine lacks it).

## Procedure

1. **Back up the originals outside `assets/`** (e.g.
   `work/<area>/antes_despill_<date>/`). Always process FROM the originals,
   never on top of an output that was already cleaned.
2. **Audit the whole cast together**:
   `audit_matte.py assets/sprites/*/portrait.png assets/sprites/*/reference.png --overlay-dir work/<area>/auditoria`.
   A character that stands out from the rest is the signal, not any number on
   its own. `violeta pintado > 1%` means that character's art is violet on
   purpose, so look at its overlay before you trust its counts.
3. **Clean** each file: `despill_matte.py <original> <tmp>`, then copy the tmp
   over the asset. Writing through a temp file also avoids the Windows lock
   error when the PNG is open in a viewer.
4. **Verify by eye**, not only by numbers:
   - side by side before/after at 3–4x zoom, on GREEN and on dark gray;
   - for a violet character, a diff map (changed pixels in green over the
     darkened art): changes must hug the silhouette and hair gaps, never cover
     the flame or the lit skin;
   - re-run `audit_matte.py`. Edge and interior should drop to near zero for
     non-violet characters.
5. **Record** the run in the area's README (what was found, backup folder).

## When it is not this skill

- 128x192 sprite frames: use `create-character-sprites/scripts/auto_repair.py`.
  The windows here assume 1000+ px canvases.
- Scenery atlases: `create-tiled-assets/scripts/key_matte.py` does its own
  unblend and edge bleed.
- A defect that is actually **painted** (the generator drew a pink object):
  that calls for a correction prompt, "one defect, one sentence", as in
  `work/portraits/README.md`. Don't use a color filter for it.

## Guardrails

- Never delete pink pixels (alpha 0 / black). That eats the outline. Re-color
  at the same luminance.
- Never widen the hue window or lower the green threshold globally to catch a
  single spot. Check the Necromante's diff map after any threshold change.
- The new portraits already come out clean: `key_and_frame.py` calls
  `despill_matte.clean()` before saving (`--no-despill` turns it off for
  diagnosis).

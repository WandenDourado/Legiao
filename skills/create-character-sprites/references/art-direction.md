# Art Direction

## Target

Create readable fantasy RPG character sprites for an isometric or top-down camera. The target should feel compact, painterly, and game-readable, with a clear floor contact point and a silhouette that survives animation.

Use franchise names supplied by the user only as broad angle and readability references. Do not copy exact costumes, proportions, line treatment, UI style, or recognizable franchise assets.

## Legiao Wizard Reference

When the user provides the blue hooded wizard reference, preserve these traits:

- Young mage silhouette with large hood and compact body proportions.
- Deep blue robe and hood with gold trim and ornamental embroidery.
- Brown hair visible under the hood, warm face, large expressive eyes.
- Brown boots, belt, pouch, and leather accents.
- Wooden staff held on the character's right side with a cyan glowing crystal.
- Cyan magic glow should be a small accent, not a background effect.

## Prompt Pattern

Use this pattern for image tools:

```text
Create [frame_count] transparent-background game sprite frames for one character animation direction.
Character reference: [brief character bible].
Animation: [walk/idle/etc.], direction: [canonical direction].
Camera/style: isometric three-quarter fantasy RPG sprite, compact readable proportions, clean silhouette, feet aligned to the same baseline, consistent costume details, transparent PNG.
Frame size: [width]x[height] px per frame.
Output: individual frames or a single row contact sheet, no background, no labels, no shadows outside the sprite silhouette unless requested.
Constraints: keep staff/weapon on the correct body side, preserve palette, avoid changing age/body shape/costume, avoid cropped parts.
```

## Size Guidance

- Prefer production-friendly frame boxes: `128x192` for painterly RPG characters, `96x128` for lighter/mobile-friendly characters, and `64x96` for simpler sprites.
- Use larger custom frames only when the silhouette or weapon genuinely needs the space; avoid odd sizes unless a legacy asset requires them.
- Use `165x246` only when replacing or matching the current Legiao wizard asset/code path.
- Keep the visible character slightly inside the frame: leave safe padding around head, staff, robe hem, and effects.
- Anchor the character at foot-center; all frames in one animation must share the same floor baseline.

---
name: create-enemy-sprites
description: Create, validate and install enemy sprite sheets for Legiao. Use when asked for a new monster, creature or enemy sprite; to turn AI-generated grids into game-ready frames; or to review, normalize and register an enemy's art, hitbox and combat stats.
---

# Create Enemy Sprites

Enemies are cheaper than player characters on purpose. A character needs five
directional rows, mirror-safety and an eight-pose walk plan
(`create-character-sprites`). An enemy needs one view and a short loop.

## The one lesson that shapes everything

**The image generator obeys art direction and ignores geometry.** This was
measured repeatedly, not assumed: prompts specifying exact per-frame widths,
fixed centres and containment radii were disregarded every single time, across
six generations and two creatures.

So the division of labour is fixed:

| Owned by the prompt | Owned by the scripts |
|---|---|
| identity, palette, finish | frame size and squareness |
| lighting approach | centring on the pivot |
| pose semantics, anatomy | per-frame scale |
| silhouette topology (what connects to what) | containment in the inscribed circle |
| what the creature *is* | matte keying and downsampling |

Never spend a generation to fix centring, scale or containment. Never expect the
generator to hit a number. Ask it for the drawing; measure and correct the rest.

## Mode selection — do this first, with the user

**Radial** — one true top-down view, rotated at runtime toward the velocity
vector. One sheet, free 360-degree facing, cheapest possible.

**Directional** — a 3/4 sheet with one row per direction, like the player
characters. Not implemented yet; build it when a creature demands it.

Radial works when the creature reads from directly above and has no rigid,
light-catching surface: slimes, swarms, insects, floating orbs, quadrupeds.
Radial fails on anything whose appeal is a face seen head-on, and on rigid
shapes (armour, shields, skulls) where a baked highlight visibly spins.

Ask the user, with a recommendation. Do not decide alone: choosing wrong costs
the entire pipeline.

## Prompt rules earned the hard way

Each of these comes from a specific failed generation.

1. **Give exactly one spatial anchor.** A wolf prompt said both "faces the top of
   the frame" and "tail toward the bottom"; the generator drew the body facing
   down and honoured the tail instruction, producing a tail growing out of the
   face. State the body layout once, as an ordered list from the top edge of the
   cell to the bottom edge, and give no other orientation cue.

2. **Distinguish tilted from rotated.** Asking for a visible face made the
   generator rotate the head to meet the camera, which dragged the whole camera
   into a frontal view. Say "tilted back along the body's own axis" and describe
   what is visible (top of the skull, foreshortened muzzle), never "facing the
   viewer".

3. **Width comes from the torso, never from the limbs.** Asking for a wider
   silhouette produced legs splayed sideways into an X with a pinched waist.
   State the rule geometrically instead: *the widest cross-section is always
   mid-body; shoulders and hips are narrower.* Geometric rules are both easier
   to obey and verifiable — `check_gait.py` measures exactly this.

4. **Limbs swing along the body axis.** Toward the top and bottom edges of the
   cell, never toward the left and right edges.

5. **Alternate the lead limb.** Real quadrupeds hold one lead leg for several
   strides, but in a short loop seen from above that reads as a limp. Structure
   the cycle as two half-strides that swap the lead, the second half mirroring
   the first.

6. **Light by marking, not by direction.** The generator ignores "zenithal
   lighting" and paints an upper-left key light every time. Stop fighting it:
   ask for flat, shadowless, overcast lighting and put the entire value
   structure into the creature's own markings (dark spine over a lighter coat).
   With no directional light present, rotation has no inconsistency to expose.
   This is the single most important radial-mode technique.

7. **Do not chase readability by distorting anatomy.** A wolf seen from above is
   genuinely about 0.30 as wide as it is long. Widening it to fill the frame
   produced the X pose. Readability comes from `RenderScale` in the `EnemyDef`,
   which costs nothing.

8. **Paste measured defects verbatim into the next prompt.** "The legs must
   alternate" is vague. "Your left paw reaches 14 and your right never passes 40"
   converges in one retry. Every regeneration prompt should quote the numbers the
   scripts produced.

9. **Regenerate conditioned on the accepted image.** Once identity and palette
   are right, attach the previous sheet and frame the request as a correction to
   named defects, restating the rules already satisfied so they are not undone.

## Workflow

1. **Design gate.** Agree with the user: mode, silhouette, palette family,
   combat stats, and scale against the ruler (a door is 150 px, the hero is
   186 px tall). Write the prompt to `work/enemy-sprites/prompt-N-<name>.md`
   with the ready-to-paste block in its own section.

2. **Generate.** The host may have no image tool; if so, hand the prompt to the
   user and continue when they return the grid. Ask for a grid of equal square
   cells on a flat `#FF00FF` matte, at least 512 px per cell.

3. **Normalize** — no generation, no judgement calls.

   ```bash
   python skills/create-enemy-sprites/scripts/normalize_radial.py raw.png \
       --out work/enemy-sprites/<name>/ --cols 4 --rows 2 --frame 128 \
       --fit uniform --extent 78
   ```

   `--fit uniform` preserves the aspect ratio: use it for anything anatomical,
   where distorting proportions per frame deforms the animal. `--fit stretch`
   with `--targets` forces exact per-frame dimensions: use it only for amorphous
   bodies, where that distortion *is* the squash-and-stretch.

4. **Validate geometry.** Pass the `--pivot` that matches the fit mode:
   `--fit stretch` centres on the centroid, `--fit uniform` on the bounding-box
   centre. Measuring with the wrong one reports drift on a correct sprite — a
   body with an appendage (the slime's trailing drip) has the two several pixels
   apart.

   ```bash
   python skills/create-enemy-sprites/scripts/validate_radial.py work/enemy-sprites/<name>/ \
       --pivot bbox      # centroid for --fit stretch
   ```

   Exit 1 rejects. Exit 3 warns — read the warnings, they usually prescribe a
   `RenderScale` or palette change rather than a regeneration.

   The flat-black warning is gated on saturation, deliberately: a dark but
   saturated body reads by hue (the crimson slime sits 75% below luminance 70
   and is perfectly legible at 0.93 saturation), while a dark *desaturated* body
   collapses into one blob (the first wolf, 89% below 70 at 0.08 saturation).

5. **Validate locomotion**, for creatures with limbs only.

   ```bash
   python skills/create-enemy-sprites/scripts/check_gait.py work/enemy-sprites/<name>/
   ```

6. **Look once**, at game scale, on the actual terrain, rotated. Composite the
   frames over the grass colour `#699032` at the intended `RenderScale` and view
   the cycle and eight rotation angles side by side. Every defect the scripts
   cannot see — the creature reading as a smudge, the lighting spinning, the
   pose looking like roadkill — shows up here and nowhere else. Do this before
   accepting, always.

7. **Assemble and install.**

   ```bash
   python skills/create-enemy-sprites/scripts/assemble_sheet.py work/enemy-sprites/<name>/ \
       --out assets/sprites/enemies/<name> --name <name> \
       --frame-time 0.07 --animation run --note "..."
   ```

8. **Register** in `internal/entity/enemy_sprite.go`. One `RegisterEnemy` call
   carries art *and* hitbox *and* combat stats — see below. Add the `EnemyType`
   constant in `enemy.go` and the spawn share in `internal/network/host_spawn.go`.

9. **Build.** `go build ./...` and `go vet ./...`.

## Resolution and filtering

Two settings decide whether the sprite looks crisp or blocky in game, and both
were originally wrong.

**Author frames larger than they are drawn.** Work out the on-screen size first
(from the door/hero ruler), then pick a frame size at or above it and set
`RenderScale` below 1 to land on that size. Frames at 128 px drawn at `1.8`
magnify by 1.8x and look like blocks. The same sprite at 256 px with
`RenderScale 0.9` is the identical size on screen but arrives by reduction.

```
slime  256 px frame x 0.575 = 147 px on screen   (was 128 x 1.15, magnified)
wolf   256 px frame x 0.9   = 230 px on screen   (was 128 x 1.8,  magnified)
```

Keep the frame at or below half the generator's source cell so the single
downsample in `normalize_radial.py` is still a reduction: a 512 px cell
comfortably yields a 256 px frame.

**Set bilinear filtering.** raylib defaults to point sampling, which is correct
for pixel art and wrong here — these are painterly paintings. `ApplySpriteFilter`
in `internal/entity/enemy_sprite.go` sets `rl.FilterBilinear`, and every sprite
texture load must call it: enemies, the local player, and the remote-player
cache in the renderer.

## Integration contract

`EnemyDef` is the single source of truth for an enemy. `NewEnemy` reads every
stat from it, so balancing means editing one registration.

- **Art and hitbox are sized together.** A 100 px sprite over a 30 px hitbox
  makes the player swing at empty space. `Radius` lives in the same struct as
  `RenderScale` for exactly this reason.
- **A circle cannot fit an elongated body.** The wolf is 180 px long and 55 px
  wide; its radius of 50 covers the torso and leaves muzzle and tail outside.
  Record the compromise in a comment.
- **The runtime rotation is `atan2(vy, vx) + 90` degrees**, because the art's
  front points at screen `-Y`. Facing is eased toward the heading at `TurnRate`
  degrees per second, never snapped — a snap reads as the silhouette teleporting.
- **Enemies separate from each other.** `Radius` also drives the separation
  steering and the overlap resolution in `EntityManager.UpdateAll`, so an
  oversized radius spreads a pack thin and an undersized one lets sprites stack.

## Colour language

Established by the two enemies that exist; keep it coherent.

- The heroes are the **lightest, least saturated** things on screen. No enemy may
  compete with them for brightness.
- Separate the enemy from the terrain by **hue** or by **value**, and say which.
  The slime is jade green at hue 150 against grass at hue 85 — a hue separation
  that survives their similar luminance (111 against 122) because the slime is
  far more saturated. The wolf is charcoal against everything — a value
  separation, with red eyes carrying the enemy tag.
- A green creature on green grass is workable, but only at a genuinely different
  hue. Grass is *yellow*-green; a jade or emerald green reads as a separate
  colour, while a leaf green does not.
- Grey-brown creatures collide with stone `#A8A595` and dirt `#B88E45`. Pick a
  side of the value scale and commit.
- Separating by value has a trap: too dark and the creature loses its *internal*
  contrast and reads as one flat blob in game. `validate_radial.py` warns above
  70% flat black, but only when saturation is also low — a dark saturated body
  still reads by hue. The first wolf was 89% flat black at 0.08 saturation and
  was unusable.
- **Recolouring is cheap and does not need a regeneration.** Rotating hue in HSV
  while preserving saturation and value keeps the whole shading structure
  intact; the crimson slime became the green one this way. Skip the magenta
  matte when rotating, or the background shifts with the body.

See `references/enemy_creation_guide.md` for the full art direction, and
`doc/art_style.md` for the game-wide visual contract.

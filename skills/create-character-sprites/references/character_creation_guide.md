# Character Sprite Guide

## Prompt

Use the reference as the identity source. Record hair, clothing, palette, symmetric props, scale, and prohibited changes (ensure visual effects like fire or glowing auras are explicitly removed). Adapt any one-sided reference detail by duplicating, centering, or removing it: every playable character must be safe to mirror. **Centering a back-mounted prop (quiver, scabbard, pack) needs two extra clauses or the generator "symmetrizes" it wrongly:** (1) demand ONE single compact cluster/tube at the exact center of the back — prompted only as "centered", the generator splits the prop into two mirrored bundles peeking over BOTH shoulders with an empty middle (archer model sheet, 2026-07-20); (2) state per-view visibility explicitly — e.g. "visible only in W/N/NW; in the front views nothing of it shows above the shoulders or beside the head" — otherwise front views grow phantom prop parts to satisfy symmetry. Describe style with observable traits such as painterly volume, soft gradients, readable silhouette, restrained outlines, and game-scale detail.

For a `128x192` frame, generate the eight poses as **two conditioned 4-pose batches**, each an unguttered 1 row x 4 columns image at `1024x384` or larger. Approve batch 1 (poses 1–4) first, then generate batch 2 (poses 5–8) conditioned on the accepted batch-1 image so the second half inherits the exact same camera facing, scale, pivot, and padding. Text alone does not keep the halves consistent — the visual reference does. Keep the torso pivot at target `x=64`, foot baseline at target `y=186`, enough transparent margin for safe recentering, and the visible character at `80%–92%` target-frame height (`84%` preferred for front/back). Require a solid `#FF00FF` matte only; prohibit that color on the character. (A single 2 rows x 4 columns grid is still accepted downstream; the batched form exists to prevent second-half inversion and to allow regenerating only the broken half.)

```text
# BATCH 1 - poses 1-4
Create frames 1-4 of one walk direction: [DIRECTION].
Character: [approved mirror-safe bible and reference]. Keep all left/right visual details symmetric.
Facing: state it explicitly AND condition on the matching model-sheet view. Crop the corresponding view from the approved five-view model sheet and pass it as a visual reference alongside the text. **Never condition a grid generation on a single vertical figure** — the generator imitates the reference's COMPOSITION and returns one vertical figure instead of the 1x4 grid (this wasted a generation). First tile the crop into the target layout with `tile_reference.py` (4 copies on the magenta matte with wide gaps) and condition on the 4-up; layout pull then works FOR the grid. For N: "seen from BEHIND — the back of the head/hair and the cape's back; the face is NOT visible". For NW: "back three-quarter view — mostly the back and cape, sliver of the far cheek at most, NO frontal face". For W: "pure left profile". Generators silently default to front-facing when the facing is only implied — a whole direction generated front-facing passes every geometric gate, so this line is the only cheap place to stop it.
Layout: EXACTLY 4 figures — no more, no fewer — in 1 row x 4 equal columns, left-to-right, evenly spaced with clear matte gaps between them (generators love to sneak in 5-7 figures; the slicer counts figures on the matte and rejects any other number). In EVERY cell keep the character horizontally CENTERED with the torso column at the cell's horizontal center, and leave a clear transparent margin on BOTH sides — the widest pose (including a flared cape/robe) must NOT touch the left or right cell edge. Target torso pivot x=64 and foot baseline y=186 after export; identical scale and padding in every cell.
Scale: request the whole character (including cape flare and any prop) at only **65–75% of the cell height and at most ~60% of the cell width**, centered, with fat transparent margins all around. Do not fear generating "too small": in the supersampled (2x) pipeline `scale_fit` up-fits the set to the target height at working resolution and `finalize_frames` still ends in a net downsample of the source art — generous margins cost NOTHING in sharpness, while a tight fit causes clip refusals and paid regenerations. Wide-garment characters especially: err well on the small side.
Background: solid #FF00FF matte only. No text, borders, panels, duplicate limbs, cropped parts, cast shadows, pixel-art conversion, or visual effects (e.g., fire, glowing hands, auras).
Walk plan: [PASTE POSES 1-4].
Secondary motion: [staff / robe / hair instructions]. Keep every prop attached and continuous between frames; keep a billowing cape controlled so it stays within the side margins.
```

```text
# BATCH 2 - poses 5-8, CONDITIONED on the approved batch-1 image (pass it as a visual reference)
Continue the SAME walk direction: [DIRECTION], keeping the identical camera facing, scale, pivot, padding, and identity shown in the reference image. Do NOT mirror, rotate, or flip the character's facing.
Layout, canvas, matte, and prohibitions: identical to batch 1.
Walk plan: [PASTE POSES 5-8]. Advance the cycle so the LEADING FOOT is the opposite one from pose 1 (see below); do not repeat pose 1.
```

## Eight-Pose Walk Plan

Use body-left/body-right, never screen-left/screen-right.

1. Contact A: left foot forward, right foot back; left heel contacts.
2. Down A: left foot supports weight; hips lower.
3. Passing A: right leg passes under the body; left foot remains planted.
4. Up A: body rises; right leg advances toward contact.
5. Contact B: right foot forward, left foot back; right heel contacts.
6. Down B: right foot supports weight; hips lower.
7. Passing B: left leg passes under the body; right foot remains planted.
8. Up B: body rises and returns toward pose 1 without duplicating it.

Arms oppose legs. Keep every held or worn prop's attachment point stable; let a long prop follow the torso with a small delayed arc, never teleport. Let clothing, hair, and loose accessories lag by one pose while retaining their silhouette and attachment points.

For the SIDE view (`W`), state the stride explicitly per contact pose in the prompt — generators default to a shuffle where one foot never overtakes the other (a moonwalk). Require: "in pose 1 the LEFT foot is clearly planted AHEAD of the right foot with visible separation; in pose 5 the RIGHT foot is clearly planted ahead of the left; during poses 3 and 7 the moving foot visibly passes the planted one". Also pin garment volume: "the cape/robe keeps the same length and volume as the model sheet" — an instruction to tuck a garment for width reasons must never shrink it relative to the other directions.

Reject the cycle when the two contact phases show the same leading foot. Review frame 1 against frame 5 first: frame 1 must visibly place the **left** foot forward and frame 5 the **right** foot forward. Then confirm frame 3 retains the left support foot and frame 7 retains the right support foot.

For a pure side view (`W`) the feet must visibly cross past each other during the passing poses (3 and 7); a walk where the feet only shuffle in place under the hem reads as a moonwalk and must be regenerated with a wider stride and clearer heel/toe contact. Deciding the lead foot from pixels is unreliable for every body type — a robe hides the feet, and with visible legs the two contact poses have near-identical silhouettes — so `structural_check.py` always returns `needs_visual_review` for a side view: confirm alternation with a single targeted look at the lower-body band of frames 1/3/5/7 rather than repeatedly reviewing the full grid.

## Lateral facing of model-sheet views (fix at the model gate, for free)

The renderer convention is the W family faces **viewer-left**: `W` = pure left profile, `SW` = front 3/4 with the gaze toward viewer-left, `NW` = back 3/4 with the head turned toward viewer-left (installed paladina rows are the ground truth). Generators frequently return SW/NW views facing viewer-RIGHT even when S/W/N are correct (archer model sheet, 2026-07-20). A model view whose lateral facing disagrees with its target row makes every downstream `tile_reference` conditioning fight the generation prompt — the historical source of paid facing retries. At the model gate, check each view's lateral facing against the convention; for a mirror-safe (fully symmetric) approved design, correct a wrong-facing view with a FREE horizontal flip of that cell before approving — never by regenerating the sheet and never by leaving the conflict for the direction prompts to win. While compositing, normalize all near-matte pixels to exact `#FF00FF`: cells patched from separate generations otherwise show mismatched magenta tones that survive into keying.

**Smallest-unit repair applies to the model sheet too.** When the gate rejects ONE view of an otherwise good sheet (wrong prop rendering, missing detail), do not regenerate the five-view sheet — the four approved views would be re-rolled and can drift. Generate a SINGLE figure of the offending view conditioned on the approved sheet image ("indistinguishable from the other views"), then composite it locally for free: mirror if the lateral facing requires it, scale the figure bbox to the incumbent view's bbox height, paste at the same baseline with a non-matte mask, and normalize the whole sheet's matte (archer SW, 2026-07-20: one cell regenerated, zero drift on the other views).

## Centering and margin (approve at the model sheet)

A character drawn edge-to-edge or off-center in its cell cannot be re-anchored to the fixed torso pivot without clipping, which forces a real regeneration. Catch this once, at the model-sheet gate, not after generating five directions: the approved model sheet must show the character centered with clear side margins at the target scale. Wide capes/robes are the common offender — approve a slightly smaller, controlled silhouette. If `scale_fit.py` reports it cannot fit "without dropping below the height floor" or that the art is "too wide/off-center", the generation (not the tooling) is at fault: regenerate that direction smaller and centered.

## Acceptance

Let the scripts decide before spending generation or vision tokens; accept "good enough" rather than chasing pixel perfection with new images.

- Each frame must reach the fixed `x=64` torso pivot and `y=186` foot baseline after normalization. A one-pixel raster tolerance may be used during the first validation pass, but repeat bounded normalization before accepting a set that still varies between frames; held props must not influence either measurement.
- The one-pixel outer frame border is fully transparent, no magenta fringe survives at visible alpha, and the visible silhouette occupies `80%–92%` of the frame height (`84%` preferred for front/back).
- Playback shows the contact, down, passing, and up phases in both halves of the cycle; poses 1/2 and 5/6 are not duplicates, and the second half is not a mirrored/inverted copy of the first.
- The red baseline contact sheet shows no vertical pop; the GIF shows no horizontal snap between frames 4 and 5.
- Alpha edges are free of magenta fringe, the character remains sharp at native gameplay size, no frame is clipped, and every design detail remains valid after horizontal mirroring.

### Accept / repair / regenerate

- **Cosmetic only** (magenta fringe, non-transparent border, small anchor drift): `validate_frames.py --score` recommends `repair`. Run `auto_repair.py` / `normalize_frames.py` and re-score. Do not regenerate; do not review with vision. A set scoring at or above the acceptance threshold with only such residue is accepted as-is — these are trivially fixable and never justify a new image.
- **Structural** (orientation flip of the second half, frozen cycle, or — when legs are visible — no lead-foot alternation): `structural_check.py` reports `hard_fail`. Regenerate only the guilty batch, pasting the reported frame indices / verdict into the prompt so it converges in one retry.
- **Scale** (silhouette height outside `80%–92%`): this is a **repair**, not a regen. Run `scale_fit.py` to rescale the whole direction's set by one shared factor to `84%` and re-anchor — motion and proportions are preserved. Do not regenerate to chase the height band. If `scale_fit.py` refuses because the fit is width-locked (a trailing cape/cloak in side views), run `diagnose_direction.py`: when it prescribes `trim_overhang`, the free chain `trim_overhang.py → scale_fit → auto_repair → normalize_frames → auto_repair → re-score` lands the set at target with a feathered, feet-safe shave of a few px off the overhang tip. Only a wrong *frame size*, or an overhang the diagnostician rules untrimmable, is a true regen.
- **Clone collapse** (batch continuity fails and the outlier frame is the only DISTINCT pose — the rest are near-duplicates): `diagnose_direction.py` prints `reorder_patch`. Never patch or discard the outlier; keep the distinct poses (contact + passing), reorder to `[contact, placeholder, pass, placeholder]`, and replace each placeholder via `patch_frame.py` with a freshly generated single cell of the missing `down`/`up` phase. Two cell generations beat regenerating a batch that has already shown collapse.
- **Height pulsing** (set spread above the limit but half medians nearly equal — generation noise or a mis-scaled patched cell): `diagnose_direction.py` prints `flatten`. Run `scale_fit.py --flatten-heights` (one bounded factor PER frame toward the common height, pose untouched), then `auto_repair → normalize_frames → re-score`. Free; never a regen. A single frame needing more than the flatten step is a `patch_frame` case.
- **Indeterminate legs** (long robe hides the feet): confirm alternation with a single targeted lower-body look, then accept or regenerate once.
- Record `accepted` or `redo`, the reason, the gate verdict (score, recommendation, structural report path), and whether a visual review was needed, in the character bible for each direction.

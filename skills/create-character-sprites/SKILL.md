---
name: create-character-sprites
description: Create and validate reference-driven 2D RPG character sprites. Use when asked for directional walk, idle, combat, or model-sheet sprites; to turn AI grids into transparent animation frames; or to preflight, review, assemble, and install a game-ready sprite sheet.
---

# Create Character Sprites

Generate and validate each animation direction in isolation. Preserve the supplied character's identity; describe observable visual traits instead of copying a named game's exact style. Do not include any visual effects (VFX) in the sprites (e.g., fire on a staff, glowing hands, mystical auras); generate only the physical character and their props.

## Cost discipline (read first)

Image generation is the dominant cost. Every regeneration and every multimodal (visual) review spends tokens, so the pipeline is built to spend the fewest of both:

- **Deterministic scripts gate the expensive LLM review.** Run the local checks (`validate_frames.py`, `structural_check.py`) *before* looking at any GIF or contact sheet with vision. Only escalate to a visual review when a script explicitly says it cannot decide.
- **Repair before you regenerate.** A candidate that fails only on cosmetic pixels (magenta fringe, non-transparent border, small anchor drift) is salvaged by `auto_repair.py` / `normalize_frames.py` with zero generation calls. Regenerate only for structural or scale defects.
- **Regenerate the smallest unit.** Generate in two 4-pose batches, not one 8-pose grid. When only the second half is wrong, regenerate that batch alone and keep the accepted first batch.
- **Feed the exact defect back into the prompt.** When a regeneration is truly needed, paste the failing check's machine verdict (e.g. "frames 5-7 face the wrong way", "lead foot did not alternate") into the next prompt so it converges in one retry instead of many blind attempts.
- **Accept "good enough".** A set that scores at or above the acceptance threshold with only easily-fixable cosmetic residue is accepted as-is. Do not chase pixel perfection with new generations.

## Export Contract

- Ship lossless RGBA PNGs with no visible magenta, a fully transparent one-pixel outer border, and one `128x192` frame size. Magenta is generation matte only.
- Use eight walk frames per direction. Generate each source cell at least 2x target size, key the matte before the single downsample, and target a visible silhouette height of `80%–92%` of the target frame (`84%` preferred for front/back views).
- Use fixed source anchors: torso center `x=64`, foot baseline `y=186`. That
  baseline is load-bearing at runtime, not just a normalization aid: it is
  copied into `CharacterDef.FootLine` and is where the game puts the
  character's collision box (`internal/entity/character_ground.go`). Drift in
  it moves what the character can walk into. Normalization may recenter an intact frame by at most 24 px and may never clip or rescale art; a frame that cannot move within its transparent margin must be regenerated.
- Characters must be **mirror-safe**. Make left/right clothing, weapons, hair accessories, and silhouettes visually symmetric in the approved model sheet; adapt a one-sided reference detail by duplicating, centering, or removing it. Never generate `E`, `SE`, or `NE` source rows.
- The Legiao sheet is exactly `S,SW,W,N,NW`, eight frames per row. The renderer mirrors `W→E`, `SW→SE`, and `NW→NE`; metadata records those mappings, fixed anchor, and the alternating left/right contact plan.

This contract is fixed: the final sheet is always 8 columns x 5 rows. Batched generation only changes the intermediate raw image; slicing a 4-pose batch with `--frame-offset` yields byte-identical frames to slicing a full 2x4 grid, so the assembled sheet is unaffected.

## Image generation step (portability)

This skill does not ship an image generator; it hands a text-plus-reference prompt to whatever generation capability the host agent provides, then validates the result locally with the bundled Python scripts. Keep the skill agnostic:

- **Codex / ChatGPT and similar hosts** with a native image tool: issue the generation prompt through that tool. `agents/openai.yaml` is host-specific interface metadata and is ignored by other runners.
- **Claude Code / Cowork or any host without a native generator**: this environment has no built-in text-to-image tool. Generation must come from a connected image MCP, or the user supplies the raw grids manually. Do not silently stall assuming a generator exists — if none is available, state that the generation step needs an image tool or user-provided grids, and proceed with slicing/validation once grids exist.
- Everything after generation (slice, key, normalize, validate, structural check, repair, review, assemble, preflight) is plain Python and runs identically on any host with Pillow. Prefer these local scripts over agent reasoning wherever a script can decide.

## Autonomy (mandatory)

Run the whole pipeline — all batches, gates, repairs, and directions — without pausing to ask for confirmation after successful steps. A passing gate is its own approval; proceed immediately to the next step, direction, and final assembly. Stop and ask ONLY when: (a) the image-generation capability is unavailable, (b) the same batch has failed regeneration twice (report the verdicts and ask how to proceed), or (c) a mandatory visual review is ambiguous. Everything else is yours to decide; the manifest records the decisions for later audit.

## Sharpness (mandatory): supersampled pipeline, single downsample

Chained LANCZOS resamples at export resolution (slice → scale_fit → harmonize, each blurring a 128x192 image a little more) are what make sprites look pixelated in-game. Resample to export size ONCE:

- Run the whole per-direction pipeline at 2x working resolution: frames `256x384`, `--anchor-x 128 --baseline 372 --max-shift 48`, `--alpha-threshold 40` (thin semi-transparent art survives 2x downsamples that 4x would erase; measuring at 40 keeps it from inflating anchors), and slice with `--min-visible-alpha 24 --minimum-source-scale 1`.
- Only after the direction passes every gate, reduce once: `finalize_frames.py --factor 2` (256x384 → 128x192, single resample), then run the final validate/structural at export size.
- Never run scale_fit, harmonize, or patch_frame on already-final 128x192 frames if the working-resolution set still exists — redo the operation at 2x and re-finalize.

## Audit trail (mandatory)

Retries that overwrite files destroy the evidence needed to diagnose token waste. Preserve history:

- **Never overwrite an attempt.** Each regeneration of a direction gets a NEW `attempts/<direction>/attempt-N/` (N increments). Never reuse `attempt-1` with `-Force`; the failed raw grids are the most valuable debugging artifact.
- **Save the raw generated image and its prompt per attempt**: `attempt-N/batch1.png`, `attempt-N/batch1-prompt.txt` (the exact text sent to the generator), same for batch 2. On regen, record what changed in the prompt and why.
- **Name reports per attempt**: `review/<DIR>-attempt-N-score.json`, `-structural.json`, `-repair.json`. Never overwrite a previous attempt's report.
- **Log every gate decision** with one `log_attempt.py` call (appends to `work/<character-id>/manifest.jsonl`): after each generate, validate, scale_fit, auto_repair, structural, visual review, accept, and regen — with the verdict and a one-line reason. This file is the timeline a later analysis reads instead of the chat transcript.

```powershell
python skills/create-character-sprites/scripts/log_attempt.py --root work/<character-id> --direction W --attempt 2 --stage validate --verdict repair --note "height 0.93 -> scale_fit" --data score=0.65
```

## Workflow

1. Read `references/character_creation_guide.md`. Choose a lowercase kebab-case `<character-id>` and create a concise bible: silhouette, palette, symmetric props, target scale, fixed anchors, and forbidden drift. With one reference view, first approve a five-view model sheet.
2. Freeze the approved bible/model sheet before animation. This gate is serial: a changed identity, asymmetric detail, or scale invalidates all direction work. Before approving, run the model-gate checklist in the guide: (a) mirror-safety decisions for every asymmetric reference detail are made WITH the user (accept the mirror swap / duplicate / center / remove) and frozen in the bible; (b) each view's LATERAL facing matches the renderer convention (W family faces viewer-left) — fix a wrong-facing view of a mirror-safe design with a free horizontal flip; (c) a defect confined to one view is repaired by regenerating THAT CELL conditioned on the sheet and compositing it locally (mask paste + matte normalized to exact #FF00FF) — never by regenerating the whole sheet, which risks drifting the approved views.
3. Plan phases. After the model gate, use an orchestrator to spawn independent workers for `S`, `SW`, `W`, `N`, and `NW` when capacity and image-generation limits permit. A worker owns one direction and an isolated `work/<character-id>/attempts/<direction>/` directory; it may regenerate only its own direction. Parallelize slicing, validation, structural check, repair, and review across ready directions. Keep model approval, final selection, sheet assembly, and final validation serial. If the image service cannot run concurrently, serialize generation but parallelize the local processing stages.
4. Generate each direction as **two conditioned 4-pose batches**, not one 8-pose grid:
   - Batch 1 = poses 1–4 (`1024x384` or larger, 4 columns x 1 row). **Gate batch 1 before generating batch 2**: slice it (`--frame-offset 0`), run `validate_frames --score` on its four frames for scale/margins/matte, and fix or regenerate batch 1 alone if it fails. A bad batch 1 caught here costs one image; caught after batch 2 it costs two.
   - Batch 2 = poses 5–8, generated **conditioned on the accepted batch-1 image** so the second half inherits the exact same camera facing, scale, pivot, and padding. This is what prevents the "second-half is mirrored/inverted" defect: pass batch 1 as a visual reference, and instruct that batch 2 keeps identical facing while advancing the walk to the opposite lead foot.
   - Keep the fixed `x=64` torso pivot, `y=186` foot baseline, ample transparent margin, and `80%–92%` visible height (`84%` front/back). Follow the guide's eight-pose plan. Never request a multi-direction final sheet, and never rely on text alone to keep the two halves consistent.
5. Slice each batch, then run the two-layer validation gate below on the assembled 8-frame set. Regenerate only the failing batch; do not force large shifts, crop, scale, or substitute another direction.

### Two-layer validation gate (cheap to expensive)

Run in this exact order and stop as early as possible. Only the last step costs vision tokens.

```powershell
# Slice both batches into one 8-frame direction folder (offset 4 for batch 2).
python skills/create-character-sprites/scripts/slice_and_stitch.py work/<character-id>/attempts/W/attempt-1/batch1.png --direction W --output-root work/<character-id>/attempts/W/attempt-1/sliced --frame-width 128 --frame-height 192 --grid-rows 1 --grid-cols 4 --frame-offset 0 --minimum-source-scale 2
python skills/create-character-sprites/scripts/slice_and_stitch.py work/<character-id>/attempts/W/attempt-1/batch2.png --direction W --output-root work/<character-id>/attempts/W/attempt-1/sliced --frame-width 128 --frame-height 192 --grid-rows 1 --grid-cols 4 --frame-offset 4 --minimum-source-scale 2

# Layer 0 - geometry/anchor normalization (no generation).
# If normalize refuses because re-anchoring would clip the art, do NOT regenerate yet:
# run scale_fit.py directly on the SLICED frames (it shrinks-to-fit and re-anchors on
# its own) and continue from its output. Regenerate only if scale_fit also refuses.
python skills/create-character-sprites/scripts/normalize_frames.py --input-root work/<character-id>/attempts/W/attempt-1/sliced --output-root work/<character-id>/attempts/W/attempt-1/frames --directions W --frame-width 128 --frame-height 192 --max-shift 24 --anchor-x 64 --baseline 186

# Layer B - cosmetic score. Recommendation: accept (exit 0) | repair (exit 3) | regen (exit 1).
python skills/create-character-sprites/scripts/validate_frames.py work/<character-id>/attempts/W/attempt-1/frames/W/*.png --frame-width 128 --frame-height 192 --require-alpha --require-transparent --require-clear-border --reject-magenta --magenta-threshold 140 --check-baseline --baseline-tolerance 1 --expected-baseline 186 --check-center --center-tolerance 1 --expected-center 64 --min-foreground-height-ratio 0.80 --max-foreground-height-ratio 0.92 --score --acceptance-threshold 0.85 --score-report work/<character-id>/review/W-score.json

# If Layer B recommends "repair": salvage without generating a new image, then re-score.
#   - silhouette scale out of the 0.80-0.92 band  -> scale_fit.py (uniform set rescale + re-anchor)
#   - magenta fringe / non-transparent border      -> auto_repair.py
#   - residual anchor drift                         -> normalize_frames.py (already run above)
python skills/create-character-sprites/scripts/scale_fit.py --input-root work/<character-id>/attempts/W/attempt-1/frames --output-root work/<character-id>/attempts/W/attempt-1/frames --directions W --frame-width 128 --frame-height 192 --target-height-ratio 0.84 --anchor-x 64 --baseline 186
python skills/create-character-sprites/scripts/auto_repair.py --input-root work/<character-id>/attempts/W/attempt-1/frames --output-root work/<character-id>/attempts/W/attempt-1/frames --directions W --frame-width 128 --frame-height 192 --report work/<character-id>/review/W-repair.json

# Layer A - structural content gate (no generation). Pass --direction so the side-view
# leg metrics run only where they apply. hard_fail -> regen the guilty batch;
# needs_visual_review -> exactly one targeted vision call.
python skills/create-character-sprites/scripts/structural_check.py work/<character-id>/attempts/W/attempt-1/frames/W --direction W --frame-width 128 --frame-height 192 --report work/<character-id>/review/W-structural.json
```

Decision policy from the gate:

- `validate_frames --score` says **regen** (wrong frame size, missing alpha/transparency): regenerate the affected batch with the defect pasted into the prompt. No visual review.
- `validate_frames --score` says **repair**: run the matching no-generation fixer(s), then re-run the score. If it now accepts, continue. No visual review, no generation.
  - **Silhouette scale out of the 0.80–0.92 band is a repair, not a regen.** Do not regenerate to chase the height band — a whole set that is uniformly a little too tall or too short is fixed by `scale_fit.py`, which rescales all eight frames by one factor (preserving motion and proportions) to land at `0.84` and re-anchors. If `scale_fit.py` refuses because a garment overhang (trailing cape/cloak, common in side views) width-locks the fit, that is STILL a repair: `diagnose_direction.py` detects it and prescribes `trim_overhang.py` (a bounded, feathered, feet-safe shave of the overhang tip), after which the chain `scale_fit → auto_repair → normalize_frames → auto_repair → re-score` lands the set at target. Scale becomes a regen only when the diagnostician says so (mismatch beyond `--max-scale-step`, or an overhang too large/too low to trim).
  - Matte fringe / non-transparent border → `auto_repair.py`; residual anchor drift → `normalize_frames.py`.
- `structural_check` reports `hard_fail` (orientation flip with 2+ flagged frames, or frozen cycle): regenerate the guilty batch (for an orientation flip, only batch 2), pasting the reported flipped-frame indices into the prompt. No visual review.
- `structural_check` reports `needs_visual_review: true`: spend exactly **one** small targeted vision call, never a full-grid re-review loop. Two sources:
  - `orientation: suspect` (a single noisy flagged frame — real inversions flip the whole batch): glance at that frame vs frame 1 for facing; accept or regen batch 2 once.
  - `leg_alternation: indeterminate` (every side view; deciding the lead foot from pixels is unreliable both for robes and for visible legs, so the script never hard-fails on legs): check the lower band of frames 1/3/5/7 for foot alternation; accept or regenerate once.
- Everything passes and no visual review is flagged: the deterministic gates are done — but the direction is NOT yet accepted. Every direction still owes the semantic review below.

**Mandatory semantic review (facing + poses + identity) — scripts cannot see this.** Every deterministic gate measures geometry; `structural_check`'s orientation is only RELATIVE within the direction. A direction generated with the wrong absolute facing (N/NW front-facing) or with non-walk poses (crouching, idle) passes every script and ships a broken sheet — this happened. Therefore, per direction, after assembly and before `finalize_frames`:

1. Run `facing_check.py --frames-root <frames> --model-sheet <five-view sheet>` — a deterministic screen that correlates each direction against the character's own model-sheet views. `suspect` verdicts make the vision look mandatory for those directions; `pass` does not waive step 2.
2. ONE vision look at the direction's 8-frame strip verifying, explicitly: (a) facing matches the direction label (S=front, SW=front-diagonal, W=side, N=back with NO face visible, NW=back-diagonal); (b) all 8 poses are walk phases — no crouch, kneel, idle, or attack; (c) identity/palette matches the model sheet; (d) for characters with props (quiver, staff, scabbard): the prop is present per the bible's per-view visibility decisions, attached, CONTINUOUS (thin traces — bow limbs, arrow shafts, straps — can break in the 2x→1x downsample), and consistent across all 8 frames. Record the answer to each of (a)(b)(c)(d) in the manifest note.
3. A `harmonize` verdict with a half-median jump above ~15 export px is a semantic red flag, not just scale noise: batch 2 may be a different action entirely. Do the step-2 look at batch 2 BEFORE accepting the harmonization.

Only after a direction is accepted, build its GIF/contact sheet/report for the delivery record:

```powershell
python skills/create-character-sprites/scripts/review_animation.py work/<character-id>/attempts/W/attempt-1/frames/W --gif work/<character-id>/review/W.gif --contact-sheet work/<character-id>/review/W.png --report work/<character-id>/review/W.json --frame-width 128 --frame-height 192
```

6. The orchestrator copies only accepted frame sets into `work/<character-id>/frames/`, then assembles and validates the single sheet. For a registered character, run renderer preflight; otherwise leave the passing package for `$install-character-sprites`.

```powershell
python skills/create-character-sprites/scripts/build_sheet.py --input-root work/<character-id>/frames --output work/<character-id>/<character-id>.png --metadata-output work/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --frames-per-direction 8 --directions S,SW,W,N,NW --anchor-x 64 --baseline 186
python skills/create-character-sprites/scripts/validate_frames.py work/<character-id>/<character-id>.png --sheet --columns 8 --rows 5 --frame-width 128 --frame-height 192 --require-alpha --require-transparent --reject-magenta --magenta-threshold 140
python skills/create-character-sprites/scripts/preflight_renderer.py --character-id <character-id> --asset assets/sprites/<character-id>/<character-id>.png --metadata work/<character-id>/<character-id>.json --frame-width 128 --frame-height 192 --columns 8 --directions S,SW,W,N,NW
```

7. Deliver the sheet, metadata, bible, approved source reference, model sheet, per-direction GIF/contact sheet/report, and an incremental attempt manifest. Record every accepted/rejected attempt with its reason, its gate verdict (score, recommendation, structural report), and whether a visual review was needed. `$install-character-sprites` copies the approved reference to `assets/sprites/<character-id>/reference.png`.

## Failure playbook (run the diagnostician, not guesses)

When ANY gate refuses — normalize clip, scale_fit refusal, validate regen/repair, structural fail — do not decide the next step yourself. Run:

```powershell
python skills/create-character-sprites/scripts/diagnose_direction.py <direction-frames-dir> --frame-width 256 --frame-height 384 --alpha-threshold 40
```

(use `--frame-width 128 --frame-height 192 --alpha-threshold 16` for export-resolution sets; for a 4-frame partial batch — e.g. the batch-1 gate — point it at the 4-frame folder and it assumes `--batch 1`, targeting regen verdicts at that batch only). It measures per-frame scale, torso position, width extents, fit caps, trimmable overhangs, continuity, clone collapse, and batch-boundary jumps, and prints one `action`: `accept`, `auto_repair`, `scale_fit`, `trim_overhang`, `harmonize`, `patch_frame N`, `reorder_patch`, `regen_batch1`, `regen_batch2`, or `regen_both` — ordered from free to most expensive. Follow it. Log the verdict to the manifest. Overriding the recommendation requires a logged justification.

Execute the printed action LITERALLY — never escalate to a costlier action, and never paraphrase the verdict in reports (paste it). Refusals answered by a free fix (`auto_repair`, `scale_fit`, `trim_overhang`, `flatten`, `harmonize`, `reorder_patch`, the script part of `patch_frame`) are repairs, not failed attempts: they do NOT count toward any regeneration limit or stop-after-N-refusals rule. Only actual regenerations count.

Diagnosing assembled sets — two mandatory flags:

- A set containing PLACEHOLDER cells (clones parked for a later `patch_frame`, e.g. after `reorder_patch`) must be diagnosed with `--ignore-frames <indices>`. Diagnosing placeholders as real content produces bogus `regen` verdicts — this once destroyed an accepted batch 1 and forced a full re-spend.
- Once a batch has passed its gate, every later full-set diagnose must pass `--accepted-batch N`. An accepted batch is never regenerated because of defects elsewhere in the set; the script redirects such verdicts to the other batch.

Two verdicts it cannot issue (they come from `structural_check` and vision): an orientation `hard_fail` still means regen batch 2 with the flipped-frame indices in the prompt, and the side-view leg review still decides moonwalk vs valid stride.

`diagnose_direction` measures GEOMETRY only (scale, anchors, width, continuity). It does not see matte residue, border pixels, or alpha defects — so its `accept` never overrides a failing `validate_frames --score`. The batch gate is `validate` exit 0, nothing else: when diagnose says `accept` but the score says `repair`, run the cosmetic fixers (`auto_repair` / `normalize_frames`) and re-score before generating anything conditioned on that batch.

Chain order when a set is undersized AND drifted (small figure floating high in its cell — the common generator output): `scale_fit` FIRST (it re-anchors internally, no shift limit), then `auto_repair → normalize_frames → auto_repair → re-score`. Running `normalize_frames` before `scale_fit` on such a set refuses on max-shift and stalls the pipeline for no reason.

**Standard post-slice step — use `repair_chain.py`, not a hand-assembled chain.** It derives every parameter from the frames themselves (size, 4-vs-8 count, 1x/2x thresholds, anchors), runs the full chain in the correct order, and on a refusal runs the diagnostician itself, applying free verdicts (`scale_fit`/`flatten`/`harmonize`/`trim_overhang`) automatically for up to 3 rounds:

```powershell
python skills/create-character-sprites/scripts/repair_chain.py --input-root work/<id>/attempts/<D>/attempt-N/batch2-sliced --output-root work/<id>/attempts/<D>/attempt-N/batch2-final --direction <D> --batch 2
```

Exit 0 = gated frames in `<output-root>/<D>` (score JSON printed — paste it in reports). Exit 2 = not free-fixable; the printed diagnose verdict (`patch_frame`/`reorder_patch`/`regen_*`) is the next action. Never re-derive the per-script flags by hand; the flag surface (`--frames-per-direction`, 2x thresholds, ...) is the most common way agents stall.

## Targeted frame patch (post-delivery edits)

When a delivered sheet needs a fix in ONE pose (e.g. "S, column 3 looks wrong"), never regenerate the direction — patch the single frame. Column K (1-8) = frame index K-1.

1. Produce a candidate for just that pose, one of:
   - **Generate one cell** (cheapest generation possible): a single image of that pose on the `#FF00FF` matte, conditioned on a reference strip of the neighboring accepted frames (K-1 and K+1) plus the model sheet, with the instruction "same character, same facing, same scale; only this pose". Any canvas size ≥2x the frame works.
   - **Hand edit** (zero generation): the user edits the exported frame in an image editor and supplies the PNG.
This applies mid-pipeline too: when one cell of a NOT-yet-accepted set is regenerated (e.g. a single wide pose blocked `scale_fit`), insert it with `patch_frame.py` pointed at the sliced set — never by slicing the raw cell over the old index, because a freshly generated cell's scale will not match the set and `scale_fit` cannot fix a single frame (it applies one factor to the whole set by design).

2. Fit and swap — the script matches the candidate to the SET's own metrics (median silhouette height of the other 7 frames, fixed pivot/baseline), keys the matte if present, backs the old frame up under `attempts/<DIR>/patches/<timestamp>/`, and refuses candidates that cannot fit:

```powershell
python skills/create-character-sprites/scripts/patch_frame.py --frames-dir work/<character-id>/frames/S --column 3 --candidate work/<character-id>/attempts/S/patches/new-pose.png
```

3. Re-validate ONLY the patched direction (`validate_frames --score` + `structural_check --direction`), rebuild the sheet with `build_sheet.py`, and re-run the sheet-level validation. The other four rows are untouched by construction. Log the patch with `log_attempt.py` (stage `patch`).

Cost: at most one single-cell generation and zero visual reviews unless `structural_check` flags one.

## Tools

- `preflight_renderer.py`: compare the requested export with the live Legiao renderer constants and asset path, then open the PNG and cross-check its alpha, dimensions, and metadata contract.
- `slice_and_stitch.py`: split a grid (full 2x4 or a partial 1x4 batch via `--grid-rows/--grid-cols/--frame-offset`), key and despill magenta, and downsample production frames once.
- `normalize_frames.py`: correct small torso/foot-anchor drift to a fixed pivot without clipping or resizing frames.
- `scale_fit.py`: repair an out-of-band silhouette scale by rescaling a whole direction's set by one shared factor to the target height and re-anchoring — preserves motion/proportions, so scale defects no longer force a regeneration. Refuses (→ regen) only when the mismatch is too large to be a scale issue. It cannot fix a SINGLE frame whose scale disagrees with the set — that is `patch_frame.py`'s job.
- `validate_frames.py`: validate geometry, alpha, matte leakage, transparent borders, silhouette scale, torso center, and foot baseline. With `--score` it also emits a 0–1 acceptability score and an accept/repair/regen recommendation (exit 0/3/1) so the orchestrator can skip needless regenerations. Silhouette-scale misses are classified as repair (fixable by `scale_fit.py`), not regen.
- `structural_check.py`: deterministic content gate — detects second-half orientation flips via upper-body content correlation (legs excluded because a correct cycle mirrors them legitimately; works for robed and legged characters alike; hard-fails only with 2+ flagged frames) and frozen cycles. Side-view leg alternation is always routed to one targeted visual check (`needs_visual_review`) because pixels cannot decide the lead foot reliably for any body type.
- `auto_repair.py`: cosmetic salvage — clears magenta fringe, despills edges, and forces a transparent 1px border without moving or rescaling the body, so cosmetic-only defects never trigger a regeneration.
- `review_animation.py`: create a looped GIF, a baseline/torso contact sheet, and an audit report for accepted directions (delivery record, not a gate).
- `log_attempt.py`: append one JSONL event per gate decision to `work/<character-id>/manifest.jsonl` — the audit timeline used to diagnose retries and token waste after the fact.
- `finalize_frames.py`: the single working-resolution → export-resolution downsample (2x → 1x) that keeps sprites sharp; run once per direction after all gates pass.
- `diagnose_direction.py`: the failure diagnostician — measures a refused set (full 8-frame direction or 4-frame partial batch) and prints the single cheapest sufficient action (accept/repair/scale_fit/trim_overhang/harmonize/patch_frame N/reorder_patch/regen batch 1/2/both). Mandatory after any gate refusal.
- `trim_overhang.py`: free un-locker for width-locked scale — feathered shave (bounded by `--max-trim`, never inside the foot zone) of a garment overhang tip so `scale_fit` can reach the height band without regeneration. Run only when `diagnose_direction.py` prescribes it.
- `repair_chain.py`: the standard post-slice orchestrator — auto-derives all parameters from the frames, runs scale_fit → auto_repair → normalize → auto_repair → validate, self-applies free diagnose verdicts, and stops with the verdict when generation is needed. Prefer it over hand-running the chain scripts.
- `facing_check.py`: absolute-facing screen — NCC-matches every direction's upper-body content against the character's own five-view model sheet and flags directions whose best-matching view belongs to the wrong facing group (front/side/back). Suspect = mandatory vision look. Part of the semantic review gate; catches whole-direction facing errors that every geometric gate misses.
- `patch_frame.py`: surgically replace one frame of an accepted direction with a new single-pose candidate (generated cell or hand edit) — keys the matte, matches the set's scale/anchor, backs up the old frame, and never touches the other frames or directions.

Install Pillow in the active environment before using the image scripts.

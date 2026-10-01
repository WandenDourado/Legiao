---
name: audit-character-sprites
description: Act as the AUDITOR and ENGINEER of the create-character-sprites pipeline. Use when supervising a sprite-generation agent (Codex or similar) that executes create-character-sprites; when writing the prompts that agent will receive; when verifying, gating, or approving its returns; when a gate refuses and the next action must be decided; or when a diagnosis should become a permanent improvement to the create-character-sprites skill. Do NOT use for generating sprites directly — that is create-character-sprites itself.
---

# Audit Character Sprites

You supervise a separate generation agent (called "Codex" here) that executes
`create-character-sprites`. You do not generate images. You have four duties,
in priority order: **audit** every return in the files, **engineer** every
diagnosis into a permanent skill improvement, **optimize tokens** so the
expensive generator runs as little as possible, and **write the prompts** the
user pastes into Codex. Read `create-character-sprites/SKILL.md` and its
`references/character_creation_guide.md` IN FULL before doing anything — you
enforce that contract; you cannot enforce what you have not read.

## Rule zero: never trust the report, audit the files

Codex has a documented history of paraphrasing verdicts wrongly, skipping
steps, and declaring work done without executing it. Every claim it makes is
unverified until you reproduce it from `work/<character-id>/` yourself. All
verification scripts are local and free — there is no cost excuse for
trusting a report.

After EVERY Codex return:

1. `ls` the paths it claims to have written. Missing file = the step did not
   happen, whatever the report says.
2. Re-run the gate yourself on the actual frames: `validate_frames --score`
   (the batch gate is ITS exit 0 — a `diagnose_direction` "accept" never
   substitutes, it only sees geometry), `structural_check --direction`,
   `facing_check` against the approved model sheet.
3. Build an 8-frame contact strip locally and do the ONE semantic vision look
   yourself: (a) facing matches the label (N/NW = back, NO face), (b) all 8
   poses are walk phases, (c) identity/palette, (d) props present, attached,
   continuous, consistent (see the parent skill's semantic review).
4. Log your audit verdict to `work/<id>/manifest.jsonl` via `log_attempt.py`
   with a `claude:` prefix, pasting scores and verdict JSONs — never
   paraphrases of them.

If Codex says a repair was applied (harmonize, trim, patch), read the manifest
entries and measure the result. A harmonize half-median jump above ~15 export
px means look at batch 2 with vision before accepting it.

## Codex prompt protocol (every prompt, no exceptions)

Prompts the user pastes into Codex MUST:

1. **Open by invoking the skill**: "Use a skill create-character-sprites (leia
   SKILL.md e references/character_creation_guide.md INTEGRAIS)" — without
   this Codex improvises outside the toolchain. Mention any rule added to the
   skill since Codex last ran ("o guide ganhou hoje uma regra sobre X; siga-a").
2. **State the workflow step** ("Estamos no passo 4: direção W, lote 2") and
   what is already ACCEPTED and therefore untouchable (`--accepted-batch`).
3. **Cover ONE stage only** and **end with an explicit STOP**: what to save,
   what to run, what JSON to paste, what to log via `log_attempt.py`, and the
   instruction to stop at the gate — approval is external (yours). Codex
   pausing after each stage is normal; "continue" replies are expected.
4. Carry the standing content rules inline: exact save paths under
   `attempts/<D>/attempt-N/` (never overwrite an attempt), conditioning
   references (tiled 4-up for batch 1, the accepted batch-1 image for batch
   2), facing stated with a NEGATIVE restriction ("N = DE COSTAS, o rosto NÃO
   é visível"), lateral facing spelled out (W family looks viewer-LEFT), the
   frozen bible decisions, layout/scale/matte numbers from the parent skill,
   and the walk plan for the exact poses.
5. On a regeneration, paste the failing check's machine verdict into the
   prompt verbatim so it converges in one retry.

## Failure protocol (gates disagree or refuse)

- Any refusal → run `diagnose_direction.py` yourself (with `--batch`,
  `--ignore-frames`, `--accepted-batch` as applicable). Execute the printed
  action LITERALLY; never escalate to a costlier one. Tooling refusals
  answered by free fixes do NOT count toward regeneration limits.
- **Suspect the metric before spending a generation.** When a gate refuses
  content that your semantic look says is right, test the metric against
  already-ACCEPTED material: if accepted frames also fail it, the tool is
  miscalibrated, not the art (this is how full-silhouette continuity IoU was
  caught: its low pairs appeared inside an accepted batch). Then fix the
  SCRIPT, not the art.
- Overriding a script verdict requires a logged justification in the manifest
  and a root-cause fix; "it looks fine" is not a justification.

## Engineer duty: no lesson dies in the chat

Every root-caused failure, false positive, or user correction becomes a
PERMANENT change, in the same session:

1. Fix the right layer: generation-prompt lessons → the guide; workflow/gate
   lessons → the parent `SKILL.md`; metric/repair lessons → the script itself.
   Frozen per-character decisions → `work/<id>/bible.md`.
2. Write the rule with its why: symptom, root cause, the failed example and
   date. Rules without their failure story get deleted by future editors.
3. **Regression-check every script change** against all previously accepted
   material (current character's accepted directions + installed characters'
   sheets) before relying on it. Keep old behavior visible where cheap (e.g.
   report both new and legacy metrics as diagnostics).
4. Append one line to `doc/changelog.md` per change.

## Token discipline (what runs where)

Everything except image generation is local and free: slicing, repair chains,
validation, structural/facing checks, tile_reference, strips, model-sheet
cell composition (mask paste + matte normalization + flips), finalize,
build_sheet, preflight, installation. Do these YOURSELF instead of prompting
Codex — a Codex round-trip costs tokens and risks improvisation. Prompt Codex
ONLY for: generating images, and stages the user explicitly delegates to it.
Regenerate the smallest unit (one cell < one batch < one direction < never
the whole sheet). Do not chase pixel perfection past a passing gate.

## User decisions (art gates)

Decisions that change the design are the user's, not yours or Codex's:
mirror-safety adaptations for asymmetric details (accept swap / duplicate /
center / remove), per-view prop visibility, and model-sheet approval. Ask
them explicitly at the MODEL GATE, before any direction is generated, and
freeze the answers in the bible. Mid-pipeline user feedback ("SW should show
the quiver") is a bible revision: update the bible, log it, and repair with
the smallest unit.

## Standing flow (per character)

1. Bible with user's frozen decisions → Codex prompt: save reference + model
   sheet → audit (5 views, symmetry, lateral facing, centering, N faceless)
   → user approval → freeze + crop views + `tile_reference` 4-up (local).
2. Per direction (S, SW, W, N, NW): prompt batch 1 → audit + accept → prompt
   batch 2 conditioned on batch 1 → audit full set (validate exit 0 +
   structural + facing_check + semantic look) → `finalize_frames` (local).
3. `build_sheet` + sheet validate with `--max-edge-spill` → present sheet and
   GIFs to the user → install via `$install-character-sprites` (registry
   order preserved) → preflight → changelog. Flag `go test ./...` if the
   sandbox has no Go.

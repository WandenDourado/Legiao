#!/usr/bin/env python3
"""Diagnose a failing direction and print ONE recommended next action.

Run this after ANY gate refusal instead of guessing. It reproduces the analysis a
human debugger would do — per-frame scale, torso position, width extents, fit caps,
continuity, batch-boundary scale jumps — and maps the findings to the cheapest
sufficient action:

  accept            all checks pass; proceed with the normal gates
  auto_repair       only matte/border residue
  scale_fit         whole set uniformly out of the height band and it fits
  trim_overhang     a garment overhang width-locks the fit but is safely trimmable
                    (run trim_overhang.py, then scale_fit + repair chain) — free
  flatten           per-frame height pulsing with no half jump (scale_fit --flatten-heights)
  harmonize         batch-boundary scale jump (frames 0-3 vs 4-7)
  patch_frame N     exactly one frame blocks the set; replace only that cell
  reorder_patch     (batch mode) the batch collapsed onto near-clone poses and the
                    continuity outlier is the only DISTINCT pose — keep the distinct
                    poses, reorder, and patch the missing phases; never patch the outlier
  regen_batch1/2    the defect is systematic and lives in one half
  regen_both        the defect is systematic across both halves

Partial batches: point it at a 4-frame folder (e.g. the batch-1 gate) and pass
--batch 1 (assumed if omitted for 4 frames). Regen verdicts then target that batch
only — a 4-frame set must NEVER produce regen_both.

Assembled sets with pending placeholders or an accepted batch:
  --ignore-frames 1,3   placeholder cells awaiting patch_frame are EXCLUDED from all
                        verdict logic; diagnosing them as real content produces bogus
                        regen verdicts (this destroyed an accepted batch once).
  --accepted-batch 1    an already-accepted batch is never the target of a regen
                        verdict from a full-set diagnose; regen is redirected to the
                        other batch.

Costs are ordered: script-only fixes are free, patch is one cell image, batch regen
is one grid image. Never jump to a costlier action than recommended without logging
a justification to the manifest.
"""

from __future__ import annotations

import argparse
import json
import math
import statistics
from pathlib import Path

from PIL import Image

from frame_analysis import foreground_bbox, mask_iou, silhouette_mask, torso_center
from trim_overhang import occupied_rows, zone_density


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input_dir", type=Path, help="Direction folder with 8 frames (working or export resolution).")
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--anchor-x", type=int, help="Torso pivot (default: frame width / 2).")
    parser.add_argument("--min-height-ratio", type=float, default=0.80)
    parser.add_argument("--max-height-ratio", type=float, default=0.92)
    parser.add_argument("--target-height-ratio", type=float, default=0.84)
    parser.add_argument("--height-max-spread", type=float, default=5.0, help="Px at export scale; scaled by frame size.")
    parser.add_argument("--continuity-iou", type=float, default=0.70)
    parser.add_argument("--edge-margin", type=int, default=2)
    parser.add_argument("--alpha-threshold", type=int, default=16,
                        help="Use 40 for supersampled (2x) frames.")
    parser.add_argument("--batch", type=int, choices=(1, 2),
                        help="Diagnosing a 4-frame partial batch; regen verdicts target this batch (assumed 1 for 4-frame input).")
    parser.add_argument("--ignore-frames", help="CSV of placeholder frame indices excluded from verdict logic.")
    parser.add_argument("--accepted-batch", type=int, choices=(1, 2),
                        help="Regen verdicts never target this already-accepted batch.")
    parser.add_argument("--baseline", type=int, help="Foot baseline (default: 186 scaled to frame height).")
    parser.add_argument("--max-trim", type=float, default=6.0,
                        help="Trimmable overhang budget per frame at export scale; must match trim_overhang.py.")
    parser.add_argument("--foot-zone", type=float, default=16.0,
                        help="Export-scale px above the baseline protected from trims; must match trim_overhang.py.")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()

    anchor_x = args.anchor_x if args.anchor_x is not None else args.frame_width // 2
    scale = args.frame_height / 192.0
    spread_limit = args.height_max_spread * scale
    baseline = args.baseline if args.baseline is not None else round(186 * scale)

    paths = sorted(args.input_dir.glob("*.png"))
    if len(paths) == 4 and args.batch is None:
        args.batch = 1
    batch_mode = args.batch is not None
    expected = 4 if batch_mode else 8
    regen_all = f"regen_batch{args.batch}" if batch_mode else "regen_both"
    if len(paths) != expected:
        print(json.dumps({"action": regen_all, "reason": f"expected {expected} frames, found {len(paths)}"}))
        return 0
    frames = []
    for path in paths:
        with Image.open(path) as image:
            if image.size != (args.frame_width, args.frame_height):
                print(json.dumps({"action": regen_all,
                                  "reason": f"{path.name} is {image.width}x{image.height}, expected {args.frame_width}x{args.frame_height}"}))
                return 0
            frames.append(image.convert("RGBA"))

    rows = []
    heights, caps = [], []
    for index, frame in enumerate(frames):
        bbox = foreground_bbox(frame, args.alpha_threshold)
        tx = torso_center(frame, alpha_limit=args.alpha_threshold)
        height = bbox[3] - bbox[1]
        left_ext, right_ext = tx - bbox[0], bbox[2] - tx
        cap = min(
            (anchor_x - args.edge_margin) / left_ext if left_ext > 0 else 9.0,
            (args.frame_width - 1 - anchor_x - args.edge_margin) / right_ext if right_ext > 0 else 9.0,
        )
        heights.append(height)
        caps.append(cap)
        rows.append({
            "frame": index, "h_ratio": round(height / args.frame_height, 3),
            "torso_x": round(tx), "ext_left": round(left_ext), "ext_right": round(right_ext),
            "fit_cap": round(cap, 3),
        })

    ignored = {int(item) for item in args.ignore_frames.split(",")} if args.ignore_frames else set()
    kept = [i for i in range(len(frames)) if i not in ignored]
    heights_eff = [heights[i] for i in kept]
    caps_eff = [caps[i] for i in kept]
    median_h = statistics.median(heights_eff)
    target_factor = (args.target_height_ratio * args.frame_height) / median_h
    fit_factor = min(min(caps_eff), target_factor)
    resulting = fit_factor * median_h / args.frame_height
    half = len(frames) // 2
    b1, b2 = statistics.median(heights[:half]), statistics.median(heights[half:])
    spread = max(heights_eff) - min(heights_eff)
    masks = [silhouette_mask(frame, args.alpha_threshold) for frame in frames]
    ious = [round(mask_iou(masks[i], masks[i + 1]), 3) for i in range(len(frames) - 1)]
    low_pairs = [i for i, iou in enumerate(ious)
                 if iou < args.continuity_iou and i not in ignored and i + 1 not in ignored]

    def trimmable(blockers: list[int]) -> list[dict] | None:
        """Per-frame overhang trims that would unlock the target fit, or None if any
        blocker cannot be trimmed safely (mirrors trim_overhang.py's guards)."""
        max_trim = args.max_trim * scale
        foot_top = baseline - args.foot_zone * scale
        avail_left = anchor_x - args.edge_margin
        avail_right = args.frame_width - 1 - anchor_x - args.edge_margin
        plans = []
        for index in blockers:
            frame = frames[index]
            bbox = foreground_bbox(frame, args.alpha_threshold)
            tx = torso_center(frame, alpha_limit=args.alpha_threshold)
            for side, ext, avail in (("left", tx - bbox[0], avail_left), ("right", bbox[2] - tx, avail_right)):
                needed = ext - (math.floor(avail / target_factor) - 1)
                if needed <= 0:
                    continue
                trim = math.ceil(needed)
                if trim > max_trim:
                    return None
                x0, x1 = (bbox[0], bbox[0] + trim - 1) if side == "left" else (bbox[2] - trim, bbox[2] - 1)
                rows = occupied_rows(frame, x0, x1, args.alpha_threshold)
                if rows and rows[1] >= foot_top:
                    return None
                if rows and zone_density(frame, x0, x1, rows[0], rows[1], args.alpha_threshold) < 0.35:
                    return None  # thin rigid prop (bow/staff), not garment mass — never trim
                plans.append({"frame": index, "side": side, "px": trim})
        return plans

    findings = []
    action, reason = "accept", "all measurements inside contract"

    # 1. Continuity (content defect) dominates: no local fix exists — unless the low
    #    pairs share ONE common frame, in which case that single frame is the anomaly
    #    and a cell patch is enough.
    if len(low_pairs) >= 2:
        involvement: dict[int, int] = {}
        for i in low_pairs:
            for f in (i, i + 1):
                involvement[f] = involvement.get(f, 0) + 1
        hot = [f for f, n in involvement.items() if n >= 2]
        others = [i for i in low_pairs if hot and hot[0] not in (i, i + 1)]
        collapse = False
        if len(hot) == 1 and batch_mode:
            # Clone-collapse guard: if every frame EXCEPT the outlier is a near-clone
            # (bimodal IoU: clone pairs high, outlier pairs very low), the outlier is
            # the only pose with real motion — patching IT would destroy the one good
            # pose. Keep the distinct poses and patch the missing phases instead.
            rest = [i for i in range(len(frames)) if i != hot[0]]
            rest_ious = [mask_iou(masks[a], masks[b]) for a in rest for b in rest if a < b]
            low_vals = [ious[i] for i in low_pairs]
            # Bars calibrated on mago-celeste W attempt-5: clone pairs 0.83-0.94 at raw
            # scale vs 0.85-0.88 fitted; genuinely distinct side-view poses 0.52-0.63.
            if rest_ious and min(rest_ious) >= 0.82 and max(low_vals) < 0.65:
                collapse = True
                action, reason = "reorder_patch", (
                    f"mode collapse: frames {rest} are near-clones (min mutual IoU {min(rest_ious):.2f}) and frame "
                    f"{hot[0]} is the only distinct pose — do NOT patch frame {hot[0]}. Keep frame {rest[0]} as the "
                    f"contact pose and frame {hot[0]} as the passing pose, reorder to [contact, placeholder, pass, "
                    f"placeholder], then patch_frame each placeholder with the missing 'down'/'up' phase (single-cell "
                    f"generations). Regenerate the batch only if a patch fails twice."
                )
                findings.append("clone_collapse")
        if collapse:
            pass
        elif len(hot) == 1 and len(others) <= 1:
            action, reason = f"patch_frame {hot[0]}", (
                f"frame {hot[0]} is the shared anomaly in the low continuity pairs {low_pairs}; replace only that cell"
            )
        elif batch_mode:
            action, reason = regen_all, f"discontinuous poses within the batch (low pairs at {low_pairs})"
        else:
            b1_low = sum(1 for i in low_pairs if i < 3)
            b2_low = sum(1 for i in low_pairs if i >= 4)
            width_blockers = [i for i, cap in enumerate(caps) if cap * median_h / args.frame_height < args.min_height_ratio]
            wb1 = any(i < half for i in width_blockers)
            wb2 = any(i >= half for i in width_blockers)
            if b1_low and not b2_low and not wb2:
                action, reason = "regen_batch1", f"discontinuous poses concentrated in frames 0-3 (low pairs at {low_pairs})"
            elif b2_low and not b1_low and not wb1:
                action, reason = "regen_batch2", f"discontinuous poses concentrated in frames 4-7 (low pairs at {low_pairs})"
            else:
                action, reason = "regen_both", f"discontinuous poses and/or width blockers across both halves (low pairs at {low_pairs})"
        findings.append("continuity_fail")
    # 2. Width blockers.
    elif resulting < args.min_height_ratio:
        blockers = [i for i, cap in enumerate(caps)
                    if cap * median_h / args.frame_height < args.min_height_ratio and i not in ignored]
        rest = [cap for i, cap in enumerate(caps) if i not in blockers and i not in ignored]
        trims = trimmable(blockers)
        if trims is not None:
            # Free beats everything: a bounded garment-overhang trim unlocks the fit
            # without any generation. Run trim_overhang.py, then scale_fit + repairs.
            detail = ", ".join(f"frame {t['frame']} {t['side']} {t['px']}px" for t in trims)
            action, reason = "trim_overhang", (
                f"width-locked by a trimmable overhang ({detail}); run trim_overhang.py then "
                f"scale_fit -> auto_repair -> normalize_frames -> auto_repair and re-score"
            )
        elif len(blockers) == 1 and rest and min(min(rest), target_factor) * median_h / args.frame_height >= args.min_height_ratio:
            action, reason = f"patch_frame {blockers[0]}", (
                f"only frame {blockers[0]} blocks the fit (cap {caps[blockers[0]]:.2f}); the other {len(frames) - 1} fit at "
                f"{min(min(rest), target_factor) * median_h / args.frame_height:.3f}"
            )
        elif batch_mode:
            action, reason = regen_all, f"frames {blockers} too wide/off-center; set would drop to {resulting:.3f}"
        else:
            guilty1 = [i for i in blockers if i < half]
            guilty2 = [i for i in blockers if i >= half]
            if guilty1 and not guilty2:
                action, reason = "regen_batch1", f"frames {guilty1} too wide/off-center; set would drop to {resulting:.3f}"
            elif guilty2 and not guilty1:
                action, reason = "regen_batch2", f"frames {guilty2} too wide/off-center; set would drop to {resulting:.3f}"
            else:
                action, reason = "regen_both", f"frames {blockers} too wide/off-center; set would drop to {resulting:.3f}"
        findings.append("width_blocked")
    # 3. Batch-boundary scale jump.
    elif not batch_mode and not ignored and spread > spread_limit and abs(b1 - b2) > spread_limit * 0.6:
        action, reason = "harmonize", f"half medians {b1:.0f}px vs {b2:.0f}px (spread {spread:.0f}px): scale_fit --harmonize-halves"
        findings.append("half_scale_jump")
    # 3b. Per-frame height pulsing without a half-boundary jump: generation noise or a
    #     mis-scaled patched cell. Free fix, one bounded factor per frame.
    elif spread > spread_limit:
        action, reason = "flatten", (
            f"height pulsing (spread {spread:.0f}px > {spread_limit:.0f}px, half medians {b1:.0f}/{b2:.0f}px "
            f"nearly equal): run scale_fit --flatten-heights, then auto_repair -> normalize_frames -> re-score"
        )
        findings.append("height_pulsing")
    # 4. Uniform out-of-band scale.
    elif not (args.min_height_ratio <= median_h / args.frame_height <= args.max_height_ratio):
        action, reason = "scale_fit", f"set uniformly at {median_h / args.frame_height:.3f}; fits at target (cap {min(caps):.2f})"
        findings.append("out_of_band")

    if args.accepted_batch and action in (f"regen_batch{args.accepted_batch}", "regen_both"):
        other = 2 if args.accepted_batch == 1 else 1
        action = f"regen_batch{other}"
        reason += f" [redirected: batch {args.accepted_batch} is accepted and never regenerated by a full-set diagnose]"

    result = {
        "action": action,
        "reason": reason,
        "findings": findings,
        "set": {
            "median_h_ratio": round(median_h / args.frame_height, 3),
            "height_spread_px": round(spread, 1),
            "half_medians": [round(b1, 1), round(b2, 1)],
            "min_fit_cap": round(min(caps), 3),
            "resulting_ratio_if_fit": round(resulting, 3),
            "consecutive_ious": ious,
            "low_pairs": low_pairs,
        },
        "frames": rows,
    }
    text = json.dumps(result, indent=2)
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(text + "\n", encoding="utf-8")
    print(text)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

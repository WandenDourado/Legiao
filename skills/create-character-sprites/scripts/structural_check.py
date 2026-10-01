#!/usr/bin/env python3
"""Deterministic Layer-A structural gate for a single walk direction.

Catches the content defects that geometry/matte validation cannot see, for free,
before any expensive multimodal review:

  * orientation flip  - frames in the second half whose facing is mirrored
                        relative to the first half (the "column 5+ inverted" bug).
  * duplicate pose    - near-identical consecutive or paired frames (dead cycle).
  * leg alternation   - contact frames should swap the leading foot. For long-robe
                        characters the legs are hidden, so this check reports
                        `indeterminate` and asks for ONE small targeted visual crop
                        instead of forcing a full-grid regeneration.

Exit code: 0 when there is no hard structural failure (the set may still carry a
`needs_visual_review` flag), 1 on a hard failure, 2 on bad input. The JSON report
is the machine-readable contract the orchestrator reads to decide the next step.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from PIL import Image

from frame_analysis import content_patch, lower_body_offset, mask_iou, silhouette_mask


def load_frames(folder: Path, width: int, height: int, count: int) -> list[Image.Image]:
    paths = sorted(folder.glob("*.png"))
    if len(paths) != count:
        raise SystemExit(f"{folder}: expected {count} PNG frames, got {len(paths)}")
    frames = []
    for path in paths:
        with Image.open(path) as image:
            if image.size != (width, height):
                raise SystemExit(f"{path}: expected {width}x{height}, got {image.width}x{image.height}")
            frames.append(image.convert("RGBA"))
    return frames


def check_orientation(frames, alpha_limit, margin, min_flagged, body_fraction):
    """Second-half frames are compared against the first-half reference facing.

    Uses interior-content correlation (not the outline) so it detects facing flips
    even on near-symmetric robe silhouettes. Two generality guards keep false
    positives (and their regeneration loops) out:
      * only the UPPER body is correlated (`body_fraction`), because the legs of a
        correct cycle legitimately mirror between halves;
      * a hard fail needs at least `min_flagged` flagged frames — real generation
        inversions flip the whole second batch, not one noisy frame. A single
        flagged frame downgrades to a visual check instead of a regen.
    """
    from frame_analysis import _ncc

    half = len(frames) // 2
    ref_patches = [
        content_patch(frames[i], alpha_limit=alpha_limit, body_fraction=body_fraction) for i in range(half)
    ]
    flipped = []
    details = []
    for index in range(half, len(frames)):
        patch = content_patch(frames[index], ref_patches[0].size, alpha_limit, body_fraction)
        mirror = patch.transpose(Image.Transpose.FLIP_LEFT_RIGHT)
        # Best match across ALL first-half poses, same facing vs mirrored, to shed
        # pose-phase noise: a true facing flip beats every same-facing reference.
        best_same = max(_ncc(patch, ref) for ref in ref_patches)
        best_mirror = max(_ncc(mirror, ref) for ref in ref_patches)
        details.append({"frame": index, "ncc_same": round(best_same, 3), "ncc_mirror": round(best_mirror, 3)})
        if best_mirror > best_same + margin:
            flipped.append(index)
    if len(flipped) >= min_flagged:
        status = "fail"
    elif flipped:
        status = "suspect"  # one noisy frame: confirm with a single visual look, don't regen
    else:
        status = "pass"
    return {"status": status, "flipped_frames": flipped, "margin": margin, "min_flagged": min_flagged, "frames": details}


def _upper_body_submask(mask):
    """Rows of the silhouette bbox covering its upper 55% (head+torso).

    Why: full-silhouette IoU was calibrated on ROBED characters (mago,
    sacerdotisa, paladina), whose garment keeps consecutive silhouettes
    overlapping ~0.72+. A bare-legged character taking the CORRECT wide
    stride demanded of side views legitimately drops full-mask IoU to
    0.35-0.47 between stride and passing phases (archer W, 2026-07-20 —
    the accepted batch 1 itself failed the old check). The torso, however,
    barely moves in a legitimate walk regardless of leg scissoring, while
    real pose-salad (pointing, crouching, gesturing, looking away) disturbs
    the upper body too. Continuity is therefore judged on the upper body.
    """
    # masks from silhouette_mask are 1-bit images already cropped to their bbox
    cut = max(1, int(round(mask.height * 0.55)))
    sub = mask.crop((0, 0, mask.width, cut))
    bbox = sub.getbbox()
    return sub.crop(bbox) if bbox else sub


def check_continuity(frames, alpha_limit, iou_floor, max_low_pairs):
    """A walk cycle is a CONTINUOUS motion: consecutive UPPER-BODY silhouettes
    overlap heavily. Legs are excluded: a correct side-view stride scissors
    them apart and legitimately collapses full-mask IoU (see
    _upper_body_submask). Full-mask IoUs are still reported as diagnostics.
    """
    masks = [silhouette_mask(frame, alpha_limit) for frame in frames]
    upper = [_upper_body_submask(m) for m in masks]
    ious = [round(mask_iou(upper[i], upper[i + 1]), 3) for i in range(len(frames) - 1)]
    full_ious = [round(mask_iou(masks[i], masks[i + 1]), 3) for i in range(len(frames) - 1)]
    low = [{"pair": f"{i}-{i+1}", "iou": ious[i]} for i in range(len(ious)) if ious[i] < iou_floor]
    if len(low) >= max_low_pairs:
        status = "fail"
    elif low:
        status = "suspect"
    else:
        status = "pass"
    return {"status": status, "iou_floor": iou_floor, "mode": "upper_body",
            "consecutive_ious": ious, "full_mask_ious": full_ious, "low_pairs": low}


def check_height_consistency(frames, alpha_limit, max_spread):
    """All 8 frames must share one scale; a batch-boundary jump (frames 0-3 at one
    height, 4-7 at another) makes the character pulse while walking. Repairable
    without generation: scale_fit --harmonize-halves."""
    heights = []
    for frame in frames:
        bbox = silhouette_mask(frame, alpha_limit).size  # cropped mask: size == bbox dims
        heights.append(bbox[1])
    spread = max(heights) - min(heights)
    half = len(frames) // 2
    import statistics
    b1 = statistics.median(heights[:half])
    b2 = statistics.median(heights[half:])
    return {
        "status": "fail" if spread > max_spread else "pass",
        "max_spread": max_spread,
        "spread": spread,
        "heights": heights,
        "half_medians": [b1, b2],
        "repair": "scale_fit --harmonize-halves" if spread > max_spread else None,
    }


def check_duplicates(frames, alpha_limit, static_iou):
    """Flag only a fully frozen cycle. A subtle walk on a robed body is legitimately
    near-static in silhouette, so we fail only when even the most-different
    consecutive pair is essentially identical (no motion anywhere)."""
    masks = [silhouette_mask(frame, alpha_limit) for frame in frames]
    consecutive = [mask_iou(masks[i], masks[i + 1]) for i in range(len(frames) - 1)]
    weakest = max(consecutive)  # the pair with the LEAST change
    strongest = min(consecutive)  # the pair with the MOST change
    static = strongest >= static_iou
    return {
        "status": "fail" if static else "pass",
        "static_iou_threshold": static_iou,
        "min_consecutive_iou": round(strongest, 4),
        "max_consecutive_iou": round(weakest, 4),
        "note": "cycle is frozen" if static else "motion present",
    }


def check_leg_alternation(frames, alpha_limit, direction, side_views):
    """Report lower-body motion metrics for a side view and request ONE visual check.

    Deciding the lead foot from pixels alone is fundamentally unreliable in a side
    view: contact A and contact B have near-identical silhouettes (the difference —
    which body-side leg is forward — lives in appearance/occlusion, not geometry),
    and a long robe hides the feet entirely. A wrong hard-fail here would trigger a
    regeneration loop, so this check NEVER hard-fails. It measures lower-body lateral
    motion as supporting evidence and returns `indeterminate`, meaning: confirm foot
    alternation with a single targeted look at the lower band of frames 1/3/5/7.

    Front/back/diagonal views alternate the lead foot in depth, where a lateral cue
    is meaningless: `not_applicable` (no hard fail, no vision cost).
    """
    if direction is not None and direction.upper() not in side_views:
        return {
            "status": "not_applicable",
            "reason": f"lateral lead-foot cue only valid for side views {sorted(side_views)}, not {direction.upper()}",
        }
    offsets = [lower_body_offset(frame, alpha_limit=alpha_limit)[0] for frame in frames]
    spread = max(offsets) - min(offsets)
    return {
        "status": "indeterminate",
        "reason": "side-view lead foot cannot be decided from pixels; confirm with one lower-band visual check of frames 1/3/5/7",
        "offset_spread": round(spread, 2),
        "measured": [{"frame": i, "offset": round(o, 2)} for i, o in enumerate(offsets)],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input_dir", type=Path)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames", type=int, default=8)
    parser.add_argument("--report", type=Path, help="Optional JSON report path.")
    parser.add_argument("--direction", help="Direction name (S, SW, W, N, NW); enables the side-view leg check only where valid.")
    parser.add_argument("--side-views", default="W,E",
                        help="Comma-separated directions where the lateral lead-foot check applies.")
    parser.add_argument("--alpha-threshold", type=int, default=16)
    parser.add_argument("--orientation-margin", type=float, default=0.15,
                        help="Mirrored content correlation must beat same-facing by this to flag a flip.")
    parser.add_argument("--orientation-min-flagged", type=int, default=2,
                        help="Flagged second-half frames required for a hard orientation fail (fewer -> visual check).")
    parser.add_argument("--orientation-body-fraction", type=float, default=0.58,
                        help="Top fraction of the silhouette used for facing correlation (legs excluded: they mirror legitimately).")
    parser.add_argument("--static-iou", type=float, default=0.9999,
                        help="If the most-changing consecutive pair is at or above this, the cycle is frozen.")
    parser.add_argument("--continuity-iou", type=float, default=0.70,
                        help="Consecutive-frame silhouette IoU below this marks a discontinuous pose jump.")
    parser.add_argument("--continuity-max-low", type=int, default=2,
                        help="Low-IoU pairs required for a hard continuity fail (fewer -> visual check).")
    parser.add_argument("--height-max-spread", type=float, default=5.0,
                        help="Max px spread of silhouette heights across the set before flagging inconsistent scale.")
    args = parser.parse_args()

    if args.frames % 2:
        print("Frame count must be even (two walk halves).", file=sys.stderr)
        return 2

    frames = load_frames(args.input_dir, args.frame_width, args.frame_height, args.frames)
    side_views = {item.strip().upper() for item in args.side_views.split(",") if item.strip()}
    orientation = check_orientation(
        frames, args.alpha_threshold, args.orientation_margin,
        args.orientation_min_flagged, args.orientation_body_fraction,
    )
    duplicates = check_duplicates(frames, args.alpha_threshold, args.static_iou)
    continuity = check_continuity(frames, args.alpha_threshold, args.continuity_iou, args.continuity_max_low)
    # The spread limit is defined at export scale (192px height); scale it for
    # supersampled inputs, like diagnose_direction does — otherwise legitimate 2x
    # sets fail with spreads that are within contract at export resolution.
    spread_limit = args.height_max_spread * (args.frame_height / 192.0)
    heights = check_height_consistency(frames, args.alpha_threshold, spread_limit)
    legs = check_leg_alternation(frames, args.alpha_threshold, args.direction, side_views)

    hard_fail = any(section["status"] == "fail" for section in (orientation, duplicates, continuity))
    needs_visual_review = (
        legs["status"] == "indeterminate"
        or orientation["status"] == "suspect"
        or continuity["status"] == "suspect"
    )
    report = {
        "input_dir": str(args.input_dir),
        "orientation": orientation,
        "duplicates": duplicates,
        "continuity": continuity,
        "height_consistency": heights,
        "leg_alternation": legs,
        "hard_fail": hard_fail,
        "needs_visual_review": needs_visual_review,
        "needs_repair": heights["status"] == "fail",
    }
    text = json.dumps(report, indent=2)
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(text + "\n", encoding="utf-8")
    print(text)
    if hard_fail:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

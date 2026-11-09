#!/usr/bin/env python3
"""Feathered trim of a garment overhang (cape/cloak tail) that width-locks scale_fit.

Failure mode this solves (discovered on side-view walks with long capes): every
frame in the set is uniform and healthy, but the trailing cape extends so far from
the torso pivot that scaling the silhouette up to the height band would clip the
canvas — scale_fit refuses, and the naive playbook escalates to a paid regeneration.
When the overhang is soft fabric far above the feet, shaving a few pixels off its
tip is visually invisible and unlocks the whole set for free.

Safety guards (the script refuses rather than damage content):
  - trims ONLY the side that binds the fit cap, ONLY beyond the cut column;
  - per-frame trim is capped (--max-trim, default 6 px at export scale, scaled up
    for supersampled frames);
  - the trimmed region must sit entirely ABOVE the foot zone (rows below
    baseline - foot-zone are protected), so feet/legs are never touched;
  - alpha is tapered linearly across the trim zone, no hard vertical cut.

Run scale_fit AFTER this script; then auto_repair + normalize_frames + a second
scale_fit pass (recentering frees extra width), auto_repair again, and re-score.
diagnose_direction.py recommends this action as `trim_overhang` when feasible.
"""

from __future__ import annotations

import argparse
import json
import math
from pathlib import Path

from PIL import Image

from frame_analysis import foreground_bbox, torso_center


def occupied_rows(image: Image.Image, x0: int, x1: int, alpha_limit: int) -> tuple[int, int] | None:
    """Bottom/top opaque rows within columns [x0, x1], or None if empty."""
    if x1 < x0:
        return None
    region = image.convert("RGBA").crop((x0, 0, x1 + 1, image.height)).getchannel("A")
    mask = region.point(lambda value: 255 if value >= alpha_limit else 0)
    bbox = mask.getbbox()
    if bbox is None:
        return None
    return bbox[1], bbox[3] - 1  # top row, bottom row


def zone_density(image: Image.Image, x0: int, x1: int, y0: int, y1: int, alpha_limit: int) -> float:
    """Opaque fill fraction of a column zone. Soft garment mass (cape hem) fills most
    of its bounding rows; a thin rigid prop (bow limb, staff, spear tip) traces a
    sparse line. Trimming garments is invisible; trimming props mutilates them."""
    if x1 < x0 or y1 < y0:
        return 0.0
    region = image.convert("RGBA").crop((x0, y0, x1 + 1, y1 + 1)).getchannel("A")
    data = region.get_flattened_data() if hasattr(region, "get_flattened_data") else region.getdata()
    opaque = sum(1 for value in data if value >= alpha_limit)
    return opaque / ((x1 - x0 + 1) * (y1 - y0 + 1))


def taper_columns(image: Image.Image, new_edge: int, old_edge: int, left_side: bool, feather: int = 4) -> None:
    """Clear alpha beyond `new_edge` (through `old_edge`) and feather a small fringe
    inside the kept region so the cut has no hard vertical line. The cleared zone is
    fully transparent, so the silhouette bbox genuinely shrinks by the trim amount."""
    pixels = image.load()
    step = -1 if left_side else 1
    column = new_edge + step
    while (column >= old_edge) if left_side else (column <= old_edge):
        for y in range(image.height):
            r, g, b, a = pixels[column, y]
            if a:
                pixels[column, y] = (r, g, b, 0)
        column += step
    for offset in range(feather):
        column = new_edge - step * offset
        factor = 0.35 + 0.65 * (offset + 1) / feather
        for y in range(image.height):
            r, g, b, a = pixels[column, y]
            if a:
                pixels[column, y] = (r, g, b, int(a * factor))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path)
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--directions", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames-per-direction", type=int, default=8)
    parser.add_argument("--anchor-x", type=int, help="Torso pivot (default: frame width / 2).")
    parser.add_argument("--baseline", type=int, help="Foot baseline (default: 186 scaled to frame height).")
    parser.add_argument("--target-height-ratio", type=float, default=0.84)
    parser.add_argument("--edge-margin", type=int, default=2)
    parser.add_argument("--max-trim", type=float, default=6.0,
                        help="Max trim per frame in px at export scale (192px height); scaled for supersampled frames.")
    parser.add_argument("--foot-zone", type=float, default=16.0,
                        help="Rows within this many export-scale px above the baseline are protected.")
    parser.add_argument("--min-zone-density", type=float, default=0.35,
                        help="Minimum opaque fill of the trim zone; sparser zones are rigid props, not garment.")
    parser.add_argument("--alpha-threshold", type=int, default=16, help="Use 40 for supersampled (2x) frames.")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()

    scale = args.frame_height / 192.0
    anchor_x = args.anchor_x if args.anchor_x is not None else args.frame_width // 2
    baseline = args.baseline if args.baseline is not None else round(186 * scale)
    max_trim = args.max_trim * scale
    foot_top = baseline - args.foot_zone * scale  # rows >= foot_top are protected

    summary = {}
    for direction in args.directions.split(","):
        direction = direction.strip()
        in_dir = args.input_root / direction
        out_dir = args.output_root / direction
        paths = sorted(in_dir.glob("*.png"))
        if len(paths) != args.frames_per_direction:
            print(f"{direction}: expected {args.frames_per_direction} frames, found {len(paths)}")
            return 1

        frames = [Image.open(path).convert("RGBA") for path in paths]
        stats = []
        for frame in frames:
            bbox = foreground_bbox(frame, args.alpha_threshold)
            tx = torso_center(frame, alpha_limit=args.alpha_threshold)
            stats.append((bbox, tx, bbox[3] - bbox[1]))
        median_h = sorted(stat[2] for stat in stats)[len(stats) // 2]
        target_factor = (args.target_height_ratio * args.frame_height) / median_h
        avail_left = anchor_x - args.edge_margin
        avail_right = args.frame_width - 1 - anchor_x - args.edge_margin

        plan = []
        for index, (bbox, tx, _height) in enumerate(stats):
            left_ext, right_ext = tx - bbox[0], bbox[2] - tx
            trims = []
            for side, ext, avail in (("left", left_ext, avail_left), ("right", right_ext, avail_right)):
                # 1 px of slack absorbs the rescale-then-anchor rounding inside scale_fit.
                allowed = math.floor(avail / target_factor) - 1
                needed = ext - allowed
                if needed > 0:
                    trims.append((side, math.ceil(needed)))
            plan.append((index, bbox, trims))

        for index, bbox, trims in plan:
            for side, trim in trims:
                if trim > max_trim:
                    print(f"{direction} frame {index}: needs {trim}px off the {side} side, above the "
                          f"{max_trim:.0f}px cap — not a trim case; run diagnose_direction for the next action")
                    return 1
                x0, x1 = (bbox[0], bbox[0] + trim - 1) if side == "left" else (bbox[2] - trim + 1, bbox[2])
                rows = occupied_rows(frames[index], x0, x1, args.alpha_threshold)
                if rows and rows[1] >= foot_top:
                    print(f"{direction} frame {index}: {side} trim zone reaches row {rows[1]} inside the foot "
                          f"zone (>= {foot_top:.0f}) — refusing to touch feet/legs; regenerate instead")
                    return 1
                if rows:
                    density = zone_density(frames[index], x0, x1, rows[0], rows[1], args.alpha_threshold)
                    if density < args.min_zone_density:
                        print(f"{direction} frame {index}: {side} trim zone density {density:.2f} < "
                              f"{args.min_zone_density:.2f} — looks like a thin rigid prop (bow/staff/spear), "
                              f"not garment mass; refusing to mutilate it. Regenerate smaller/centered instead.")
                        return 1

        out_dir.mkdir(parents=True, exist_ok=True)
        applied = []
        for (index, bbox, trims), path in zip(plan, paths):
            frame = frames[index]
            for side, trim in trims:
                if side == "left":
                    taper_columns(frame, bbox[0] + trim, bbox[0], left_side=True)
                else:
                    taper_columns(frame, bbox[2] - trim, bbox[2], left_side=False)
            frame.save(out_dir / path.name)
            applied.append({"frame": index, "trims": [{"side": s, "px": t} for s, t in trims]})
        summary[direction] = applied
        trimmed = sum(1 for entry in applied if entry["trims"])
        print(f"OK: {direction} trimmed overhang on {trimmed}/{len(applied)} frames "
              f"(target factor {target_factor:.3f}); now run scale_fit -> auto_repair -> "
              f"normalize_frames -> scale_fit -> auto_repair -> re-score")

    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

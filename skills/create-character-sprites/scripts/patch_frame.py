#!/usr/bin/env python3
"""Surgically replace ONE frame of an accepted direction with a new candidate.

Post-delivery fixes ("S column 3 looks wrong") must not regenerate the whole
direction. This script takes a single candidate image — a freshly generated one-pose
cell (any size, magenta matte allowed) or a hand-edited RGBA frame — and:

  1. keys the magenta matte if present;
  2. scales the candidate by ONE factor so its silhouette height matches the median
     of the direction's OTHER frames (not a fixed target: consistency with the set
     is what matters), and anchors it on the set's torso pivot and foot baseline;
  3. refuses if the result would clip the canvas;
  4. backs up the old frame to attempts/<DIR>/patches/<timestamp>/ and swaps in
     the new one.

After a swap, re-run validate_frames --score and structural_check on the direction,
then rebuild the sheet with build_sheet.py. Cost: one single-cell generation (or
zero, for a hand edit) plus scripts — never a batch regeneration.

Column/frame numbering: sheet column K (1-8) = frame index K-1 (file 00{K-1}.png).
"""

from __future__ import annotations

import argparse
import datetime as _dt
import shutil
import statistics
from pathlib import Path

from PIL import Image

from auto_repair import clear_border
from frame_analysis import body_anchor, foreground_bbox, torso_center, visible_magenta_pixels
from slice_and_stitch import clear_near_transparent_pixels, matte_to_alpha


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--frames-dir", required=True, type=Path,
                        help="Accepted direction folder, e.g. work/<id>/frames/S.")
    parser.add_argument("--frame", type=int, help="Frame index 0-7.")
    parser.add_argument("--column", type=int, help="Sheet column 1-8 (alias for --frame column-1).")
    parser.add_argument("--candidate", required=True, type=Path,
                        help="New art: one-pose cell (any size; #FF00FF matte allowed) or a keyed RGBA frame.")
    parser.add_argument("--backup-root", type=Path,
                        help="Where to store the replaced frame (default: <frames-dir>/../../attempts/<DIR>/patches).")
    parser.add_argument("--frame-width", type=int, default=128)
    parser.add_argument("--frame-height", type=int, default=192)
    parser.add_argument("--anchor-x", type=int, default=64)
    parser.add_argument("--baseline", type=int, default=186)
    parser.add_argument("--matte-threshold", type=int, default=120)
    parser.add_argument("--matte-feather-threshold", type=int, default=190)
    parser.add_argument("--alpha-threshold", type=int, default=16)
    parser.add_argument("--edge-margin", type=int, default=1)
    parser.add_argument("--min-matte-pixels", type=int, default=500,
                        help="Visible magenta pixels above this triggers matte keying of the candidate.")
    args = parser.parse_args()

    if (args.frame is None) == (args.column is None):
        raise SystemExit("Provide exactly one of --frame (0-7) or --column (1-8).")
    index = args.frame if args.frame is not None else args.column - 1
    if not 0 <= index <= 7:
        raise SystemExit("Frame index must be 0-7 (column 1-8).")

    target_path = args.frames_dir / f"{index:03d}.png"
    if not target_path.is_file():
        raise SystemExit(f"{target_path} does not exist.")
    others = [p for p in sorted(args.frames_dir.glob("*.png")) if p != target_path]
    if len(others) not in (3, 7):
        raise SystemExit(f"{args.frames_dir}: expected 8 frames — or 4 for a partial-batch patch (found {len(others) + 1}).")

    # Set metrics from the 7 kept frames: the candidate must match THEM.
    heights = []
    for path in others:
        with Image.open(path) as image:
            bbox = foreground_bbox(image.convert("RGBA"), args.alpha_threshold)
            heights.append(bbox[3] - bbox[1])
    target_height = statistics.median(heights)

    with Image.open(args.candidate) as source:
        candidate = source.convert("RGBA")
    if visible_magenta_pixels(candidate, 140, args.alpha_threshold) > args.min_matte_pixels:
        candidate = clear_near_transparent_pixels(
            matte_to_alpha(candidate, args.matte_threshold, args.matte_feather_threshold)
        )
    bbox = foreground_bbox(candidate, args.alpha_threshold)
    cand_height = bbox[3] - bbox[1]
    factor = target_height / cand_height
    scaled = candidate.resize(
        (max(1, round(candidate.width * factor)), max(1, round(candidate.height * factor))),
        Image.Resampling.LANCZOS,
    )
    scaled = clear_near_transparent_pixels(scaled)
    tx = torso_center(scaled, alpha_limit=args.alpha_threshold)
    _, by = body_anchor(scaled, alpha_limit=args.alpha_threshold)
    offset = (round(args.anchor_x - tx), args.baseline - by)
    # Judge fit by the VISIBLE silhouette (alpha >= threshold); near-invisible feather
    # pixels must not veto a fit, they are cleaned below by the transparent border.
    sb = foreground_bbox(scaled, args.alpha_threshold)
    # A hem/foot extending a hair deeper than the set's is common; nudge up within the
    # validator's baseline tolerance instead of refusing over 1-2px.
    bottom_overflow = (sb[3] + offset[1]) - (args.frame_height - args.edge_margin)
    if 0 < bottom_overflow <= 2:
        offset = (offset[0], offset[1] - bottom_overflow)
        print(f"note: nudged {bottom_overflow}px up to keep the hem inside the transparent border")
    if (sb[0] + offset[0] < args.edge_margin or sb[2] + offset[0] > args.frame_width - args.edge_margin
            or sb[1] + offset[1] < args.edge_margin or sb[3] + offset[1] > args.frame_height - args.edge_margin):
        raise SystemExit(
            "candidate does not fit the frame at the set's scale/anchor (too wide or off-center); "
            "generate the pose smaller/centered and retry."
        )
    canvas = Image.new("RGBA", (args.frame_width, args.frame_height))
    canvas.alpha_composite(scaled, offset)
    canvas = clear_border(clear_near_transparent_pixels(canvas), 1)

    direction = args.frames_dir.name
    backup_root = args.backup_root or (args.frames_dir.parent.parent / "attempts" / direction / "patches")
    stamp = _dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    backup_dir = backup_root / stamp
    backup_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(target_path, backup_dir / target_path.name)
    canvas.save(target_path)
    print(f"OK: patched {direction} frame {index} (column {index + 1}); old frame backed up to {backup_dir}")
    print("Next: re-run validate_frames --score and structural_check on this direction, then rebuild the sheet.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

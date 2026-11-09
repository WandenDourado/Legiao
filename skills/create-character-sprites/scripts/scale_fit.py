#!/usr/bin/env python3
"""Uniformly rescale a direction's frame set to the target silhouette height, then
re-anchor. This is a Layer-B repair: it fixes an out-of-band silhouette scale WITHOUT
a new image generation.

Safety: every frame in the direction is scaled by the SAME factor (derived from the
set's median silhouette height), so relative motion, proportions, and inter-frame
consistency are preserved exactly. Only a global zoom + recenter is applied; frames are
never distorted independently. Content is re-centered on the fixed torso pivot and foot
baseline afterwards. If a scaled frame would clip the canvas horizontally the script
refuses and asks for regeneration instead.

Use it when validate_frames.py --score reports an out-of-range foreground height. After
running, re-validate to confirm the set now sits inside the accepted band.
"""

from __future__ import annotations

import argparse
import statistics
from pathlib import Path

from PIL import Image

from frame_analysis import body_anchor, foreground_bbox, torso_center


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


def scale_and_anchor(frame: Image.Image, factor: float, frame_w: int, frame_h: int,
                     anchor_x: int, baseline: int, alpha_limit: int,
                     body_left: float, body_right: float, torso_top: float, torso_bottom: float) -> Image.Image:
    """Zoom the whole frame by `factor`, then place it on the fixed anchor."""
    scaled = frame.resize((round(frame.width * factor), round(frame.height * factor)), Image.Resampling.LANCZOS)
    tx = torso_center(scaled, torso_top, torso_bottom, alpha_limit)
    _, by = body_anchor(scaled, body_left, body_right, alpha_limit)
    offset_x = round(anchor_x - tx)
    offset_y = baseline - by
    # Clip check at the SAME alpha threshold as the width cap: resampling spreads a
    # faint sub-threshold halo past the silhouette edge, and failing on that halo
    # rejects tight-but-legitimate fits. Sub-threshold residue cropped at the canvas
    # border is harmless (validators ignore it; auto_repair clears it).
    left, top, right, bottom = foreground_bbox(scaled, alpha_limit)  # PIL-style exclusive right/bottom
    if left + offset_x < 0 or right + offset_x > frame_w or top + offset_y < 0 or bottom + offset_y > frame_h:
        raise SystemExit(
            "scaled+anchored frame would clip the canvas; regenerate this direction with more padding instead"
        )
    canvas = Image.new("RGBA", (frame_w, frame_h))
    canvas.alpha_composite(scaled, (offset_x, offset_y))
    return canvas


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path)
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--directions", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames-per-direction", type=int, default=8)
    parser.add_argument("--target-height-ratio", type=float, default=0.84,
                        help="Median silhouette height the set is scaled to (0.84 default, inside the 0.80-0.92 band).")
    parser.add_argument("--min-height-ratio", type=float, default=0.80,
                        help="Floor for width-driven shrink; below this the set is too wide/off-center and must be regenerated.")
    parser.add_argument("--edge-margin", type=int, default=2, help="Transparent px kept between the silhouette and each side edge.")
    parser.add_argument("--harmonize-halves", action="store_true",
                        help="First unify a scale jump between frames 0-3 and 4-7 (batch boundary) before the whole-set fit.")
    parser.add_argument("--anchor-x", type=int, default=64)
    parser.add_argument("--baseline", type=int, default=186)
    parser.add_argument("--flatten-heights", action="store_true",
                        help="Per-frame height equalization: rescale EACH frame by its own small factor so every "
                             "silhouette lands at the same height (min of target and the set's width cap). Use when "
                             "a set pulses in height (generation noise / a mis-scaled patch) — the uniform-factor "
                             "doctrine preserves motion but cannot fix per-frame scale errors. Factors are bounded "
                             "by --max-flatten-step; the pose itself is untouched (global zoom only).")
    parser.add_argument("--max-flatten-step", type=float, default=0.12,
                        help="Max per-frame deviation from 1.0 allowed in --flatten-heights mode.")
    parser.add_argument("--max-scale-step", type=float, default=0.35,
                        help="Refuse a rescale larger than this fraction; a huge mismatch means regenerate.")
    parser.add_argument("--alpha-threshold", type=int, default=16)
    parser.add_argument("--body-left", type=float, default=0.30)
    parser.add_argument("--body-right", type=float, default=0.70)
    parser.add_argument("--torso-top", type=float, default=0.25)
    parser.add_argument("--torso-bottom", type=float, default=0.65)
    args = parser.parse_args()

    if not 0 < args.target_height_ratio < 1:
        raise SystemExit("--target-height-ratio must be between 0 and 1.")
    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    for direction in directions:
        frames = load_frames(args.input_root / direction, args.frame_width, args.frame_height, args.frames_per_direction)
        heights = []
        for frame in frames:
            bbox = foreground_bbox(frame, args.alpha_threshold)
            heights.append(bbox[3] - bbox[1])
        if args.harmonize_halves:
            # Unify a batch-boundary scale jump: rescale the second half by ONE factor
            # so its median silhouette height matches the first half's, before the
            # whole-set fit below. Uniform within each half -> motion is preserved.
            half = len(frames) // 2
            m1 = statistics.median(heights[:half])
            m2 = statistics.median(heights[half:])
            if abs(m1 - m2) > 1:
                hfactor = m1 / m2
                for i in range(half, len(frames)):
                    frame = frames[i]
                    resized = frame.resize(
                        (max(1, round(frame.width * hfactor)), max(1, round(frame.height * hfactor))),
                        Image.Resampling.LANCZOS,
                    )
                    # refit onto the canvas centered; final anchoring happens below
                    canvas = Image.new("RGBA", (args.frame_width, args.frame_height))
                    canvas.alpha_composite(
                        resized,
                        ((args.frame_width - resized.width) // 2, args.frame_height - resized.height),
                    )
                    frames[i] = canvas
                    bbox = foreground_bbox(canvas, args.alpha_threshold)
                    heights[i] = bbox[3] - bbox[1]
                print(f"{direction}: harmonized halves ({m2:.0f}px -> {m1:.0f}px, factor {hfactor:.3f} on frames {half}-{len(frames)-1})")
        median_height = statistics.median(heights)
        target_px = args.target_height_ratio * args.frame_height
        if args.flatten_heights:
            caps_px = []
            for frame, height in zip(frames, heights):
                bbox = foreground_bbox(frame, args.alpha_threshold)
                tx = torso_center(frame, args.torso_top, args.torso_bottom, args.alpha_threshold)
                left_ext, right_ext = tx - bbox[0], bbox[2] - tx
                cap = min(
                    (args.anchor_x - args.edge_margin) / left_ext if left_ext > 0 else float("inf"),
                    (args.frame_width - 1 - args.anchor_x - args.edge_margin) / right_ext if right_ext > 0 else float("inf"),
                )
                caps_px.append(cap * height)
            final_px = min(target_px, min(caps_px))
            if final_px / args.frame_height < args.min_height_ratio:
                raise SystemExit(
                    f"{direction}: equalized height would be {final_px / args.frame_height:.3f} (floor "
                    f"{args.min_height_ratio:.2f}); run diagnose_direction for the next action."
                )
            factors = [final_px / height for height in heights]
            for index, item in enumerate(factors):
                if abs(item - 1.0) > args.max_flatten_step:
                    raise SystemExit(
                        f"{direction}: frame {index} needs a {item:.3f}x rescale (>{args.max_flatten_step:.0%} "
                        f"flatten step) — too far off the set; patch or regenerate that cell instead."
                    )
            out_dir = args.output_root / direction
            out_dir.mkdir(parents=True, exist_ok=True)
            for index, (frame, item) in enumerate(zip(frames, factors)):
                fitted = scale_and_anchor(
                    frame, item, args.frame_width, args.frame_height, args.anchor_x, args.baseline,
                    args.alpha_threshold, args.body_left, args.body_right, args.torso_top, args.torso_bottom,
                )
                fitted.save(out_dir / f"{index:03d}.png")
            print(f"OK: {direction} heights equalized to {final_px / args.frame_height:.3f} "
                  f"(per-frame factors {', '.join(f'{item:.3f}' for item in factors)}), "
                  f"re-anchored to ({args.anchor_x},{args.baseline})")
            continue
        factor = target_px / median_height
        # Width-aware clamp: pinning the torso to anchor_x can push a wide garment off
        # the canvas. Compute the largest factor that keeps every frame inside the
        # canvas when centered, and shrink toward it (never below the height floor) so
        # a slightly-wide set is salvaged instead of regenerated.
        width_caps = []
        for frame in frames:
            bbox = foreground_bbox(frame, args.alpha_threshold)
            tx = torso_center(frame, args.torso_top, args.torso_bottom, args.alpha_threshold)
            left_ext, right_ext = tx - bbox[0], bbox[2] - tx
            cap_left = (args.anchor_x - args.edge_margin) / left_ext if left_ext > 0 else float("inf")
            cap_right = (args.frame_width - 1 - args.anchor_x - args.edge_margin) / right_ext if right_ext > 0 else float("inf")
            width_caps.append(min(cap_left, cap_right))
        width_cap = min(width_caps)
        if width_cap < factor:
            factor = width_cap  # shrink to fit the widest pose within the canvas
        resulting_ratio = factor * median_height / args.frame_height
        if resulting_ratio < args.min_height_ratio:
            raise SystemExit(
                f"{direction}: cannot fit the silhouette within the canvas and keep the torso centered "
                f"without dropping to {resulting_ratio:.3f} height (floor {args.min_height_ratio:.2f}); "
                f"the generation is too wide/off-center - regenerate with the character centered and clear side margins."
            )
        if abs(factor - 1.0) > args.max_scale_step:
            raise SystemExit(
                f"{direction}: needs a {factor:.2f}x rescale (>{args.max_scale_step:.0%} step); regenerate with the correct scale instead."
            )
        out_dir = args.output_root / direction
        out_dir.mkdir(parents=True, exist_ok=True)
        for index, frame in enumerate(frames):
            fitted = scale_and_anchor(
                frame, factor, args.frame_width, args.frame_height, args.anchor_x, args.baseline,
                args.alpha_threshold, args.body_left, args.body_right, args.torso_top, args.torso_bottom,
            )
            fitted.save(out_dir / f"{index:03d}.png")
        before = statistics.median(heights) / args.frame_height
        print(f"OK: {direction} rescaled {factor:.3f}x (median height {before:.3f} -> ~{args.target_height_ratio:.2f}), re-anchored to ({args.anchor_x},{args.baseline})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

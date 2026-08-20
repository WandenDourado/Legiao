#!/usr/bin/env python3
"""Single final downsample from working resolution to export resolution.

Sharpness rule: resample ONCE. The pipeline therefore runs at a supersampled
working resolution (2x: 256x384, anchors 128/372) through slice, normalize,
scale_fit, auto_repair, and patch, and this script performs the one and only
reduction to the export size (128x192). Never chain LANCZOS passes at export
resolution — each pass blurs, and the compounding is what makes sprites look
pixelated in-game.
"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image

from auto_repair import clear_border
from slice_and_stitch import clear_near_transparent_pixels


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path, help="Working-resolution frame dirs.")
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--directions", required=True)
    parser.add_argument("--factor", type=int, default=2, help="Integer supersample factor being reduced.")
    parser.add_argument("--frame-width", type=int, default=128, help="Export frame width.")
    parser.add_argument("--frame-height", type=int, default=192, help="Export frame height.")
    parser.add_argument("--frames-per-direction", type=int, default=8)
    args = parser.parse_args()

    if args.factor < 1:
        raise SystemExit("--factor must be >= 1.")
    expected = (args.frame_width * args.factor, args.frame_height * args.factor)
    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    for direction in directions:
        in_dir = args.input_root / direction
        paths = sorted(in_dir.glob("*.png"))
        if len(paths) != args.frames_per_direction:
            raise SystemExit(f"{in_dir}: expected {args.frames_per_direction} frames, got {len(paths)}")
        out_dir = args.output_root / direction
        out_dir.mkdir(parents=True, exist_ok=True)
        for path in paths:
            with Image.open(path) as source:
                frame = source.convert("RGBA")
            if frame.size != expected:
                raise SystemExit(f"{path}: expected working size {expected[0]}x{expected[1]}, got {frame.width}x{frame.height}")
            reduced = frame.resize((args.frame_width, args.frame_height), Image.Resampling.LANCZOS)
            reduced = clear_border(clear_near_transparent_pixels(reduced), 1)
            reduced.save(out_dir / path.name)
        print(f"OK: {direction} finalized {len(paths)} frames {expected[0]}x{expected[1]} -> {args.frame_width}x{args.frame_height} (single resample)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

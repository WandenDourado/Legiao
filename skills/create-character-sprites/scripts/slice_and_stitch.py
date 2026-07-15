#!/usr/bin/env python3
"""Split a 2x4 AI grid into keyed RGBA frames for one direction."""

from __future__ import annotations

import argparse
from pathlib import Path

try:
    from PIL import Image
except ImportError as exc:  # pragma: no cover
    raise SystemExit("Pillow is required. Install with: python -m pip install pillow") from exc

from frame_analysis import magenta_distance


def matte_to_alpha(image: Image.Image, threshold: int) -> Image.Image:
    """Key and despill a magenta generation matte."""
    pixels = []
    rgba = image.convert("RGBA")
    data = rgba.get_flattened_data() if hasattr(rgba, "get_flattened_data") else rgba.getdata()
    for red, green, blue, alpha in data:
        distance = magenta_distance(red, green, blue)
        if distance < threshold:
            key_strength = 1 - distance / threshold
            alpha = round(alpha * (1 - key_strength))
            spill = max(0, min(red, blue) - green)
            green = min(255, round(green + spill * key_strength))
        pixels.append((red, green, blue, alpha))
    result = Image.new("RGBA", image.size)
    result.putdata(pixels)
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, help="Unguttered 2x4 grid image.")
    parser.add_argument("--direction", required=True, help="Direction folder name, for example S.")
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--matte-threshold", type=int, default=160)
    parser.add_argument("--minimum-source-scale", type=float, default=1.0)
    args = parser.parse_args()

    if args.frame_width <= 0 or args.frame_height <= 0 or not 0 < args.matte_threshold <= 255:
        raise SystemExit("Frame dimensions and --matte-threshold must be positive; threshold must be at most 255.")
    if args.minimum_source_scale < 1:
        raise SystemExit("--minimum-source-scale must be at least 1.")
    if not args.input.is_file():
        raise SystemExit(f"Input does not exist: {args.input}")

    with Image.open(args.input) as source:
        if source.width % 4 or source.height % 2:
            raise SystemExit("Grid dimensions must divide exactly into 4 columns and 2 rows; do not use gutters.")
        crop_width, crop_height = source.width // 4, source.height // 2
        if crop_width * args.frame_height != crop_height * args.frame_width:
            raise SystemExit(
                f"Grid cells are {crop_width}x{crop_height}, not the requested {args.frame_width}x{args.frame_height} ratio."
            )
        source_scale = min(crop_width / args.frame_width, crop_height / args.frame_height)
        if source_scale < args.minimum_source_scale:
            raise SystemExit(
                f"Grid cells are only {source_scale:.2f}x the target frame; expected at least "
                f"{args.minimum_source_scale:.2f}x for the requested export quality."
            )
        source = source.convert("RGBA")
        frames = [
            source.crop((column * crop_width, row * crop_height, (column + 1) * crop_width, (row + 1) * crop_height))
            for row in range(2)
            for column in range(4)
        ]

    output_dir = args.output_root / args.direction.strip().upper()
    output_dir.mkdir(parents=True, exist_ok=True)
    resampling = Image.Resampling.LANCZOS
    for index, frame in enumerate(frames):
        frame = frame.resize((args.frame_width, args.frame_height), resampling)
        matte_to_alpha(frame, args.matte_threshold).save(output_dir / f"{index:03d}.png")

    print(f"OK: wrote 8 RGBA frames to {output_dir} from {source_scale:.2f}x source cells")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

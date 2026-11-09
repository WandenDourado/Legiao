#!/usr/bin/env python3
"""Layer-B cosmetic salvage for sliced/normalized frames.

Fixes the small, pixel-level defects that otherwise trigger a full regeneration:
residual magenta fringe, faint semi-transparent halo, and a non-transparent 1px
frame border. It NEVER moves, scales, crops, or repaints the character body, so it
cannot change the animation content — only clean the matte. Run it before deciding
to regenerate: a set that only failed on these cosmetic criteria becomes acceptable
without another image-generation call.

Output: repaired PNGs plus a JSON report of pixels cleaned per frame. Exit code is
always 0 on success (repair is best-effort); re-run validate_frames.py afterwards to
confirm acceptance.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from PIL import Image

from frame_analysis import magenta_distance, nontransparent_border_pixels, visible_magenta_pixels
from slice_and_stitch import clear_near_transparent_pixels


def clear_border(image: Image.Image, border: int) -> Image.Image:
    """Force the outer `border` ring to full transparency without touching the body."""
    result = image.convert("RGBA").copy()
    pixels = result.load()
    width, height = result.size
    for y in range(height):
        for x in range(width):
            if x < border or y < border or x >= width - border or y >= height - border:
                pixels[x, y] = (0, 0, 0, 0)
    return result


def despill_edges(image: Image.Image, alpha_limit: int, matte_threshold: int) -> Image.Image:
    """Neutralize magenta on partially transparent edge pixels (green-channel repair)."""
    rgba = image.convert("RGBA")
    data = rgba.get_flattened_data() if hasattr(rgba, "get_flattened_data") else rgba.getdata()
    out = []
    for red, green, blue, alpha in data:
        if 0 < alpha < 255 and magenta_distance(red, green, blue) < matte_threshold:
            spill = max(0, min(red, blue) - green)
            green = min(255, green + spill)
        out.append((red, green, blue, alpha))
    result = Image.new("RGBA", rgba.size)
    result.putdata(out)
    return result


def unmix_matte_edge(image: Image.Image, pink_margin: int = 18, min_alpha_fraction: float = 0.12) -> tuple[Image.Image, int]:
    """Un-blend the magenta matte from semi-transparent edge pixels.

    An anti-aliased edge pixel is c = a*fg + (1-a)*matte. Keying sets the alpha but
    leaves c contaminated, which renders as a pink outline over any real background
    (green-boost despill only desaturates it). Solving fg = (c - (1-a)*M) / a with
    M = (255, 0, 255) recovers the true edge color. Pixels too transparent to unmix
    reliably are cleared. Returns (repaired image, pixels fixed)."""
    rgba = image.convert("RGBA")
    pixels = rgba.load()
    fixed = 0
    for y in range(rgba.height):
        for x in range(rgba.width):
            r, g, b, a = pixels[x, y]
            if a == 0 or a == 255:
                continue
            if not (r > g + pink_margin and b > g + pink_margin):
                continue
            alpha = a / 255.0
            if alpha <= min_alpha_fraction:
                pixels[x, y] = (0, 0, 0, 0)
            else:
                spill = (1 - alpha) * 255
                pixels[x, y] = (
                    max(0, min(255, round((r - spill) / alpha))),
                    max(0, min(255, round(g / alpha))),
                    max(0, min(255, round((b - spill) / alpha))),
                    a,
                )
            fixed += 1
    return rgba, fixed


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path)
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--directions", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames-per-direction", type=int, default=8)
    parser.add_argument("--border-pixels", type=int, default=1)
    parser.add_argument("--alpha-threshold", type=int, default=16)
    parser.add_argument("--matte-threshold", type=int, default=140)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()

    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    if not directions or args.border_pixels < 1:
        raise SystemExit("Provide directions and a border width of at least 1px.")

    report: dict[str, list[dict]] = {}
    for direction in directions:
        in_dir = args.input_root / direction
        out_dir = args.output_root / direction
        out_dir.mkdir(parents=True, exist_ok=True)
        paths = sorted(in_dir.glob("*.png"))
        if len(paths) != args.frames_per_direction:
            raise SystemExit(f"{in_dir}: expected {args.frames_per_direction} frames, got {len(paths)}")
        entries = []
        for index, path in enumerate(paths):
            with Image.open(path) as source:
                frame = source.convert("RGBA")
                if frame.size != (args.frame_width, args.frame_height):
                    raise SystemExit(f"{path}: expected {args.frame_width}x{args.frame_height}")
            before_matte = visible_magenta_pixels(frame, args.matte_threshold, args.alpha_threshold)
            before_border = nontransparent_border_pixels(frame, args.border_pixels, args.alpha_threshold)
            repaired = despill_edges(frame, args.alpha_threshold, args.matte_threshold)
            repaired, unmixed = unmix_matte_edge(repaired)
            repaired = clear_near_transparent_pixels(repaired)
            repaired = clear_border(repaired, args.border_pixels)
            after_matte = visible_magenta_pixels(repaired, args.matte_threshold, args.alpha_threshold)
            after_border = nontransparent_border_pixels(repaired, args.border_pixels, args.alpha_threshold)
            repaired.save(out_dir / f"{index:03d}.png")
            entries.append({
                "frame": index,
                "matte_before": before_matte, "matte_after": after_matte,
                "border_before": before_border, "border_after": after_border,
                "edge_unmixed": unmixed,
            })
        report[direction] = entries
        cleaned = sum(e["matte_before"] - e["matte_after"] + e["border_before"] - e["border_after"] for e in entries)
        unmixed_total = sum(e["edge_unmixed"] for e in entries)
        print(f"OK: {direction} repaired {args.frames_per_direction} frames, cleaned {cleaned} matte/border pixels, "
              f"unmixed {unmixed_total} pink edge pixels")

    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

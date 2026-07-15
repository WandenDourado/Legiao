#!/usr/bin/env python3
"""Validate sprite frame geometry, matte cleanup, and foot-baseline stability."""

from __future__ import annotations

import argparse
import glob
import struct
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:  # pragma: no cover - fallback is intentionally dependency-light.
    Image = None

if Image is not None:
    from frame_analysis import body_anchor, visible_magenta_pixels


PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"


def png_size(path: Path) -> tuple[int, int]:
    with path.open("rb") as handle:
        if handle.read(8) != PNG_SIGNATURE:
            raise ValueError("not a PNG file")
        header = handle.read(25)
    return struct.unpack(">II", header[8:16])


def expand_inputs(patterns: list[str]) -> list[Path]:
    paths: list[Path] = []
    for pattern in patterns:
        matches = glob.glob(pattern)
        if matches:
            paths.extend(Path(match) for match in matches)
        elif Path(pattern).exists():
            paths.append(Path(pattern))
    return sorted(set(paths))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("inputs", nargs="+", help="PNG paths or glob patterns.")
    parser.add_argument("--frame-width", type=int, required=True)
    parser.add_argument("--frame-height", type=int, required=True)
    parser.add_argument("--sheet", action="store_true", help="Treat each input as a sheet.")
    parser.add_argument("--columns", type=int, help="Expected sheet columns.")
    parser.add_argument("--rows", type=int, help="Expected sheet rows.")
    parser.add_argument("--require-alpha", action="store_true")
    parser.add_argument("--require-transparent", action="store_true")
    parser.add_argument("--reject-magenta", action="store_true")
    parser.add_argument("--magenta-threshold", type=int, default=96)
    parser.add_argument("--alpha-threshold", type=int, default=16)
    parser.add_argument("--check-baseline", action="store_true")
    parser.add_argument("--baseline-tolerance", type=int, default=2)
    parser.add_argument("--body-left", type=float, default=0.30)
    parser.add_argument("--body-right", type=float, default=0.70)
    args = parser.parse_args()

    needs_pillow = args.require_alpha or args.require_transparent or args.reject_magenta or args.check_baseline
    if Image is None and needs_pillow:
        print("Pillow is required for alpha, matte, or baseline validation.", file=sys.stderr)
        return 2
    if args.sheet and (not args.columns or not args.rows):
        print("--sheet requires --columns and --rows.", file=sys.stderr)
        return 2
    if args.sheet and args.check_baseline:
        print("--check-baseline accepts individual frames, not a sheet.", file=sys.stderr)
        return 2
    if not 0 < args.magenta_threshold <= 255 or not 0 < args.alpha_threshold <= 255:
        print("Alpha and magenta thresholds must be between 1 and 255.", file=sys.stderr)
        return 2
    if args.baseline_tolerance < 0 or not 0 <= args.body_left < args.body_right <= 1:
        print("Invalid baseline tolerance or body range.", file=sys.stderr)
        return 2

    paths = expand_inputs(args.inputs)
    if not paths:
        print("No PNG files matched.", file=sys.stderr)
        return 2
    expected_width = args.frame_width * args.columns if args.sheet else args.frame_width
    expected_height = args.frame_height * args.rows if args.sheet else args.frame_height
    failures: list[str] = []
    baselines: list[tuple[Path, int]] = []

    for path in paths:
        try:
            if Image is None:
                width, height = png_size(path)
                mode = None
            else:
                with Image.open(path) as source:
                    width, height, mode = source.width, source.height, source.mode
                    rgba = source.convert("RGBA")
                    has_transparency = rgba.getchannel("A").getextrema()[0] < 255
                    matte_pixels = visible_magenta_pixels(rgba, args.magenta_threshold, args.alpha_threshold)
                    baseline = body_anchor(rgba, args.body_left, args.body_right, args.alpha_threshold) if args.check_baseline else None
            if (width, height) != (expected_width, expected_height):
                failures.append(f"{path}: expected {expected_width}x{expected_height}, got {width}x{height}")
            if args.require_alpha and mode is not None and "A" not in mode:
                failures.append(f"{path}: expected alpha channel, got mode {mode}")
            if args.require_transparent and not has_transparency:
                failures.append(f"{path}: expected visible transparency")
            if args.reject_magenta and matte_pixels:
                failures.append(f"{path}: found {matte_pixels} visible magenta-matte pixels")
            if baseline is not None:
                baselines.append((path, baseline[1]))
        except Exception as exc:  # noqa: BLE001 - report every unreadable asset.
            failures.append(f"{path}: {exc}")

    if args.check_baseline and baselines:
        values = [value for _, value in baselines]
        spread = max(values) - min(values)
        if spread > args.baseline_tolerance:
            details = ", ".join(f"{path.name}={value}" for path, value in baselines)
            failures.append(f"baseline spread {spread}px exceeds {args.baseline_tolerance}px ({details})")

    if failures:
        print("Validation failed:")
        print("\n".join(f"- {failure}" for failure in failures))
        return 1

    checks = ["geometry"]
    if args.require_alpha:
        checks.append("alpha")
    if args.require_transparent:
        checks.append("transparency")
    if args.reject_magenta:
        checks.append("no magenta matte")
    if args.check_baseline:
        checks.append("baseline=" + ",".join(str(value) for _, value in baselines))
    print(f"OK: {len(paths)} file(s), expected {expected_width}x{expected_height}, {'; '.join(checks)}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

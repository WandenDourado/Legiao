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
    from frame_analysis import body_anchor, foreground_bbox, nontransparent_border_pixels, torso_center, visible_magenta_pixels


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
    parser.add_argument("--expected-baseline", type=int, help="Require each frame's foot baseline to match this y coordinate.")
    parser.add_argument("--check-center", action="store_true")
    parser.add_argument("--center-tolerance", type=int, default=2)
    parser.add_argument("--expected-center", type=int, help="Require each frame's torso center to match this x coordinate.")
    parser.add_argument("--body-left", type=float, default=0.30)
    parser.add_argument("--body-right", type=float, default=0.70)
    parser.add_argument("--torso-top", type=float, default=0.25)
    parser.add_argument("--torso-bottom", type=float, default=0.65)
    parser.add_argument("--require-clear-border", action="store_true")
    parser.add_argument("--border-pixels", type=int, default=1)
    parser.add_argument("--border-alpha-limit", type=int, default=4)
    parser.add_argument("--min-foreground-height-ratio", type=float)
    parser.add_argument("--max-foreground-height-ratio", type=float)
    args = parser.parse_args()

    needs_pillow = (
        args.require_alpha
        or args.require_transparent
        or args.reject_magenta
        or args.check_baseline
        or args.check_center
        or args.expected_baseline is not None
        or args.expected_center is not None
        or args.require_clear_border
        or args.min_foreground_height_ratio is not None
        or args.max_foreground_height_ratio is not None
    )
    if Image is None and needs_pillow:
        print("Pillow is required for alpha, matte, or baseline validation.", file=sys.stderr)
        return 2
    if args.sheet and (not args.columns or not args.rows):
        print("--sheet requires --columns and --rows.", file=sys.stderr)
        return 2
    if args.sheet and (
        args.check_baseline
        or args.check_center
        or args.expected_baseline is not None
        or args.expected_center is not None
        or args.min_foreground_height_ratio is not None
        or args.max_foreground_height_ratio is not None
    ):
        print("Anchor and foreground-occupancy checks accept individual frames, not a sheet.", file=sys.stderr)
        return 2
    if not 0 < args.magenta_threshold <= 255 or not 0 < args.alpha_threshold <= 255:
        print("Alpha and magenta thresholds must be between 1 and 255.", file=sys.stderr)
        return 2
    if (
        args.baseline_tolerance < 0
        or args.center_tolerance < 0
        or args.border_pixels < 1
        or not 0 <= args.border_alpha_limit <= 255
        or not 0 <= args.body_left < args.body_right <= 1
        or not 0 <= args.torso_top < args.torso_bottom <= 1
        or (args.min_foreground_height_ratio is not None and not 0 < args.min_foreground_height_ratio <= 1)
        or (args.max_foreground_height_ratio is not None and not 0 < args.max_foreground_height_ratio <= 1)
        or (
            args.min_foreground_height_ratio is not None
            and args.max_foreground_height_ratio is not None
            and args.min_foreground_height_ratio > args.max_foreground_height_ratio
        )
        or (args.expected_baseline is not None and not 0 <= args.expected_baseline < args.frame_height)
        or (args.expected_center is not None and not 0 <= args.expected_center < args.frame_width)
    ):
        print("Invalid tolerance, border, body/torso range, or foreground-height ratio.", file=sys.stderr)
        return 2

    paths = expand_inputs(args.inputs)
    if not paths:
        print("No PNG files matched.", file=sys.stderr)
        return 2
    expected_width = args.frame_width * args.columns if args.sheet else args.frame_width
    expected_height = args.frame_height * args.rows if args.sheet else args.frame_height
    failures: list[str] = []
    baselines: list[tuple[Path, int]] = []
    centers: list[tuple[Path, float]] = []

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
                    baseline = body_anchor(rgba, args.body_left, args.body_right, args.alpha_threshold) if (args.check_baseline or args.expected_baseline is not None) else None
                    center = torso_center(rgba, args.torso_top, args.torso_bottom, args.alpha_threshold) if (args.check_center or args.expected_center is not None) else None
                    foreground = foreground_bbox(rgba, args.alpha_threshold) if (
                        args.min_foreground_height_ratio is not None or args.max_foreground_height_ratio is not None
                    ) else None
                    border_pixels = nontransparent_border_pixels(rgba, args.border_pixels, args.border_alpha_limit) if args.require_clear_border else 0
            if (width, height) != (expected_width, expected_height):
                failures.append(f"{path}: expected {expected_width}x{expected_height}, got {width}x{height}")
            if args.require_alpha and mode is not None and "A" not in mode:
                failures.append(f"{path}: expected alpha channel, got mode {mode}")
            if args.require_transparent and not has_transparency:
                failures.append(f"{path}: expected visible transparency")
            if args.reject_magenta and matte_pixels:
                failures.append(f"{path}: found {matte_pixels} visible magenta-matte pixels")
            if args.require_clear_border and border_pixels:
                failures.append(f"{path}: found {border_pixels} non-transparent pixels in the {args.border_pixels}px frame border")
            if baseline is not None:
                baselines.append((path, baseline[1]))
            if center is not None:
                centers.append((path, center))
            if foreground is not None:
                height_ratio = (foreground[3] - foreground[1]) / height
                if args.min_foreground_height_ratio is not None and height_ratio < args.min_foreground_height_ratio:
                    failures.append(
                        f"{path}: foreground height {height_ratio:.3f} is below {args.min_foreground_height_ratio:.3f}"
                    )
                if args.max_foreground_height_ratio is not None and height_ratio > args.max_foreground_height_ratio:
                    failures.append(
                        f"{path}: foreground height {height_ratio:.3f} exceeds {args.max_foreground_height_ratio:.3f}"
                    )
        except Exception as exc:  # noqa: BLE001 - report every unreadable asset.
            failures.append(f"{path}: {exc}")

    if args.check_baseline and baselines:
        values = [value for _, value in baselines]
        spread = max(values) - min(values)
        if spread > args.baseline_tolerance:
            details = ", ".join(f"{path.name}={value}" for path, value in baselines)
            failures.append(f"baseline spread {spread}px exceeds {args.baseline_tolerance}px ({details})")

    if args.check_center and centers:
        values = [value for _, value in centers]
        spread = max(values) - min(values)
        if spread > args.center_tolerance:
            details = ", ".join(f"{path.name}={value:.1f}" for path, value in centers)
            failures.append(f"torso-center spread {spread:.1f}px exceeds {args.center_tolerance}px ({details})")

    if args.expected_baseline is not None:
        for path, value in baselines:
            if abs(value - args.expected_baseline) > args.baseline_tolerance:
                failures.append(
                    f"{path}: foot baseline {value}px differs from expected {args.expected_baseline}px by more than {args.baseline_tolerance}px"
                )
    if args.expected_center is not None:
        for path, value in centers:
            if abs(value - args.expected_center) > args.center_tolerance:
                failures.append(
                    f"{path}: torso center {value:.1f}px differs from expected {args.expected_center}px by more than {args.center_tolerance}px"
                )

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
    if args.check_center:
        checks.append("torso-center=" + ",".join(f"{value:.1f}" for _, value in centers))
    if args.expected_baseline is not None:
        checks.append(f"expected-baseline={args.expected_baseline}")
    if args.expected_center is not None:
        checks.append(f"expected-center={args.expected_center}")
    if args.require_clear_border:
        checks.append("clear frame border")
    print(f"OK: {len(paths)} file(s), expected {expected_width}x{expected_height}, {'; '.join(checks)}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

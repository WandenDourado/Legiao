#!/usr/bin/env python3
"""Validate sprite frame files or a sprite sheet grid."""

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


PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"


def png_size(path: Path) -> tuple[int, int]:
    with path.open("rb") as handle:
        signature = handle.read(8)
        if signature != PNG_SIGNATURE:
            raise ValueError("not a PNG file")
        header = handle.read(25)
    return struct.unpack(">II", header[8:16])


def image_info(path: Path) -> tuple[int, int, str | None]:
    if Image is None:
        width, height = png_size(path)
        return width, height, None
    with Image.open(path) as image:
        return image.width, image.height, image.mode


def expand_inputs(patterns: list[str]) -> list[Path]:
    paths: list[Path] = []
    for pattern in patterns:
        matches = glob.glob(pattern)
        if matches:
            paths.extend(Path(match) for match in matches)
        else:
            path = Path(pattern)
            if path.exists():
                paths.append(path)
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
    args = parser.parse_args()

    paths = expand_inputs(args.inputs)
    if not paths:
        print("No PNG files matched.", file=sys.stderr)
        return 2

    if args.sheet and (not args.columns or not args.rows):
        print("--sheet requires --columns and --rows.", file=sys.stderr)
        return 2

    expected_width = args.frame_width * args.columns if args.sheet else args.frame_width
    expected_height = args.frame_height * args.rows if args.sheet else args.frame_height
    failures: list[str] = []

    for path in paths:
        try:
            width, height, mode = image_info(path)
        except Exception as exc:  # noqa: BLE001 - report every unreadable asset.
            failures.append(f"{path}: {exc}")
            continue

        if (width, height) != (expected_width, expected_height):
            failures.append(f"{path}: expected {expected_width}x{expected_height}, got {width}x{height}")
        if args.require_alpha and mode is not None and "A" not in mode:
            failures.append(f"{path}: expected alpha channel, got mode {mode}")

    if failures:
        print("Validation failed:")
        for failure in failures:
            print(f"- {failure}")
        return 1

    alpha_note = "alpha checked" if Image is not None and args.require_alpha else "alpha not checked"
    print(f"OK: {len(paths)} file(s), expected {expected_width}x{expected_height}, {alpha_note}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

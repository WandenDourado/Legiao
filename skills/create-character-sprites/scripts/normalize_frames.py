#!/usr/bin/env python3
"""Align small foot-anchor drift without changing frame dimensions or clipping art."""

from __future__ import annotations

import argparse
import statistics
from pathlib import Path

from PIL import Image

from frame_analysis import body_anchor


def read_frames(folder: Path, frame_width: int, frame_height: int, expected_count: int) -> list[Image.Image]:
    paths = sorted(folder.glob("*.png"))
    if len(paths) != expected_count:
        raise ValueError(f"{folder}: expected {expected_count} PNG frames, got {len(paths)}")
    frames = []
    for path in paths:
        with Image.open(path) as image:
            if image.size != (frame_width, frame_height):
                raise ValueError(f"{path}: expected {frame_width}x{frame_height}, got {image.width}x{image.height}")
            frames.append(image.convert("RGBA"))
    return frames


def shifted(frame: Image.Image, dx: int, dy: int) -> Image.Image:
    bbox = frame.getchannel("A").getbbox()
    if bbox and (bbox[0] + dx < 0 or bbox[1] + dy < 0 or bbox[2] + dx > frame.width or bbox[3] + dy > frame.height):
        raise ValueError("alignment would clip visible art; regenerate this direction with more padding")
    result = Image.new("RGBA", frame.size)
    result.alpha_composite(frame, (dx, dy))
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path)
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--directions", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames-per-direction", type=int, default=8)
    parser.add_argument("--max-shift", type=int, default=4)
    parser.add_argument("--body-left", type=float, default=0.30)
    parser.add_argument("--body-right", type=float, default=0.70)
    parser.add_argument("--alpha-threshold", type=int, default=16)
    args = parser.parse_args()

    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    if not directions or args.max_shift < 0 or not 0 <= args.body_left < args.body_right <= 1:
        raise SystemExit("Provide directions, a non-negative max shift, and valid body bounds.")

    all_frames: dict[str, list[Image.Image]] = {}
    all_anchors: dict[str, list[tuple[float, int]]] = {}
    for direction in directions:
        frames = read_frames(args.input_root / direction, args.frame_width, args.frame_height, args.frames_per_direction)
        all_frames[direction] = frames
        all_anchors[direction] = [body_anchor(frame, args.body_left, args.body_right, args.alpha_threshold) for frame in frames]

    shifts: dict[str, list[tuple[int, int]]] = {}
    for direction, anchors in all_anchors.items():
        target_x = round(statistics.median(anchor[0] for anchor in anchors))
        target_y = round(statistics.median(anchor[1] for anchor in anchors))
        shifts[direction] = [(target_x - round(x), target_y - y) for x, y in anchors]
        if any(max(abs(dx), abs(dy)) > args.max_shift for dx, dy in shifts[direction]):
            raise SystemExit(f"{direction}: anchor drift exceeds {args.max_shift}px; regenerate instead of forcing alignment.")
        for frame, (dx, dy) in zip(all_frames[direction], shifts[direction], strict=True):
            shifted(frame, dx, dy)

    for direction, frames in all_frames.items():
        output = args.output_root / direction
        output.mkdir(parents=True, exist_ok=True)
        for index, (frame, (dx, dy)) in enumerate(zip(frames, shifts[direction], strict=True)):
            shifted(frame, dx, dy).save(output / f"{index:03d}.png")
        print(f"OK: {direction} anchors normalized with shifts {shifts[direction]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

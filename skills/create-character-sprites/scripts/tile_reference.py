#!/usr/bin/env python3
"""Tile a single-view reference crop into the grid layout the generator must produce.

Why: conditioning a 1x4 batch generation on a SINGLE vertical figure (e.g. a
model-sheet back-view crop) pulls the output toward the reference's composition —
the generator returns one vertical figure instead of the 4-figure grid, wasting the
generation. The reference must therefore look like the desired layout: the same
figure tiled N times on the magenta matte with clear gaps. Facing/identity signal
is preserved; layout pull now works FOR the grid instead of against it.

Usage:
  python tile_reference.py --input model_N_back.png --output model_N_back_4up.png \
      [--count 4] [--gap-fraction 0.35] [--matte FF00FF]
"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--count", type=int, default=4)
    parser.add_argument("--gap-fraction", type=float, default=0.35,
                        help="Horizontal gap between figures as a fraction of figure width.")
    parser.add_argument("--margin-fraction", type=float, default=0.12,
                        help="Vertical margin above/below as a fraction of figure height.")
    parser.add_argument("--matte", default="FF00FF")
    args = parser.parse_args()

    figure = Image.open(args.input).convert("RGBA")
    matte = tuple(int(args.matte[i:i + 2], 16) for i in (0, 2, 4)) + (255,)
    gap = round(figure.width * args.gap_fraction)
    margin = round(figure.height * args.margin_fraction)
    width = args.count * figure.width + (args.count + 1) * gap
    height = figure.height + 2 * margin
    canvas = Image.new("RGBA", (width, height), matte)
    for index in range(args.count):
        x = gap + index * (figure.width + gap)
        canvas.alpha_composite(figure, (x, margin))
    args.output.parent.mkdir(parents=True, exist_ok=True)
    canvas.convert("RGB").save(args.output)
    print(f"OK: {args.count}-up reference {width}x{height} written to {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

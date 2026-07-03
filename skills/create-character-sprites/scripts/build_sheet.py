#!/usr/bin/env python3
"""Build a directional sprite sheet and JSON metadata from frame folders."""

from __future__ import annotations

import argparse
import json
from pathlib import Path


try:
    from PIL import Image
except ImportError as exc:  # pragma: no cover
    raise SystemExit("Pillow is required. Install with: python -m pip install pillow") from exc


def direction_files(root: Path, direction: str, frames_per_direction: int) -> list[Path]:
    files = sorted((root / direction).glob("*.png"))
    if len(files) != frames_per_direction:
        raise ValueError(
            f"{direction}: expected {frames_per_direction} PNG files in {root / direction}, got {len(files)}"
        )
    return files


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, help="Folder containing direction subfolders.")
    parser.add_argument("--output", required=True, help="Output sprite sheet PNG.")
    parser.add_argument("--metadata-output", required=True, help="Output metadata JSON.")
    parser.add_argument("--directions", default="S,N,E,W,SE,SW,NE,NW")
    parser.add_argument("--animation", default="walk")
    parser.add_argument("--frame-width", type=int, required=True)
    parser.add_argument("--frame-height", type=int, required=True)
    parser.add_argument("--frames-per-direction", type=int, required=True)
    parser.add_argument("--frame-time", type=float, default=0.12)
    parser.add_argument("--origin", default="foot-center")
    args = parser.parse_args()

    input_root = Path(args.input_root)
    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    output = Path(args.output)
    metadata_output = Path(args.metadata_output)

    sheet = Image.new(
        "RGBA",
        (args.frame_width * args.frames_per_direction, args.frame_height * len(directions)),
        (0, 0, 0, 0),
    )

    rows: dict[str, int] = {}
    for row, direction in enumerate(directions):
        rows[direction] = row
        for column, frame_path in enumerate(direction_files(input_root, direction, args.frames_per_direction)):
            with Image.open(frame_path) as frame:
                frame = frame.convert("RGBA")
                if frame.size != (args.frame_width, args.frame_height):
                    raise ValueError(
                        f"{frame_path}: expected {args.frame_width}x{args.frame_height}, got {frame.size}"
                    )
                sheet.alpha_composite(frame, (column * args.frame_width, row * args.frame_height))

    output.parent.mkdir(parents=True, exist_ok=True)
    metadata_output.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(output)

    metadata = {
        "image": output.name,
        "frame_width": args.frame_width,
        "frame_height": args.frame_height,
        "origin": args.origin,
        "directions": directions,
        "animations": {
            args.animation: {
                "frames_per_direction": args.frames_per_direction,
                "frame_time_seconds": args.frame_time,
                "rows": rows,
            }
        },
    }
    metadata_output.write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    print(f"OK: wrote {output} and {metadata_output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

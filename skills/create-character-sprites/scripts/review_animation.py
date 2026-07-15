#!/usr/bin/env python3
"""Create GIF and contact-sheet animation reviews with a body-baseline overlay."""

from __future__ import annotations

import argparse
import json
import statistics
from pathlib import Path

from PIL import Image, ImageDraw

from frame_analysis import body_anchor


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input_dir", type=Path)
    parser.add_argument("--gif", required=True, type=Path)
    parser.add_argument("--contact-sheet", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--frames", type=int, default=8)
    parser.add_argument("--frame-time-ms", type=int, default=120)
    parser.add_argument("--body-left", type=float, default=0.30)
    parser.add_argument("--body-right", type=float, default=0.70)
    parser.add_argument("--alpha-threshold", type=int, default=16)
    args = parser.parse_args()

    paths = sorted(args.input_dir.glob("*.png"))
    if len(paths) != args.frames:
        raise SystemExit(f"Expected {args.frames} frames in {args.input_dir}, got {len(paths)}")
    frames = []
    for path in paths:
        with Image.open(path) as image:
            if image.size != (args.frame_width, args.frame_height):
                raise SystemExit(f"{path}: expected {args.frame_width}x{args.frame_height}")
            frames.append(image.convert("RGBA"))
    anchors = [body_anchor(frame, args.body_left, args.body_right, args.alpha_threshold) for frame in frames]
    baselines = [anchor[1] for anchor in anchors]
    baseline = round(statistics.median(baselines))

    args.gif.parent.mkdir(parents=True, exist_ok=True)
    frames[0].save(args.gif, save_all=True, append_images=frames[1:], duration=args.frame_time_ms, loop=0, disposal=2)
    contact = Image.new("RGBA", (args.frame_width * len(frames), args.frame_height))
    for index, frame in enumerate(frames):
        contact.alpha_composite(frame, (index * args.frame_width, 0))
    ImageDraw.Draw(contact).line((0, baseline, contact.width - 1, baseline), fill=(255, 64, 64, 220), width=1)
    args.contact_sheet.parent.mkdir(parents=True, exist_ok=True)
    contact.save(args.contact_sheet)

    report = {
        "frames": [path.name for path in paths],
        "anchors": [{"x": x, "baseline": y} for x, y in anchors],
        "baseline_spread": max(baselines) - min(baselines),
        "baseline_reference": baseline,
    }
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"OK: wrote {args.gif}, {args.contact_sheet}, and {args.report}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

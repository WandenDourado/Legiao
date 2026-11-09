#!/usr/bin/env python3
"""Screen every direction for WRONG ABSOLUTE FACING (front where back belongs, etc.).

Why this exists: all other gates measure geometry (scale, anchors, continuity), and
structural orientation is only RELATIVE within one direction. A whole direction
generated with the wrong facing — N/NW rendered front-facing — passes every
deterministic gate (it shipped a broken sheet once). This is the missing screen.

Method: content correlation against the character's OWN approved five-view model
sheet. Each model view (S, SW, W, N, NW) yields a normalized upper-body grayscale
template (face/hair/torso detail carries the facing signal). Every frame of every
direction is matched (NCC, mirror-tolerant) against all five templates; a
direction is SUSPECT when some view from a DIFFERENT facing group (front-ish
{S,SW} / side {W} / back-ish {N,NW}) beats the expected group's best score by more
than --margin. Self-calibrated per character; no color heuristics (skin/hair tone
tricks fail — warm hair reads as skin).

Verdicts: "pass" or "suspect" (mandatory single vision look; a confirmed wrong
facing means regenerating that direction conditioned on the matching model-sheet
view crop). NEVER a hard fail: pixels can cheaply doubt a facing, not prove it.
Exit 0 = all pass, 3 = at least one suspect.

This is a SCREEN, not a substitute for the mandatory per-direction semantic review
(facing + walk poses + identity) in the SKILL playbook.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from PIL import Image

from frame_analysis import _ncc, content_patch, foreground_bbox
from slice_and_stitch import matte_to_alpha

GROUPS = {"S": "front", "SW": "front", "SE": "front",
          "W": "side", "E": "side",
          "N": "back", "NW": "back", "NE": "back"}
PATCH_SIZE = (48, 64)


def model_templates(sheet_path: Path, order: list[str], alpha_limit: int,
                    matte_threshold: int, feather_threshold: int, body_fraction: float):
    """Split the five-view model sheet into per-view upper-body templates."""
    keyed = matte_to_alpha(Image.open(sheet_path).convert("RGBA"), matte_threshold, feather_threshold)
    alpha = keyed.getchannel("A")
    width, height = keyed.size
    columns = [False] * width
    for x in range(width):
        for y in range(0, height, 4):
            if alpha.getpixel((x, y)) >= alpha_limit:
                columns[x] = True
                break
    runs, start = [], None
    for x, occupied in enumerate(columns + [False]):
        if occupied and start is None:
            start = x
        elif not occupied and start is not None:
            if x - start > width // 40:
                runs.append((start, x))
            start = None
    # Touching figures (cape edges meeting) merge two views into one run: split the
    # widest runs at their occupancy valley until the count matches.
    occupancy = [0] * width
    for x in range(width):
        occupancy[x] = sum(1 for y in range(0, height, 4) if alpha.getpixel((x, y)) >= alpha_limit)
    while len(runs) < len(order):
        widest = max(range(len(runs)), key=lambda i: runs[i][1] - runs[i][0])
        x0, x1 = runs.pop(widest)
        lo = x0 + (x1 - x0) // 5
        hi = x1 - (x1 - x0) // 5
        valley = min(range(lo, hi), key=lambda x: occupancy[x])
        runs[widest:widest] = [(x0, valley), (valley, x1)]
    if len(runs) != len(order):
        raise SystemExit(f"{sheet_path}: found {len(runs)} figures, expected {len(order)} ({','.join(order)})")
    templates = {}
    for view, (x0, x1) in zip(order, runs):
        cell = keyed.crop((x0, 0, x1, height))
        cell = cell.crop(foreground_bbox(cell, alpha_limit))
        templates[view] = content_patch(cell, PATCH_SIZE, alpha_limit, body_fraction)
    return templates


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--frames-root", required=True, type=Path, help="Folder containing one subfolder per direction.")
    parser.add_argument("--model-sheet", required=True, type=Path, help="Approved five-view model sheet (magenta matte).")
    parser.add_argument("--model-order", default="S,SW,W,N,NW", help="Left-to-right view order in the model sheet.")
    parser.add_argument("--directions", default="S,SW,W,N,NW")
    parser.add_argument("--alpha-threshold", type=int, default=16, help="Use 40 for supersampled (2x) frames.")
    parser.add_argument("--matte-threshold", type=int, default=120)
    parser.add_argument("--matte-feather-threshold", type=int, default=190)
    parser.add_argument("--body-fraction", type=float, default=0.5)
    parser.add_argument("--margin", type=float, default=0.05,
                        help="How much a wrong-group view must beat the expected group to raise suspicion.")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()

    order = [item.strip().upper() for item in args.model_order.split(",") if item.strip()]
    directions = [item.strip().upper() for item in args.directions.split(",") if item.strip()]
    templates = model_templates(args.model_sheet, order, args.alpha_threshold,
                                args.matte_threshold, args.matte_feather_threshold, args.body_fraction)

    result = {"model_sheet": str(args.model_sheet), "directions": {}, "verdict": "pass"}
    for direction in directions:
        folder = args.frames_root / direction
        paths = sorted(folder.glob("*.png"))
        if not paths:
            print(f"{folder}: no frames found")
            return 1
        totals = {view: 0.0 for view in order}
        for path in paths:
            with Image.open(path) as image:
                patch = content_patch(image.convert("RGBA"), PATCH_SIZE, args.alpha_threshold, args.body_fraction)
            mirrored = patch.transpose(Image.Transpose.FLIP_LEFT_RIGHT)
            for view, template in templates.items():
                totals[view] += max(_ncc(patch, template), _ncc(mirrored, template))
        scores = {view: round(total / len(paths), 3) for view, total in totals.items()}
        expected_group = GROUPS.get(direction, "front")
        best_own = max(score for view, score in scores.items() if GROUPS.get(view) == expected_group)
        offenders = {view: score for view, score in scores.items()
                     if GROUPS.get(view) != expected_group and score > best_own + args.margin}
        status = "suspect" if offenders else "pass"
        if offenders:
            result["verdict"] = "suspect"
        result["directions"][direction] = {"expected_group": expected_group, "scores": scores,
                                           "best_own_group": round(best_own, 3),
                                           "offenders": offenders, "status": status}

    text = json.dumps(result, indent=2)
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(text + "\n", encoding="utf-8")
    print(text)
    if result["verdict"] == "suspect":
        print("SUSPECT facing above: confirm each flagged direction with ONE vision look; a confirmed "
              "wrong facing means regenerating that direction conditioned on the matching model-sheet view crop.")
        return 3
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

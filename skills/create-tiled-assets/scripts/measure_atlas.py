#!/usr/bin/env python3
"""Measure real alpha-bounds rectangles of connected regions in an atlas PNG.

Outputs, per detected piece: exact source rect, occupied 128px cells, and a
suggested bottom-center anchor. These measured numbers feed the manifest;
never trust the generation prompt's intended layout.

Usage:
  python measure_atlas.py assets/tilesets/village_vegetation.png --grid 128
  python measure_atlas.py atlas.png --grid 128 --json out.json
  python measure_atlas.py atlas.png --alpha-threshold 16 --min-area 64
"""
import argparse
import json
import sys

from PIL import Image


def find_regions(alpha, w, h, threshold, min_area):
    """Connected-component scan (4-connectivity, iterative flood fill)."""
    visited = bytearray(w * h)
    regions = []
    px = alpha
    for start in range(w * h):
        if visited[start] or px[start] <= threshold:
            continue
        stack = [start]
        visited[start] = 1
        minx = maxx = start % w
        miny = maxy = start // w
        area = 0
        while stack:
            i = stack.pop()
            area += 1
            x, y = i % w, i // w
            if x < minx: minx = x
            if x > maxx: maxx = x
            if y < miny: miny = y
            if y > maxy: maxy = y
            for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
                if 0 <= nx < w and 0 <= ny < h:
                    j = ny * w + nx
                    if not visited[j] and px[j] > threshold:
                        visited[j] = 1
                        stack.append(j)
        if area >= min_area:
            regions.append({"x": minx, "y": miny,
                            "width": maxx - minx + 1, "height": maxy - miny + 1,
                            "area": area})
    return regions


def merge_close(regions, gap):
    """Merge regions whose bounding boxes are within `gap` px (e.g. flowers
    drawn as several petals). Repeats until stable."""
    changed = True
    while changed:
        changed = False
        out = []
        while regions:
            r = regions.pop()
            merged = False
            for o in out:
                if (r["x"] - gap <= o["x"] + o["width"] and o["x"] - gap <= r["x"] + r["width"]
                        and r["y"] - gap <= o["y"] + o["height"] and o["y"] - gap <= r["y"] + r["height"]):
                    nx, ny = min(o["x"], r["x"]), min(o["y"], r["y"])
                    o["width"] = max(o["x"] + o["width"], r["x"] + r["width"]) - nx
                    o["height"] = max(o["y"] + o["height"], r["y"] + r["height"]) - ny
                    o["x"], o["y"] = nx, ny
                    o["area"] += r["area"]
                    merged = changed = True
                    break
            if not merged:
                out.append(r)
        regions = out
    return regions


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("atlas")
    ap.add_argument("--grid", type=int, default=128, help="base cell size (default 128)")
    ap.add_argument("--alpha-threshold", type=int, default=8)
    ap.add_argument("--min-area", type=int, default=64, help="ignore specks below this pixel count")
    ap.add_argument("--merge-gap", type=int, default=12, help="merge regions closer than this (px)")
    ap.add_argument("--json", help="also write results to this JSON file")
    args = ap.parse_args()

    im = Image.open(args.atlas).convert("RGBA")
    w, h = im.size
    alpha = list(im.getchannel("A").getdata())

    regions = find_regions(alpha, w, h, args.alpha_threshold, args.min_area)
    regions = merge_close(regions, args.merge_gap)
    regions.sort(key=lambda r: (r["y"] // args.grid, r["x"]))

    pieces = []
    for i, r in enumerate(regions):
        cells = {"col0": r["x"] // args.grid, "row0": r["y"] // args.grid,
                 "col1": (r["x"] + r["width"] - 1) // args.grid,
                 "row1": (r["y"] + r["height"] - 1) // args.grid}
        pieces.append({
            "id": f"piece_{i:02d}",
            "source": {"x": r["x"], "y": r["y"], "width": r["width"], "height": r["height"]},
            "cells": cells,
            "gridAligned": (r["width"] <= args.grid and r["height"] <= args.grid
                            and cells["col0"] == cells["col1"] and cells["row0"] == cells["row1"]),
            "suggestedAnchor": {"x": r["width"] // 2, "y": max(0, r["height"] - 8)},
            "opaqueArea": r["area"],
        })

    report = {"atlas": args.atlas, "size": {"width": w, "height": h},
              "grid": args.grid, "pieceCount": len(pieces), "pieces": pieces}
    print(json.dumps(report, indent=2))
    if args.json:
        with open(args.json, "w", encoding="utf-8") as f:
            json.dump(report, f, indent=2)
    if not pieces:
        print("WARNING: no opaque regions found", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

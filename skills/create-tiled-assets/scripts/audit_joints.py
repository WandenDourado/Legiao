#!/usr/bin/env python3
"""Audit placed runs of connecting pieces (fence, wall, path) for gaps.

Butt-joining pieces by width leaves 5-7 px holes at every joint, because a
piece's bounding box starts before its rail does. This script renders the run
exactly as the game would and scans it for holes, so joints are checked by
measurement instead of by eye.

Pieces sharing an anchor axis and standing next to each other are grouped into
a chain; the script finds the chain's own rail/band line (the scan line with
the most opaque pixels) and reports every hole in it. Holes at least
--min-opening wide are reported as deliberate openings (a gate) instead of
defects — check those against the player footprint.

Usage:
  python audit_joints.py assets/maps/world_01.json --manifest assets/fences_manifest.json --layer fences
  python audit_joints.py map.json --manifest m.json --layer fences --max-gap 2 --min-opening 128
"""
import argparse
import sys
from collections import defaultdict

from scene_utils import collect_objects, load_manifests, render, runs_of


def chains(group, key_lo, key_hi, max_gap):
    """Split pieces sharing an axis into contiguous chains."""
    group = sorted(group, key=key_lo)
    out, current = [], []
    for obj in group:
        if current and key_lo(obj) - key_hi(current[-1]) > max_gap:
            out.append(current); current = []
        current.append(obj)
    if current:
        out.append(current)
    return [c for c in out if len(c) > 1]


def scan_line(alpha, x_range, y_range, horizontal):
    """Row (or column) with the most opaque pixels — the run's own rail line."""
    best, best_count = None, -1
    outer = y_range if horizontal else x_range
    inner = x_range if horizontal else y_range
    for o in outer:
        count = sum(1 for i in inner if (alpha[i, o] if horizontal else alpha[o, i]) > 40)
        if count > best_count:
            best, best_count = o, count
    return best


def report(label, holes, offset, max_gap, min_opening):
    problems = 0
    printed = False
    for start, end in holes:
        width = end - start + 1
        if width <= max_gap:
            continue
        printed = True
        if width >= min_opening:
            print(f"  ABERTURA: {width} px em {offset + start}..{offset + end}")
        else:
            problems += 1
            print(f"  FALHA: {width} px em {offset + start}..{offset + end}")
    if not printed:
        print("  contínuo")
    return problems


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", action="append", required=True)
    ap.add_argument("--layer", action="append", required=True)
    ap.add_argument("--max-gap", type=int, default=3, help="tolerated hole width in px (default 3)")
    ap.add_argument("--min-opening", type=int, default=128,
                    help="hole at least this wide counts as a deliberate opening (default: one cell)")
    ap.add_argument("--chain-gap", type=int, default=64,
                    help="max distance between two pieces of the same run (default 64)")
    args = ap.parse_args()

    objects, _ = collect_objects(args.map, args.layer)
    pieces, atlases = load_manifests(args.manifest)
    objects = [o for o in objects if o["name"] in pieces]
    canvas, x0, y0 = render(objects, pieces, atlases, margin=100)
    alpha = canvas.getchannel("A").load()

    def box(obj):
        p = pieces[obj["name"]]
        left = obj["x"] - p["anchor"]["x"] - x0
        top = obj["y"] - p["anchor"]["y"] - y0
        return int(left), int(top), int(left + p["source"]["width"]), int(top + p["source"]["height"])

    def vertical_axis(obj):
        """World x of the piece's own vertical band (its densest column)."""
        left, top, right, bottom = box(obj)
        col = scan_line(alpha, range(left, right), range(top, bottom), horizontal=False)
        return col + x0

    problems = 0
    rows, cols = defaultdict(list), defaultdict(list)
    for obj in objects:
        rows[obj["y"]].append(obj)
        # Runs are grouped by the band they share, not by the object x: a corner
        # and the straight below it have different anchors but the same band.
        cols[round(vertical_axis(obj) / 16)].append(obj)

    for world_y, group in sorted(rows.items()):
        for chain in chains(group, lambda o: box(o)[0], lambda o: box(o)[2], args.chain_gap):
            left = min(box(o)[0] for o in chain)
            right = max(box(o)[2] for o in chain)
            top = min(box(o)[1] for o in chain)
            bottom = max(box(o)[3] for o in chain)
            row = scan_line(alpha, range(left, right), range(top, bottom), horizontal=True)
            solid = [alpha[x, row] > 40 for x in range(left, right)]
            holes = [h for h in runs_of([not s for s in solid]) if h[0] > 0 and h[1] < len(solid) - 1]
            print(f"run horizontal y={world_y}: {len(chain)} peças, x {left + x0}..{right + x0}")
            problems += report("h", holes, left + x0, args.max_gap, args.min_opening)

    for world_x, group in sorted(cols.items()):
        for chain in chains(group, lambda o: box(o)[1], lambda o: box(o)[3], args.chain_gap):
            left = min(box(o)[0] for o in chain)
            right = max(box(o)[2] for o in chain)
            top = min(box(o)[1] for o in chain)
            bottom = max(box(o)[3] for o in chain)
            column = scan_line(alpha, range(left, right), range(top, bottom), horizontal=False)
            solid = [alpha[column, y] > 40 for y in range(top, bottom)]
            holes = [h for h in runs_of([not s for s in solid]) if h[0] > 0 and h[1] < len(solid) - 1]
            print(f"run vertical x~{world_x * 16}: {len(chain)} peças, y {top + y0}..{bottom + y0}")
            problems += report("v", holes, top + y0, args.max_gap, args.min_opening)

    if problems:
        print(f"\n{problems} falha(s) de emenda — reposicione pelas conexões medidas, não pela largura",
              file=sys.stderr)
        return 1
    print("\nOK: nenhuma falha de emenda")
    return 0


if __name__ == "__main__":
    sys.exit(main())

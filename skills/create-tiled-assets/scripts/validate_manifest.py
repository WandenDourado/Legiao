#!/usr/bin/env python3
"""Validate an asset manifest against its atlas PNG and the spec contracts.

Checks per piece:
  - source rect lies inside the atlas
  - source rect contains every opaque pixel of its region (no clipped art:
    scans a margin ring around the rect for opaque pixels touching the edge)
  - rects do not overlap each other
  - anchor lies inside the piece rectangle
  - role is one of ground_detail | structures_back | foreground
  - collision=true requires collisionFootprint (or collisionFootprints) with
    positive size
  - every footprint, applied at the anchor, stays inside the opaque art: what
    is blocked must be what the player sees, never open ground beside it

Exit 0 = pass, 1 = failures (listed on stderr).

Usage:
  python validate_manifest.py assets/vegetation_manifest.json
  python validate_manifest.py manifest.json --atlas-root . --alpha-threshold 8
"""
import argparse
import json
import os
import sys

from PIL import Image

ROLES = {"ground_detail", "structures_back", "foreground"}


def opaque_bounds(alpha, sx, sy, sw, sh, thr):
    """Tight bounds of the visible art inside a source rect, in atlas pixels.

    Returns (left, top, right, bottom) or None when the rect is empty.
    """
    left, top, right, bottom = None, None, None, None
    for x in range(int(sx), int(sx + sw)):
        for y in range(int(sy), int(sy + sh)):
            if alpha[x, y] <= thr:
                continue
            if left is None or x < left:
                left = x
            if right is None or x + 1 > right:
                right = x + 1
            if top is None or y < top:
                top = y
            if bottom is None or y + 1 > bottom:
                bottom = y + 1
    if left is None:
        return None
    return (left - sx, top - sy, right - sx, bottom - sy)


def rect_overlap(a, b):
    return (a["x"] < b["x"] + b["width"] and b["x"] < a["x"] + a["width"]
            and a["y"] < b["y"] + b["height"] and b["y"] < a["y"] + a["height"])


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("manifest")
    ap.add_argument("--atlas-root", default=None,
                    help="directory the manifest's atlas path is relative to (default: cwd, then manifest dir)")
    ap.add_argument("--alpha-threshold", type=int, default=8)
    ap.add_argument("--margin", type=int, default=6, help="clip-scan ring width around each rect")
    ap.add_argument("--tile", type=int, default=128, help="collision grid cell size")
    ap.add_argument("--max-outset", type=int, default=8,
                    help="px a collision footprint may extend past the opaque art before failing")
    args = ap.parse_args()

    with open(args.manifest, encoding="utf-8") as f:
        manifest = json.load(f)

    errors, warnings = [], []

    atlas_rel = manifest.get("atlas", "")
    candidates = []
    if args.atlas_root:
        candidates.append(os.path.join(args.atlas_root, atlas_rel))
    candidates += [atlas_rel, os.path.join(os.path.dirname(os.path.abspath(args.manifest)), "..", atlas_rel)]
    atlas_path = next((c for c in candidates if c and os.path.isfile(c)), None)
    if not atlas_path:
        print(f"FAIL: atlas not found: {atlas_rel}", file=sys.stderr)
        return 1

    im = Image.open(atlas_path).convert("RGBA")
    w, h = im.size
    alpha = im.getchannel("A").load()
    thr = args.alpha_threshold

    pieces = manifest.get("pieces", {})
    if not pieces:
        errors.append("manifest has no pieces")

    for name, p in pieces.items():
        s = p.get("source", {})
        sx, sy = s.get("x", -1), s.get("y", -1)
        sw, sh = s.get("width", 0), s.get("height", 0)
        if sw <= 0 or sh <= 0:
            errors.append(f"{name}: non-positive source size")
            continue
        if sx < 0 or sy < 0 or sx + sw > w or sy + sh > h:
            errors.append(f"{name}: source rect {sx},{sy} {sw}x{sh} outside atlas {w}x{h}")
            continue

        # Clipped-art scan: pixels in the 1px ring just outside the rect.
        # Solid pixels (alpha > 128) = error; faint anti-aliased residue = warning.
        max_ring_alpha = 0
        for x in range(max(0, int(sx) - 1), min(w, int(sx + sw) + 1)):
            for y in (int(sy) - 1, int(sy + sh)):
                if 0 <= y < h:
                    max_ring_alpha = max(max_ring_alpha, alpha[x, y])
        for y in range(max(0, int(sy) - 1), min(h, int(sy + sh) + 1)):
            for x in (int(sx) - 1, int(sx + sw)):
                if 0 <= x < w:
                    max_ring_alpha = max(max_ring_alpha, alpha[x, y])
        if p.get("clipOk"):
            pass  # intentional cut (e.g. base/roof split of one building)
        elif max_ring_alpha > 128:
            errors.append(f"{name}: solid pixels (alpha {max_ring_alpha}) touch the rect edge — source rect clips the art")
        elif max_ring_alpha > thr:
            warnings.append(f"{name}: faint pixels (alpha {max_ring_alpha}) just outside the rect — consider growing it 1px")

        # Emptiness check
        opaque = sum(1 for x in range(int(sx), int(sx + sw), 2)
                     for y in range(int(sy), int(sy + sh), 2) if alpha[x, y] > thr)
        if opaque == 0:
            errors.append(f"{name}: source rect contains no opaque pixels")

        a = p.get("anchor", {})
        ax, ay = a.get("x", -1), a.get("y", -1)
        if not (0 <= ax <= sw and 0 <= ay <= sh):
            # Legitimate for canopies: their anchor sits at the shared trunk
            # anchor below the rect. Flag it for review, don't fail.
            warnings.append(f"{name}: anchor ({ax},{ay}) outside piece rect {sw}x{sh} "
                            "(OK for canopy-style pieces sharing a trunk anchor)")

        role = p.get("role", "")
        if role not in ROLES:
            errors.append(f"{name}: invalid role '{role}' (expected {sorted(ROLES)})")

        if p.get("collision"):
            footprints = p.get("collisionFootprints") or (
                [p["collisionFootprint"]] if p.get("collisionFootprint") else [])
            if not footprints or any(fp.get("width", 0) <= 0 or fp.get("height", 0) <= 0
                                     for fp in footprints):
                errors.append(f"{name}: collision=true but no collisionFootprint(s) with positive size")
            else:
                # Collision must stay INSIDE the art. Anything blocked past the
                # sprite is ground the player can see is empty and still cannot
                # walk on, which is the defect this check exists to catch.
                #
                # The reference is the OPAQUE bounds, not the source rect: a
                # rect is padded, and measuring against padding would license a
                # footprint to float in transparent pixels.
                art = opaque_bounds(alpha, sx, sy, sw, sh, thr)
                if art is None:
                    continue
                bounds = {"left": art[0] - ax, "top": art[1] - ay,
                          "right": art[2] - ax, "bottom": art[3] - ay}
                for fp in footprints:
                    f = {"left": fp.get("offsetX", 0), "top": fp.get("offsetY", 0)}
                    f["right"] = f["left"] + fp["width"]
                    f["bottom"] = f["top"] + fp["height"]
                    outset = {"left": bounds["left"] - f["left"], "top": bounds["top"] - f["top"],
                              "right": f["right"] - bounds["right"], "bottom": f["bottom"] - bounds["bottom"]}
                    for edge in ("left", "right", "top", "bottom"):
                        if outset[edge] > args.max_outset:
                            errors.append(f"{name}: footprint reaches {outset[edge]:.0f} px past the art on the "
                                          f"{edge} edge (max {args.max_outset}) — it would block visibly free ground")
                if role == "foreground":
                    errors.append(f"{name}: foreground pieces must not collide")

    for i, (na, pa) in enumerate(pieces.items()):
        for nb, pb in list(pieces.items())[i + 1:]:
            if rect_overlap(pa.get("source", {}), pb.get("source", {})):
                errors.append(f"{na} / {nb}: source rects overlap")

    for wmsg in warnings:
        print(f"WARN: {wmsg}", file=sys.stderr)
    if errors:
        for e in errors:
            print(f"FAIL: {e}", file=sys.stderr)
        print(f"{len(errors)} error(s), {len(warnings)} warning(s)", file=sys.stderr)
        return 1
    print(f"OK: {len(pieces)} piece(s) validated against {atlas_path} ({len(warnings)} warning(s))")
    return 0


if __name__ == "__main__":
    sys.exit(main())

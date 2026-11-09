#!/usr/bin/env python3
"""Render manifest-driven map objects to a PNG so the assembly can be LOOKED at
before the user ever runs the game. Optionally overlays the collision layer.

Every fence/house defect in this project's history was visible in a render like
this one and invisible in the manifest numbers. Render, look, then integrate.

Usage:
  python render_scene.py assets/maps/world_01.json --manifest assets/fences_manifest.json \
      --layer fences --collision --out /tmp/fences.png --scale 0.5
  python render_scene.py map.json --manifest a.json --manifest b.json --out scene.png
"""
import argparse
import sys

from PIL import Image, ImageDraw

from scene_utils import collect_objects, load_manifests, render


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", action="append", required=True)
    ap.add_argument("--layer", action="append", help="objectgroup layer(s) to render (default: all)")
    ap.add_argument("--out", required=True)
    ap.add_argument("--collision", action="store_true",
                    help="overlay solid space: painted collision cells AND manifest footprints")
    ap.add_argument("--grass", action="store_true", help="draw a flat green background instead of transparency")
    ap.add_argument("--scale", type=float, default=1.0)
    ap.add_argument("--margin", type=int, default=200)
    args = ap.parse_args()

    objects, tmap = collect_objects(args.map, args.layer)
    pieces, atlases = load_manifests(args.manifest)
    bg = (101, 144, 50, 255) if args.grass else None
    canvas, x0, y0 = render(objects, pieces, atlases, margin=args.margin, background=bg)

    if args.collision:
        tile = tmap.get("tilewidth", 128)
        width = tmap.get("width", 0)
        layer = next((l for l in tmap["layers"] if l.get("name") == "collision"), None)
        if layer:
            overlay = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
            draw = ImageDraw.Draw(overlay)
            for i, gid in enumerate(layer["data"]):
                if not gid:
                    continue
                cx, cy = (i % width) * tile - x0, (i // width) * tile - y0
                if -tile <= cx <= canvas.width and -tile <= cy <= canvas.height:
                    draw.rectangle([cx, cy, cx + tile, cy + tile],
                                   fill=(255, 60, 60, 60), outline=(255, 60, 60, 150))
        else:
            overlay = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
            draw = ImageDraw.Draw(overlay)
        # Manifest footprints are the other half of solid space, and on a map
        # like world_01 they are ALL of it — the painted layer is empty there.
        # Drawing only the cells would show an obstacle-free village.
        for obj in objects:
            piece = pieces.get(obj.get("name"))
            if not piece or not piece.get("collision"):
                continue
            for fp in (piece.get("collisionFootprints")
                       or ([piece["collisionFootprint"]] if piece.get("collisionFootprint") else [])):
                fx, fy = obj["x"] + fp["offsetX"] - x0, obj["y"] + fp["offsetY"] - y0
                draw.rectangle([fx, fy, fx + fp["width"], fy + fp["height"]],
                               fill=(255, 60, 60, 60), outline=(255, 60, 60, 200))
        canvas = Image.alpha_composite(overlay, canvas) if bg is None else Image.alpha_composite(canvas, overlay)

    if args.scale != 1.0:
        canvas = canvas.resize((int(canvas.width * args.scale), int(canvas.height * args.scale)), Image.LANCZOS)
    canvas.save(args.out)
    print(f"OK: {len(objects)} object(s) -> {args.out} ({canvas.width}x{canvas.height}, origin {x0},{y0})")
    return 0


if __name__ == "__main__":
    sys.exit(main())

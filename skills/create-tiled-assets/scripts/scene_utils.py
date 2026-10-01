"""Shared helpers: load manifests, collect map objects, render a scene.

Used by render_scene.py and audit_joints.py so both see exactly what the game
renderer would draw: each piece blitted at (object position - anchor).
"""
import json
import os

from PIL import Image


def load_manifests(paths, root="."):
    """Return (pieces, atlases): name -> piece dict, atlas path -> Image."""
    pieces, atlases = {}, {}
    for path in paths:
        with open(path, encoding="utf-8") as f:
            data = json.load(f)
        atlas_rel = data["atlas"]
        candidates = [os.path.join(root, atlas_rel), atlas_rel,
                      os.path.join(os.path.dirname(os.path.abspath(path)), "..", atlas_rel)]
        atlas_path = next((c for c in candidates if os.path.isfile(c)), None)
        if not atlas_path:
            raise SystemExit(f"atlas not found for {path}: {atlas_rel}")
        if atlas_path not in atlases:
            atlases[atlas_path] = Image.open(atlas_path).convert("RGBA")
        for name, piece in data["pieces"].items():
            piece = dict(piece)
            piece["_atlas"] = atlas_path
            pieces[name] = piece
    return pieces, atlases


def collect_objects(map_path, layers=None):
    """Objects of the given objectgroup layers as dicts with name/x/y/layer."""
    with open(map_path, encoding="utf-8") as f:
        tmap = json.load(f)
    out = []
    for layer in tmap.get("layers", []):
        if layer.get("type") != "objectgroup":
            continue
        if layers and layer.get("name") not in layers:
            continue
        for obj in layer.get("objects", []):
            out.append({"name": obj.get("name", ""), "x": obj.get("x", 0),
                        "y": obj.get("y", 0), "layer": layer.get("name")})
    return out, tmap


def piece_image(piece, atlases):
    s = piece["source"]
    return atlases[piece["_atlas"]].crop((s["x"], s["y"], s["x"] + s["width"], s["y"] + s["height"]))


def render(objects, pieces, atlases, margin=200, background=None, roles=None):
    """Render objects to an RGBA image. Returns (image, origin_x, origin_y)."""
    boxes = []
    for obj in objects:
        piece = pieces.get(obj["name"])
        if not piece or (roles and piece.get("role") not in roles):
            continue
        s, a = piece["source"], piece["anchor"]
        left, top = obj["x"] - a["x"], obj["y"] - a["y"]
        boxes.append((left, top, left + s["width"], top + s["height"]))
    if not boxes:
        raise SystemExit("no objects resolved to manifest pieces")
    x0 = min(b[0] for b in boxes) - margin
    y0 = min(b[1] for b in boxes) - margin
    x1 = max(b[2] for b in boxes) + margin
    y1 = max(b[3] for b in boxes) + margin
    canvas = Image.new("RGBA", (int(x1 - x0), int(y1 - y0)), background or (0, 0, 0, 0))
    order = {"ground_detail": 0, "structures_back": 1, "foreground": 2}
    for obj in sorted(objects, key=lambda o: order.get((pieces.get(o["name"]) or {}).get("role"), 1)):
        piece = pieces.get(obj["name"])
        if not piece or (roles and piece.get("role") not in roles):
            continue
        a = piece["anchor"]
        canvas.alpha_composite(piece_image(piece, atlases),
                               (int(obj["x"] - a["x"] - x0), int(obj["y"] - a["y"] - y0)))
    return canvas, int(x0), int(y0)


def runs_of(flags):
    """Contiguous True ranges of a boolean list, as (start, end) inclusive."""
    out, start = [], None
    for i, flag in enumerate(flags):
        if flag and start is None:
            start = i
        elif not flag and start is not None:
            out.append((start, i - 1)); start = None
    if start is not None:
        out.append((start, len(flags) - 1))
    return out

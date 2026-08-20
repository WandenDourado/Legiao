#!/usr/bin/env python3
"""Validate a Tiled map's object layers against one or more asset manifests.

Checks:
  - every object in the given objectgroup layer(s) resolves to a manifest piece
  - manifest pieces that collide are not ALSO covered by painted cells in the
    `collision` tile layer (double-collision, reported as warnings). This is
    the defect the fences shipped with: every piece had its own footprint AND
    a hand-painted cell under it, so a ~48 px rail blocked a whole 128 px cell.
    It is a warning and not an error because the same signal has a legitimate
    reading — a prop standing inside a thicket painted impassable on purpose,
    which is most of world_02. Read the ratio: a run of pieces each losing
    most of its cells to paint is the fence defect; scattered pieces fully
    inside a painted mass is map design.
  - objects lie inside map bounds
  - reports manifest pieces never used by the map (informational)

Usage:
  python validate_map.py assets/maps/world_01.json --manifest assets/vegetation_manifest.json
  python validate_map.py map.json --manifest a.json --manifest b.json --layer vegetation --layer scenery
"""
import argparse
import json
import sys

# Objectgroups que NAO carregam arte de manifesto, e por isso nao entram na
# validacao por padrao. Cada uma existe por um motivo diferente:
#   spawn   - marcadores de posicao (jogador e hordas), sem desenho
#   trails  - polilinhas; a fita e desenhada ao longo da curva, nao e uma peca
#   portals - retangulo de gatilho; o portal e desenhado so com primitivas
# Cobrar peca de manifesto delas reprova mapa correto.
# Layers whose objects are never manifest art. `zones` and `collision` hold
# plain rectangles — territory bounds, climax gates, the creek banks of map 4 —
# so checking them against the manifest reported every one as a missing piece.
# That failed world_03 and world_05 too, not just the map this was added for.
NON_ART_LAYERS = {"spawn", "trails", "portals", "zones", "collision"}


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", action="append", required=True)
    ap.add_argument("--layer", action="append",
                    help="objectgroup layer name(s) to validate (default: all objectgroups except spawn)")
    args = ap.parse_args()

    with open(args.map, encoding="utf-8") as f:
        tmap = json.load(f)

    pieces = {}
    for mpath in args.manifest:
        with open(mpath, encoding="utf-8") as f:
            m = json.load(f)
        for name, p in m.get("pieces", {}).items():
            if name in pieces:
                print(f"WARN: piece '{name}' defined in multiple manifests", file=sys.stderr)
            pieces[name] = p

    map_w = tmap.get("width", 0) * tmap.get("tilewidth", 128)
    map_h = tmap.get("height", 0) * tmap.get("tileheight", 128)

    errors, warnings, used = [], [], set()
    layers = [l for l in tmap.get("layers", []) if l.get("type") == "objectgroup"]
    if args.layer:
        layers = [l for l in layers if l.get("name") in args.layer]
        found = {l.get("name") for l in layers}
        for want in args.layer:
            if want not in found:
                errors.append(f"objectgroup layer '{want}' not found in map")
    else:
        layers = [l for l in layers if l.get("name") not in NON_ART_LAYERS]

    for layer in layers:
        for obj in layer.get("objects", []):
            name = obj.get("name", "")
            if not name:
                errors.append(f"layer '{layer['name']}': unnamed object id={obj.get('id')}")
                continue
            if name not in pieces:
                errors.append(f"layer '{layer['name']}': object '{name}' (id={obj.get('id')}) has no manifest piece")
                continue
            used.add(name)
            x, y = obj.get("x", 0), obj.get("y", 0)
            if not (0 <= x <= map_w and 0 <= y <= map_h):
                errors.append(f"layer '{layer['name']}': '{name}' anchor ({x},{y}) outside map {map_w}x{map_h}")

    # Dupla colisao: footprint de manifesto sob celula pintada.
    tw = tmap.get("tilewidth", 128)
    th = tmap.get("tileheight", 128)
    cols = tmap.get("width", 0)
    painted = set()
    for layer in tmap.get("layers", []):
        if layer.get("name") == "collision" and layer.get("type") == "tilelayer":
            painted = {(i % cols, i // cols) for i, gid in enumerate(layer.get("data", [])) if gid}
    if painted:
        for layer in layers:
            for obj in layer.get("objects", []):
                piece = pieces.get(obj.get("name"))
                if not piece or not piece.get("collision"):
                    continue
                fps = piece.get("collisionFootprints") or (
                    [piece["collisionFootprint"]] if piece.get("collisionFootprint") else [])
                touched, covered = 0, 0
                for fp in fps:
                    x, y = obj.get("x", 0) + fp["offsetX"], obj.get("y", 0) + fp["offsetY"]
                    for row in range(int(y // th), int((y + fp["height"] - 0.01) // th) + 1):
                        for col in range(int(x // tw), int((x + fp["width"] - 0.01) // tw) + 1):
                            touched += 1
                            if (col, row) in painted:
                                covered += 1
                if not touched:
                    continue
                where = f"'{obj.get('name')}' at ({obj.get('x')},{obj.get('y')})"
                if covered == touched:
                    warnings.append(f"{where}: footprint fully buried under painted collision cells — "
                                    "either the piece is decoration inside a mass painted impassable "
                                    "on purpose (dense woods, world edge), or the cells are leftovers "
                                    "and the piece is blocking a whole cell instead of its own shape")
                elif covered:
                    warnings.append(f"{where}: {covered}/{touched} footprint cells are also painted — "
                                    "the piece blocks more than its art there")

    unused = sorted(set(pieces) - used)
    if unused:
        print(f"INFO: manifest pieces never placed: {', '.join(unused)}", file=sys.stderr)

    for w in warnings:
        print(f"WARN: {w}", file=sys.stderr)
    if errors:
        for e in errors:
            print(f"FAIL: {e}", file=sys.stderr)
        print(f"{len(errors)} error(s)", file=sys.stderr)
        return 1
    print(f"OK: {sum(len(l.get('objects', [])) for l in layers)} object(s) in {len(layers)} layer(s) resolve to manifest pieces")
    return 0


if __name__ == "__main__":
    sys.exit(main())

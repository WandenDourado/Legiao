#!/usr/bin/env python3
"""Renderiza o mapa inteiro (terreno + toppings + objetos) para conferencia.

O motor mistura as bordas do terreno num shader; aqui a borda sai dura, o que
basta para julgar LAYOUT: forma das regioes, caminho, densidade, vazios.

Uso:
  python render_map.py assets/maps/world_01.json \
      --manifest assets/vegetation_manifest.json --manifest assets/buildings_manifest.json \
      --manifest assets/fences_manifest.json --toppings assets/tilesets/terrain_toppings.png \
      --out /tmp/mapa.png --scale 0.25 --collision
"""
import argparse
import os
import sys

from PIL import Image, ImageDraw

from map_utils import footprints_of, load

TERRAIN_TEXTURE = {
    1: "terrain_grass.png", 2: "terrain_dirt.png", 3: "terrain_stone.png",
    4: "terrain_dark_grass.png", 5: "terrain_dark_grass_sparse.png",
    6: "terrain_bare_soil.png", 7: "terrain_forest_grass.png", 8: "terrain_dirt.png",
    9: "terrain_siege_gravel.png", 10: "terrain_dark_flagstone.png",
    # A terceira pilha (terrain_mask.go): o chao de castelo. Faltava aqui, e o
    # sintoma era um mapa de castelo renderizando com o chao PRETO — o script
    # nao achava textura para 11/13/14 e desenhava nada. O motor sempre soube
    # (terrain_renderer.go); era so a ferramenta que nao.
    11: "terrain_castle_blocks.png", 12: "terrain_castle_water.png",
    13: "terrain_castle_carpet.png", 14: "terrain_castle_stone.png",
}
ROLE_ORDER = {"ground_detail": 0, "structures_back": 1, "foreground": 2}


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", action="append", default=[])
    ap.add_argument("--tilesets-dir", default="assets/tilesets")
    # Pode repetir: um mapa de transicao usa dois kits de topping ao mesmo
    # tempo, e um render que so conhece um mente sobre metade do chao.
    ap.add_argument("--toppings", action="append",
                    default=None, help="PNG:firstgid (pode repetir)")
    ap.add_argument("--out", required=True)
    ap.add_argument("--scale", type=float, default=0.25)
    ap.add_argument("--collision", action="store_true")
    ap.add_argument("--grid", type=int, default=0, help="desenha uma grade a cada N celulas")
    args = ap.parse_args()

    tmap = load(args.map)
    W, H = tmap["width"], tmap["height"]
    T = tmap["tilewidth"]
    canvas = Image.new("RGBA", (W * T, H * T), (20, 20, 20, 255))

    textures = {}
    for kind, filename in TERRAIN_TEXTURE.items():
        path = os.path.join(args.tilesets_dir, filename)
        if os.path.isfile(path):
            textures[kind] = Image.open(path).convert("RGBA").resize((T, T), Image.LANCZOS)

    ground = next(l for l in tmap["layers"] if l.get("name") == "ground")
    for i, kind in enumerate(ground["data"]):
        if kind in textures:
            canvas.alpha_composite(textures[kind], ((i % W) * T, (i // W) * T))

    detail = next((l for l in tmap["layers"] if l.get("name") == "ground_detail"), None)
    kits = args.toppings or ["assets/tilesets/terrain_toppings.png:400",
                             "assets/tilesets/terrain_toppings_dark.png:432"]
    for kit in kits:
        path, _, gid_text = kit.rpartition(":")
        first = int(gid_text)
        if not detail or not os.path.isfile(path):
            continue
        sheet = Image.open(path).convert("RGBA")
        cols, rows = sheet.width // T, sheet.height // T
        for i, gid in enumerate(detail["data"]):
            tid = gid - first
            # A faixa importa: com dois kits, o gid do escuro e maior que o
            # firstgid do claro, e sem o limite superior o kit claro tentaria
            # desenhar tile 32+ e recortaria fora da folha.
            if gid == 0 or not 0 <= tid < cols * rows:
                continue
            sx, sy = (tid % cols) * T, (tid // cols) * T
            canvas.alpha_composite(sheet.crop((sx, sy, sx + T, sy + T)), ((i % W) * T, (i // W) * T))

    pieces, atlases = {}, {}
    for path in args.manifest:
        data = load(path)
        atlas_path = data["atlas"]
        if not os.path.isfile(atlas_path):
            atlas_path = os.path.join(os.path.dirname(path), "..", data["atlas"])
        atlases.setdefault(data["atlas"], Image.open(atlas_path).convert("RGBA"))
        for name, piece in data["pieces"].items():
            piece = dict(piece); piece["_atlas"] = data["atlas"]
            pieces[name] = piece

    objects = [(o, l.get("name")) for l in tmap["layers"] if l.get("type") == "objectgroup"
               for o in l.get("objects", [])]
    objects.sort(key=lambda pair: (ROLE_ORDER.get((pieces.get(pair[0].get("name")) or {}).get("role"), 1),
                                   pair[0]["y"]))
    for obj, _ in objects:
        piece = pieces.get(obj.get("name"))
        if not piece:
            continue
        s, a = piece["source"], piece["anchor"]
        art = atlases[piece["_atlas"]].crop((s["x"], s["y"], s["x"] + s["width"], s["y"] + s["height"]))
        canvas.alpha_composite(art, (int(obj["x"] - a["x"]), int(obj["y"] - a["y"])))

    draw = ImageDraw.Draw(canvas, "RGBA")
    if args.collision:
        # O espaco solido tem DUAS formas e o render tem que mostrar as duas.
        # Desenhar so a camada pintada leva a conclusao errada: no world_01 ela
        # esta vazia, porque casa, cerca e arvore colidem pelo manifesto.
        collision = next((l for l in tmap["layers"] if l.get("name") == "collision"), None)
        if collision:
            for i, gid in enumerate(collision["data"]):
                if gid:
                    x, y = (i % W) * T, (i // W) * T
                    draw.rectangle([x, y, x + T, y + T], fill=(255, 60, 60, 55), outline=(255, 60, 60, 130))
        for obj, _ in objects:
            piece = pieces.get(obj.get("name"))
            if not piece:
                continue
            for fp in footprints_of(piece):
                x, y = obj["x"] + fp["offsetX"], obj["y"] + fp["offsetY"]
                draw.rectangle([x, y, x + fp["width"], y + fp["height"]],
                               fill=(255, 60, 60, 55), outline=(255, 60, 60, 200))
    if args.grid:
        for col in range(0, W + 1, args.grid):
            draw.line([(col * T, 0), (col * T, H * T)], fill=(255, 255, 255, 60))
        for row in range(0, H + 1, args.grid):
            draw.line([(0, row * T), (W * T, row * T)], fill=(255, 255, 255, 60))
    for obj, layer_name in objects:
        if obj.get("name") == "player_spawn":
            draw.ellipse([obj["x"] - 30, obj["y"] - 30, obj["x"] + 30, obj["y"] + 30],
                         outline=(255, 255, 0, 220), width=6)

    if args.scale != 1.0:
        canvas = canvas.resize((max(1, int(canvas.width * args.scale)),
                                max(1, int(canvas.height * args.scale))), Image.LANCZOS)
    canvas.save(args.out)
    print(f"OK: {args.out} ({canvas.width}x{canvas.height})")
    return 0


if __name__ == "__main__":
    sys.exit(main())

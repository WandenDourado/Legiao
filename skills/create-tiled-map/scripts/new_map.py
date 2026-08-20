#!/usr/bin/env python3
"""Cria um mapa Tiled vazio com as camadas que o motor espera.

Ordem das camadas e obrigatoria: ground, ground_detail, structures_back,
foreground, collision (invisivel), e as camadas de objetos por categoria.
O terreno base comeca todo em grama.

Uso:
  python new_map.py --out assets/maps/world_02.json --width 60 --height 45
"""
import argparse
import sys

from map_utils import OBJECT_LAYERS, TERRAIN, save

# Referencia relativa ao arquivo do mapa (assets/maps/*.json).
TOPPINGS_TSX = "../tilesets/terrain_toppings.tsx"
TOPPINGS_FIRSTGID = 400


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--out", required=True)
    ap.add_argument("--width", type=int, default=60, help="largura em celulas")
    ap.add_argument("--height", type=int, default=45, help="altura em celulas")
    ap.add_argument("--tile", type=int, default=128)
    ap.add_argument("--base", default="grass", choices=list(TERRAIN))
    args = ap.parse_args()

    n = args.width * args.height
    def tiles(name, fill=0, visible=True):
        return {"data": [fill] * n, "height": args.height, "width": args.width,
                "name": name, "opacity": 1, "type": "tilelayer", "visible": visible,
                "x": 0, "y": 0}

    layers = [
        tiles("ground", TERRAIN[args.base]),
        tiles("ground_detail"),
        tiles("structures_back"),
        tiles("foreground"),
        tiles("collision", visible=False),
    ]
    # Todas as objectgroups que o motor sabe ler, inclusive as vazias: uma
    # camada ausente e um erro silencioso no Tiled (o objeto e desenhado numa
    # camada qualquer e o loader nunca o encontra), enquanto uma camada vazia
    # nao custa nada. `trails` e `portals` entram aqui pelo mesmo motivo que
    # `fences`: quem for editar o mapa a mao ja acha o lugar pronto.
    for name in OBJECT_LAYERS:
        layers.append({"name": name, "opacity": 1, "type": "objectgroup",
                       "visible": True, "x": 0, "y": 0, "objects": []})
    for i, layer in enumerate(layers, start=1):
        layer["id"] = i

    # O tileset dos toppings ja entra registrado: scatter_toppings.py grava
    # gids a partir de 400 e, sem esta entrada, TilesetForGID nao resolve e a
    # camada ground_detail simplesmente nao aparece no jogo.
    tilesets = [{"firstgid": TOPPINGS_FIRSTGID, "source": TOPPINGS_TSX}]

    tmap = {"compressionlevel": -1, "height": args.height, "infinite": False,
            "layers": layers, "nextlayerid": len(layers) + 1, "nextobjectid": 1,
            "orientation": "orthogonal", "renderorder": "right-down",
            "tiledversion": "1.12.1", "tileheight": args.tile, "tilesets": tilesets,
            "tilewidth": args.tile, "type": "map", "version": "1.10",
            "width": args.width}
    save(tmap, args.out)
    print(f"OK: {args.width}x{args.height} celulas ({args.width * args.tile}x{args.height * args.tile} px) -> {args.out}")
    print("Proximo: pintar terreno (paint_terrain.py), colocar objetos, spawn, e so entao toppings.")
    return 0


if __name__ == "__main__":
    sys.exit(main())

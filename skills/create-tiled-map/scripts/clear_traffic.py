#!/usr/bin/env python3
"""Limpa topping onde o mapa nao aguenta mais detalhe. Rode depois do scatter.

Faz duas coisas, as duas por medicao:

1. Zonas de trafego alto (vao do portao, soleira da porta, boca da ponte)
   ficam limpas de proposito: e chao pisado todo dia, nao canteiro.
2. Faz valer o teto de elementos por janela 3x3 — o mesmo que audit_layout.py
   cobra. O espalhamento decide celula a celula e nao enxerga a janela
   inteira; aqui, onde objeto e topping se somam acima do teto, o topping
   cede (sempre o que tem mais vizinhos).

Uso:
  python clear_traffic.py mapa.json --zone 28,29,4,4 --zone 25,38,4,2
  python clear_traffic.py mapa.json --max-per-window 4
"""
import argparse
import sys

from map_utils import load, save, set_cell, tile_layer


def topping_cells(detail, W):
    return {(i % W, i // W) for i, gid in enumerate(detail["data"]) if gid}


def object_cells(tmap):
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    cells = set()
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup" or layer.get("name") == "spawn":
            continue
        for obj in layer.get("objects", []):
            cells.add((int(obj["x"]) // tw, int(obj["y"]) // th))
    return cells


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--zone", action="append", default=[],
                    help="COL,LIN,LARG,ALT de area de trafego alto (pode repetir)")
    ap.add_argument("--window", type=int, default=3)
    ap.add_argument("--max-per-window", type=int, default=4)
    ap.add_argument("--limit", type=int, default=500, help="teto de remocoes")
    args = ap.parse_args()

    tmap = load(args.map)
    detail = tile_layer(tmap, "ground_detail")
    W, H = tmap["width"], tmap["height"]

    cleared = 0
    for zone in args.zone:
        col0, row0, w, h = (int(v) for v in zone.split(","))
        for row in range(row0, row0 + h):
            for col in range(col0, col0 + w):
                i = row * W + col
                if 0 <= i < len(detail["data"]) and detail["data"][i]:
                    set_cell(detail, W, col, row, 0)
                    cleared += 1

    objects = object_cells(tmap)
    win = args.window
    thinned = 0
    for _ in range(args.limit):
        tops = topping_cells(detail, W)
        marks = tops | objects
        worst = None
        for row in range(H - win + 1):
            for col in range(W - win + 1):
                cells = [(col + dx, row + dy) for dy in range(win) for dx in range(win)]
                if sum(1 for c in cells if c in marks) <= args.max_per_window:
                    continue
                worst = max((c for c in cells if c in tops),
                            key=lambda c: sum(1 for d in cells if d in marks and d != c),
                            default=None)
                if worst:
                    break
            if worst:
                break
        if not worst:
            break
        set_cell(detail, W, worst[0], worst[1], 0)
        thinned += 1

    save(tmap, args.map)
    print(f"{cleared} toppings removidos das zonas de trafego alto")
    print(f"{thinned} toppings removidos de janelas {win}x{win} acima de {args.max_per_window}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

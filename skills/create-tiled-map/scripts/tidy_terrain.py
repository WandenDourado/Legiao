#!/usr/bin/env python3
"""Dissolve as ilhas de terreno sem motivo deixadas pela borda irregular.

Uma transicao de material precisa de motivo visivel. Um quadrado de terra
sozinho no meio do campo nao tem — e ruido do --ragged, nao layout. Rode
depois de pintar o terreno e antes de colocar objetos.

A ilha vai para o material que a CERCA, nao para a grama. Mandar tudo para a
grama funcionava enquanto o unico chao de fundo era o verde; num mapa de bioma
escuro isso abria retalhos de grama clara no meio da mata fechada — exatamente
o tipo de transicao sem motivo que este script existe para remover. Vale
tambem para o chao verde: uma ilha de pedra dentro de um patio de terra vira
terra, e nao grama brotando no meio do calcamento.

Uso:
  python tidy_terrain.py assets/maps/world_01.json --min-island 5
"""
import argparse
import sys
from collections import Counter, deque

from map_utils import load, save, set_cell, tile_layer


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--min-island", type=int, default=5,
                    help="componente menor que isso vira grama")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    tmap = load(args.map)
    ground = tile_layer(tmap, "ground")
    W, H = tmap["width"], tmap["height"]
    data = ground["data"]
    seen, removed = set(), 0

    for row in range(H):
        for col in range(W):
            start, kind = (col, row), data[row * W + col]
            if start in seen:
                continue
            component, queue = {start}, deque([start])
            border = Counter()
            seen.add(start)
            while queue:
                c, r = queue.popleft()
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    nxt = (c + dx, r + dy)
                    if not (0 <= nxt[0] < W and 0 <= nxt[1] < H):
                        continue
                    neighbour = data[nxt[1] * W + nxt[0]]
                    if neighbour != kind:
                        border[neighbour] += 1     # quem cerca a ilha
                    elif nxt not in seen:
                        seen.add(nxt); component.add(nxt); queue.append(nxt)
            if len(component) >= args.min_island or not border:
                continue
            # O material que mais cerca a ilha e o que ela deveria ter sido.
            into = border.most_common(1)[0][0]
            removed += len(component)
            if not args.dry_run:
                for c, r in component:
                    set_cell(ground, W, c, r, into)

    if not args.dry_run:
        save(tmap, args.map)
    print(f"{removed} celulas de terreno solto dissolvidas no material vizinho"
          f"{' (simulacao)' if args.dry_run else ''}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""Audita o layout do mapa: alcancabilidade, poluicao e vazio.

Tres perguntas que so a medicao responde:
  1. O jogador consegue chegar aonde precisa? (flood fill a partir do spawn,
     com colisao da camada + footprints dos manifestos)
  2. Tem regiao poluida? (janelas com decoracao demais)
  3. Tem regiao morta? (blocos grandes sem nenhum detalhe nem objeto)

Uso:
  python audit_layout.py assets/maps/world_01.json \
      --manifest assets/vegetation_manifest.json --manifest assets/buildings_manifest.json \
      --goal caminho_floresta
"""
import argparse
import sys
from collections import deque

from map_utils import (PAVED, blocked_cells, cells_of, footprints_of, load,
                       manifests_for, spawn_cell, tile_layer)


def flood(tmap, blocked, start):
    W, H = tmap["width"], tmap["height"]
    seen = {start}
    queue = deque([start])
    while queue:
        col, row = queue.popleft()
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nxt = (col + dx, row + dy)
            if (0 <= nxt[0] < W and 0 <= nxt[1] < H
                    and nxt not in seen and nxt not in blocked):
                seen.add(nxt); queue.append(nxt)
    return seen


# A camada `vegetation` nao guarda so vegetacao: o motor amarra CADA manifesto
# a uma camada (internal/tilemap/vegetation.go), e os props de pedra do bioma
# escuro caem nessa. A regra "nada nasce em chao construido" e sobre PLANTA —
# um monolito em pe na laje de uma fortaleza e exatamente o que se espera ali,
# e foi por isso que a moldura do portao do world_03 reprovou oito vezes numa
# auditoria que estava certa sobre plantas e errada sobre pedra.
GROWS_NOT = ("standing_stone", "rune_stone")


def grows(name):
    return not name.startswith(GROWS_NOT)


def misplaced_vegetation(tmap, manifest_paths, margin):
    """Vegetacao que nasceu onde ninguem planta.

    Mato no meio do calcamento, ou colado nele, denuncia que a peca foi posta
    por coordenada e nao por leitura do chao. Mesma coisa para arbusto dentro
    do footprint de uma casa. Sao os dois erros que o olho perdoa no editor e
    nao perdoa no jogo.
    """
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    W, H = tmap["width"], tmap["height"]
    ground = tile_layer(tmap, "ground")
    pieces = manifests_for(tmap, manifest_paths)

    def paved(col, row):
        if not (0 <= col < W and 0 <= row < H):
            return False
        return ground["data"][row * W + col] in PAVED

    inside = set()
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup" or layer.get("name") != "buildings":
            continue
        for obj in layer.get("objects", []):
            piece = pieces.get(obj.get("name"))
            if not piece or not piece.get("collision"):
                continue
            for fp in footprints_of(piece):
                x, y = obj["x"] + fp["offsetX"], obj["y"] + fp["offsetY"]
                for row in range(int(y // th), int((y + fp["height"] - 0.01) // th) + 1):
                    for col in range(int(x // tw), int((x + fp["width"] - 0.01) // tw) + 1):
                        inside.add((col, row))

    problems = []
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup" or layer.get("name") != "vegetation":
            continue
        for obj in layer.get("objects", []):
            if obj.get("name", "").startswith("tree_canopy"):
                continue          # a copa acompanha o tronco, nao e uma peca no chao
            if not grows(obj.get("name", "")):
                continue          # pedra nao brota; ver GROWS_NOT
            cell = (int(obj["x"]) // tw, int(obj["y"]) // th)
            if cell in inside:
                problems.append((obj["name"], cell, "dentro do footprint de uma casa"))
                continue
            # a ancora cai na divisa entre a celula e a de cima; as duas contam
            near = [(cell[0] + dx, cell[1] + dy - k)
                    for k in (0, 1)
                    for dy in range(-margin, margin + 1)
                    for dx in range(-margin, margin + 1)]
            if any(paved(c, r) for c, r in near):
                problems.append((obj["name"], cell, "em caminho, ou colada nele"))
    return problems


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", action="append", default=[])
    ap.add_argument("--vegetation-margin", type=int, default=0,
                    help="celulas de folga exigidas entre vegetacao e caminho")
    ap.add_argument("--goal", action="append", default=[],
                    help="nome de objeto que precisa ser alcancavel a pe")
    ap.add_argument("--window", type=int, default=3, help="lado da janela de poluicao")
    ap.add_argument("--max-per-window", type=int, default=4)
    ap.add_argument("--empty-block", type=int, default=8,
                    help="lado do bloco considerado morto se nao tiver nada")
    args = ap.parse_args()

    tmap = load(args.map)
    W, H = tmap["width"], tmap["height"]
    blocked = blocked_cells(tmap, args.manifest)
    spawn = spawn_cell(tmap)
    problems = 0

    print(f"mapa {W}x{H} celulas | {len(blocked)} celulas solidas "
          f"({100 * len(blocked) / (W * H):.1f}%)")

    # 1. alcancabilidade
    if not spawn:
        print("FALHA: nao existe objeto player_spawn", file=sys.stderr); problems += 1
    elif spawn in blocked:
        print(f"FALHA: spawn {spawn} esta dentro de uma celula solida", file=sys.stderr); problems += 1
    else:
        reachable = flood(tmap, blocked, spawn)
        free = W * H - len(blocked)
        print(f"alcancavel a pe: {len(reachable)} de {free} celulas livres "
              f"({100 * len(reachable) / max(1, free):.1f}%)")
        if len(reachable) / max(1, free) < 0.5:
            print("AVISO: menos da metade do mapa livre e alcancavel — cerca fechada demais?")
        tw, th = tmap["tilewidth"], tmap["tileheight"]
        for goal in args.goal:
            found = None
            for layer in tmap["layers"]:
                if layer.get("type") != "objectgroup":
                    continue
                for obj in layer.get("objects", []):
                    if obj.get("name") == goal:
                        found = (int(obj["x"]) // tw, int(obj["y"]) // th)
            if not found:
                print(f"FALHA: objetivo '{goal}' nao existe no mapa", file=sys.stderr); problems += 1
            elif found not in reachable:
                print(f"FALHA: objetivo '{goal}' em {found} nao e alcancavel do spawn",
                      file=sys.stderr); problems += 1
            else:
                print(f"objetivo '{goal}' alcancavel")

    # 2. poluicao
    decorated = cells_of(tmap, "ground_detail")
    # `zones`, `collision` and `portals` hold rectangles, not art. Counting one
    # as a decorated cell registers its top-left CORNER — so a 28x12 climax zone
    # became "one element" at an arbitrary spot and pushed real windows over the
    # ceiling. world_03's eleven zones have the same problem.
    NON_ART = {"spawn", "zones", "collision", "portals", "trails"}
    for layer in tmap["layers"]:
        if layer.get("type") == "objectgroup" and layer.get("name") not in NON_ART:
            for obj in layer.get("objects", []):
                decorated.add((int(obj["x"]) // tmap["tilewidth"],
                               int(obj["y"]) // tmap["tileheight"]))
    win = args.window
    crowded = []
    for row in range(0, H - win + 1):
        for col in range(0, W - win + 1):
            count = sum(1 for dy in range(win) for dx in range(win)
                        if (col + dx, row + dy) in decorated)
            if count > args.max_per_window:
                crowded.append(((col, row), count))
    if crowded:
        problems += 1
        print(f"FALHA: {len(crowded)} janela(s) {win}x{win} com mais de {args.max_per_window} "
              f"elementos — salada de frutas", file=sys.stderr)
        for (cell, count) in crowded[:5]:
            print(f"   em {cell}: {count} elementos", file=sys.stderr)
    else:
        print(f"nenhuma janela {win}x{win} acima de {args.max_per_window} elementos")

    # 3. vegetacao no lugar errado
    misplaced = misplaced_vegetation(tmap, args.manifest, args.vegetation_margin)
    if misplaced:
        problems += 1
        print(f"FALHA: {len(misplaced)} peca(s) de vegetacao no lugar errado", file=sys.stderr)
        for name, cell, why in misplaced[:5]:
            print(f"   {name} em {cell}: {why}", file=sys.stderr)
    else:
        print("nenhuma vegetacao em caminho nem dentro de casa")

    # 4. vazio
    block = args.empty_block
    dead = []
    for row in range(0, H - block + 1, block):
        for col in range(0, W - block + 1, block):
            cells = {(col + dx, row + dy) for dy in range(block) for dx in range(block)}
            if cells & blocked:
                continue
            if not (cells & decorated):
                dead.append((col, row))
    if dead:
        print(f"AVISO: {len(dead)} bloco(s) {block}x{block} sem nenhum detalhe nem objeto: "
              f"{dead[:6]}{' ...' if len(dead) > 6 else ''}")
    else:
        print(f"nenhum bloco {block}x{block} completamente vazio")

    # densidade por bioma
    ground = tile_layer(tmap, "ground")
    per = {}
    for i, terrain in enumerate(ground["data"]):
        cell = (i % W, i // W)
        stats = per.setdefault(terrain, [0, 0])
        stats[0] += 1
        if cell in decorated:
            stats[1] += 1
    for terrain, (cells, dec) in sorted(per.items()):
        print(f"terreno {terrain}: {dec}/{cells} decorado ({100 * dec / max(1, cells):.1f}%)")

    if problems:
        print(f"\n{problems} problema(s) de layout", file=sys.stderr)
        return 1
    print("\nOK: layout consistente")
    return 0


if __name__ == "__main__":
    sys.exit(main())

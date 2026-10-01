#!/usr/bin/env python3
"""Espalha toppings por CONTEXTO, nao por sorteio uniforme.

Um mapa fica plausivel quando o detalhe conta a mesma historia do lugar: folha
junta perto de arvore, chao pisado dentro do lote cercado, e uma vila cuidada
nao tem poca de agua nem fuligem no calcamento. As regras vivem num JSON
(ver references/topping_rules.json) para poderem mudar por mapa.

Tambem evita os dois extremos: espacamento minimo entre pecas iguais e teto de
pecas por janela 3x3 (contra poluicao), e o relatorio final mostra a densidade
por bioma (contra vazio).

Uso:
  python scatter_toppings.py mapa.json --rules regras.json
  python scatter_toppings.py mapa.json --rules regras.json --seed 3 --dry-run
"""
import argparse
import json
import random
import sys
from collections import defaultdict

from map_utils import cells_of, load, save, tile_layer


def matching_objects(tmap, spec):
    """Objetos que casam com {layer, prefix} do contexto."""
    out = []
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup":
            continue
        if spec.get("layer") and layer.get("name") != spec["layer"]:
            continue
        for obj in layer.get("objects", []):
            if spec.get("prefix") and not obj.get("name", "").startswith(spec["prefix"]):
                continue
            out.append(obj)
    return out


def object_cells(tmap):
    """Celulas que ja tem um objeto em cima (arvore, casa, cerca, prop).

    Objeto conta como elemento na janela 3x3 igual a um topping — e assim que
    audit_layout.py mede poluicao. Ignorar as camadas de objeto aqui era o que
    fazia o espalhamento passar e a auditoria reprovar logo depois.
    O spawn fica de fora: nao e arte, e um marcador.
    """
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    cells = set()
    for layer in tmap["layers"]:
        if layer.get("type") != "objectgroup" or layer.get("name") == "spawn":
            continue
        for obj in layer.get("objects", []):
            cells.add((int(obj["x"]) // tw, int(obj["y"]) // th))
    return cells


def context_cells(tmap, ctx):
    """Celulas sob influencia de um contexto."""
    tw, th = tmap["tilewidth"], tmap["tileheight"]
    radius = ctx.get("radius", 2)
    near = ctx.get("near", {})
    seeds = set()
    if "terrain" in near:
        seeds |= {c for c in cells_of(tmap, "ground", lambda g: g == near["terrain"])}
    else:
        for obj in matching_objects(tmap, near):
            seeds.add((int(obj["x"]) // tw, int(obj["y"]) // th))
    cells = set()
    for cx, cy in seeds:
        for dy in range(-radius, radius + 1):
            for dx in range(-radius, radius + 1):
                if dx * dx + dy * dy <= radius * radius + 0.5:
                    cells.add((cx + dx, cy + dy))
    if near.get("outside"):
        cells -= seeds
    return cells


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--rules", required=True)
    ap.add_argument("--seed", type=int, default=1)
    ap.add_argument("--dry-run", action="store_true", help="so relata, nao grava")
    args = ap.parse_args()

    random.seed(args.seed)
    tmap = load(args.map)
    rules = load(args.rules)
    firstgid = rules["firstgid"]
    # firstgid por BIOMA: um mapa de transicao usa dois kits de topping ao
    # mesmo tempo (a orla clara continua com o kit da vila, a mata sombria usa
    # o proprio), e sem isso seria preciso rodar o espalhamento duas vezes —
    # o que nao compoe, porque a limpeza abaixo apagaria a rodada anterior.
    biome_gid = {b["terrain"]: b.get("firstgid", firstgid)
                 for b in rules["biomes"].values()}
    lowest_gid = min([firstgid, *biome_gid.values()])

    # Um id de tile alem do tileset vira um gid que TilesetForGID nao resolve:
    # a peca simplesmente nao aparece no jogo, sem erro nenhum. Ja aconteceu
    # (indices 36 e 37 num tileset de 32 pecas), entao a conferencia e aqui.
    tilecount = rules.get("tilecount", 32)
    for label, biome in rules["biomes"].items():
        bad = sorted(int(k) for k in biome["tiles"] if not 0 <= int(k) < tilecount)
        if bad:
            raise SystemExit(f"bioma '{label}': tile(s) {bad} fora do tileset "
                             f"de {tilecount} pecas")
    for ctx in rules.get("contexts", []):
        bad = sorted(int(k) for k in ctx.get("multiply", {}) if not 0 <= int(k) < tilecount)
        if bad:
            raise SystemExit(f"contexto '{ctx.get('name')}': tile(s) {bad} fora do "
                             f"tileset de {tilecount} pecas")
    W, H = tmap["width"], tmap["height"]
    ground = tile_layer(tmap, "ground")
    detail = tile_layer(tmap, "ground_detail")

    # limpa toppings de rodadas anteriores (mantem outros tiles de detalhe)
    for i, gid in enumerate(detail["data"]):
        if gid >= lowest_gid:
            detail["data"][i] = 0

    occupied = set()
    for name in ("structures_back", "foreground", "collision", "ground_detail"):
        occupied |= cells_of(tmap, name)
    objects = object_cells(tmap)
    occupied |= objects
    spacing = rules.get("spacing", {})
    from map_utils import spawn_cell
    spawn = spawn_cell(tmap)
    if spawn:
        r = spacing.get("keep_clear_radius_around_spawn", 2)
        occupied |= {(spawn[0] + dx, spawn[1] + dy)
                     for dy in range(-r, r + 1) for dx in range(-r, r + 1)}

    ctx_cells = [(ctx, context_cells(tmap, ctx)) for ctx in rules.get("contexts", [])]
    by_terrain = {b["terrain"]: b for b in rules["biomes"].values()}

    placed = {}                      # (col,row) -> tile id
    per_biome = defaultdict(int)
    total_cells = defaultdict(int)
    min_dist = spacing.get("min_same_tile_distance", 4)
    max_3x3 = spacing.get("max_in_3x3", 2)
    same_tile_positions = defaultdict(list)

    order = [(i % W, i // W) for i in range(W * H)]
    random.shuffle(order)
    for col, row in order:
        terrain = ground["data"][row * W + col]
        biome = by_terrain.get(terrain)
        if not biome:
            continue
        total_cells[terrain] += 1
        if (col, row) in occupied or (col, row) in placed:
            continue
        weights = {int(k): float(v) for k, v in biome["tiles"].items()}
        density = biome["density"]
        for ctx, cells in ctx_cells:
            if (col, row) not in cells:
                continue
            density *= ctx.get("density", 1.0)
            for tid, factor in ctx.get("multiply", {}).items():
                if int(tid) in weights:
                    weights[int(tid)] *= float(factor)
        if random.random() > density:
            continue
        # Arvore e casa contam na janela igual a um topping: o orcamento e de
        # elementos visiveis, nao de tiles.
        neighbours = sum(1 for dy in (-1, 0, 1) for dx in (-1, 0, 1)
                         if (col + dx, row + dy) in placed
                         or (col + dx, row + dy) in objects)
        if neighbours >= max_3x3:
            continue
        pool = [(t, w) for t, w in weights.items() if w > 0]
        if not pool:
            continue
        total = sum(w for _, w in pool)
        pick, acc = pool[-1][0], random.random() * total
        for tid, w in pool:
            acc -= w
            if acc <= 0:
                pick = tid
                break
        # A chave inclui o tileset: com dois kits no mesmo mapa, o id 3 do kit
        # claro e o id 3 do escuro sao pecas diferentes, e o espacamento minimo
        # entre peca IGUAL nao deve amarrar as duas.
        key = (biome_gid.get(terrain, firstgid), pick)
        if any(abs(col - px) + abs(row - py) < min_dist for px, py in same_tile_positions[key]):
            continue
        placed[(col, row)] = (pick, terrain)
        same_tile_positions[key].append((col, row))
        per_biome[terrain] += 1

    for (col, row), (tid, terrain) in placed.items():
        detail["data"][row * W + col] = biome_gid.get(terrain, firstgid) + tid

    if not args.dry_run:
        save(tmap, args.map)
    print(f"{len(placed)} toppings colocados ({'simulacao' if args.dry_run else 'gravado'})")
    for terrain, count in sorted(per_biome.items()):
        cells = total_cells[terrain] or 1
        print(f"  terreno {terrain}: {count} em {cells} celulas ({100 * count / cells:.1f}%)")
    return 0


if __name__ == "__main__":
    sys.exit(main())

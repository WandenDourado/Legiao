#!/usr/bin/env python3
"""Pinta regioes de terreno na camada ground.

O motor mistura as bordas sozinho (mascara de 8 vizinhos), entao aqui so se
declara QUAL terreno ocupa cada celula — nunca se pinta transicao.

Formas (coordenadas em CELULAS):
  --rect COL,LIN,LARG,ALT
  --disc COL,LIN,RAIO
  --path COL,LIN COL,LIN ...   (com --width, traca uma faixa entre os pontos)
  --band LIN_TOPO,LIN_BASE     (faixa horizontal de borda organica; * = aberto)

Uso:
  python paint_terrain.py mapa.json --type stone --rect 20,30,8,6
  python paint_terrain.py mapa.json --type dirt --path 24,44 24,30 36,22 --width 3 --jitter 1
  python paint_terrain.py mapa.json --type stone --disc 30,20,5 --ragged 2
  python paint_terrain.py mapa.json --type dark_grass --band '*,24' --noise 3.5

Sobre --band: e a forma de uma TRANSICAO DE BIOMA, e nao substitui --rect.
Faixa de bioma nao termina numa linha reta nem numa borda serrilhada celula a
celula: ela tem entrante, peninsula e ilha. Isso vem de ruido suave (duas
oitavas reamostradas por bicubica) somado a fronteira, nao de sorteio por
celula — sorteio por celula da chuvisco, nao faixa.

Bandas se declaram EMPILHANDO, do bioma mais externo para o mais interno:
cada passada sobrescreve a anterior no que avanca, e o que sobra embaixo vira
a faixa de transicao. Para a pilha escura (terra nua -> rala -> escura):

  --type bare_soil    --band '*,41'
  --type sparse_grass --band '*,34'
  --type dark_grass   --band '*,24'
"""
import argparse
import random
import sys

from map_utils import TERRAIN, load, save, set_cell, tile_layer

# Duas oitavas: a grossa desenha o contorno da faixa, a fina belisca a borda.
# So a grossa da uma linha quase reta que se desloca em bloco; so a fina da
# chuvisco. As duas juntas dao entrante, peninsula e ilha.
NOISE_OCTAVES = (((5, 4), 1.0), ((13, 10), 0.5))


def _catmull_rom(control, width):
    """Reamostra `control` para `width` valores com uma spline Catmull-Rom.

    Interpolacao linear entre os pontos de controle deixaria um bico em cada
    um deles, e bico na fronteira le como recorte, nao como mata. Catmull-Rom
    passa por todos os pontos com tangente continua, que e o que faz a borda
    parecer desenhada a mao.
    """
    n = len(control)
    out = []
    for i in range(width):
        t = i * (n - 1) / max(1, width - 1)
        k = min(int(t), n - 2)
        f = t - k
        p0 = control[max(0, k - 1)]
        p1, p2 = control[k], control[k + 1]
        p3 = control[min(n - 1, k + 2)]
        out.append(0.5 * ((2 * p1)
                          + (-p0 + p2) * f
                          + (2 * p0 - 5 * p1 + 4 * p2 - p3) * f * f
                          + (-p0 + 3 * p1 - 3 * p2 + p3) * f * f * f))
    return out


# A rampa precisa de ruido 2D COERENTE, e essa e a diferenca entre uma mancha e
# um retangulo. A primeira versao usava um ruido 1D por linha, com semente
# propria em cada uma: cada linha virava uma corrida horizontal independente e o
# empilhamento delas produzia degraus retangulares - em jogo lia como retalho
# colado, pior que a fronteira reta que veio substituir.
#
# Duas oitavas de grade grossa reamostradas por Catmull-Rom nos DOIS eixos: a
# grossa da a mancha, a fina belisca a borda dela.
RAMP_OCTAVES = (((9, 7), 1.0), ((19, 15), 0.55))


def _resample_2d(grid, cols, rows, width, height):
    """Catmull-Rom separavel: primeiro nas linhas, depois nas colunas."""
    wide = [_catmull_rom(row, width) for row in grid]
    out = []
    for x in range(width):
        column = [wide[r][x] for r in range(rows)]
        out.append(_catmull_rom(column, height))
    # out[x][y] -> devolve indexado por [y][x], que e como o mapa e varrido
    return [[out[x][y] for x in range(width)] for y in range(height)]


def ramp_noise(width, height, seed):
    """Campo em [-1, 1] sobre o mapa inteiro, suave nos dois eixos."""
    rng = random.Random(seed)
    total = [[0.0] * width for _ in range(height)]
    for (cols, rows), weight in RAMP_OCTAVES:
        grid = [[rng.uniform(-1.0, 1.0) for _ in range(cols)] for _ in range(rows)]
        field = _resample_2d(grid, cols, rows, width, height)
        for y in range(height):
            for x in range(width):
                total[y][x] += weight * field[y][x]
    peak = max(abs(v) for row in total for v in row) or 1.0
    return [[v / peak for v in row] for row in total]


def boundary_noise(width, seed):
    """Deslocamento em [-1, 1] por coluna, suave o bastante para virar borda."""
    rng = random.Random(seed)
    total = [0.0] * width
    for (cols, _rows), weight in NOISE_OCTAVES:
        control = [rng.uniform(-1.0, 1.0) for _ in range(cols)]
        for i, value in enumerate(_catmull_rom(control, width)):
            total[i] += weight * value
    peak = max(abs(v) for v in total) or 1.0
    return [v / peak for v in total]


def parse_edge(text):
    """'*' e uma borda aberta (a faixa vai ate o fim do mapa daquele lado)."""
    return None if text.strip() == "*" else int(text)


def line_cells(a, b):
    (x0, y0), (x1, y1) = a, b
    dx, dy = abs(x1 - x0), abs(y1 - y0)
    sx, sy = (1 if x0 < x1 else -1), (1 if y0 < y1 else -1)
    err = dx - dy
    out = []
    while True:
        out.append((x0, y0))
        if (x0, y0) == (x1, y1):
            return out
        e2 = 2 * err
        if e2 > -dy:
            err -= dy; x0 += sx
        if e2 < dx:
            err += dx; y0 += sy


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--type", required=True, choices=list(TERRAIN))
    ap.add_argument("--rect", help="COL,LIN,LARG,ALT")
    ap.add_argument("--disc", help="COL,LIN,RAIO")
    ap.add_argument("--path", nargs="+", help="COL,LIN COL,LIN ...")
    ap.add_argument("--band", help="LIN_TOPO,LIN_BASE — faixa horizontal; '*' = borda aberta")
    ap.add_argument("--ramp", help="LIN_ZERO,LIN_CHEIA — mancha o material entrando aos poucos")
    ap.add_argument("--noise", type=float, default=3.0,
                    help="amplitude do ruido da borda do --band, em celulas")
    ap.add_argument("--width", type=int, default=3, help="largura da faixa do --path em celulas")
    ap.add_argument("--jitter", type=int, default=0,
                    help="varia a largura da faixa +-N celulas, deixando a borda organica")
    ap.add_argument("--ragged", type=int, default=0,
                    help="quebra a borda de rect/disc em ate N celulas")
    ap.add_argument("--seed", type=int, default=7)
    args = ap.parse_args()

    random.seed(args.seed)
    tmap = load(args.map)
    layer = tile_layer(tmap, "ground")
    W, H = tmap["width"], tmap["height"]
    value = TERRAIN[args.type]
    painted = 0

    def put(col, row):
        nonlocal painted
        if 0 <= col < W and 0 <= row < H and set_cell(layer, W, col, row, value):
            painted += 1

    if args.rect:
        x, y, w, h = (int(v) for v in args.rect.split(","))
        for row in range(y, y + h):
            for col in range(x, x + w):
                edge = col in (x, x + w - 1) or row in (y, y + h - 1)
                if edge and args.ragged and random.random() < 0.35:
                    continue
                put(col, row)
                if edge and args.ragged and random.random() < 0.25:
                    put(col + random.randint(-args.ragged, args.ragged),
                        row + random.randint(-args.ragged, args.ragged))

    if args.disc:
        cx, cy, r = (int(v) for v in args.disc.split(","))
        for row in range(cy - r - args.ragged, cy + r + args.ragged + 1):
            for col in range(cx - r - args.ragged, cx + r + args.ragged + 1):
                d = ((col - cx) ** 2 + (row - cy) ** 2) ** 0.5
                limit = r + (random.uniform(-args.ragged, args.ragged) if args.ragged else 0)
                if d <= limit:
                    put(col, row)

    if args.band:
        top_text, bottom_text = args.band.split(",")
        top, bottom = parse_edge(top_text), parse_edge(bottom_text)
        # Cada borda ganha o proprio ruido: bordas correlacionadas produziriam
        # uma faixa de espessura constante serpenteando, que le como fita, nao
        # como bioma.
        top_wobble = boundary_noise(W, args.seed) if top is not None else None
        bottom_wobble = boundary_noise(W, args.seed + 991) if bottom is not None else None
        for col in range(W):
            lo = 0 if top is None else top + top_wobble[col] * args.noise
            hi = H if bottom is None else bottom + bottom_wobble[col] * args.noise
            for row in range(max(0, int(round(lo))), min(H, int(round(hi)) + 1)):
                put(col, row)

    if args.ramp:
        zero_row, full_row = (int(v) for v in args.ramp.split(","))
        # Duas oitavas de ruido 2D: a grossa faz a mancha, a fina belisca a
        # borda dela. Sorteio por celula daria chuvisco, e chuvisco de dois
        # biomas le como sujeira, nao como transicao.
        field = ramp_noise(W, H, args.seed)
        span = (full_row - zero_row) or 1
        for row in range(H):
            t = (row - zero_row) / span
            if t <= 0:
                continue
            for col in range(W):
                # O campo desloca o limiar, entao o bioma avanca por linguas e
                # deixa ilhas para tras — que e como uma mata toma a outra.
                if min(t, 1.0) > 0.5 + field[row][col] * 0.48:
                    put(col, row)

    if args.path:
        points = [tuple(int(v) for v in p.split(",")) for p in args.path]
        for a, b in zip(points, points[1:]):
            for col, row in line_cells(a, b):
                half = args.width / 2
                if args.jitter:
                    half += random.uniform(-args.jitter, args.jitter)
                span = int(round(half))
                for dy in range(-span, span + 1):
                    for dx in range(-span, span + 1):
                        if dx * dx + dy * dy <= half * half + 0.5:
                            put(col + dx, row + dy)

    save(tmap, args.map)
    print(f"OK: {painted} celulas pintadas de {args.type}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""Fecha um lote com cerca emendando as pecas pelos PONTOS DE CONEXAO medidos.

Emendar peca por largura de source deixa buraco: varias pecas tem 5-7 px de
margem transparente antes do poste, e essa margem vira vao visivel na junta.
Aqui cada peca e medida no alfa do atlas — onde o trilho realmente comeca e
termina — e a proxima peca e posta para continuar exatamente nesse pixel.
Nunca derive posicao de cerca de aritmetica de celula.

Nao pinta colisao. A cerca colide pelo MANIFESTO: cada peca tem os proprios
retangulos em `collisionFootprints`, com o vao do portao aberto livre por
construcao. Pintar celula por baixo dela bloquearia 128 px onde a peca ocupa
~48, que era exatamente o defeito. `--paint-collision` existe so para cerca de
um kit que ainda nao tenha footprint no manifesto.

Imprime o vao livre do portao em colunas — que e o que voce precisa para levar
o caminho ate ele.

Uso:
  python place_fence.py assets/maps/world_01.json --at 3020,3871 \
      --north fence_corner_nw,fence_h_middle,fence_gate_h_open,fence_h_middle,fence_corner_ne \
      --south fence_corner_sw,fence_h_middle,fence_gate_h_closed,fence_h_middle,fence_corner_se \
      --sides 2
"""
import argparse
import sys

from PIL import Image

from map_utils import add_object, load, save, set_cell, tile_layer

RAIL_LO, RAIL_HI = 40, 160   # faixa de altura dos trilhos, acima da linha do chao
OPAQUE = 40                  # alfa a partir do qual o pixel conta como arte


class Pieces:
    """Manifesto de cerca + medicoes feitas no alfa do atlas."""

    def __init__(self, manifest_path):
        data = load(manifest_path)
        self.pieces = data["pieces"]
        self.atlas = Image.open(data["atlas"])
        self._cache = {}

    def image(self, name):
        if name not in self._cache:
            s = self.pieces[name]["source"]
            self._cache[name] = self.atlas.crop(
                (s["x"], s["y"], s["x"] + s["width"], s["y"] + s["height"]))
        return self._cache[name]

    def anchor_y(self, name):
        return self.pieces[name]["anchor"]["y"]

    def connect_x(self, name):
        """Primeira e ultima coluna com trilho, relativas a ancora."""
        img = self.image(name)
        alpha = img.getchannel("A").load()
        ay = self.anchor_y(name)
        lo, hi = max(0, ay - RAIL_HI), max(1, ay - RAIL_LO)
        cols = [x for x in range(img.width)
                if any(alpha[x, y] > OPAQUE for y in range(lo, hi))]
        return cols[0], cols[-1]

    def band_center_x(self, name):
        """Centro da banda vertical, relativo a ancora."""
        img = self.image(name)
        alpha = img.getchannel("A").load()
        row = 1 if self.anchor_y(name) >= img.height else img.height - 2
        cols = [x for x in range(img.width) if alpha[x, row] > OPAQUE]
        return (cols[0] + cols[-1]) // 2

    def connect_y(self, name):
        """Primeira e ultima linha com banda, relativas a ancora."""
        img = self.image(name)
        alpha = img.getchannel("A").load()
        ay, cx = self.anchor_y(name), self.band_center_x(name)
        rows = [y for y in range(img.height)
                if any(alpha[x, y] > OPAQUE
                       for x in range(max(0, cx - 15), cx + 15))]
        return rows[0] - ay, rows[-1] - ay


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("map")
    ap.add_argument("--manifest", default="assets/fences_manifest.json")
    ap.add_argument("--at", required=True,
                    help="X,Y em PIXELS do canto noroeste (linha de chao do trecho norte)")
    ap.add_argument("--north", required=True, help="pecas do trecho norte, separadas por virgula")
    ap.add_argument("--south", required=True, help="pecas do trecho sul")
    ap.add_argument("--sides", type=int, default=2,
                    help="quantos fence_v_middle em cada lateral (define a altura do lote)")
    ap.add_argument("--paint-collision", action="store_true",
                    help="pinta celulas na camada `collision` (legado: so para kit de cerca "
                         "sem collisionFootprints no manifesto; ver docstring)")
    args = ap.parse_args()

    tmap = load(args.map)
    P = Pieces(args.manifest)
    T = tmap["tilewidth"]
    X0, Y0 = (int(v) for v in args.at.split(","))
    placed = []                       # (nome, x, y)

    def run_h(start_x, y, names):
        cursor = start_x
        for name in names:
            left, right = P.connect_x(name)
            x = cursor - left
            placed.append((name, x, y))
            cursor = x + right + 1
        return cursor

    def run_v(band_x, start_y, names):
        cursor = start_y
        for name in names:
            top, bottom = P.connect_y(name)
            y = cursor - top
            placed.append((name, band_x - P.band_center_x(name), y))
            cursor = y + bottom + 1
        return cursor

    north = args.north.split(",")
    south = args.south.split(",")
    run_h(X0, Y0, north)
    nw, ne = placed[0], placed[-1]
    left_band = nw[1] + P.band_center_x(nw[0])
    right_band = ne[1] + P.band_center_x(ne[0])

    # As laterais continuam a banda que ja desce do canto norte.
    sides = ["fence_v_middle"] * args.sides
    end_left = run_v(left_band, nw[2] + P.connect_y(nw[0])[1] + 1, sides)
    run_v(right_band, ne[2] + P.connect_y(ne[0])[1] + 1, sides)

    # O trecho sul encaixa onde a lateral termina.
    y_south = end_left - P.connect_y(south[0])[0]
    sw_x = left_band - P.band_center_x(south[0])
    run_h(sw_x + P.connect_x(south[0])[0], y_south, south)

    gate_cols = set()
    for name, x, y in placed:
        add_object(tmap, "fences", name, 0, 0, pixel=(int(x), int(y)))
        if "gate" in name and "open" in name:
            left, right = P.connect_x(name)
            gate_cols |= {c for c in range(tmap["width"])
                          if x + left <= c * T + T / 2 <= x + right}

    marked = set()
    if args.paint_collision:
        collision = tile_layer(tmap, "collision")
        W = tmap["width"]
        for name, x, y in placed:
            if "gate" in name and "open" in name:
                continue          # o vao do portao e por onde se passa
            # Canto e tê tem trilho E banda: marcar so um dos dois deixa duas
            # celulas abertas em cada lateral, e o lote vaza.
            joint = "corner" in name or "tee" in name
            if joint or "_h_" in name:
                row = (int(y) - 1) // T
                left, right = P.connect_x(name)
                marked |= {(c, row)
                           for c in range((int(x) + left) // T, (int(x) + right) // T + 1)
                           if x + left <= c * T + T / 2 <= x + right}
            if joint or "_v_" in name:
                col = int(x + P.band_center_x(name)) // T
                top, bottom = P.connect_y(name)
                marked |= {(col, r)
                           for r in range((int(y) + top) // T, (int(y) + bottom) // T + 1)
                           if y + top <= r * T + T / 2 <= y + bottom}
        for col, row in marked:
            set_cell(collision, W, col, row, 1)

    save(tmap, args.map)
    xs = [x for _, x, _ in placed]
    ys = [y for _, _, y in placed]
    colisao = f"{len(marked)} celulas pintadas (legado)" if args.paint_collision \
        else "colisao vem do manifesto"
    print(f"{len(placed)} pecas, {colisao}")
    print(f"lote: colunas {min(xs) // T}-{max(xs) // T}, linhas {min(ys) // T}-{max(ys) // T}")
    print(f"vao livre do portao nas colunas {sorted(gate_cols)}"
          if gate_cols else "AVISO: nenhum portao aberto — o lote fica selado")
    return 0


if __name__ == "__main__":
    sys.exit(main())

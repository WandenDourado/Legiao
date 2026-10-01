#!/usr/bin/env python3
"""Normaliza uma sheet radial (inimigo rotacionavel) gerada por IA.

O gerador nao obedece geometria numerica. Em vez de insistir no prompt, o
alinhamento e feito aqui, de forma deterministica:

  1. chroma key do matte magenta + despill da franja
  2. recentragem pelo centroide no centro geometrico da celula
  3. escala nao-uniforme de cada frame para a largura/altura alvo do ciclo
     (a escala nao-uniforme E a animacao de squash: aplica-la aqui e legitimo)
  4. verificacao do circulo inscrito
  5. um unico downsample ate o frame final

Uso:
  python radial_normalize.py sheet.png --out frames/ --cols 3 --rows 2 \
      --frame 128 --targets 68x68,72x64,76x60,70x66,62x74,66x70
"""
import argparse
import math
from pathlib import Path

from PIL import Image


def key_matte(cell: Image.Image) -> Image.Image:
    """Magenta puro -> transparente, com despill da franja rosada."""
    cell = cell.convert("RGBA")
    px = cell.load()
    w, h = cell.size
    for y in range(h):
        for x in range(w):
            r, g, b, _ = px[x, y]
            # "magenta-ness": vermelho e azul altos, verde baixo
            m = min(r, b) - g
            if m > 90:
                px[x, y] = (0, 0, 0, 0)
            elif m > 30:
                # franja: alpha proporcional + despill do azul
                a = int(255 * (1 - (m - 30) / 60))
                px[x, y] = (r, g, min(b, g + r // 3), a)
    return cell


def bbox_and_centroid(cell: Image.Image, alpha_min: int = 24):
    px = cell.load()
    w, h = cell.size
    xs, ys = [], []
    for y in range(h):
        for x in range(w):
            if px[x, y][3] >= alpha_min:
                xs.append(x)
                ys.append(y)
    if not xs:
        raise ValueError("celula vazia")
    return (min(xs), min(ys), max(xs) + 1, max(ys) + 1), (sum(xs) / len(xs), sum(ys) / len(ys))


def max_radius(cell: Image.Image, alpha_min: int = 24) -> float:
    px = cell.load()
    w, h = cell.size
    cx, cy = w / 2, h / 2
    best = 0.0
    for y in range(h):
        for x in range(w):
            if px[x, y][3] >= alpha_min:
                best = max(best, math.hypot(x - cx, y - cy))
    return best


def normalize_stretch(cell: Image.Image, target_w: float, target_h: float, out_size: int) -> Image.Image:
    """Escala nao-uniforme para largura/altura alvo, recentrando pelo centroide.

    Para corpos amorfos (slime): distorcer a proporcao por frame E a animacao de
    squash, entao aplica-la aqui e legitimo e garante amplitude exata.
    """
    (x0, y0, x1, y1), (ccx, ccy) = bbox_and_centroid(cell)
    art = cell.crop((x0, y0, x1, y1))

    # trabalha em 2x o tamanho final e reduz uma unica vez no fim
    work = out_size * 2
    tw = max(1, round(work * target_w))
    th = max(1, round(work * target_h))
    art = art.resize((tw, th), Image.LANCZOS)

    # centroide dentro do recorte, reescalado
    off_x = (ccx - x0) / (x1 - x0) * tw
    off_y = (ccy - y0) / (y1 - y0) * th

    canvas = Image.new("RGBA", (work, work), (0, 0, 0, 0))
    canvas.alpha_composite(art, (round(work / 2 - off_x), round(work / 2 - off_y)))
    return canvas.resize((out_size, out_size), Image.LANCZOS)


def normalize_uniform(cell: Image.Image, target_extent: float, out_size: int) -> Image.Image:
    """Escala uniforme ate a maior dimensao, recentrando pela bounding box.

    Para corpos anatomicos (lobo): esticar a proporcao por frame deformaria o
    animal, entao o aspecto da arte e preservado e so o tamanho e o centro sao
    corrigidos. O pivo e o centro da bbox, nao o centroide: numa forma alongada
    com cauda, o centroide e puxado para tras e a rotacao ficaria excentrica.
    """
    (x0, y0, x1, y1), _ = bbox_and_centroid(cell)
    art = cell.crop((x0, y0, x1, y1))
    w, h = art.size

    work = out_size * 2
    # a maior dimensao encosta no alvo; a outra acompanha pelo mesmo fator
    factor = (work * target_extent) / max(w, h)
    art = art.resize((max(1, round(w * factor)), max(1, round(h * factor))), Image.LANCZOS)

    canvas = Image.new("RGBA", (work, work), (0, 0, 0, 0))
    canvas.alpha_composite(art, (round((work - art.size[0]) / 2), round((work - art.size[1]) / 2)))
    return canvas.resize((out_size, out_size), Image.LANCZOS)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("sheet")
    ap.add_argument("--out", required=True)
    ap.add_argument("--cols", type=int, default=3)
    ap.add_argument("--rows", type=int, default=2)
    ap.add_argument("--frame", type=int, default=128)
    ap.add_argument(
        "--fit",
        choices=("stretch", "uniform"),
        default="stretch",
        help="stretch: alvo por frame (corpos amorfos). uniform: preserva o aspecto (anatomicos).",
    )
    ap.add_argument("--targets", help="modo stretch, ex: 68x68,72x64,...")
    ap.add_argument("--extent", type=int, default=78, help="modo uniform: %% do frame na maior dimensao")
    args = ap.parse_args()

    targets = []
    if args.fit == "stretch":
        if not args.targets:
            ap.error("--targets e obrigatorio no modo stretch")
        for chunk in args.targets.split(","):
            w, h = chunk.lower().split("x")
            targets.append((int(w) / 100, int(h) / 100))

    sheet = Image.open(args.sheet).convert("RGBA")
    W, H = sheet.size
    cw, chh = W // args.cols, H // args.rows
    outdir = Path(args.out)
    outdir.mkdir(parents=True, exist_ok=True)

    limit = args.frame / 2
    print(f"{'frame':>6} {'alvo':>9} {'raio':>7} {'limite':>7}  veredito")
    for i in range(args.cols * args.rows):
        row, col = divmod(i, args.cols)
        cell = sheet.crop((col * cw, row * chh, (col + 1) * cw, (row + 1) * chh))
        cell = key_matte(cell)

        if args.fit == "stretch":
            tw, th = targets[i]
            norm = normalize_stretch(cell, tw, th, args.frame)
            label = f"{tw:.0%}x{th:.0%}"
        else:
            norm = normalize_uniform(cell, args.extent / 100, args.frame)
            label = f"{args.extent}% unif"

        r = max_radius(norm)
        ok = "ok" if r <= limit else "ESTOURA"
        print(f"{i + 1:>6} {label:>9} {r:7.1f} {limit:7.1f}  {ok}")
        norm.save(outdir / f"frame_{i + 1:02d}.png")

    print(f"\n{args.cols * args.rows} frames em {outdir}")


if __name__ == "__main__":
    main()

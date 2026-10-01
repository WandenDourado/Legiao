#!/usr/bin/env python3
"""Valida frames radiais ja normalizados. Nao gera nada, nao corrige nada.

Verifica o que quebra um sprite que gira em runtime:

  containment  todo pixel dentro do circulo inscrito, senao as quinas somem
               em alguns angulos e nao em outros (o sprite "pisca" ao girar)
  centering    o centro tem que ser o pivo de rotacao, senao o bicho orbita
  fringe       residuo magenta da chave de croma
  occupancy    quanto do frame o corpo ocupa; corpo estreito nao le no jogo
  value        proporcao do corpo em preto chapado; mata a leitura em escala

Saida legivel + exit code: 0 aprovado, 1 reprovado, 3 aprovado com ressalva.

Uso:
  python validate_radial.py frames/ [--frame 128] [--background 105,144,50]
"""
import argparse
import math
import sys
from pathlib import Path

from PIL import Image

ALPHA_MIN = 60

# Limites. Derivados dos dois inimigos ja produzidos (slime e lobo), nao de
# teoria: sao os valores que separaram os casos aprovados dos reprovados.
CONTAINMENT_TARGET = 0.82   # fracao do raio inscrito; 0.80 = folga confortavel
CENTER_TOLERANCE = 4.0      # px de desvio do centro do frame
FRINGE_WARN = 0.02          # 2% dos pixels com residuo magenta
FRINGE_FAIL = 0.06
OCCUPANCY_WARN = 0.15       # slime 0.33, lobo 0.11 (compensado por RenderScale)
FLAT_BLACK_WARN = 0.70      # lobo v1 tinha 0.89 e era ilegivel; v2 0.75; v3 ok
SATURATION_RESCUE = 0.45    # acima disso o corpo le por matiz mesmo sendo escuro


def load_frames(root: Path):
    files = sorted(root.glob("frame_*.png"))
    if not files:
        raise SystemExit(f"nenhum frame_*.png em {root}")
    return [(f.name, Image.open(f).convert("RGBA")) for f in files]


def measure(img: Image.Image, pivot: str):
    px = img.load()
    w, h = img.size
    xs, ys, lums, sats, fringe = [], [], [], [], 0
    for y in range(h):
        for x in range(w):
            r, g, b, a = px[x, y]
            if a < ALPHA_MIN:
                continue
            xs.append(x)
            ys.append(y)
            lums.append(0.299 * r + 0.587 * g + 0.114 * b)
            hi, lo = max(r, g, b), min(r, g, b)
            sats.append((hi - lo) / hi if hi else 0.0)
            # residuo do matte: vermelho e azul altos com verde baixo
            if min(r, b) - g > 18:
                fringe += 1
    if not xs:
        return None

    cx, cy = w / 2, h / 2
    max_r = max(math.hypot(x - cx, y - cy) for x, y in zip(xs, ys))
    # O pivo tem que ser medido do mesmo jeito que normalize_radial.py centrou,
    # senao a checagem acusa deslocamento num sprite correto: num corpo com
    # apendice (a gota do slime) o centroide e o centro da bbox nao coincidem.
    if pivot == "bbox":
        px_c = (min(xs) + max(xs)) / 2
        py_c = (min(ys) + max(ys)) / 2
    else:
        px_c = sum(xs) / len(xs)
        py_c = sum(ys) / len(ys)
    lums.sort()
    n = len(lums)
    return {
        "count": n,
        "max_radius": max_r,
        "inscribed": min(w, h) / 2,
        "drift": math.hypot(px_c - cx, py_c - cy),
        "fringe": fringe / n,
        "occupancy": n / (w * h),
        "flat_black": sum(1 for v in lums if v < 70) / n,
        "median_lum": lums[n // 2],
        "median_sat": sorted(sats)[len(sats) // 2],
        "iqr": lums[3 * n // 4] - lums[n // 4],
        "square": w == h,
    }


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("frames_dir")
    ap.add_argument("--frame", type=int, default=128)
    ap.add_argument("--background", default="105,144,50", help="cor de fundo do jogo, para contraste")
    ap.add_argument(
        "--pivot",
        choices=("centroid", "bbox"),
        default="centroid",
        help="como o pivo foi definido na normalizacao: centroid para --fit stretch, bbox para --fit uniform",
    )
    args = ap.parse_args()

    bg = [int(v) for v in args.background.split(",")]
    bg_lum = 0.299 * bg[0] + 0.587 * bg[1] + 0.114 * bg[2]

    frames = load_frames(Path(args.frames_dir))
    failures, warnings = [], []

    print(f"{'frame':>12} {'raio':>7} {'limite':>7} {'desvio':>7} {'franja':>7} {'ocupa':>7} {'preto':>7}")
    stats = []
    for name, img in frames:
        m = measure(img, args.pivot)
        if m is None:
            failures.append(f"{name}: frame vazio")
            continue
        stats.append(m)
        target = m["inscribed"] * CONTAINMENT_TARGET
        flags = []
        if not m["square"]:
            failures.append(f"{name}: frame nao e quadrado ({img.size})")
        if m["max_radius"] > m["inscribed"]:
            failures.append(f"{name}: estoura o circulo inscrito ({m['max_radius']:.0f} > {m['inscribed']:.0f})")
            flags.append("ESTOURA")
        elif m["max_radius"] > target:
            warnings.append(f"{name}: pouca folga ate o circulo inscrito ({m['max_radius']:.0f} / {m['inscribed']:.0f})")
            flags.append("justo")
        if m["drift"] > CENTER_TOLERANCE:
            failures.append(f"{name}: fora de centro em {m['drift']:.1f} px")
            flags.append("DESCENTRADO")
        if m["fringe"] > FRINGE_FAIL:
            failures.append(f"{name}: franja magenta de {m['fringe']:.1%}")
        elif m["fringe"] > FRINGE_WARN:
            warnings.append(f"{name}: franja magenta de {m['fringe']:.1%}")

        print(
            f"{name:>12} {m['max_radius']:7.1f} {m['inscribed']:7.1f} {m['drift']:7.1f}"
            f" {m['fringe']:6.1%} {m['occupancy']:6.1%} {m['flat_black']:6.1%}  {' '.join(flags)}"
        )

    if stats:
        occ = sum(s["occupancy"] for s in stats) / len(stats)
        blk = sum(s["flat_black"] for s in stats) / len(stats)
        med = sum(s["median_lum"] for s in stats) / len(stats)
        sat = sum(s["median_sat"] for s in stats) / len(stats)
        print()
        print(
            f"media: ocupacao {occ:.1%} | preto chapado {blk:.1%} | "
            f"luminancia mediana {med:.0f} (fundo {bg_lum:.0f}) | saturacao mediana {sat:.2f}"
        )
        if occ < OCCUPANCY_WARN:
            warnings.append(
                f"corpo ocupa so {occ:.1%} do frame. Nao alargue a anatomia para compensar - "
                f"isso produz pose espalhada. Aumente o RenderScale no EnemyDef."
            )
        # Escuro so e problema quando tambem e dessaturado. Um corpo escuro e
        # saturado le por matiz (o slime carmesim tem 75% abaixo de 70 e le
        # perfeitamente); um corpo escuro e cinza vira mancha (lobo v1, 89%).
        if blk > FLAT_BLACK_WARN and sat < SATURATION_RESCUE:
            warnings.append(
                f"{blk:.1%} do corpo esta abaixo de luminancia 70 com saturacao de apenas "
                f"{sat:.2f}. Em escala de jogo o interior vira uma mancha unica; clareie a "
                f"base e reserve o escuro para as marcas."
            )
        if abs(med - bg_lum) < 25 and sat < SATURATION_RESCUE:
            warnings.append(
                f"luminancia mediana {med:.0f} muito proxima do fundo {bg_lum:.0f} e sem "
                f"saturacao para compensar; separe por valor ou por matiz."
            )

    print()
    for w in warnings:
        print(f"  AVISO   {w}")
    for f in failures:
        print(f"  FALHA   {f}")

    if failures:
        print("\nREPROVADO")
        sys.exit(1)
    if warnings:
        print("\nAPROVADO COM RESSALVA")
        sys.exit(3)
    print("\nAPROVADO")
    sys.exit(0)


if __name__ == "__main__":
    main()

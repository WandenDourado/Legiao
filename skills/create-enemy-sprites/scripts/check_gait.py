#!/usr/bin/env python3
"""Checa a locomocao de um inimigo com pernas, em frames radiais normalizados.

Tres defeitos que passam batido numa revisao visual apressada e que so foram
identificados por medicao durante a producao do lobo:

  ampulheta   a silhueta fica mais larga nas pontas que no meio. Acontece quando
              se pede largura e o gerador abre as pernas para os lados em vez de
              move-las no eixo do corpo. Le como pele esticada, nao como corrida.

  manqueira   sempre a mesma perna dianteira avanca. Um galope real ate mantem a
              perna de guia por varias passadas, mas num loop curto visto de cima
              isso le como o animal arrastando um membro. O ciclo precisa ser
              duas meias-passadas que trocam a guia.

  ciclo morto os frames mudam pouco demais entre si e a animacao nao le.

Nao se aplica a corpos amorfos (slime): use so em inimigos com membros.

Uso:
  python check_gait.py frames/ [--center-band 54,74]
"""
import argparse
import sys
from pathlib import Path

from PIL import Image

ALPHA_MIN = 60

# A ampulheta real medida no lobo v2 tinha meio/ponta = 0.59. O falso positivo
# do v4 aprovado tinha 0.92 (dois pixels de diferenca na fase de extensao
# traseira, que e esperado). 0.85 separa os dois casos com folga.
WAIST_RATIO_FAIL = 0.85
# Avanco minimo, em px, para considerar que um lado esta liderando.
LEAD_THRESHOLD = 8
# Assimetria tolerada entre o alcance maximo de cada lado.
LEAD_BALANCE_TOLERANCE = 10
# Mudanca minima de silhueta entre frames consecutivos. Slime 10.5%, lobo 19.6%.
MOTION_WARN = 0.06


def load(root: Path):
    files = sorted(root.glob("frame_*.png"))
    if not files:
        raise SystemExit(f"nenhum frame_*.png em {root}")
    return [(f.name, Image.open(f).convert("RGBA")) for f in files]


def silhouette(img):
    px = img.load()
    w, h = img.size
    rows = {}
    mask = set()
    for y in range(h):
        line = [x for x in range(w) if px[x, y][3] >= ALPHA_MIN]
        if line:
            rows[y] = line
            mask.update((x, y) for x in line)
    return rows, mask


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("frames_dir")
    ap.add_argument(
        "--center-band",
        default="54,74",
        help="faixa x da cabeca/eixo, excluida da medicao de avanco lateral",
    )
    args = ap.parse_args()
    band_lo, band_hi = (int(v) for v in args.center_band.split(","))

    frames = load(Path(args.frames_dir))
    failures, warnings = [], []

    print("=== silhueta: o meio do corpo deve ser a secao mais larga ===")
    print(f"{'frame':>12} {'ombro':>7} {'meio':>7} {'quadril':>7} {'razao':>7}  veredito")
    masks = []
    leads = []
    for name, img in frames:
        rows, mask = silhouette(img)
        masks.append(mask)
        if not rows:
            failures.append(f"{name}: frame vazio")
            continue
        ys = sorted(rows)
        top, height = ys[0], ys[-1] - ys[0]

        def width_at(frac):
            line = rows.get(int(top + height * frac), [])
            return max(line) - min(line) if line else 0

        shoulder, middle, hip = width_at(0.25), width_at(0.50), width_at(0.75)
        ends = max(shoulder, hip)
        ratio = middle / ends if ends else 1.0
        verdict = "ok"
        if ratio < WAIST_RATIO_FAIL:
            verdict = "AMPULHETA"
            failures.append(
                f"{name}: cintura estrangulada (meio {middle} vs pontas {ends}). "
                f"As pernas estao abrindo para os lados em vez de se moverem no eixo do corpo."
            )
        print(f"{name:>12} {shoulder:7d} {middle:7d} {hip:7d} {ratio:7.2f}  {verdict}")

        # avanco de cada lado, ignorando a faixa central (cabeca/eixo)
        left = [y for (x, y) in mask if x < band_lo]
        right = [y for (x, y) in mask if x > band_hi]
        top_l = min(left) if left else 10**6
        top_r = min(right) if right else 10**6
        leads.append(top_r - top_l)  # positivo = esquerda avanca

    print()
    print("=== alternancia dos membros dianteiros ===")
    print(f"{'frame':>12} {'avanco':>8}  guia")
    for (name, _), d in zip(frames, leads):
        side = "esquerda" if d > LEAD_THRESHOLD else ("direita" if d < -LEAD_THRESHOLD else "simetrico")
        print(f"{name:>12} {d:+8d}  {side}")

    if leads:
        best_left, best_right = max(leads), min(leads)
        alternates = best_left > LEAD_THRESHOLD and best_right < -LEAD_THRESHOLD
        balance = abs(abs(best_left) - abs(best_right))
        print()
        print(f"alcance maximo: esquerda {best_left:+d} | direita {best_right:+d} | desequilibrio {balance}")
        if not alternates:
            failures.append(
                "so um lado avanca no ciclo inteiro: o animal manca. Reestruture em duas "
                "meias-passadas trocando a perna de guia, com a segunda metade espelhando a primeira."
            )
        elif balance > LEAD_BALANCE_TOLERANCE:
            warnings.append(
                f"os dois lados avancam, mas de forma desigual (desequilibrio {balance}); "
                f"uma passada fica mais curta que a outra."
            )

    print()
    print("=== movimento entre frames ===")
    total = 0.0
    for i in range(len(masks)):
        j = (i + 1) % len(masks)
        union = masks[i] | masks[j]
        change = len(masks[i] ^ masks[j]) / len(union) if union else 0
        total += change
        print(f"  {frames[i][0]} -> {frames[j][0]}: {change:6.1%}")
    if masks:
        avg = total / len(masks)
        print(f"  media: {avg:.1%}")
        if avg < MOTION_WARN:
            warnings.append(f"pouca variacao entre frames ({avg:.1%}); a animacao vai parecer parada.")

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

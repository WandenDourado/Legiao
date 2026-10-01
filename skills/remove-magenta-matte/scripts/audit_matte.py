#!/usr/bin/env python3
"""Acha rosa de matte que sobrou num PNG RGBA ja recortado — e mostra onde.

    python3 skills/remove-magenta-matte/scripts/audit_matte.py a.png [b.png ...] \
        [--overlay-dir work/<pasta>/auditoria]

Para cada imagem imprime tres numeros e a lista dos maiores focos:

  borda   rosa na faixa de 8 px do contorno (franja de antialias)
  miolo   rosa que DESTOA da vizinhanca, longe da borda (matte preso em fresta
          de cabelo, vao entre flechas) — o que escapou da primeira limpeza
  violeta fracao do miolo que e violeta/magenta pintado. Acima de ~1% o
          personagem e "banhado de violeta" (Necromante) e os dois numeros de
          cima incluem luz pintada: ai so o overlay decide.

Com --overlay-dir grava <nome>_audit.png: a figura sobre VERDE (o rosa salta
no verde e some no cinza escuro — conferir so no fundo escuro deixou passar o
cabelo da Sacerdotisa), com circulo amarelo em cada foco de miolo e pixels de
borda pintados de ciano.

Numero nao e veredito. Compare o elenco inteiro de uma vez: o personagem fora
da curva e o sinal (foi assim que o Necromante denunciou a regra que comia a
chama dele).
"""
import argparse
import os
import sys

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage


def hue(R, G, B):
    mx = np.maximum(np.maximum(R, G), B); mn = np.minimum(np.minimum(R, G), B)
    d = mx - mn + 1e-6
    return np.where(mx == R, ((G - B) / d) % 6,
                    np.where(mx == G, (B - R) / d + 2, (R - G) / d + 4)) * 60


def audit(path, overlay_dir=None):
    a = np.asarray(Image.open(path).convert("RGBA")).astype(np.float64)
    R, G, B, A = a[..., 0], a[..., 1], a[..., 2], a[..., 3]
    h = hue(R, G, B)
    s = np.minimum(R - G, B - G)
    dist = ndimage.distance_transform_edt(A >= 250)
    violeta = (B - R) > 0.30 * B + 10
    rosa = (A > 20) & (h >= 280) & (h <= 350) & ~violeta

    borda = rosa & (dist <= 8) & (s > 25)
    op = A >= 128
    sm = ndimage.median_filter(np.where(op, s, -255), size=15)
    miolo = rosa & op & (dist > 8) & (s > 8) & ((s - np.maximum(sm, -60)) > 10)

    deep = (A >= 250) & (dist > 4)
    tint = deep & (h >= 250) & (h <= 345) & (s > 15)
    frac = tint.sum() / max(deep.sum(), 1)

    print(f"\n{path}")
    print(f"  borda {int(borda.sum()):6d}   miolo {int(miolo.sum()):6d}   "
          f"violeta pintado {frac*100:.1f}%" + ("  <- modo estrito" if frac > 0.01 else ""))
    lab, n = ndimage.label(ndimage.binary_dilation(miolo, iterations=2))
    focos = []
    if n:
        tam = ndimage.sum(miolo, lab, range(1, n + 1))
        objs = ndimage.find_objects(lab)
        for i in np.argsort(-tam)[:8]:
            sl = objs[i]
            m = miolo[sl] & (lab[sl] == i + 1)
            cor = np.median(a[sl][m][:, :3], 0).astype(int).tolist()
            focos.append(sl)
            print(f"    foco {int(tam[i]):4d} px  x{sl[1].start}-{sl[1].stop} "
                  f"y{sl[0].start}-{sl[0].stop}  cor {cor}")

    if overlay_dir:
        os.makedirs(overlay_dir, exist_ok=True)
        im = Image.open(path).convert("RGBA")
        fundo = Image.new("RGBA", im.size, (40, 90, 40, 255))
        fundo.alpha_composite(im)
        v = np.asarray(fundo.convert("RGB")).copy()
        v[borda] = [0, 255, 255]
        out = Image.fromarray(v)
        d = ImageDraw.Draw(out)
        for sl in ndimage.find_objects(lab) if n else []:
            cy = (sl[0].start + sl[0].stop) / 2; cx = (sl[1].start + sl[1].stop) / 2
            r = max(sl[0].stop - sl[0].start, sl[1].stop - sl[1].start) / 2 + 10
            d.ellipse((cx - r, cy - r, cx + r, cy + r), outline=(255, 255, 0), width=3)
        nome = os.path.splitext(os.path.basename(path))[0]
        pai = os.path.basename(os.path.dirname(os.path.abspath(path)))
        dst = os.path.join(overlay_dir, f"{pai}_{nome}_audit.png")
        out.save(dst)
        print(f"  overlay: {dst}")
    return int(borda.sum()), int(miolo.sum()), frac


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("pngs", nargs="+")
    ap.add_argument("--overlay-dir")
    args = ap.parse_args()
    for p in args.pngs:
        audit(p, args.overlay_dir)
    return 0


if __name__ == "__main__":
    sys.exit(main())

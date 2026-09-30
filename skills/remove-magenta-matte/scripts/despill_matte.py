#!/usr/bin/env python3
"""Remove a franja rosa do matte magenta que sobrou na borda dos retratos.

    python3 skills/remove-magenta-matte/scripts/despill_matte.py entrada.png saida.png

O que sobrou nao e matte puro (esse o key_and_frame.py ja tirou): e matte
MISTURADO com a pintura — antialias da borda, reduzido pelo LANCZOS, e a luz de
rebote magenta que o gerador pintou nas pontas do cabelo. Por isso nao se apaga
nada: cada pixel pego mantem sua luminancia (o traco escuro continua escuro, o
brilho continua brilho) e troca so a COR pela cor da pintura logo ali dentro.

Um pixel so e mexido quando as quatro travas concordam:
  1. esta na faixa de borda (ate BAND px de um pixel nao opaco);
  2. tem assinatura de matte: R e B acima de G (o matte e 236,3,226);
  3. matiz entre HUE_LO e HUE_HI (magenta/rosa), e nao e violeta de azul
     dominante;
  4. o MIOLO vizinho nao e magenta/violeta. Se a pintura por dentro ja e dessa
     cor (chama do Necromante, dedos iluminados por ela), o rosa da borda e luz
     pintada, e fica.

Depois, um segundo passo (interior()) pega matte preso DENTRO da figura —
frestas do cabelo, vao entre flechas — por anomalia local de cor.

No anel mais externo (ate RING px) o alfa tambem cai pela fracao de matte
estimada: ali o rosa era cobertura de fundo, nao tinta.
"""
import sys, numpy as np
from PIL import Image
from scipy import ndimage

M = (236.0, 3.0, 226.0)   # matte medido nas geracoes (README de work/portraits)
BAND = 8                  # px de faixa de borda onde se procura franja
RING = 2.0                # anel externo em que o alfa tambem e corrigido
HUE_LO, HUE_HI = 265.0, 352.0
MIN_S = 6.0               # excesso minimo de R e B sobre G
DEEP = 4                  # px para dentro da silhueta que contam como miolo
CTX = 61                  # janela (px) em que se olha o miolo vizinho
CTX_FRAC = 0.12           # fracao de miolo magenta/violeta acima da qual a borda e pintura
REF_SIGMA = 4.0           # raio da media de cor do miolo usada como referencia
INT_WIN = 15              # janela da mediana local no passo de miolo
INT_EXC = 10              # quanto o pixel precisa destoar da vizinhanca
STRICT_TINT = 0.01        # acima desta fracao de miolo violeta -> modo estrito
STRICT_G = 25             # no modo estrito, so matte escuro (verde quase zero)


def hue(R, G, B):
    mx = np.maximum(np.maximum(R, G), B); mn = np.minimum(np.minimum(R, G), B)
    d = mx - mn + 1e-6
    return np.where(mx == R, ((G - B) / d) % 6, np.where(mx == G, (B - R) / d + 2, (R - G) / d + 4)) * 60


def lum(R, G, B):
    return 0.299 * R + 0.587 * G + 0.114 * B


def clean(im):
    a = np.asarray(im.convert("RGBA")).astype(np.float64)
    R, G, B, A = a[..., 0], a[..., 1], a[..., 2], a[..., 3]
    h = hue(R, G, B)
    s = np.minimum(R - G, B - G)
    dist = ndimage.distance_transform_edt(A >= 250)
    band = (A > 0) & (dist <= BAND)

    deep = (A >= 250) & (dist > DEEP)
    tint = deep & (h >= 250) & (h <= 345) & (s > 15)
    frac = ndimage.uniform_filter(tint.astype(float), CTX) / np.maximum(
        ndimage.uniform_filter(deep.astype(float), CTX), 1e-6)
    pintado = frac > CTX_FRAC
    violeta = (B - R) > 0.30 * B + 10
    cand = band & (h >= HUE_LO) & (h <= HUE_HI) & (s > MIN_S) & ~violeta
    hit = cand & ~pintado

    # fracao de matte na mistura, e a cor que sobra tirando-a
    k = np.clip(np.minimum((R - G) / (M[0] - M[1]), (B - G) / (M[2] - M[1])), 0, 0.95)
    one = 1 - k
    L = np.clip(lum((R - k * M[0]) / one, (G - k * M[1]) / one, (B - k * M[2]) / one), 0, 255)

    # cor de referencia: media do miolo limpo ao redor; onde nao ha miolo perto,
    # o pixel de miolo mais proximo
    ok = deep & ~tint
    w = ndimage.gaussian_filter(ok.astype(float), REF_SIGMA)
    ref = np.stack([ndimage.gaussian_filter(a[..., c] * ok, REF_SIGMA) for c in range(3)], -1)
    ref = ref / np.maximum(w, 1e-6)[..., None]
    _, (iy, ix) = ndimage.distance_transform_edt(~ok, return_indices=True)
    near = a[..., :3][iy, ix]
    ref = np.where((w > 0.02)[..., None], ref, near)
    Lr = np.maximum(lum(ref[..., 0], ref[..., 1], ref[..., 2]), 1.0)

    novo = ref * (L / Lr)[..., None]
    # se escalar a referencia estourar um canal, mistura para cinza da mesma luminancia
    over = novo.max(-1)
    t = np.clip((over - 255) / np.maximum(over - L, 1e-6), 0, 1)[..., None]
    novo = novo * (1 - t) + L[..., None] * t

    out = a.copy()
    for c in range(3):
        out[..., c] = np.where(hit, novo[..., c], a[..., c])
    anel = hit & (dist <= RING)
    out[..., 3] = np.where(anel, A * (1 - k), A)

    out, n_int = interior(out, ok, tint.sum() / max(deep.sum(), 1), pintado)

    out = np.clip(np.rint(out), 0, 255).astype(np.uint8)
    out[out[..., 3] < 4] = 0
    return Image.fromarray(out, "RGBA"), int(hit.sum()), int((cand & pintado).sum()), n_int


def interior(a, ok, tint_global, pintado):
    """Segundo passo: matte preso DENTRO da figura, longe da borda.

    Frestas entre mechas de cabelo, vao entre flechas e corda do arco: bolsoes
    pequenos demais para o key_and_frame.py virar fundo, que ficaram opacos com
    a cor de matte misturada a da pintura. A faixa de borda do primeiro passo
    nao chega neles (achado no cabelo da Sacerdotisa, 20 px para dentro).

    O criterio e ANOMALIA LOCAL: o pixel tem bem mais excesso de R e B sobre G
    do que a mediana dos vizinhos opacos. Luz pintada e larga e acompanha a
    vizinhanca; matte preso e um ponto que destoa dela.

    Personagem cuja pintura e banhada de violeta (Necromante: ~8% do miolo,
    contra 0% nos outros quatro) entra em modo estrito — so conta pixel com
    assinatura de matte escuro (G < STRICT_G), porque ali rosa no peito e no
    pescoco e luz da chama.
    """
    R, G, B, A = a[..., 0], a[..., 1], a[..., 2], a[..., 3]
    h = hue(R, G, B)
    s = np.minimum(R - G, B - G)
    op = A >= 128
    sm = ndimage.median_filter(np.where(op, s, -255), size=INT_WIN)
    exc = s - np.maximum(sm, -60)
    violeta = (B - R) > 0.30 * B + 10
    hit = op & (h >= 280) & (h <= 350) & (s > 8) & (exc > INT_EXC) & ~violeta & ~pintado
    if tint_global > STRICT_TINT:
        hit &= G < STRICT_G
    # a borda antialiasada do bolsao carrega o mesmo tom, mais fraco
    hit |= ndimage.binary_dilation(hit, iterations=1) & op & (h >= 280) & (h <= 350) & (s > 4) & ~violeta & ~pintado
    if not hit.any():
        return a, 0

    k = np.clip(np.minimum((R - G) / (M[0] - M[1]), (B - G) / (M[2] - M[1])), 0, 0.95)
    one = 1 - k
    L = np.clip(lum((R - k * M[0]) / one, (G - k * M[1]) / one, (B - k * M[2]) / one), 0, 255)
    src = op & ~hit
    w = ndimage.gaussian_filter(src.astype(float), REF_SIGMA)
    ref = np.stack([ndimage.gaussian_filter(a[..., c] * src, REF_SIGMA) for c in range(3)], -1)
    ref = ref / np.maximum(w, 1e-6)[..., None]
    _, (iy, ix) = ndimage.distance_transform_edt(~src, return_indices=True)
    ref = np.where((w > 0.02)[..., None], ref, a[..., :3][iy, ix])
    Lr = np.maximum(lum(ref[..., 0], ref[..., 1], ref[..., 2]), 1.0)
    novo = ref * (L / Lr)[..., None]
    over = novo.max(-1)
    t = np.clip((over - 255) / np.maximum(over - L, 1e-6), 0, 1)[..., None]
    novo = novo * (1 - t) + L[..., None] * t
    out = a.copy()
    for c in range(3):
        out[..., c] = np.where(hit, novo[..., c], a[..., c])
    return out, int(hit.sum())


if __name__ == "__main__":
    src, dst = sys.argv[1], sys.argv[2]
    o, n, p, ni = clean(Image.open(src))
    o.save(dst, optimize=True)
    print(f"{src}: {n} px de franja limpos na borda, {ni} no miolo, {p} rosa/violeta protegidos como pintura")

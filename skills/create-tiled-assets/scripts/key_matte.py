#!/usr/bin/env python3
"""Key out a solid #FF00FF magenta matte from a generated sheet and despill
edges, producing the clean RGBA runtime atlas. Same convention as
create-character-sprites: magenta is generation-only, never shipped.

If the source already has meaningful transparency and no magenta background,
it is passed through unchanged.

Usage:
  python key_matte.py assets/tilesets/village_buildings_source.png --output assets/tilesets/village_buildings.png
  python key_matte.py source.png --output atlas.png --tolerance 90 --despill 2
"""
import argparse
import sys

from PIL import Image

MAGENTA = (255, 0, 255)


def magenta_distance(r, g, b):
    return abs(r - MAGENTA[0]) + g + abs(b - MAGENTA[2])


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("source")
    ap.add_argument("--output", required=True)
    ap.add_argument("--tolerance", type=int, default=120,
                    help="max |r-255|+g+|b-255| distance treated as matte (default 120)")
    ap.add_argument("--despill", type=int, default=2,
                    help="edge despill radius in px (default 2)")
    ap.add_argument("--hard", action="store_true",
                    help="disable soft chroma keying (only exact-matte pixels are removed)")
    ap.add_argument("--soft-spread", type=int, default=255,
                    help="divisor of the magenta cast when estimating coverage; 255 is the "
                         "value derived from the blend equation and should rarely change")
    ap.add_argument("--min-recovery", type=float, default=0.45,
                    help="lower bound of the un-blend divisor; prevents a green halo")
    ap.add_argument("--bleed", type=int, default=4,
                    help="passes of edge colour bleed into semi-transparent pixels (0 disables)")
    ap.add_argument("--solid-alpha", type=int, default=250,
                    help="alpha at or above which a pixel is treated as solid art")
    ap.add_argument("--cast-clamp", type=float, default=0.85,
                    help="share of the residual magenta cast removed after un-blending (0..1)")
    args = ap.parse_args()

    im = Image.open(args.source).convert("RGBA")
    w, h = im.size
    px = im.load()

    # Detect whether the sheet actually uses a magenta matte: sample corners.
    corners = [px[0, 0], px[w - 1, 0], px[0, h - 1], px[w - 1, h - 1]]
    matte_corners = sum(1 for (r, g, b, a) in corners
                        if a > 0 and magenta_distance(r, g, b) <= args.tolerance)
    already_alpha = sum(1 for (r, g, b, a) in corners if a == 0)
    if matte_corners == 0:
        if already_alpha >= 3:
            im.save(args.output)
            print(f"OK: source already transparent, copied unchanged -> {args.output}")
            return 0
        print("FAIL: no magenta matte detected at corners and background is not "
              "transparent — is this sheet on a different background color?", file=sys.stderr)
        return 1

    # Pass 1: key the matte.
    keyed = 0
    for y in range(h):
        for x in range(w):
            r, g, b, a = px[x, y]
            if a > 0 and magenta_distance(r, g, b) <= args.tolerance:
                px[x, y] = (0, 0, 0, 0)
                keyed += 1

    # Pass 1b: soft key. Generators paint a feathered edge that blends INTO the
    # matte, producing fully opaque pink pixels a distance key cannot reach —
    # that halo is the single most common defect in a keyed sheet.
    #
    # Derivation (matte m = (255, 0, 255), coverage k, art colour a):
    #   observed = a*(1-k) + m*k, so g = g_a*(1-k) and min(r,b) = min(r_a,b_a)*(1-k) + 255k.
    # Project art satisfies min(r_a,b_a) <= g_a (brown, green, grey, blue and
    # white all do; magenta in the art is forbidden by the prompt), therefore
    #   k <= (min(r,b) - g) / 255,
    # which is why the divisor is 255. Using a small divisor overestimates k and
    # the un-blend then explodes the green channel into a bright halo.
    softened = 0
    if not args.hard and args.soft_spread > 0:
        for y in range(h):
            for x in range(w):
                r, g, b, a = px[x, y]
                if a == 0:
                    continue
                cast = min(r, b) - g
                if cast <= 0:
                    continue
                k = min(1.0, cast / args.soft_spread)
                if k <= 0.02:
                    continue
                if k >= 0.98:
                    px[x, y] = (0, 0, 0, 0)
                else:
                    inv = 1.0 - k
                    # O ganho da recuperacao e limitado: dividir por inv perto de
                    # zero amplifica o verde do pixel quase-matte e cria um halo
                    # verde em volta de cada peca (visto nos toppings).
                    inv_c = max(inv, args.min_recovery)
                    nr = min(255, max(0, int((r - 255 * k) / inv_c)))
                    ng = min(255, max(0, int(g / inv_c)))
                    nb = min(255, max(0, int((b - 255 * k) / inv_c)))
                    # Un-blending recovers most of the colour, but a wide feather
                    # leaves a mauve ring: the residual cast is clamped, since no
                    # legitimate art colour in this project sits above green in
                    # BOTH red and blue.
                    residual = min(nr, nb) - ng
                    if residual > 0:
                        nr -= int(residual * args.cast_clamp)
                        nb -= int(residual * args.cast_clamp)
                    px[x, y] = (nr, ng, nb, int(a * inv))
                softened += 1

    # Pass 2: despill — pixels adjacent to transparency with a magenta cast
    # (r and b clearly above g) get their cast pulled toward neutral and a
    # softened alpha, killing the pink fringe without eroding solid art.
    alpha = im.getchannel("A").load()
    fringe = []
    rad = max(0, args.despill)
    if rad:
        for y in range(h):
            for x in range(w):
                if alpha[x, y] == 0:
                    continue
                near_hole = any(0 <= x + dx < w and 0 <= y + dy < h and alpha[x + dx, y + dy] == 0
                                for dx in range(-rad, rad + 1) for dy in range(-rad, rad + 1))
                if not near_hole:
                    continue
                r, g, b, a = px[x, y]
                if r > g + 40 and b > g + 40:  # magenta-cast fringe
                    fringe.append((x, y, r, g, b, a))
        for x, y, r, g, b, a in fringe:
            cast = min(r - g, b - g)
            px[x, y] = (r - cast // 2, g, b - cast // 2, min(a, max(0, a - cast // 2)))

    # Pass 3: edge colour bleed. The feather the generator painted is a real
    # mixture of art and matte, so un-blending can only approximate it and a
    # faint mauve ring survives. Standard atlas fix: keep the alpha ramp but
    # replace the colour of semi-transparent pixels with the colour of nearby
    # solid art, so the fade carries the piece's own hue.
    bled = 0
    if args.bleed > 0:
        alpha = [[px[x, y][3] for x in range(w)] for y in range(h)]
        known = [[a >= args.solid_alpha for a in row] for row in alpha]
        for _ in range(args.bleed):
            updates = []
            for y in range(h):
                for x in range(w):
                    if alpha[y][x] == 0 or known[y][x]:
                        continue
                    acc, n = [0, 0, 0], 0
                    for dy in (-1, 0, 1):
                        for dx in (-1, 0, 1):
                            nx, ny = x + dx, y + dy
                            if 0 <= nx < w and 0 <= ny < h and known[ny][nx]:
                                c = px[nx, ny]
                                acc[0] += c[0]; acc[1] += c[1]; acc[2] += c[2]; n += 1
                    if n:
                        updates.append((x, y, acc[0] // n, acc[1] // n, acc[2] // n))
            if not updates:
                break
            for x, y, r, g, b in updates:
                px[x, y] = (r, g, b, alpha[y][x])   # alpha preservado
                known[y][x] = True
            bled += len(updates)

    # Pass 4: global cast clamp. Sprayed/dusted elements keep a pink tint well
    # inside the artwork, where no edge rule reaches. Project art never has BOTH
    # red and blue above green (browns, greens, greys, blues and whites all
    # fail that test), so any pixel that does is matte contamination.
    clamped = 0
    if args.cast_clamp > 0:
        for y in range(h):
            for x in range(w):
                r, g, b, a = px[x, y]
                if a == 0:
                    continue
                residual = min(r, b) - g
                if residual <= 2:
                    continue
                cut = int(residual * args.cast_clamp)
                px[x, y] = (max(0, r - cut), g, max(0, b - cut), a)
                clamped += 1

    im.save(args.output)
    print(f"OK: clamped {clamped} tinted px, keyed {keyed} matte px, softened {softened} blended px, "
          f"bled {bled} edge px, despilled {len(fringe)} fringe px -> {args.output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

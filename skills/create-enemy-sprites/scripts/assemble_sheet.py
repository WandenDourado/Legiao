#!/usr/bin/env python3
"""Monta a sheet final e o manifesto a partir dos frames normalizados.

A sheet radial e uma tira horizontal de frames quadrados: uma linha, N colunas.
O manifesto registra o que o runtime precisa (tamanho do frame, ancora, tempo
por frame) e o que uma auditoria futura precisa (o raio maximo medido, o modo
de ajuste usado, as decisoes que nao sao obvias no PNG).

Uso:
  python assemble_sheet.py frames/ --out assets/sprites/enemies/wolf \
      --name wolf --frame-time 0.07 --animation run \
      --note "duas meias-passadas alternando a perna de guia"
"""
import argparse
import json
import math
from pathlib import Path

from PIL import Image

ALPHA_MIN = 24


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("frames_dir")
    ap.add_argument("--out", required=True, help="diretorio de destino do asset")
    ap.add_argument("--name", required=True, help="nome do inimigo, vira <name>.png/.json")
    ap.add_argument("--frame-time", type=float, required=True)
    ap.add_argument("--animation", default="idle")
    ap.add_argument("--fit", choices=("stretch", "uniform"), default="uniform")
    ap.add_argument("--note", action="append", default=[], help="pode repetir")
    args = ap.parse_args()

    files = sorted(Path(args.frames_dir).glob("frame_*.png"))
    if not files:
        raise SystemExit(f"nenhum frame_*.png em {args.frames_dir}")

    frames = [Image.open(f).convert("RGBA") for f in files]
    fw, fh = frames[0].size
    for f, path in zip(frames, files):
        if f.size != (fw, fh):
            raise SystemExit(f"{path.name} tem {f.size}, esperado {(fw, fh)}")

    sheet = Image.new("RGBA", (fw * len(frames), fh), (0, 0, 0, 0))
    for i, f in enumerate(frames):
        sheet.paste(f, (i * fw, 0))

    outdir = Path(args.out)
    outdir.mkdir(parents=True, exist_ok=True)
    png = outdir / f"{args.name}.png"
    sheet.save(png)

    # raio maximo real, para o manifesto registrar a folga ate o circulo inscrito
    px = sheet.load()
    cx, cy = fw / 2, fh / 2
    max_r = 0.0
    for i in range(len(frames)):
        for y in range(fh):
            for x in range(fw):
                if px[i * fw + x, y][3] >= ALPHA_MIN:
                    max_r = max(max_r, math.hypot(x - cx, y - cy))

    manifest = {
        "image": png.name,
        "mode": "radial",
        "frame_width": fw,
        "frame_height": fh,
        "origin": "center",
        "anchor": {"x": fw // 2, "y": fh // 2},
        "front": "up",
        "rotation": "runtime",
        "inscribed_radius": min(fw, fh) // 2,
        "measured_max_radius": round(max_r, 1),
        "fit_mode": args.fit,
        "animations": {
            args.animation: {
                "frames": len(frames),
                "frame_time_seconds": args.frame_time,
                "loop": True,
            }
        },
        "notes": [
            "Radial mode: one top-down view rotated at runtime toward the velocity vector.",
            "The sprite's front points to the top of the frame; rotation is atan2(vy,vx)+90 degrees.",
            "Geometry was normalized by scripts/normalize_radial.py, not requested from the "
            "image generator, which ignores numeric geometry instructions.",
        ]
        + args.note,
    }
    (outdir / f"{args.name}.json").write_text(json.dumps(manifest, indent=2) + "\n")

    print(f"{png}  {sheet.size}  {len(frames)} frames")
    print(f"raio maximo {max_r:.1f} / inscrito {min(fw, fh) // 2}")


if __name__ == "__main__":
    main()

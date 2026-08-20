#!/usr/bin/env python3
"""One-command repair pipeline: sliced frames in, gated frames out.

This is the standard post-slice step. It replaces the hand-assembled chain
(scale_fit -> auto_repair -> normalize_frames -> auto_repair -> validate) whose
many flags (--frames-per-direction, --anchor-x, --baseline, --alpha-threshold,
2x vs 1x values...) repeatedly stalled executing agents. Everything is derived
from the frames themselves:

  - frame size read from the PNGs; supersample scale = height/192
  - anchor-x = 64*scale, baseline = 186*scale, alpha threshold 16 (1x) / 40 (2x)
  - frame count auto-detected: 8 = full direction, 4 = partial batch (pass
    --batch 1|2 so refusal verdicts target the right batch; default 1)

On a validate refusal it runs diagnose_direction itself and, when the verdict is
a FREE action (scale_fit / flatten / harmonize / trim_overhang), applies it and
re-runs the chain (max 3 rounds). Anything costlier is printed as the final
verdict for the agent to execute (patch_frame / reorder_patch / regen_*).

Usage:
  python repair_chain.py --input-root <root-with-<D>-subdir> --output-root OUT \
      --direction SW [--batch 2] [--target-height-ratio 0.84]

Exit codes: 0 = accepted (final frames in OUT/<D>, score report printed);
2 = not free-fixable, the printed diagnose verdict is the next action.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path

from PIL import Image

SCRIPTS = Path(__file__).resolve().parent
FREE_ACTIONS = {"scale_fit", "flatten", "harmonize", "trim_overhang", "accept", "auto_repair"}


def run(script: str, *extra: str) -> subprocess.CompletedProcess:
    return subprocess.run([sys.executable, str(SCRIPTS / script), *extra], capture_output=True, text=True)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input-root", required=True, type=Path)
    parser.add_argument("--output-root", required=True, type=Path)
    parser.add_argument("--direction", required=True)
    parser.add_argument("--batch", type=int, choices=(1, 2),
                        help="Partial-batch label for diagnose verdicts (default 1 when 4 frames).")
    parser.add_argument("--target-height-ratio", type=float, default=0.84)
    parser.add_argument("--acceptance-threshold", type=float, default=0.85)
    parser.add_argument("--max-rounds", type=int, default=3)
    args = parser.parse_args()

    direction = args.direction.strip().upper()
    in_dir = args.input_root / direction
    paths = sorted(in_dir.glob("*.png"))
    if len(paths) not in (4, 8):
        print(f"{in_dir}: expected 4 or 8 frames, found {len(paths)}")
        return 1
    count = len(paths)
    with Image.open(paths[0]) as image:
        width, height = image.size
    scale = height / 192.0
    anchor_x = round(64 * scale)
    baseline = round(186 * scale)
    alpha = "40" if scale > 1 else "16"
    border = str(max(1, round(scale)))
    tol = str(max(1, round(scale)))
    batch = args.batch if args.batch is not None else (1 if count == 4 else None)

    geometry = ["--frame-width", str(width), "--frame-height", str(height),
                "--frames-per-direction", str(count)]
    anchoring = ["--anchor-x", str(anchor_x), "--baseline", str(baseline), "--alpha-threshold", alpha]

    stage_root = Path(tempfile.mkdtemp(prefix=f"repair-{direction}-"))
    current = args.input_root
    step = 0

    def apply(script: str, *extra: str, tag: str) -> tuple[bool, str]:
        nonlocal current, step
        step += 1
        out = stage_root / f"{step:02d}-{tag}"
        result = run(script, "--input-root", str(current), "--output-root", str(out),
                     "--directions", direction, *geometry, *extra)
        message = (result.stdout + result.stderr).strip()
        if result.returncode != 0:
            return False, message
        current = out
        return True, message

    pending: list[tuple[str, tuple[str, ...], str]] = [("scale_fit.py", tuple(anchoring), "scale")]
    verdict = None
    for round_index in range(args.max_rounds):
        for script, extra, tag in pending:
            ok, message = apply(script, *extra, tag=tag)
            print(f"[{tag}] {message.splitlines()[-1] if message else 'ok'}")
            if not ok and script != "scale_fit.py":
                break
            if not ok:
                break  # scale_fit refusal falls through to diagnose below
        pending = []
        for script, extra, tag in (("auto_repair.py", ("--border-pixels", border, "--matte-threshold", "140",
                                                       "--alpha-threshold", alpha), "repair"),
                                   ("normalize_frames.py", (*anchoring, "--max-shift", str(round(24 * scale))), "anchor"),
                                   ("auto_repair.py", ("--border-pixels", border, "--matte-threshold", "140",
                                                       "--alpha-threshold", alpha), "repair2")):
            ok, message = apply(script, *extra, tag=tag)
            print(f"[{tag}] {message.splitlines()[-1] if message else 'ok'}")
            if not ok:
                break

        validate = run("validate_frames.py", *[str(p) for p in sorted((current / direction).glob("*.png"))],
                       "--frame-width", str(width), "--frame-height", str(height),
                       "--require-alpha", "--require-transparent", "--require-clear-border",
                       "--border-pixels", border, "--reject-magenta", "--magenta-threshold", "140",
                       "--check-baseline", "--baseline-tolerance", tol, "--expected-baseline", str(baseline),
                       "--check-center", "--center-tolerance", tol, "--expected-center", str(anchor_x),
                       "--min-foreground-height-ratio", "0.80", "--max-foreground-height-ratio", "0.92",
                       "--score", "--acceptance-threshold", str(args.acceptance_threshold))
        try:
            summary = json.loads(validate.stdout)
        except json.JSONDecodeError:
            print(validate.stdout + validate.stderr)
            return 1
        print(f"[gate] score {summary['score']} -> {summary['recommendation']}")
        if validate.returncode == 0:
            final = args.output_root / direction
            final.mkdir(parents=True, exist_ok=True)
            for path in sorted((current / direction).glob("*.png")):
                (final / path.name).write_bytes(path.read_bytes())
            print(json.dumps(summary, indent=2))
            print(f"OK: {direction} accepted; final frames in {final}")
            return 0

        diagnose_args = [str(current / direction), "--frame-width", str(width), "--frame-height", str(height),
                         "--anchor-x", str(anchor_x), "--alpha-threshold", alpha]
        if batch is not None:
            diagnose_args += ["--batch", str(batch)]
        diagnosis = run("diagnose_direction.py", *diagnose_args)
        try:
            verdict = json.loads(diagnosis.stdout)
        except json.JSONDecodeError:
            print(diagnosis.stdout + diagnosis.stderr)
            return 1
        action = verdict["action"].split()[0]
        print(f"[diagnose] {verdict['action']} | {verdict['reason']}")
        if action == "accept":
            # Geometry fine but the score still refuses: cosmetic residue only —
            # loop once more (auto_repair runs again); if it persists, surface it.
            continue
        if action not in FREE_ACTIONS:
            print(json.dumps(verdict, indent=2))
            print(f"NOT free-fixable: execute '{verdict['action']}' (see reason above).")
            return 2
        if action == "trim_overhang":
            pending = [("trim_overhang.py", (*anchoring, "--target-height-ratio", str(args.target_height_ratio)), "trim"),
                       ("scale_fit.py", tuple(anchoring), "scale")]
        elif action == "flatten":
            pending = [("scale_fit.py", (*anchoring, "--flatten-heights"), "flatten")]
        elif action == "harmonize":
            pending = [("scale_fit.py", (*anchoring, "--harmonize-halves"), "harmonize")]
        else:
            pending = [("scale_fit.py", tuple(anchoring), "scale")]

    print("Max repair rounds reached without acceptance; run diagnose_direction manually.")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Verify a Legiao sprite export against the renderer constants and asset path."""

from __future__ import annotations

import argparse
import re
from pathlib import Path


def constant(source: str, name: str) -> int:
    match = re.search(rf"\b{re.escape(name)}\s*=\s*(\d+)", source)
    if not match:
        raise ValueError(f"could not find {name}")
    return int(match.group(1))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--project-root", type=Path, default=Path("."))
    parser.add_argument("--renderer-source", type=Path, default=Path("internal/entity/player_sprite.go"))
    parser.add_argument("--asset", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--columns", required=True, type=int)
    parser.add_argument("--directions", required=True)
    args = parser.parse_args()

    source_path = args.project_root / args.renderer_source
    try:
        source = source_path.read_text(encoding="utf-8")
        loaded_asset = re.search(r'assets\.Path\("([^"]+)"\)', source).group(1)
        row_order = re.search(r"Row order:\s*([^\.]+)", source).group(1).replace(" ", "")
        expected = {
            "frame width": constant(source, "WizardFrameWidth"),
            "frame height": constant(source, "WizardFrameHeight"),
            "columns": constant(source, "WizardColumns"),
            "rows": constant(source, "WizardRows"),
        }
    except (AttributeError, OSError, ValueError) as exc:
        raise SystemExit(f"Could not read renderer contract: {exc}") from exc

    directions = ",".join(item.strip().upper() for item in args.directions.split(",") if item.strip())
    supplied = {"frame width": args.frame_width, "frame height": args.frame_height, "columns": args.columns, "rows": len(directions.split(","))}
    failures = [f"{name}: renderer={expected[name]}, export={value}" for name, value in supplied.items() if expected[name] != value]
    if Path(loaded_asset).as_posix() != Path(args.asset).as_posix():
        failures.append(f"asset: renderer={loaded_asset}, export={args.asset}")
    if row_order != directions:
        failures.append(f"directions: renderer={row_order}, export={directions}")
    if failures:
        print("Renderer preflight failed:")
        print("\n".join(f"- {failure}" for failure in failures))
        return 1
    print(f"OK: renderer accepts {args.frame_width}x{args.frame_height}, {directions}, asset {args.asset}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

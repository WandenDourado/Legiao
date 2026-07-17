#!/usr/bin/env python3
"""Verify a Legiao sprite export against a registered character definition."""

from __future__ import annotations

import argparse
import re
from pathlib import Path


def character_definition(source: str, character_id: str) -> dict[str, str]:
    constants = {
        name: value
        for name, value in re.findall(r'(Char\w+)\s+CharacterType\s*=\s*"([^"]+)"', source)
    }
    type_name = next((name for name, value in constants.items() if value == character_id), None)
    if not type_name:
        raise ValueError(f"could not find character type for {character_id!r}")

    pattern = r"RegisterCharacter\(CharacterDef\s*\{(.*?)\}\s*\)"
    for block in re.findall(pattern, source, re.DOTALL):
        if re.search(rf"\bType:\s*{re.escape(type_name)}\b", block):
            fields = {
                key: quoted or numeric
                for key, quoted, numeric in re.findall(
                    r'(SpritePath|FrameWidth|FrameHeight|Columns|Rows):\s*(?:"([^"]+)"|(\d+))', block
                )
            }
            return fields
    raise ValueError(f"could not find CharacterDef for {character_id!r}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--project-root", type=Path, default=Path("."))
    parser.add_argument("--character-source", type=Path, default=Path("internal/entity/character.go"))
    parser.add_argument("--character-id", required=True)
    parser.add_argument("--asset", required=True)
    parser.add_argument("--frame-width", required=True, type=int)
    parser.add_argument("--frame-height", required=True, type=int)
    parser.add_argument("--columns", required=True, type=int)
    parser.add_argument("--directions", required=True)
    args = parser.parse_args()

    try:
        source = (args.project_root / args.character_source).read_text(encoding="utf-8")
        definition = character_definition(source, args.character_id)
        expected = {
            "frame width": int(definition["FrameWidth"]),
            "frame height": int(definition["FrameHeight"]),
            "columns": int(definition["Columns"]),
            "rows": int(definition["Rows"]),
        }
    except (KeyError, OSError, ValueError) as exc:
        raise SystemExit(f"Could not read character contract: {exc}") from exc

    directions = ",".join(item.strip().upper() for item in args.directions.split(",") if item.strip())
    supplied = {
        "frame width": args.frame_width,
        "frame height": args.frame_height,
        "columns": args.columns,
        "rows": len(directions.split(",")),
    }
    failures = [f"{name}: registry={expected[name]}, export={value}" for name, value in supplied.items() if expected[name] != value]
    if Path(definition["SpritePath"]).as_posix() != Path(args.asset).as_posix():
        failures.append(f"asset: registry={definition['SpritePath']}, export={args.asset}")
    if directions != "S,SW,W,N,NW":
        failures.append(f"directions: renderer=S,SW,W,N,NW, export={directions}")
    if failures:
        print("Renderer preflight failed:")
        print("\n".join(f"- {failure}" for failure in failures))
        return 1
    print(f"OK: {args.character_id} accepts {args.frame_width}x{args.frame_height}, {directions}, asset {args.asset}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

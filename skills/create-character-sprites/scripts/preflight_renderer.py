#!/usr/bin/env python3
"""Verify a Legiao sprite export against a registered character definition."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

try:
    from PIL import Image
except ImportError as exc:  # pragma: no cover
    raise SystemExit("Pillow is required. Install with: python -m pip install pillow") from exc


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
    parser.add_argument("--metadata", type=Path, help="Optional sheet metadata to validate the mirrored-direction contract.")
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
    asset_path = Path(args.asset)
    if not asset_path.is_absolute():
        asset_path = args.project_root / asset_path
    try:
        with Image.open(asset_path) as asset:
            expected_size = (args.frame_width * args.columns, args.frame_height * len(directions.split(",")))
            if asset.size != expected_size:
                failures.append(f"asset dimensions: expected {expected_size[0]}x{expected_size[1]}, got {asset.width}x{asset.height}")
            if "A" not in asset.mode:
                failures.append(f"asset mode: expected an alpha channel, got {asset.mode}")
            elif asset.convert("RGBA").getchannel("A").getextrema()[0] >= 255:
                failures.append("asset alpha: expected transparent pixels")
    except OSError as exc:
        failures.append(f"asset: cannot open {asset_path}: {exc}")
    if args.metadata:
        try:
            metadata = json.loads(args.metadata.read_text(encoding="utf-8"))
            if metadata.get("image") != Path(args.asset).name:
                failures.append(f"metadata image: expected {Path(args.asset).name}, got {metadata.get('image')}")
            if metadata.get("frame_width") != args.frame_width:
                failures.append(f"metadata frame_width: expected {args.frame_width}, got {metadata.get('frame_width')}")
            if metadata.get("frame_height") != args.frame_height:
                failures.append(f"metadata frame_height: expected {args.frame_height}, got {metadata.get('frame_height')}")
            if metadata.get("origin") != "foot-center":
                failures.append(f"metadata origin: expected foot-center, got {metadata.get('origin')}")
            if metadata.get("directions") != ["S", "SW", "W", "N", "NW"]:
                failures.append("metadata directions must be S,SW,W,N,NW")
            if metadata.get("mirror_safe") is not True:
                failures.append("metadata must declare mirror_safe=true")
            if metadata.get("mirrored_directions") != {"E": "W", "SE": "SW", "NE": "NW"}:
                failures.append("metadata mirrored_directions must map E/W, SE/SW, and NE/NW")
            expected_walk_cycle = {
                "frame_0": "left-foot contact",
                "frame_2": "right foot passes with left planted",
                "frame_4": "right-foot contact",
                "frame_6": "left foot passes with right planted",
            }
            if metadata.get("walk_cycle") != expected_walk_cycle:
                failures.append("metadata must declare the alternating left/right walk cycle")
            anchor = metadata.get("anchor")
            expected_anchor = {"x": args.frame_width // 2, "y": args.frame_height - 6}
            if anchor != expected_anchor:
                failures.append(f"metadata anchor: expected {expected_anchor}, got {anchor}")
            walk = metadata.get("animations", {}).get("walk", {})
            if walk.get("frames_per_direction") != args.columns:
                failures.append(f"metadata walk frames_per_direction: expected {args.columns}, got {walk.get('frames_per_direction')}")
            expected_rows = {direction: index for index, direction in enumerate(["S", "SW", "W", "N", "NW"])}
            if walk.get("rows") != expected_rows:
                failures.append(f"metadata walk rows: expected {expected_rows}, got {walk.get('rows')}")
        except (OSError, ValueError, TypeError) as exc:
            failures.append(f"metadata: {exc}")
    if failures:
        print("Renderer preflight failed:")
        print("\n".join(f"- {failure}" for failure in failures))
        return 1
    print(f"OK: {args.character_id} accepts {args.frame_width}x{args.frame_height}, {directions}, asset {args.asset}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

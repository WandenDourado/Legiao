#!/usr/bin/env python3
"""Append one gate/decision event to the character's manifest (JSON Lines).

The manifest is the audit trail that lets a later analysis reconstruct exactly what
happened without the chat transcript: every generation, gate verdict, repair, visual
check, and accept/regen decision, in order, with timestamps. Call it after EVERY
pipeline decision — it costs nothing and preserves the information that image
overwrites and report renames destroy.

Example:
  python log_attempt.py --root work/mago-celeste --direction W --attempt 2 \
      --stage validate --verdict repair --note "height 0.93 -> scale_fit" \
      --data score=0.65 recommendation=repair
"""

from __future__ import annotations

import argparse
import datetime as _dt
import json
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True, type=Path, help="Character work dir, e.g. work/<character-id>.")
    parser.add_argument("--direction", required=True, help="S, SW, W, N, NW, or 'model' for the model-sheet phase.")
    parser.add_argument("--attempt", required=True, type=int)
    parser.add_argument("--stage", required=True,
                        help="generate | slice | normalize | validate | scale_fit | auto_repair | structural | visual_review | accept | regen")
    parser.add_argument("--verdict", required=True,
                        help="ok | fail | repair | regen | accept | suspect | indeterminate | refused")
    parser.add_argument("--note", default="", help="Free-text detail (what failed, what was decided, prompt tweak).")
    parser.add_argument("--data", nargs="*", default=[], help="Extra key=value pairs to record.")
    args = parser.parse_args()

    entry = {
        "ts": _dt.datetime.now().isoformat(timespec="seconds"),
        "direction": args.direction.upper(),
        "attempt": args.attempt,
        "stage": args.stage,
        "verdict": args.verdict,
        "note": args.note,
    }
    for pair in args.data:
        key, _, value = pair.partition("=")
        if key:
            entry[key] = value
    args.root.mkdir(parents=True, exist_ok=True)
    manifest = args.root / "manifest.jsonl"
    with manifest.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(entry, ensure_ascii=False) + "\n")
    print(f"logged: {entry['direction']} attempt {entry['attempt']} {entry['stage']}={entry['verdict']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

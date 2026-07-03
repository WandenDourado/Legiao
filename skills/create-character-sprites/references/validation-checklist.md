# Validation Checklist

Use this checklist before accepting a direction or final sheet.

## Technical

- PNG files exist and are readable.
- Every frame has the same pixel dimensions.
- Transparent background is present when required.
- No labels, grid text, background panels, or cropped body parts.
- Frame count matches the requested animation.
- Final sheet width equals `frame_width * columns`; height equals `frame_height * rows`.

## Animation

- Feet share a stable floor baseline.
- Head height and body scale do not jump between frames.
- Walk cycle alternates legs clearly.
- Robe, sleeves, hair, staff, and glow move plausibly without becoming new objects.
- Idle frame or frame 0 is usable as a standing pose.

## Direction

- Direction is readable at gameplay size.
- Cardinal directions are distinct.
- Diagonals are not just front/back frames with shifted eyes.
- Mirrored rows do not break asymmetrical details.

## Character Consistency

- Costume palette remains stable.
- Hood, robe trim, belt, pouch, boots, and staff stay recognizable.
- Face age and expression stay consistent.
- Magic glow stays attached to the staff/crystal and does not become background decoration.

## Legiao Integration

- Prefer a market-oriented sprite spec first, such as `128x192` with `6` walk frames and transparent PNG, then adapt code/import settings to that spec.
- Use `165x246`, `6` frames only if replacing the current wizard sheet directly.
- Current engine rows are `N`, `S`, `W`, `SW`, `NW`; `E`, `SE`, `NE` are mirrored from `W`, `SW`, `NW` respectively.
- A full 8-direction source sheet should include metadata so code changes can map rows intentionally.

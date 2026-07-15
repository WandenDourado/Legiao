# Character Sprite Guide

## Prompt

Use the reference as the identity source. Record hair, clothing, palette, props, body-side asymmetries, scale, and prohibited changes. Describe style with observable traits such as painterly volume, soft gradients, readable silhouette, restrained outlines, and game-scale detail.

For a `128x192` frame, request an unguttered 2 rows x 4 columns image at `1024x768` or larger (4:3). The second grid row becomes frames 5-8, so it must use the same baseline, pivot, scale, and padding as the first. Require a solid `#FF00FF` matte only; prohibit that color on the character.

```text
Create exactly 8 frames for one walk direction: [DIRECTION].
Character: [approved bible and reference]. Preserve [asymmetric prop] on its body side.
Layout: unguttered 2 rows x 4 columns, reading left-to-right then top-to-bottom, 4:3 canvas, identical foot baseline and pivot in all cells.
Background: solid #FF00FF matte only. No text, borders, panels, duplicate limbs, cropped parts, cast shadows, or pixel-art conversion.
Walk plan: [PASTE THE EIGHT-POSE PLAN].
Secondary motion: [staff / robe / hair instructions]. Keep every prop attached and continuous between frames.
```

## Eight-Pose Walk Plan

Use body-left/body-right, never screen-left/screen-right.

1. Contact A: left foot forward, right foot back; left heel contacts.
2. Down A: left foot supports weight; hips lower.
3. Passing A: right leg passes under the body; left foot remains planted.
4. Up A: body rises; right leg advances toward contact.
5. Contact B: right foot forward, left foot back; right heel contacts.
6. Down B: right foot supports weight; hips lower.
7. Passing B: left leg passes under the body; right foot remains planted.
8. Up B: body rises and returns toward pose 1 without duplicating it.

Arms oppose legs. Keep the staff hand grip stable; let the tip follow the torso with a small delayed arc, never teleport. Let robes, hair, and loose accessories lag by one pose while retaining their silhouette and attachment points.

## Acceptance

- The body baseline differs by at most 2 px after normalization; the staff must not influence the measurement. Narrow the body range when needed.
- Playback shows the contact, down, passing, and up phases in both halves of the cycle; poses 1/2 and 5/6 are not duplicates.
- The red baseline contact sheet shows no vertical pop; the GIF shows no horizontal snap between frames 4 and 5.
- Alpha edges are free of magenta fringe, the character remains sharp at native gameplay size, and no frame is clipped.
- Record `accepted` or `redo`, the reason, and the review report path in the character bible for each direction.

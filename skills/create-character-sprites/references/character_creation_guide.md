# Character Sprite Guide

## Prompt

Use the reference as the identity source. Record hair, clothing, palette, symmetric props, scale, and prohibited changes (ensure visual effects like fire or glowing auras are explicitly removed). Adapt any one-sided reference detail by duplicating, centering, or removing it: every playable character must be safe to mirror. Describe style with observable traits such as painterly volume, soft gradients, readable silhouette, restrained outlines, and game-scale detail.

For a `128x192` frame, request an unguttered 2 rows x 4 columns image at `1024x768` or larger (4:3). The second grid row becomes frames 5-8, so it must use the same baseline, pivot, scale, and padding as the first. Keep the torso pivot at target `x=64`, foot baseline at target `y=186`, enough transparent margin for safe recentering, and the visible character at `80%–92%` target-frame height (`84%` preferred for front/back). Require a solid `#FF00FF` matte only; prohibit that color on the character.

```text
Create exactly 8 frames for one walk direction: [DIRECTION].
Character: [approved mirror-safe bible and reference]. Keep all left/right visual details symmetric.
Layout: unguttered 2 rows x 4 columns, reading left-to-right then top-to-bottom, 4:3 canvas. Target torso pivot x=64 and foot baseline y=186 after export; keep an ample transparent safety inset, at identical scale and padding in every cell.
Background: solid #FF00FF matte only. No text, borders, panels, duplicate limbs, cropped parts, cast shadows, pixel-art conversion, or visual effects (e.g., fire, glowing hands, auras).
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

Arms oppose legs. Keep every held or worn prop's attachment point stable; let a long prop follow the torso with a small delayed arc, never teleport. Let clothing, hair, and loose accessories lag by one pose while retaining their silhouette and attachment points.

Reject the cycle when the two contact phases show the same leading foot. Review frame 1 against frame 5 first: frame 1 must visibly place the **left** foot forward and frame 5 the **right** foot forward. Then confirm frame 3 retains the left support foot and frame 7 retains the right support foot.

## Acceptance

- Each frame must reach the fixed `x=64` torso pivot and `y=186` foot baseline after normalization. A one-pixel raster tolerance may be used during the first validation pass, but repeat bounded normalization before accepting a set that still varies between frames; held props must not influence either measurement.
- The one-pixel outer frame border is fully transparent, no magenta fringe survives at visible alpha, and the visible silhouette occupies `80%–92%` of the frame height (`84%` preferred for front/back).
- Playback shows the contact, down, passing, and up phases in both halves of the cycle; poses 1/2 and 5/6 are not duplicates.
- The red baseline contact sheet shows no vertical pop; the GIF shows no horizontal snap between frames 4 and 5.
- Alpha edges are free of magenta fringe, the character remains sharp at native gameplay size, no frame is clipped, and every design detail remains valid after horizontal mirroring.
- Record `accepted` or `redo`, the reason, and the review report path in the character bible for each direction.

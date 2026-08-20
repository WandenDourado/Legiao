# Controls

Use this doc for keyboard, mouse, touch joystick, sprint, and aim behavior.

## Desktop

| Action | Input |
|---|---|
| Move | WASD or arrow keys |
| Sprint | Left Shift |
| Aim | Mouse position in world space |
| Fire | Left mouse button (limited by the character's attack speed) |
| Primary skill | Q |
| Ultimate | R |
| Test mode (no cooldowns) | F2 |
| Restart stage after Game Over (host only) | F5 |

Desktop mouse position is screen-space and must be converted before attack targeting:

```go
screenPos := rl.GetMousePosition()
worldPos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
```

## Android

Android controls are screen-relative and computed each frame in `game.ProcessInput`.

| Control | Position |
|---|---|
| Movement joystick | center at `screenWidth * 0.15`, `screenHeight * 0.80` |
| Aim/action joystick | center at `screenWidth * 0.85`, `screenHeight * 0.80` |
| Base radius | `screenHeight * 0.08` |

`internal/input/touch.go` tracks stable touch IDs so movement and attack touches do not steal each other.

## Sprint

| Platform | Trigger |
|---|---|
| Desktop | Left Shift |
| Android | Movement joystick magnitude greater than `entity.SprintThreshold` (`0.70`) |

Sprint currently changes animation playback speed, not the movement speed constant.

## Aim And Fire

`input.AimJoystick` is a dual-function action control:

- Quick tap (no drag): fires in the **last configured** aim direction.
- Press and drag: updates `AimDir` to the drag direction, then fires in it.
- Release: fires in the aimed direction (which also becomes the persisted direction for the next tap).

The configured direction persists across touches: once the player aims by
dragging, a subsequent tap (press without dragging) fires in that saved
direction instead of a fresh default. `AimDir` is only reset to the default
`(0,1)` at construction, never on a new touch.

Drag vs. tap is decided by a screen-relative distance threshold, not a fixed
pixel count: `AimTapMaxDrag = screenHeight * 0.05` for the attack joystick and
`SkillTapMaxDrag = screenHeight * 0.04` for the skill button (see
`internal/input/touch.go`). Both are larger than the attack button radius
(`sh*0.06`), so a tap that only drifts inside the button still fires in the
saved direction; only a real drag beyond the button reconfigures the aim. A touch that moves less than this distance (and
lasts under `AimTapMaxDuration`, 0.150s) stays a tap-to-fire in the saved
direction; only a real drag past the threshold overwrites the saved aim. The
screen-relative threshold keeps ordinary finger tremor on high-DPI screens from
being misclassified as aiming.

The aim direction is converted to a world-space target relative to the player:

```go
targetX := p.Position.X + aimDir.X*100
targetY := p.Position.Y + aimDir.Y*100
```

Both desktop and Android send attacks through the same network path: clients send `MsgAttack`; the host creates projectiles.

The host also decides whether the attack is allowed at all: every character has
an attack speed, and a request that arrives before the cadence has recharged is
dropped. On Android the fire button dims while it recharges. Skill cooldowns
work the same way and are drawn as counters (Q/R buttons on Android, a pip bar
at the bottom on desktop). See `doc/combat_rules.md`.

### Skill button (Android)

`ui.SkillButton` is the on-screen Q (fireball) button, positioned above the
attack button. It now behaves like a second aim joystick with the same
tap/drag/release model as AimJoystick:

- Quick tap (no drag): casts the **last configured** skill direction.
- Press and drag: aims the projectile to the drag direction, then casts.
- Release: casts in the aimed direction, which also becomes the persisted
  direction for the next tap.

`SkillButton.SkillDir` persists across touches (reset only to `(0,1)` at
construction), so the player can aim the fireball independently of the
attack button. `castAbilityAt` is called with `sb.SkillDir` **only** —
it never falls back to `aj.AimDir`, so the cast can never inherit the
attack button's direction. When the player is dragging, `Draw()` shows an
aim line + arrow so they can see the projectile direction.

### Ultimate button (Android) and R key (desktop)

Every character has an ultimate (supreme) skill in ability slot 1:

- Desktop: **R** casts the ultimate at the mouse-aim world position (mirrors
  the Q key, which casts slot 0).
- Android: a golden **R** button (`ui.NewUltimateButton`, same
  `ui.SkillButton` type with `IsUltimate: true`) sits above the Q button with
  the same tap/drag-aim behavior and its own persisted direction. Its
  geometry comes from `AndroidControlGeom.Ult*`, stacked with the same
  non-overlap gap rule, so the three action circles form a column.

Both paths call `castAbilityAt(p, idx, tx, ty)` (`internal/game/input_handler.go`),
which resolves `ability.AbilityAt(charType, idx)` — 0 = primary, 1 = ultimate —
and sends the cast through the same `MsgSkill` host-authoritative flow.

### Shared control geometry (Android)

All on-screen controls derive their positions from
`input.ComputeAndroidControlGeom(sw, sh)`, used by the fire-button hit-rect
(`AimJoystick.Update`), the skill button layout (`SkillButton.Layout`), and
the rendered visuals (`DrawAimJoystick` / `SkillButton.Draw`). This guarantees
the touch hit-areas always match what is drawn and that the fire and skill
buttons never overlap. `AimJoystick.Update` also receives the skill button's
circle and refuses to claim a touch that falls inside it, so the fire button
can never "steal" the skill touch.

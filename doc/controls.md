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

### Aim indicator (Android)

`ui.AimIndicator` (`internal/ui/aim_indicator.go`) draws a discreet chevron on
the ground in front of the local player showing where the next shot will go.
It exists because the three saved directions above are only ever drawn next to
the BUTTON, at the bottom-right corner, while the player is looking at the
character: a tap fires in a direction configured seconds ago and nothing on the
character says which one.

| | |
|---|---|
| Anchor | `p.Position` — the point the projectile is BORN, not the soles and not the sprite's feet |
| Placement | a plain circle: the tip lands at exactly `Dir` and the chevron points at exactly `Dir` |
| Distance | `aimChevronDistance = 120` world px — clears the sprite in every direction (the figure reaches ~99 px above Position and ~103 below) |
| Resting alpha | `aimRestAlpha = 0.24` — the lowest value that still reads at 1:1 over bright grass |
| Aiming alpha | `aimActiveAlpha = 0.85`, eased at `aimFadeRate` so it swells rather than blinks |
| Colour | white at rest; `attackAimAccent` (light red) on the fire button, `rl.Orange` on Q, `rl.Gold` on R |
| Platform | Android only (`cfg.FullScreen`); on desktop the cursor already shows the aim |

Which of the three directions it shows is resolved in
`game.resolveAimHint` (`internal/game/aim_hint.go`): the button under the
finger wins, in the order R → Q → attack (the column's own top-down order);
with no finger anywhere it falls back to the ATTACK direction, because that is
what a tap on the biggest button would fire.

The drawn direction is **chased** towards the resolved one (`aimTurnRate`), not
assigned to it. The three buttons hold three independent directions, so
releasing Q hands the mark back to the attack aim: assigned, it would teleport
to the other side of the character; chased, it swings there.

Two things it got wrong on the first pass, both worth not repeating:

- **It was anchored at the soles** (`entity.GroundPoint`), because a mark on the
  ground reads better than one floating at chest height. It does read better,
  and it was wrong: `ProcessInput` aims at `p.Position + dir*100` and the host
  spawns the projectile from `Position`, so the mark was a parallel ray one
  `GroundOffset` (~103 px) below the fireball. Same angle, wrong line — and on
  a phone that reads as the spell ignoring the indicator.
- **The placement was an ellipse** (vertical axis squashed to 0.55) so it would
  look like it lay on the ground. A squash does not only move the mark, it
  ROTATES it: a 45-degree aim came out drawn at 28.8, and every direction that
  was not exactly horizontal or vertical was off by up to 16 degrees. The one
  number the thing exists to communicate was the one number it was distorting.

It is drawn inside the camera block in `game.DrawFrame`, **after the bodies and
before the health bars** — drawn before the bodies, the player's own sprite
would cover the mark whenever the aim points at it. Nothing about it crosses the
network: it is local drawing for the local player, and no other player sees it.

Its visual can be judged without building an APK through the
`preview-skill-effects` harness, which mirrors this draw code in canvas.

### The action cluster (Android)

The three action buttons are a CLUSTER: the basic attack is the big circle in
the middle and the two abilities orbit it. Before this they were a vertical
column of three same-sized circles — reachable, but saying nothing about which
one is the main action, and making the thumb travel the whole height of the
stack to reach the ultimate.

All the numbers live in `input.ComputeAndroidControlGeom`
(`internal/input/touch.go`):

| | |
|---|---|
| Attack radius | `sh * attackRadiusFrac` (0.085) — it is the anchor and the only button pressed dozens of times a minute |
| Attack centre | `sw*0.855`, `sh*0.80` |
| Ability radius | `attackRadiusFrac * skillRadiusRatio` (0.62 of the attack) |
| Orbit radius | attack + skill + `orbitGapRatio` (0.30) of the attack — so an ability circle can never intersect the attack circle, whatever the screen |
| Q at | `skillAngleQ` = 200 degrees (left and slightly up) |
| R at | `skillAngleR` = 252 degrees (up and slightly left) |

Both abilities sit ABOVE the attack button, on the side the right thumb sweeps
towards: below or to the right of it is either off-screen or under the palm.
The 52 degrees between them is what keeps THEM apart — at this orbit that is a
chord of ~1.68 attack-radii between centres, against the ~1.24 the two circles
occupy.

The attack is hit-tested as a **circle** (`AimJoystick.Update`), not as the
bounding rect it used to be: a square's corners reach 1.41 radii out, towards
ability circles whose inner edge is at 1.30. `AttackRect()` survives only for
`TouchState`, whose attack tracking nothing downstream reads.

Drawing order in `game.DrawFrame` is outside-in — the abilities, then the
attack. Drawn first, the big circle would have its ring covered by the two
orbiting it and the middle would stop reading as the middle.

### One gesture at a time (Android)

Aiming is a HELD drag, and the drag routinely crosses the other buttons — they
are one thumb-width away, which is the point of a cluster. Each one was arming
itself on the way past and firing the moment the finger came up: the player
asked for an angle and got an ultimate.

`input.ActionLock` (`internal/input/action_lock.go`) fixes it with a single
owner. The first of the three buttons to take a touch owns the gesture; the
other two refuse new touches and draw themselves dimmed (`actionLockedDim`)
until it lets go. It is one owner and not a per-button flag because the rule is
about the GESTURE: while one button is being aimed, nothing else in the cluster
exists.

**The movement joystick is deliberately outside the lock.** Walking while
aiming and casting is the game working as intended, on both platforms, and the
joystick lives on the other side of the screen where no aim drag reaches it.

Release is detected by scanning touches, so any path that stops updating the
buttons has to free the lock explicitly or the cluster stays dead for the rest
of the run. `game.cancelAndroidAim` is that call, and it runs on: a finished
run, a body frozen inside a portal, a desktop session, and an ultimate that
stopped being available mid-aim.

### Button icons

The three buttons carried TEXT — "FIRE", "Q", "R". A keyboard letter on a
device with no keyboard says nothing about what the button does, and "FIRE" is
only legible to people who read English. They are now glyphs (sword, star,
star with a ring) loaded from `assets/ui/icon_*.png` by
`internal/ui/action_icons.go`.

The PNGs are WHITE with alpha, so one file serves the lit, dimmed and locked
states through tinting instead of three copies of the same drawing. They carry
their own translucent disc, which is why the button's fill is weak
(`actionFillAlpha`): a strong fill under the icon turns the button into a blob
and the glyph stops reading. The RING (`actionRingAlpha`) is what draws the
button's edge against the map, since the fill is too faint to.

**All three buttons wear one colour**, `ui.ActionNeutral`. They were red,
orange and gold for a day — one hue per slot — and on a screen already carrying
health bars, damage numbers, spell light and a wave counter, three saturated
circles in the corner read as three ALERTS. The cue was redundant anyway: the
glyphs differ and the middle button is half again as big as the other two. What
separates the buttons now is size, position and drawing, none of which competes
with the fight for attention. The ground chevron (`ui.AimIndicator`) lost its
per-button tint for the same reason and wears the same neutral. The cache is per SESSION, not per map (three 256x256 textures, buttons
exist on every map); `ui.UnloadActionIcons` runs once, from `game.RunGameLoop`.
`Label` survives as the fallback for a missing file: a button with nothing on
it is worse than a button with the wrong-language word.

All three are drawn by one function, `ui.DrawActionButton`
(`internal/ui/action_button.go`). They used to be drawn in two packages by two
functions and had drifted apart — different fills, different rings, one with a
knob at the finger and one without.

### Shared control geometry (Android)

All on-screen controls derive their positions from
`input.ComputeAndroidControlGeom(sw, sh)`, used by the fire-button hit-rect
(`AimJoystick.Update`), the skill button layout (`SkillButton.Layout`), and
the rendered visuals (`DrawAimJoystick` / `SkillButton.Draw`). This guarantees
the touch hit-areas always match what is drawn and that the fire and skill
buttons never overlap. `AimJoystick.Update` also receives the skill button's
circle and refuses to claim a touch that falls inside it, so the fire button
can never "steal" the skill touch.

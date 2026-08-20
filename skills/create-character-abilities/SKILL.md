---
name: create-character-abilities
description: Create a new character ability/spell (magia) for Legiao, or improve an existing one. Use when asked to implement a skill for a character (projectile, aura, buff, area effect, shield, heal, summon), when wiring a spell into the host/client network flow, or when a spell exists but looks visually poor and needs to be made beautiful. Covers gameplay wiring AND the procedural-visuals craft (raylib primitives, particles, blend modes — never images). Do NOT use for character sprites/art assets (that is create-character-sprites / install-character-sprites) or for Android builds.
---

# Create Character Abilities

Implement spells that are host-authoritative, replicated to clients, and
drawn 100% procedurally — raylib primitives, gradients, particles, and blend
modes. **You never use image assets for spell visuals.** The spell description
you receive says what the spell DOES and the feeling it should convey; the
exact visual representation is your craft decision, guided by this skill.

Before writing code, read `AGENTS.md`, `doc/coding_patterns.md`, and skim the
four reference implementations (see "Worked examples" below). They are the
living style guide; new spells must feel like they belong next to them.

## Architecture: three layers, no switches

Spells follow a Strategy + Registry pattern. Adding one requires ZERO edits to
input, renderer, or per-character switch statements.

| Layer | Package | Owns |
|---|---|---|
| Effect | `internal/skill` | State structs, simulation step, ALL drawing. One spell = `<spell>.go` (entity + visuals) + `<spell>_manager.go` (collection on `skill.Manager`). |
| Ability | `internal/ability` | Thin `Skill` interface wrapper: `ID()`, `Cooldown()`, `Cast(ctx)`, `Draw(m)`. Registered in `binds.go`. |
| Network | `internal/network` | Host-authoritative tick (`host_<spell>.go`), broadcast to clients, client-side event handling + visual advancement. |

### Wiring checklist (follow in order)

1. `internal/skill/<spell>.go` — constants (damage, range, cooldown, radii)
   at the top with doc comments; the effect struct; `New<Spell>()`; `Update`
   (host, decides life), `AdvanceVisual` (client, never decides removal that
   the host owns); `Draw`.
2. `internal/skill/<spell>_manager.go` — collection on `Manager` (map + its
   own `sync.RWMutex` field added to the struct in `fireball_manager.go`),
   `Add/Remove/GetAll`, `Spawn<Spell>(m, ownerID, ...)`, host `Step<Spell>s`
   returning events for the caller to broadcast, client `Advance<Spell>s`,
   and `Draw<Spell>s`.
3. `internal/ability/<spell>.go` — the `Skill` implementation. `Cast` spawns
   on `ctx.Host.SkillManager()` then calls `ctx.Host.BroadcastSkill` (or
   `BroadcastSkillDir` for aimed effects that need direction replicated).
4. `internal/ability/binds.go` — `New<Spell>Skill()` + `BindAbility(entity.
   Char<X>, "<spell_id>")`. This is the ONLY place character↔spell mapping
   exists.
5. `internal/network/protocol.go` — new payload/message type ONLY if an
   existing channel cannot carry it (see "Sync patterns").
6. `internal/network/host_<spell>.go` — `handle<Spell>Tick(dt)` called from
   `UpdateSimulation` in `host.go`, plus the broadcast helper. Host applies
   ALL damage/heal/absorb; clients never mutate gameplay state.
7. `internal/network/client.go` — handle the incoming event: spawn/update the
   visual in `ClientSkills` (nil-check and create with `skill.NewManager()`).
8. Client per-frame advancement — extend `network.AdvanceClientSkills` (or
   the client branch in `game/loop.go`) so the visual animates on clients.
9. Rendering is automatic: `ability.DrawAll` iterates the registry for both
   host (`CurrentHost.Skills`) and client (`ClientSkills`). Your `Draw(m)`
   just delegates to the manager's draw method. Everything draws in WORLD
   space inside the existing `BeginMode2D` block — never screen space.
10. Gating is free: `HandleSkillMessage` already enforces character binding
    and your `Cooldown()` value. Do not add per-spell gating switches.

### Hard rules inherited from the repo

- One responsibility per file; split before ~150 lines.
- Every collection guarded by its own mutex; return snapshots from getters.
- Unique IDs: `generateID()` has only 260 outcomes — anything that spawns
  multiple instances per frame MUST use an atomic counter (see `arrow.go`
  `nextArrowID`).
- Never simulate gameplay in `Draw` or in client code. `Update` = host truth;
  `AdvanceVisual` = client cosmetics.
- Update `doc/changelog.md` (one line) when done.

## Sync patterns (pick the cheapest that works)

1. **One-shot spawn event** — effect is deterministic after launch (arrow
   volley: origin + direction; sanctuary: center). Broadcast once; clients
   replicate the visual locally. Cheapest and preferred.
2. **Combat-event stream** — effect has evolving gameplay state the clients
   must mirror (shield strength: `"shield"` events with current HP; heals:
   `"heal"` events). Reuse `broadcastCombatEvent` with a new `EventType`
   before inventing a message.
3. **Snapshot sync** — only for things already snapshot-synced (enemies,
   projectiles). Don't add new snapshot arrays for spell visuals.

Client-side visuals may diverge slightly from host truth (a client arrow may
fly through an enemy the host already removed). That is accepted across the
codebase — health/death always comes from host events, never from client sim.

## The visual craft: making spells beautiful

This is the part that separates a gray circle from a spell. Every good effect
in this codebase is built from the same five ingredients, layered in order:

### 1. Layer recipe (back to front)

1. **Solid identity shapes, normal blending** — whatever makes the effect
   READ as what it is. An arrow is a wooden `DrawLineEx` shaft + steel
   `DrawTriangle` head + fletching lines FIRST; the magic is decoration.
   If the effect is pure energy (fireball, aura), skip to layer 2.
2. **Additive glow core** — inside `rl.BeginBlendMode(rl.BlendAdditive)`:
   stacked `DrawCircle`/`DrawCircleGradient` from large-dim to small-bright.
   Additive blending makes overlapping light ADD UP — this is what makes
   energy look luminous instead of painted. `DrawCircleGradient(x, y, r,
   rl.Fade(color, a), rl.Blank)` is the single most useful glow primitive.
3. **Structure lines** — rings (`DrawRing`), arcs (`DrawRing` with partial
   start/end angles), rays (`DrawLineV` from center). These give the energy
   an edge and a silhouette; pure glow without structure looks like fog.
4. **Particles** — `ParticleEmitter` (`particle.go`): `Emit` for trails,
   `Burst` for impacts. Particles already damp velocity and shrink; you only
   choose count, speed range, life, radius, color.
5. **Motion** (see below) — nothing static. Ever.

### 2. Color: 3 tones per spell, hottest = smallest

Pick a tight palette and stick to it: a deep/dark outer tone, a saturated
mid tone, and a near-white hot core. The BRIGHTEST color always occupies the
SMALLEST area (fireball: red 55% alpha outer, orange 85% mid, yellow core at
0.32×radius). Reversing this ratio is the #1 amateur mistake.

Respect character identity palettes already established:

| Character | Palette | Feeling |
|---|---|---|
| Mago | red → orange → yellow | destructive heat |
| Sacerdotisa | gold → divine yellow → white | serene holiness |
| Paladina | gold → warm white (+ cool blue accents on break) | protective valor |
| Arqueiro | wood/steel naturals + green energy accents | precise, natural |

A new character's spell should claim its own palette and reuse it across all
its future spells.

### 3. Motion: sine pulses, rotation, easing

- **Pulse**: `1 + 0.06*sin(time*3.2)` on a radius — subtle breathing. Keep
  amplitude ≤ ~8%; large pulses look like blinking.
- **Rotation**: accumulate `Time` in the struct; rotate arc start angles
  (`spin := s.Time * 70`). TWO layers counter-rotating at different speeds
  (`70` vs `-45`) instantly read as "living magic".
- **Orbits**: motes on `cos/sin(time*k + i*offset)`; multiply the Y
  component by ~0.45 to squash the orbit into a pseudo-3D ellipse around a
  character; add a vertical `sin` bob per mote.
- **Easing**: derive `progress := 1 - ttl/maxTTL` and ease with it —
  explosion rings grow `r*(0.3+progress*1.1)` while alpha falls
  `(1-progress)`. Fast-out + fade-out reads as a shockwave.
- **Trails**: emit a few particles per frame BEHIND the mover, tapering
  radius toward the tail (`fireball.go emitTrail`), or afterimage motes with
  short life (`0.2–0.45s`).

### 4. State transitions: never pop, always resolve

Every effect needs an entrance, a life, and an exit:

- **Entrance**: spawn burst (`Burst` with 20–40 particles) or expanding ring.
- **Life**: the pulse/rotation loop.
- **Exit**: a fade window (`FadeAlpha` ramp like sanctuary's 1.5s) or a
  dramatic break (shield: expanding ring + two-color shatter burst over
  0.45s, driven by a `Breaking` countdown). Removing an effect the same
  frame its gameplay ends looks like a bug even when it isn't.
- **Feedback**: gameplay events deserve visual punctuation — the shield
  flashes (`HitFlash`) and sparks on every absorbed hit; impacts explode.
  If the player can't SEE that a mechanic triggered, the mechanic is
  unfinished, and intensity should track state (shield opacity scales with
  remaining strength: `0.35 + 0.65*ratio`, never fully dim while active).

### 5. Scale and readability

- World scale: player sprites are 128×192 (RenderScale 1.15); enemies are
  ~15px-radius circles. Auras that wrap a character: radius ~90–100.
  Projectile visual bodies: ~40–60 long. Use `world.Bounds`-scale numbers,
  never screen constants.
- The effect must read at gameplay zoom in under a second. Squint test: if
  you blur your eyes, the silhouette + palette should still say "arrows
  fanning out" / "protective bubble". Detail that only shows in screenshots
  is wasted; silhouette and motion are what read in combat.
- Alpha discipline: additive layers stack fast. Keep individual layer alphas
  ≤ ~0.85 and let the blending create the hotspot.
- Performance: cap per-frame emissions (2–6 trail particles), rely on the
  emitter's built-in decay/pruning, and keep per-effect particle counts in
  the dozens, not thousands.

## Worked examples (read the code, copy the techniques)

| Spell | Files | Techniques it demonstrates |
|---|---|---|
| Bola de Fogo (mago) | `skill/fireball.go`, `explosion.go`, `fire_ground.go` | 3-tone additive core; tapered trail; impact = explosion ring + burst + lingering ground zone |
| Santuário (sacerdotisa) | `skill/sanctuary.go`, `sanctuary_particles.go` | area aura; life-ratio ramps; graceful 1.5s fade-out; floating sacred motes; layered floor rings |
| Rajada de Flechas (arqueiro) | `skill/arrow.go`, `arrow_manager.go` | solid-shapes-first (reads as a real arrow); cone/fan math in `VolleyDirections`; atomic unique IDs; one-shot spawn sync |
| Escudo Sagrado (paladina) | `skill/shield.go`, `shield_manager.go` | follow-the-owner anchor; counter-rotating arcs; squashed orbit motes; HP-scaled intensity; hit flash; shatter exit; combat-event sync |

## Definition of done

1. `go build ./cmd/desktop` and `go vet ./...` pass (on the user's machine if
   the sandbox lacks Go).
2. Cast works from the desktop Q-key path AND the Android skill button path
   (both go through `castPrimaryAbility` — no extra wiring, just verify).
3. Host sees the effect; a connected client sees the effect; damage/heal
   numbers change only via host events.
4. Edge cases: caster dies mid-effect; recast while active (define stack /
   refresh / reject explicitly); zero-length aim vector.
5. Every effect resolves visually (fade/shatter), nothing pops out.
6. Cooldown returned by `Cooldown()` and felt in-game.
7. One line appended to `doc/changelog.md`; area docs updated if the network
   protocol gained a message.

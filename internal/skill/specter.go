package skill

// Specter: one undead minion of the Necromante's ultimate (Legião Espectral).
// Visual reference: Yorick's Ghoul — a hunched teal corpse that is mostly a
// gaping fanged maw, with long arms ending in bandaged forearms and bone
// claws. Rendered 100% procedurally in specter_draw.go.

import rl "github.com/gen2brain/raylib-go/raylib"

// Legion / specter tuning. Specters have NO time limit: the legion lasts
// until every specter has been killed. They are the fastest-attacking
// creatures in the game (highest attack speed), with medium health; enemies
// fight back at their own, slower attack cadence.
const (
	// LegionCount is how many specters a single cast summons.
	LegionCount = 30
	// LegionCooldown is the caster's cooldown in seconds.
	LegionCooldown float32 = 60.0
	// LegionOrbitRadius is the idle circle the specters keep around the owner.
	LegionOrbitRadius float32 = 510
	// LegionLeashRadius is how far from the owner a specter may hunt.
	LegionLeashRadius float32 = 650
	// SpecterSpeed is the hunting speed (fast — faster than any enemy).
	SpecterSpeed float32 = 340

	// The three numbers below decide the duel the ultimate is built around,
	// and they were tuned against the world_02 pack after it was played.
	//
	// Before: 35 health, 7 damage every 0.25 s (28 dmg/s). Against a wolf of
	// 40 health / 18 damage / 0.7 s, a specter needed 1.43 s to kill and died
	// in 1.66 s — it won by 0.23 s and took roughly one wolf with it. Thirty
	// specters therefore traded about one-for-one and the legion was spent
	// halfway through a sixty-wolf pack, which is not an ultimate.
	//
	// Now: kills that wolf in 0.66 s and survives 2.6 s, so one specter is
	// worth about four wolves and the legion clears the pack.
	//
	// The skill's identity survives the buff because it is a RATIO, not a
	// number: high damage per second against low health. Against a 300-health
	// target a specter still deals only ~160 damage before dying, so the
	// weakness against a single tough enemy is unchanged.

	// SpecterMaxHealth is a specter's life pool.
	SpecterMaxHealth float32 = 60
	// SpecterDamage is dealt per bite.
	SpecterDamage float32 = 11
	// SpecterAttackEvery is the bite interval — the HIGHEST attack speed in
	// the game (enemies attack every ~0.7-1 s), so each specter deals ~61 dmg/s.
	SpecterAttackEvery float32 = 0.18
	// SpecterRadius is the contact/visual radius of one specter.
	SpecterRadius float32 = 15
	// specterLungeTime is the bite lunge animation length.
	specterLungeTime float32 = 0.16
	// specterDissolve is how long the death dissolve visual lasts.
	specterDissolve float32 = 0.45
	// specterSpawnTime is the rise-from-the-ground entrance duration.
	specterSpawnTime float32 = 0.5
)

// Undead palette: dark gray corpse-flesh with necro-purple energy.
var (
	specterFlesh = rl.NewColor(98, 94, 106, 255)  // dead gray flesh
	specterShade = rl.NewColor(50, 44, 60, 255)   // darker gray-violet shading
	specterMaw   = rl.NewColor(22, 10, 34, 255)   // near-black maw cavity
	specterBone  = rl.NewColor(232, 224, 200, 255)
	specterWrap  = rl.NewColor(92, 60, 78, 255)   // rotten bandage wraps
	specterGlow  = rl.NewColor(150, 60, 220, 255) // arcane purple aura
	specterLight = rl.NewColor(224, 198, 255, 255) // pale violet bite flash
)

// Specter is one summoned ghoul. Position/behavior/health are simulated by
// the legion (host authoritative; clients replicate cosmetically).
type Specter struct {
	HomeAngle float32 // slot angle on the summoning circle
	Position  rl.Vector2
	Facing    rl.Vector2 // last movement/attack direction (for lean/lunge)
	Age       float32
	Phase     float32 // individual bob/wiggle phase
	Health    float32
	Dying     bool
	DieAge    float32
	// Combat/animation timers (host drives combat; lungeT is shared visual).
	hitTimer  float32
	hurtTimer float32
	lungeT    float32
}

// gone reports whether the dissolve visual has fully finished.
func (s *Specter) gone() bool { return s.Dying && s.DieAge > specterDissolve }

// alpha is the current global opacity: fades in on spawn, out on death.
func (s *Specter) alpha() float32 {
	a := clamp01(s.Age / specterSpawnTime)
	if s.Dying {
		a *= clamp01(1 - s.DieAge/specterDissolve)
	}
	return a
}

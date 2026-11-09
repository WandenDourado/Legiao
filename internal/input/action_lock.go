package input

// ActionSlot names one of the three on-screen action buttons.
type ActionSlot uint8

const (
	// SlotNone means no button owns the gesture.
	SlotNone ActionSlot = iota
	SlotAttack
	SlotSkill
	SlotUltimate
)

// ActionLock is what stops the player from casting by accident while aiming.
//
// Aiming on Android is a HELD gesture: the thumb presses a button and drags
// away from it to set the angle. The drag routinely travels over the other two
// buttons in the cluster — they are one thumb-width away, that is the whole
// point of a cluster — and every one of them was arming itself on the way past
// and firing the moment the finger came up. The player asked for an angle and
// got an ultimate.
//
// So the first button to take a touch owns the gesture, and the other two stop
// answering until it lets go. It is a single owner and not a per-button flag
// because the rule is about the GESTURE, not about the buttons: while one is
// being aimed, nothing else in the cluster exists.
//
// The movement joystick is deliberately outside this: walking while aiming and
// casting is the game working as intended, on both platforms, and the joystick
// lives on the other side of the screen where no aim drag reaches it.
type ActionLock struct {
	owner ActionSlot
}

// Claim gives slot the gesture if it is free, or confirms it already has it.
// Returns false when another slot owns it — the caller must then ignore the
// touch entirely, not merely skip firing.
func (l *ActionLock) Claim(slot ActionSlot) bool {
	// SlotNone is the "nobody" value, so claiming it would set the lock to
	// free and hand every button the gesture at once. A button that reached
	// here with an unset Slot is a wiring bug, and failing loudly (it simply
	// stops responding) beats silently disabling the whole rule.
	if slot == SlotNone {
		return false
	}
	if l.owner == SlotNone || l.owner == slot {
		l.owner = slot
		return true
	}
	return false
}

// Release hands the gesture back, but only from the slot that owns it. A slot
// that never claimed cannot free another one's lock — without that guard a
// button releasing an old touch on the same frame another button claimed
// would unlock the cluster mid-aim.
func (l *ActionLock) Release(slot ActionSlot) {
	if l.owner == slot {
		l.owner = SlotNone
	}
}

// LockedFor reports that another slot owns the gesture, so this one must draw
// itself disabled and refuse new touches.
func (l *ActionLock) LockedFor(slot ActionSlot) bool {
	return l.owner != SlotNone && l.owner != slot
}

// Busy reports that any slot owns the gesture.
func (l *ActionLock) Busy() bool { return l.owner != SlotNone }

// Reset frees the gesture unconditionally. For the frames where the buttons
// stop being updated at all — a finished run, a body waiting inside a portal —
// because the release is detected by scanning touches, and code that returns
// before that scan would leave the cluster locked forever.
func (l *ActionLock) Reset() { l.owner = SlotNone }

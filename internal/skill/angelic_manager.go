package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Angelic Area collection (Sacerdotisa ultimate) ---

// ActivateAngelic consecrates the altar for ownerID at the cast position —
// the area stays FIXED on the ground (it does not follow the caster).
// Recasting replaces the altar; there is never more than one per caster.
func ActivateAngelic(m *Manager, ownerID string, center rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if m.Angelics == nil {
		m.Angelics = make(map[string]*AngelicArea)
	}
	m.Angelics[ownerID] = NewAngelicArea(ownerID, center)
}

// HasAngelic reports whether ownerID still has an altar standing.
//
// Espelha HasLegion, e existe pelo mesmo motivo: o ultimo suspiro precisa
// saber quando o efeito do heroi INVOCADO se gastou, para tirar o NPC de
// campo. Sem isto, um heroi de ultimate que nao e legiao ficaria plantado ali
// ate o fim da fase.
func (m *Manager) HasAngelic(ownerID string) bool {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	_, ok := m.Angelics[ownerID]
	return ok
}

// ConsumeAngelicResurrections returns the areas whose one-time resurrection
// has not fired yet, marking them consumed. Host-only.
func (m *Manager) ConsumeAngelicResurrections() []*AngelicArea {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	pending := make([]*AngelicArea, 0)
	for _, a := range m.Angelics {
		if a.ResurrectPending {
			a.ResurrectPending = false
			pending = append(pending, a)
		}
	}
	return pending
}

// StepAngelics advances all areas on the HOST and returns the heal events due
// this tick (same fractional-accumulation pattern as the Sanctuary). Finished
// areas are pruned.
func (m *Manager) StepAngelics(dt float32, allies map[string]PlayerHealTarget) []SanctuaryHealEvent {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	events := make([]SanctuaryHealEvent, 0)
	for id, a := range m.Angelics {
		a.Update(dt)
		if a.IsHealing() {
			a.HealAccum += AngelicHealPerSec * dt
			whole := int(a.HealAccum)
			if whole > 0 {
				a.HealAccum -= float32(whole)
				for pid, t := range allies {
					if t.IsDead || t.Health >= t.MaxHealth {
						continue
					}
					if !a.Contains(rl.NewVector2(t.X, t.Y)) {
						continue
					}
					amount := float32(whole)
					if t.Health+amount > t.MaxHealth {
						amount = t.MaxHealth - t.Health
					}
					events = append(events, SanctuaryHealEvent{PlayerID: pid, Amount: amount})
				}
			}
		}
		if a.Finished() {
			delete(m.Angelics, id)
		}
	}
	return events
}

// AdvanceAngelics advances areas on CLIENTS (visuals only) and prunes them.
func (m *Manager) AdvanceAngelics(dt float32) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	for id, a := range m.Angelics {
		a.Update(dt)
		if a.Finished() {
			delete(m.Angelics, id)
		}
	}
}

// DrawAngelics renders every angelic area (world space).
func (m *Manager) DrawAngelics() {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, a := range m.Angelics {
		a.Draw()
	}
}

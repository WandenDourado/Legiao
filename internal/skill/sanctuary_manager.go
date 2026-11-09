package skill

import (
	"github.com/WandenDourado/Legiao/internal/sound"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A COLECAO E INDEXADA PELO DONO, nao por um ID proprio de efeito.
//
// Enquanto o Santuario era um lugar no chao, duas conjuracoes eram duas
// zonas, e a chave tinha de ser unica por conjuracao. Agora a aura e ancorada
// na Sacerdotisa: ela tem UMA, e relancar so pode significar renovar. A chave
// por dono e o que torna isso verdade em vez de convencao — e e a mesma forma
// que o Escudo Sagrado e o Avatar ja usavam (ver clear_owner.go, que separa
// as colecoes exatamente por esse criterio).

// AddSanctuary registers a sanctuary in the manager (thread-safe).
func (m *Manager) AddSanctuary(s *Sanctuary, fx *SanctuaryFX) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	m.Sanctuaries[s.OwnerID] = s
	if m.sanctuaryFX == nil {
		m.sanctuaryFX = make(map[string]*SanctuaryFX)
	}
	m.sanctuaryFX[s.OwnerID] = fx
}

// RemoveSanctuary deletes a sanctuary by owner ID.
func (m *Manager) RemoveSanctuary(ownerID string) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	delete(m.Sanctuaries, ownerID)
	if m.sanctuaryFX != nil {
		delete(m.sanctuaryFX, ownerID)
	}
}

// GetAllSanctuaries returns a snapshot of active sanctuaries.
func (m *Manager) GetAllSanctuaries() []*Sanctuary {
	m.sanctMutex.RLock()
	defer m.sanctMutex.RUnlock()
	list := make([]*Sanctuary, 0, len(m.Sanctuaries))
	for _, s := range m.Sanctuaries {
		list = append(list, s)
	}
	return list
}

// SpawnSanctuary creates the aura owned by ownerID, or renews it if the owner
// already has one (nunca empilha: uma Sacerdotisa, um Santuario).
func SpawnSanctuary(m *Manager, ownerID string, center rl.Vector2) {
	// Soa no lancamento E no relancamento: para quem esta perto, renovar a
	// aura e uma conjuracao nova. O prazo e o da propria aura, entao o canto
	// desvanece junto com ela.
	sound.PlayAtFor(sound.SFXSanctuary, center, SanctuaryDuration)
	m.sanctMutex.Lock()
	if s, ok := m.Sanctuaries[ownerID]; ok && !s.Dead {
		s.Refresh(center)
		m.sanctMutex.Unlock()
		return
	}
	m.sanctMutex.Unlock()
	m.AddSanctuary(NewSanctuary(ownerID, center), NewSanctuaryFX())
}

// SetSanctuaryAnchor keeps the aura glued to its owner's current position.
// Chamada todo quadro pelo host (host_sanctuary.go) e pelo cliente
// (client_skills.go) — e o que faz a magia seguir quem a lancou.
func (m *Manager) SetSanctuaryAnchor(ownerID string, pos rl.Vector2) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	if s, ok := m.Sanctuaries[ownerID]; ok {
		s.Position = pos
	}
}

// SanctuaryMoveMultiplier devolve o multiplicador de velocidade de quem
// carrega uma aura ativa: 1.25 enquanto ela cura, 1 no resto do tempo.
//
// Uma funcao e nao um campo em PlayerState porque o bonus JA e derivavel de
// um estado que host e cliente compartilham (a aura existe nos dois, com o
// mesmo relogio). Transmitir "ela esta rapida" seria mandar pela rede algo
// que os dois lados ja sabem — doc/skill_performance.md, Regra 6.
func (m *Manager) SanctuaryMoveMultiplier(ownerID string) float32 {
	m.sanctMutex.RLock()
	defer m.sanctMutex.RUnlock()
	s, ok := m.Sanctuaries[ownerID]
	if !ok || s.Dead || !s.IsHealing() {
		return 1
	}
	return 1 + SanctuarySpeedBonus
}

// UpdateSanctuaries advances every sanctuary's lifetime + visuals. It returns
// heal events accumulated this tick for the caller to apply/broadcast. Each
// sanctuary is aged here; the host keeps authoritative ownership of removal.
func (m *Manager) UpdateSanctuaries(dt float32, allies map[string]PlayerHealTarget) []SanctuaryHealEvent {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	allEvents := make([]SanctuaryHealEvent, 0)
	for id, s := range m.Sanctuaries {
		s.Age += dt
		// advance visuals
		if fx := m.sanctuaryFX[id]; fx != nil {
			fx.update(dt)
		}
		// healing happens during the active window
		events := StepSanctuaryHealing(s, dt, allies)
		allEvents = append(allEvents, events...)
		// fully dissipated -> remove
		if s.Age > SanctuaryDuration+SanctuaryFade {
			s.Dead = true
			delete(m.Sanctuaries, id)
			if m.sanctuaryFX != nil {
				delete(m.sanctuaryFX, id)
			}
		}
	}
	return allEvents
}

// DrawSanctuaries renders all active sanctuaries (world space). Call inside a
// BeginMode2D block, after the floor and before/with the players as desired.
func (m *Manager) DrawSanctuaries() {
	m.sanctMutex.RLock()
	defer m.sanctMutex.RUnlock()
	// Ordem estavel, nao a do mapa: ver draw_order.go.
	for _, id := range orderedIDs(m.Sanctuaries) {
		s := m.Sanctuaries[id]
		a := s.FadeAlpha()
		if a <= 0 {
			continue
		}
		// fx nulo e possivel (um teste pode registrar uma aura sem visual);
		// desenhar e opcional, curar nao e.
		if fx := m.sanctuaryFX[id]; fx != nil {
			fx.draw(s, a)
		}
	}
}

// AdvanceSanctuaries ages sanctuaries and advances their procedural FX without
// applying healing. Used on clients, where HP is already synced via combat
// events; this only keeps the visuals moving and prunes finished sanctuaries.
func (m *Manager) AdvanceSanctuaries(dt float32) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	for id, s := range m.Sanctuaries {
		s.Age += dt
		if fx := m.sanctuaryFX[id]; fx != nil {
			fx.update(dt)
		}
		if s.Age > SanctuaryDuration+SanctuaryFade {
			s.Dead = true
			delete(m.Sanctuaries, id)
			if m.sanctuaryFX != nil {
				delete(m.sanctuaryFX, id)
			}
		}
	}
}

package skill

import rl "github.com/gen2brain/raylib-go/raylib"

// --- Selo da Chuva de Meteoros (marca do conjurador) ---
//
// A colecao e indexada POR DONO, como o Avatar e o Escudo: um mago, um selo.
// Relancar a suprema renova o relogio em vez de empilhar uma segunda figura no
// mesmo lugar (duas copias do mesmo desenho so somam brilho e custo).

// ActivateMeteorSigil acende o selo de ownerID, ou renova o dele.
func ActivateMeteorSigil(m *Manager, ownerID string, pos rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if m.MeteorSigils == nil {
		m.MeteorSigils = make(map[string]*MeteorSigil)
	}
	if s, ok := m.MeteorSigils[ownerID]; ok {
		s.Remaining = MeteorRainDuration
		s.Age = 0
		s.Position = pos
		return
	}
	m.MeteorSigils[ownerID] = NewMeteorSigil(ownerID, pos)
}

// SetMeteorSigilAnchor mantem o selo grudado nos pes do dono.
//
// E chamada para TODO jogador a cada quadro (host_ultimate.go e
// client_skills.go), como a do Avatar: quem nao tem selo simplesmente nao
// aparece no mapa e a chamada nao faz nada.
func (m *Manager) SetMeteorSigilAnchor(ownerID string, pos rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if s, ok := m.MeteorSigils[ownerID]; ok {
		s.Position = pos
	}
}

// UpdateMeteorSigils envelhece todos os selos e recolhe os que acabaram. Roda
// igual no host e no cliente: o selo e so desenho, entao os dois lados o
// tocam pelo mesmo relogio e nenhum evento de rede e gasto com ele depois da
// ativacao.
func (m *Manager) UpdateMeteorSigils(dt float32) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	for id, s := range m.MeteorSigils {
		s.Update(dt)
		if s.Finished() {
			delete(m.MeteorSigils, id)
		}
	}
}

// DrawMeteorSigils desenha todos os selos (espaco de mundo).
func (m *Manager) DrawMeteorSigils() {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	// Ordem estavel, nao a do mapa: ver draw_order.go.
	for _, id := range orderedIDs(m.MeteorSigils) {
		m.MeteorSigils[id].Draw()
	}
}

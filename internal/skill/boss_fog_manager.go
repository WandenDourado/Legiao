package skill

import rl "github.com/gen2brain/raylib-go/raylib"

// --- A nevoa (chefe do mapa 7) ---

// ActivateDarkFog poe a nevoa no ar sobre uma regiao. Reconjurar substitui a
// anterior; nunca ha duas.
func ActivateDarkFog(m *Manager, bounds rl.Rectangle) {
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	m.Fog = NewDarkFog(bounds)
}

// FogActive reporta se a nevoa esta no ar.
func (m *Manager) FogActive() bool {
	m.bossMutex.RLock()
	defer m.bossMutex.RUnlock()
	return m.Fog != nil
}

// AdvanceFog envelhece a nevoa e devolve quantos passos de dano cabem neste
// quadro e quanto vale cada um. Zero passos quando nao ha nevoa.
func (m *Manager) AdvanceFog(dt float32) (times int, damage float32) {
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	if m.Fog == nil {
		return 0, 0
	}
	times, damage = m.Fog.Tick(dt)
	if !m.Fog.Update(dt) {
		m.Fog = nil
	}
	return times, damage
}

// FogContains reporta se um ponto esta sob a nevoa.
func (m *Manager) FogContains(p rl.Vector2) bool {
	m.bossMutex.RLock()
	defer m.bossMutex.RUnlock()
	return m.Fog != nil && m.Fog.Contains(p)
}

// DrawFog desenha a nevoa, se houver.
func (m *Manager) DrawFog() {
	m.bossMutex.RLock()
	defer m.bossMutex.RUnlock()
	if m.Fog != nil {
		m.Fog.Draw()
	}
}

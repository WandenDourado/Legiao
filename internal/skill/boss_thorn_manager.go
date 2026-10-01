package skill

import rl "github.com/gen2brain/raylib-go/raylib"

// --- Colecao de espinhoes (chefe do mapa 7) ---

// SpawnThorns marca varios pontos de uma vez.
//
// Recebe a lista pronta em vez de descobrir as posicoes sozinha: quem sabe onde
// os jogadores estao e o host, e a FOTO tem de ser tirada num instante so. Se
// cada espinho lesse a posicao no momento em que fosse criado, os ultimos da
// lista mirariam alguns milissegundos depois dos primeiros — pouco, mas o
// suficiente para o conjunto perder o alinhamento que faz a rajada ler como um
// unico golpe.
func SpawnThorns(m *Manager, centers []rl.Vector2) {
	if len(centers) == 0 {
		return
	}
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	for _, c := range centers {
		m.Thorns = append(m.Thorns, NewThorn(c))
	}
}

// AdvanceThorns envelhece os espinhoes e descarta os que acabaram.
func (m *Manager) AdvanceThorns(dt float32) {
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	alive := m.Thorns[:0]
	for _, t := range m.Thorns {
		if t.Update(dt) {
			alive = append(alive, t)
		}
	}
	m.Thorns = alive
}

// ThornsErupting devolve os espinhoes que irromperam NESTE quadro, marcando-os
// como resolvidos. So o host chama.
func (m *Manager) ThornsErupting() []*Thorn {
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	var out []*Thorn
	for _, t := range m.Thorns {
		if t.Erupting() {
			out = append(out, t)
		}
	}
	return out
}

// DrawThorns desenha todos.
func (m *Manager) DrawThorns() {
	m.bossMutex.RLock()
	defer m.bossMutex.RUnlock()
	for _, t := range m.Thorns {
		t.Draw()
	}
}

// ClearBossEffects apaga espinhoes e nevoa. Chamado no reinicio de fase: um
// espinho sobrevivente marcaria o chao de uma tentativa que ja acabou.
func (m *Manager) ClearBossEffects() {
	m.bossMutex.Lock()
	defer m.bossMutex.Unlock()
	m.Thorns = nil
	m.Fog = nil
}

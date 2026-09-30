package sound

// PlayUI toca um efeito SEM espacializacao, no volume cheio do barramento da
// categoria dele (nada de atenuacao por distancia, nada de exigir
// SetListener). E o que a previa sonora do slider de volume precisa
// (internal/ui, tela de Configuracoes): la nao existe jogador local, entao
// PlayAt (sfx.go) nao serve — ele silenciaria tudo por falta de ouvinte.
//
// Ignora de proposito o minGap do catalogo: e um unico disparo ao SOLTAR o
// slider (ver doc/settings.md), nao uma rajada que precise de protecao
// contra ruido branco.
func PlayUI(id SFX) {
	if !sfxReady.Load() {
		return
	}
	def, ok := sfxDefs[id]
	if !ok {
		return
	}
	vol := busVolume(def.bus)
	if vol <= 0 {
		return
	}
	sfxMutex.Lock()
	defer sfxMutex.Unlock()
	if len(pending) >= pendingCap {
		return
	}
	pending = append(pending, sfxRequest{id: id, vol: vol, life: 0})
}

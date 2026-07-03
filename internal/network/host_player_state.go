package network

// Host player state helpers keep authoritative player snapshots up to date.

// UpdatePlayerState updates local movement and animation in the authoritative host map.
func (h *Host) UpdatePlayerState(state PlayerState) {
	h.playersMutex.Lock()
	if p, ok := h.players[state.PlayerID]; ok {
		p.X = state.X
		p.Y = state.Y
		p.Color = state.Color
		p.CurrentFrame = state.CurrentFrame
		p.CurrentRow = state.CurrentRow
		p.IsSprinting = state.IsSprinting
		p.VelX = state.VelX
		p.VelY = state.VelY
	}
	h.playersMutex.Unlock()
	h.BroadcastStateUpdate()
}

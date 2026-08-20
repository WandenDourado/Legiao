package network

// Local side of the test-mode switch. Whoever holds authority applies it:
// a host flips its own gate directly, a client asks the host for it.

// SetLocalTestMode routes the local player's test-mode switch to the host.
func SetLocalTestMode(enabled bool) {
	switch Role {
	case "host":
		if CurrentHost != nil {
			CurrentHost.SetTestMode(LocalPlayerID, enabled)
		}
	case "client":
		SendMessage(Message{Type: MsgTestMode, Payload: MustMarshal(TestModePayload{
			PlayerID: LocalPlayerID,
			Enabled:  enabled,
		})})
	}
}

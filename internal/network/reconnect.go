package network

// Client-side reconnection: noticing the host is gone (client_liveness.go
// does the noticing) and getting back in with the SAME identity.
//
// The identity that makes a reconnect possible lives in ui/menu.go: the
// player ID is generated once per process and reused on every attempt,
// instead of minting a new one per connection the way it used to. LocalPlayerColor
// and LocalPlayerCharacter (globals.go) are cached at the original join so
// the rejoin can resend exactly what the host is expecting to recognize.

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"
)

// ReconnectWindow is how long the client keeps trying before giving up and
// returning to the menu. Matches host.ReconnectGrace in duration —
// deliberately not a shared constant, because a reconnecting client by
// definition has no Host object to read one from.
const ReconnectWindow = 5 * time.Minute

// Backoff between dial attempts: 1s to start, doubling up to a 5s ceiling.
const (
	reconnectBackoffStart = 1 * time.Second
	reconnectBackoffMax   = 5 * time.Second
	dialTimeout           = 3 * time.Second
)

var (
	reconnectMu      sync.Mutex
	reconnecting     bool
	reconnectSince   time.Time
	reconnectBackoff time.Duration
	nextAttemptAt    time.Time
	gaveUp           bool
)

// Reconnecting reports whether the client is currently trying to get back to
// the host. The game loop freezes local input and draws the overlay while
// this is true (ui.DrawReconnectOverlay).
func Reconnecting() bool {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()
	return reconnecting
}

// ReconnectRemaining is how much of ReconnectWindow is left, for the
// overlay's countdown. Zero when not reconnecting.
func ReconnectRemaining() time.Duration {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()
	if !reconnecting {
		return 0
	}
	left := ReconnectWindow - time.Since(reconnectSince)
	if left < 0 {
		return 0
	}
	return left
}

// GaveUp reports whether the client exhausted ReconnectWindow with no host in
// reach. The UI reads this once to show a clear message and return to the
// menu instead of leaving the app in limbo; ClearGaveUp resets it so a later
// session starts clean.
func GaveUp() bool {
	reconnectMu.Lock()
	defer reconnectMu.Unlock()
	return gaveUp
}

// ClearGaveUp resets the flag GaveUp reports. Call once the menu has taken
// the player back.
func ClearGaveUp() {
	reconnectMu.Lock()
	gaveUp = false
	reconnectMu.Unlock()
}

// TickReconnect drives the client-side reconnect state machine. Called once
// per frame from the game loop for the client role — a single delegated
// call, like network.CurrentHost.UpdateSimulation is for the host; the state
// machine itself lives entirely here, not in loop.go.
func TickReconnect() {
	if Role != "client" || CurrentClient == nil {
		return
	}
	if !Reconnecting() {
		if silenceSince() > clientSilenceTimeout {
			triggerReconnect("silencio do host por mais de " + clientSilenceTimeout.String())
		}
		return
	}
	if ReconnectRemaining() <= 0 {
		giveUp()
		return
	}
	reconnectMu.Lock()
	due := !time.Now().Before(nextAttemptAt)
	reconnectMu.Unlock()
	if due {
		attemptReconnect()
	}
}

// triggerReconnect enters the reconnecting state: local input freezes (the
// game loop reads Reconnecting()), the mirrored world keeps drawing whatever
// it last had, and the overlay starts counting down. No-op if already
// reconnecting, so a stale readLoop error arriving after silence already
// triggered a reconnect cannot restart the backoff clock.
func triggerReconnect(reason string) {
	reconnectMu.Lock()
	if reconnecting {
		reconnectMu.Unlock()
		return
	}
	reconnecting = true
	reconnectSince = time.Now()
	reconnectBackoff = reconnectBackoffStart
	nextAttemptAt = time.Now()
	reconnectMu.Unlock()

	log.Printf("[Reconnect] conexao perdida (%s); tentando reconectar a %s", reason, ServerAddress)
	CurrentClient.dropConn()
}

// attemptReconnect tries once to re-establish the TCP connection and rejoin
// with the same identity. Always reschedules the next attempt first (doubling
// the backoff up to reconnectBackoffMax), so a dial that hangs up to
// dialTimeout cannot stall the next try beyond it.
func attemptReconnect() {
	reconnectMu.Lock()
	backoff := reconnectBackoff
	if reconnectBackoff < reconnectBackoffMax {
		reconnectBackoff *= 2
		if reconnectBackoff > reconnectBackoffMax {
			reconnectBackoff = reconnectBackoffMax
		}
	}
	nextAttemptAt = time.Now().Add(backoff)
	reconnectMu.Unlock()

	log.Printf("[Reconnect] tentando %s...", ServerAddress)
	conn, err := net.DialTimeout("tcp", ServerAddress, dialTimeout)
	if err != nil {
		log.Printf("[Reconnect] falhou: %v", err)
		return
	}
	if CurrentClient == nil {
		conn.Close()
		return
	}
	setClientKeepAlive(conn)
	CurrentClient.rebind(conn)
	rejoin()

	reconnectMu.Lock()
	reconnecting = false
	reconnectMu.Unlock()
	log.Printf("[Reconnect] reconectado a %s", ServerAddress)
}

// rejoin resends MsgJoin with the SAME id/color/character so the host
// recognizes this as a reconnect (host_rejoin.go) instead of a fresh spawn,
// and clears everything that only made sense for the map state minutes ago.
func rejoin() {
	resetLiveness()
	ResetInterpolation()
	RemoteEnemiesMutex.Lock()
	RemoteEnemies = make(map[string]EnemyState)
	RemoteEnemiesMutex.Unlock()
	RemoteProjectilesMutex.Lock()
	RemoteProjectiles = make(map[string]ProjectileState)
	RemoteProjectilesMutex.Unlock()
	if ClientSkills != nil {
		ClientSkills.Reset()
	}
	SendMessage(Message{Type: MsgJoin, Payload: MustMarshal(JoinPayload{
		PlayerID:  LocalPlayerID,
		Color:     LocalPlayerColor,
		Character: LocalPlayerCharacter,
	})})
}

// giveUp ends the session after the reconnect window runs out with no host
// in reach.
func giveUp() {
	reconnectMu.Lock()
	reconnecting = false
	gaveUp = true
	reconnectMu.Unlock()
	log.Printf("[Reconnect] janela de %s esgotada sem reconectar; desistindo", ReconnectWindow)
	if CurrentClient != nil {
		CurrentClient.dropConn()
	}
}

// SessionEnded reports whether the client gave up reconnecting and the game
// loop should return to the menu instead of continuing this session. True
// exactly once GaveUp fires.
func SessionEnded() bool { return GaveUp() }

// ResetClientSession clears client-role state so a fresh ui.ShowMenu ->
// host/join can start clean after a give-up.
//
// LocalPlayerID is deliberately left alone: it is good for the life of the
// process (see connectToHost in ui/menu.go), and reusing it for a brand-new
// join is harmless — the host has already forgotten the old slot after
// ReconnectGrace expired, which is what got the client here in the first
// place.
//
// GaveUp is deliberately left set: ui.ShowMenu reads it once to show the
// "could not reconnect" message and is the one that clears it
// (network.ClearGaveUp), not this function.
func ResetClientSession() {
	reconnectMu.Lock()
	reconnecting = false
	reconnectMu.Unlock()

	Role = ""
	CurrentClient = nil
	GameOver = false
	LocalPlayerDead = false

	RemotePlayersMutex.Lock()
	RemotePlayers = nil
	RemotePlayersMutex.Unlock()
	RemoteEnemiesMutex.Lock()
	RemoteEnemies = nil
	RemoteEnemiesMutex.Unlock()
	RemoteProjectilesMutex.Lock()
	RemoteProjectiles = nil
	RemoteProjectilesMutex.Unlock()
	ClientSkills = nil

	ResetInterpolation()
	SetDialogueState(DialogueState{})
}

// dropConn tears down the current connection without touching Client
// identity. Send starts failing silently until rebind gives it a writer
// again.
func (c *Client) dropConn() {
	c.mu.Lock()
	conn := c.conn
	c.conn, c.writer = nil, nil
	c.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
}

// rebind swaps in a freshly dialed connection and starts a new read loop for
// it. Guarded by the same lock Send uses, so nobody writes to half-swapped
// state — the exact race the task brief calls out.
func (c *Client) rebind(conn net.Conn) {
	c.mu.Lock()
	c.conn = conn
	c.writer = bufio.NewWriter(conn)
	c.mu.Unlock()
	go c.readLoop(conn)
}

package network

// Detecting that the host is gone, on the client.
//
// The TCP error path (a decode failing in Client.readLoop) is NOT the
// primary detector: a locked Android screen pauses the app without
// necessarily breaking the socket, so a real outage can sit there silent,
// with no decode error ever coming, for as long as the OS lets the process
// live. The clock is what actually notices.
//
// The host publishes at SnapshotHz (20 Hz, broadcast_rate.go), so silence is
// a fast, reliable signal on its own: normal jitter between messages is
// milliseconds, never seconds.

import (
	"sync"
	"time"
)

// clientSilenceTimeout is how long the client goes without hearing ANY
// message from the host before it treats the connection as lost.
const clientSilenceTimeout = 5 * time.Second

var (
	livenessMu    sync.Mutex
	lastMessageAt time.Time
)

// markMessageReceived records that a message just arrived from the host.
// Called from Client.readLoop for every successfully decoded message, not
// only state_update — any traffic at all is proof the connection is alive.
func markMessageReceived() {
	livenessMu.Lock()
	lastMessageAt = time.Now()
	livenessMu.Unlock()
}

// resetLiveness restarts the silence clock. Called on a fresh connect and
// again on every successful reconnect, so the 5 s timeout is measured from
// "we just confirmed this connection works", not from whenever the last
// message on the PREVIOUS connection happened to arrive.
func resetLiveness() {
	livenessMu.Lock()
	lastMessageAt = time.Now()
	livenessMu.Unlock()
}

// silenceSince is how long it has been since the last message from the host.
// Zero (time never set) reports as no silence: there is nothing to time out
// before the first message of the session has even had a chance to arrive.
func silenceSince() time.Duration {
	livenessMu.Lock()
	last := lastMessageAt
	livenessMu.Unlock()
	if last.IsZero() {
		return 0
	}
	return time.Since(last)
}

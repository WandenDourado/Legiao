package network

// TCP keepalive for both sides of a connection.
//
// The 5 s silence timeout (client_liveness.go) is what actually notices a
// dropped host on the client — it does not depend on this. Keepalive is the
// belt: without it, an idle TCP connection to an Android device whose screen
// locked can sit "established" from the OS's point of view for a very long
// time (the platform's own TCP timeout is much longer than anything this
// game can wait), so probing keeps the connection honest instead of relying
// on that timeout ever firing.

import (
	"net"
	"time"
)

// keepAlivePeriod is how often the OS probes an idle connection. Shorter
// than clientSilenceTimeout so a probe failure has a chance to close the
// socket (and unblock a pending read) before the application-level timeout
// would have fired anyway.
const keepAlivePeriod = 3 * time.Second

// setKeepAlive enables TCP keepalive on conn if it is a *net.TCPConn. No-op
// (and no error) for anything else — tests exercise this over net.Pipe/other
// transports that do not support it, and that is not a failure worth
// reporting.
func setKeepAlive(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(keepAlivePeriod)
}

// setClientKeepAlive is setKeepAlive, named for readability at the client's
// call sites (ConnectClient, attemptReconnect).
func setClientKeepAlive(conn net.Conn) { setKeepAlive(conn) }

package network

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	// DiscoveryPort is the UDP port used for host discovery
	DiscoveryPort = 9001
	// DiscoveryInterval is how often the host sends discovery broadcasts
	DiscoveryInterval = 3 * time.Second
	// TCPScanTimeout is how long to wait for TCP scan responses
	TCPScanTimeout = 300 * time.Millisecond
	// QueryTimeout is how long to wait for query responses
	QueryTimeout = 2 * time.Second
)

var (
	// discoveredHosts stores IPs found via discovery (client-side)
	discoveredHosts []string
	discoveredMutex sync.Mutex

	// Per-mechanism run/stop state so the listener, query sender, and TCP scan
	// can run concurrently. A single shared flag previously let only the first
	// scheduled goroutine start, silently disabling the other two.
	listenerRunning  bool
	listenerStopChan chan struct{}

	queryRunning  bool
	queryStopChan chan struct{}

	scanRunning  bool
	scanStopChan chan struct{}

	// queryConn is the UDP socket for sending queries and receiving responses
	queryConn *net.UDPConn

	// Separate flag for query responder (host-side)
	queryResponderRunning bool
)

// StartDiscoveryBroadcaster sends UDP broadcast announcements so clients can find the host.
func StartDiscoveryBroadcaster(port int, stopChan chan struct{}) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", DiscoveryPort))
	if err != nil {
		log.Printf("[Discovery] Failed to resolve broadcast address: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("[Discovery] Failed to create broadcast socket: %v", err)
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("LEGION_HOST:0.0.0.0:%d", port)

	log.Printf("[Discovery] Started broadcasting on port %d", DiscoveryPort)

	ticker := time.NewTicker(DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := conn.Write([]byte(message))
			if err != nil {
				log.Printf("[Discovery] Broadcast error: %v", err)
			}
		case <-stopChan:
			log.Printf("[Discovery] Stopped broadcasting")
			return
		}
	}
}

// StartDiscoveryListener listens for UDP broadcast announcements from hosts.
func StartDiscoveryListener() {
	if listenerRunning {
		return
	}
	listenerRunning = true
	listenerStopChan = make(chan struct{})

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DiscoveryPort))
	if err != nil {
		log.Printf("[Discovery] Failed to resolve listen address: %v", err)
		listenerRunning = false
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[Discovery] Failed to listen on discovery port: %v", err)
		listenerRunning = false
		return
	}

	log.Printf("[Discovery] Listening for hosts on UDP port %d", DiscoveryPort)

	go func() {
		defer conn.Close()
		buf := make([]byte, 1024)

		for {
			select {
			case <-listenerStopChan:
				log.Printf("[Discovery] Stopped listening")
				return
			default:
				conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, src, err := conn.ReadFromUDP(buf)
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					log.Printf("[Discovery] Read error: %v", err)
					continue
				}

				msg := strings.TrimSpace(string(buf[:n]))
				if strings.HasPrefix(msg, "LEGION_HOST:") {
					hostInfo := strings.TrimPrefix(msg, "LEGION_HOST:")
					hostInfo = strings.Replace(hostInfo, "0.0.0.0", src.IP.String(), 1)
					addDiscoveredHost(hostInfo)
				}
			}
		}
	}()
}

// StartQuerySender sends UDP query packets to discover hosts.
// Runs in a goroutine to avoid blocking.
func StartQuerySender(gamePort int) {
	if queryRunning {
		return
	}
	queryRunning = true
	queryStopChan = make(chan struct{})

	// Create a UDP socket for receiving responses
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Printf("[Discovery] Failed to resolve query address: %v", err)
		queryRunning = false
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[Discovery] Failed to create query socket: %v", err)
		queryRunning = false
		return
	}
	queryConn = conn

	// Get the local port we're listening on
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Printf("[Discovery] Query sender started, listening on port %d", localAddr.Port)

	// Start listening for responses in a goroutine
	go receiveQueryResponses(conn)

	// Send queries to broadcast address
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", DiscoveryPort))
	if err != nil {
		log.Printf("[Discovery] Failed to resolve broadcast address: %v", err)
		return
	}

	queryMsg := fmt.Sprintf("LEGION_QUERY:%d", localAddr.Port)

	log.Printf("[Discovery] Sending queries to broadcast...")

	// Send query immediately, then tick
	conn.WriteToUDP([]byte(queryMsg), broadcastAddr)

	// Run query loop in a goroutine so it doesn't block
	go func() {
		ticker := time.NewTicker(DiscoveryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := conn.WriteToUDP([]byte(queryMsg), broadcastAddr)
				if err != nil {
					log.Printf("[Discovery] Query send error: %v", err)
				}
			case <-queryStopChan:
				log.Printf("[Discovery] Stopped query sender")
				return
			}
		}
	}()
}

// receiveQueryResponses listens for direct responses from hosts
func receiveQueryResponses(conn *net.UDPConn) {
	buf := make([]byte, 1024)

	for {
		select {
		case <-queryStopChan:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(QueryTimeout))
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("[Discovery] Query response error: %v", err)
				return
			}

			msg := strings.TrimSpace(string(buf[:n]))
			if strings.HasPrefix(msg, "LEGION_RESPONSE:") {
				hostInfo := strings.TrimPrefix(msg, "LEGION_RESPONSE:")
				// Replace 0.0.0.0 with actual source IP
				hostInfo = strings.Replace(hostInfo, "0.0.0.0", src.IP.String(), 1)
				addDiscoveredHost(hostInfo)
			}
		}
	}
}

// StartQueryResponder listens for query packets and responds directly to the client.
// This runs on the host side alongside the broadcaster.
func StartQueryResponder(gamePort int, stopChan chan struct{}) {
	if queryResponderRunning {
		return
	}
	queryResponderRunning = true

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DiscoveryPort))
	if err != nil {
		log.Printf("[Discovery] Failed to resolve responder address: %v", err)
		queryResponderRunning = false
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[Discovery] Failed to create responder socket: %v", err)
		queryResponderRunning = false
		return
	}
	defer conn.Close()

	log.Printf("[Discovery] Query responder started on port %d", DiscoveryPort)

	buf := make([]byte, 1024)

	for {
		select {
		case <-stopChan:
			log.Printf("[Discovery] Stopped query responder")
			queryResponderRunning = false
			return
		default:
			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("[Discovery] Responder read error: %v", err)
				queryResponderRunning = false
				return
			}

			msg := strings.TrimSpace(string(buf[:n]))
			if strings.HasPrefix(msg, "LEGION_QUERY:") {
				// Client is asking for our info, respond directly
				response := fmt.Sprintf("LEGION_RESPONSE:0.0.0.0:%d", gamePort)
				_, err := conn.WriteToUDP([]byte(response), src)
				if err != nil {
					log.Printf("[Discovery] Response send error: %v", err)
				} else {
					log.Printf("[Discovery] Responded to query from %s", src.String())
				}
			}
		}
	}
}

// StartTCPScan scans the local subnet for hosts running on the game port.
func StartTCPScan(gamePort int) {
	if scanRunning {
		return
	}
	scanRunning = true
	scanStopChan = make(chan struct{})

	log.Printf("[Discovery] Starting TCP scan for hosts on port %d", gamePort)

	go func() {
		defer func() {
			scanRunning = false
		}()

		localIP := getOutboundIP()
		if localIP == "127.0.0.1" {
			// Outbound probe failed (no default route / restricted data).
			// Fall back to the first non-loopback interface address so the
			// scan can still run on Android without mobile data.
			if fallback := firstLANIP(); fallback != "" {
				localIP = fallback
				log.Printf("[Discovery] Outbound probe failed, using interface IP %s", localIP)
			} else {
				log.Printf("[Discovery] Cannot determine local subnet, skipping TCP scan")
				return
			}
		}

		ipParts := strings.Split(localIP, ".")
		if len(ipParts) != 4 {
			return
		}

		subnet := fmt.Sprintf("%s.%s.%s", ipParts[0], ipParts[1], ipParts[2])
		log.Printf("[Discovery] Scanning subnet %s.* for port %d", subnet, gamePort)

		semaphore := make(chan struct{}, 20)
		var wg sync.WaitGroup

		stopped := false
		for i := 1; i <= 254 && !stopped; i++ {
			select {
			case <-scanStopChan:
				stopped = true
				continue
			default:
			}

			targetIP := fmt.Sprintf("%s.%d", subnet, i)
			if targetIP == localIP {
				continue
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(ip string) {
				defer wg.Done()
				defer func() { <-semaphore }()

				conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, gamePort), TCPScanTimeout)
				if err != nil {
					return
				}
				conn.Close()

				hostInfo := fmt.Sprintf("%s:%d", ip, gamePort)
				log.Printf("[Discovery] TCP scan found host: %s", hostInfo)
				addDiscoveredHost(hostInfo)
			}(targetIP)
		}

		wg.Wait()
		log.Printf("[Discovery] TCP scan complete, found %d hosts", len(discoveredHosts))
	}()
}

// getOutboundIP gets the preferred outbound IP address.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// firstLANIP returns the first non-loopback IPv4 address found on any network
// interface. Used as a fallback when the outbound probe (getOutboundIP) fails
// so the TCP scan can still run.
func firstLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

// addDiscoveredHost adds a host to the discovered list (avoiding duplicates).
func addDiscoveredHost(hostInfo string) {
	discoveredMutex.Lock()
	defer discoveredMutex.Unlock()

	for _, h := range discoveredHosts {
		if h == hostInfo {
			return
		}
	}
	log.Printf("[Discovery] Found host: %s", hostInfo)
	discoveredHosts = append(discoveredHosts, hostInfo)
}

// GetDiscoveredHosts returns the list of discovered hosts (IP:port).
func GetDiscoveredHosts() []string {
	discoveredMutex.Lock()
	defer discoveredMutex.Unlock()
	result := make([]string, len(discoveredHosts))
	copy(result, discoveredHosts)
	return result
}

// ClearDiscoveredHosts clears the list of discovered hosts.
func ClearDiscoveredHosts() {
	discoveredMutex.Lock()
	defer discoveredMutex.Unlock()
	discoveredHosts = nil
}

// StopDiscovery stops all discovery mechanisms.
func StopDiscovery() {
	if listenerStopChan != nil {
		close(listenerStopChan)
		listenerStopChan = nil
	}
	if queryStopChan != nil {
		close(queryStopChan)
		queryStopChan = nil
		if queryConn != nil {
			queryConn.Close()
			queryConn = nil
		}
	}
	if scanStopChan != nil {
		close(scanStopChan)
		scanStopChan = nil
	}
	listenerRunning = false
	queryRunning = false
	scanRunning = false
	// Also stop query responder if running
	if queryResponderRunning {
		queryResponderRunning = false
	}
}

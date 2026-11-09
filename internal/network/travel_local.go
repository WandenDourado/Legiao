package network

// Travelling between maps, for the whole party at once.
//
// Two things had to change together here. The map used to be a purely local
// affair — each machine swapped it on its own — and that is why the party
// could be split across two worlds with the host still simulating the one it
// had left. And the network layer does not own the loaded map (the game
// package does), so an arrival is QUEUED here and applied by the game loop on
// its next frame, exactly like a stage reset in reset_local.go.
//
// The decision belongs to one machine only. StartTravel is called by the host
// (or by a solo session, which is its own host), and it both announces the
// destination and queues it for the local player; a client never calls it, it
// only ever consumes what arrived.

import (
	"log"
	"sync"
)

var (
	travelMutex   sync.Mutex
	pendingTravel *TravelPayload
)

// StartTravel sends the party to targetMap. Broadcasting first and queueing
// second is deliberate: the host is about to reload its own world and retarget
// the simulation, and clients that hear about it a frame earlier simply start
// loading a moment sooner.
//
// Solo play (no host object) still goes through here, so the portal has ONE
// path in the code for one player and for four.
func StartTravel(targetMap, targetSpawn string) {
	if CurrentHost != nil {
		CurrentHost.broadcast(Message{Type: MsgTravel, Payload: MustMarshal(TravelPayload{
			TargetMap:   targetMap,
			TargetSpawn: targetSpawn,
		})})
	}
	log.Printf("[Travel] o grupo segue para %s", targetMap)
	RequestLocalTravel(TravelPayload{TargetMap: targetMap, TargetSpawn: targetSpawn})
}

// RequestLocalTravel queues an arrival for the game loop to apply.
//
// The last request wins. Two destinations in the same frame cannot happen from
// one host, and if they ever did, obeying the newest is the only answer that
// leaves everyone on the same map.
func RequestLocalTravel(t TravelPayload) {
	travelMutex.Lock()
	pendingTravel = &t
	travelMutex.Unlock()

	// The two snapshots held for interpolation belong to the map being left.
	// Without this, every remote player would SLIDE across the new map from
	// wherever they stood in the old one — the same trap RequestLocalReset
	// already covers.
	ResetInterpolation()
}

// ConsumeLocalTravel returns the queued destination and clears the request.
// The second result is false when there is nothing pending.
func ConsumeLocalTravel() (TravelPayload, bool) {
	travelMutex.Lock()
	defer travelMutex.Unlock()
	if pendingTravel == nil {
		return TravelPayload{}, false
	}
	t := *pendingTravel
	pendingTravel = nil
	return t, true
}

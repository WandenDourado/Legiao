package network

import (
    "log"
    "sync"

    "github.com/WandenDourado/Legiao/internal/skill"
)

// Global singleton instances for the current process.
// remotePlayers holds the latest known states of all players (including the local one).
// It is updated by the host when broadcasting state and by clients when receiving updates.

var (
    CurrentHost   *Host
    CurrentClient *Client
    // Role indicates whether this instance is acting as a host or client.
    Role string
    // LocalPlayerID stores the ID of the local player (set on join)
    LocalPlayerID string
    // LocalPlayerColor and LocalPlayerCharacter are cached at the original
    // join so a reconnect (reconnect.go) can resend the exact same identity
    // the host is expecting instead of asking the player to pick again.
    LocalPlayerColor     string
    LocalPlayerCharacter string
    // ServerAddress stores the IP:port being used (host's listen address or client's connected address)
    ServerAddress string
    // RemotePlayers is the shared map of all players, updated by network layer
    RemotePlayers      map[string]PlayerState
    RemotePlayersMutex sync.Mutex

    // NEW: RemoteEnemies holds the latest known states of all enemies (updated by host broadcast)
    RemoteEnemies      map[string]EnemyState
    RemoteEnemiesMutex sync.Mutex

    // NEW: RemoteProjectiles holds the latest known states of all projectiles
    RemoteProjectiles      map[string]ProjectileState
    RemoteProjectilesMutex sync.Mutex

    // NEW: Game state flags
    GameOver      bool
    LocalPlayerDead bool
    // LocalPlayerInPortal mirrors the host's PlayerState.InPortal for the
    // local player (game.SyncLocalPlayer). ProcessInput freezes on it the
    // same way it freezes on GameOver, and the renderer stops drawing the
    // local player while it is true — see host_portal_presence.go.
    LocalPlayerInPortal bool
    // LocalRespawnIn is how many seconds are left before the local player is
    // revived. The host counts it down and ships it in PlayerState; this is
    // only the mirror the HUD reads.
    LocalRespawnIn float32
    // TestMode is on when the local player pressed F2: no skill cooldowns and
    // no attack-speed cap. The host applies it per player, so this flag is
    // just what the local machine believes and draws.
    TestMode bool

    // CurrentWave is the latest horde state received from (or produced by) the
    // host. Read by the HUD on every frame, written by the network layer.
    CurrentWave      WaveState
    CurrentWaveMutex sync.Mutex

    // ClientSkills holds skill visuals (fireballs, explosions, ground fire,
    // sanctuaries) received from the host for rendering on a client. Healing
    // itself is synced via combat events, so this manager is visuals-only.
    ClientSkills *skill.Manager
)

// SendMessage sends a Message to the appropriate peer depending on the role.
func SendMessage(msg Message) error {
    switch Role {
    case "host":
        if CurrentHost != nil {
            // Host broadcasts the message to all peers (including itself).
            CurrentHost.broadcast(msg)
            return nil
        }
    case "client":
        if CurrentClient != nil {
            return CurrentClient.Send(msg)
        }
    }
    log.Printf("network.SendMessage called with no active connection (role=%s)", Role)
    return nil
}

// GetAllPlayers returns a copy of all known players (safe for reading)
func GetAllPlayers() map[string]PlayerState {
    RemotePlayersMutex.Lock()
    defer RemotePlayersMutex.Unlock()
    result := make(map[string]PlayerState, len(RemotePlayers))
    for k, v := range RemotePlayers {
        result[k] = v
    }
    return result
}

// UpdatePlayerState updates or adds a player in the shared state
func UpdatePlayerState(p PlayerState) {
    RemotePlayersMutex.Lock()
    defer RemotePlayersMutex.Unlock()
    if RemotePlayers == nil {
        RemotePlayers = make(map[string]PlayerState)
    }
    RemotePlayers[p.PlayerID] = p
}

// RemovePlayerState removes a player from the shared state
func RemovePlayerState(playerID string) {
    RemotePlayersMutex.Lock()
    defer RemotePlayersMutex.Unlock()
    delete(RemotePlayers, playerID)
}

// pruneRemotePlayersLocked removes from RemotePlayers any ID not present in
// current (the host's authoritative player list just broadcast). Called by
// the host role only, with RemotePlayersMutex already held. Without this, a
// player removed from h.players (tickAbsence, a bot taken over) lingers in
// RemotePlayers forever, since BroadcastStateUpdate/BroadcastRoster only ever
// write entries, never delete them.
func pruneRemotePlayersLocked(current []PlayerState) {
    live := make(map[string]bool, len(current))
    for _, p := range current {
        live[p.PlayerID] = true
    }
    for id := range RemotePlayers {
        if !live[id] {
            delete(RemotePlayers, id)
        }
    }
}

// SetWaveState stores the latest horde state.
func SetWaveState(w WaveState) {
    CurrentWaveMutex.Lock()
    defer CurrentWaveMutex.Unlock()
    CurrentWave = w
}

// GetWaveState returns a copy of the latest horde state.
func GetWaveState() WaveState {
    CurrentWaveMutex.Lock()
    defer CurrentWaveMutex.Unlock()
    return CurrentWave
}

// GetAllEnemies returns a copy of all known enemies (safe for reading)
func GetAllEnemies() map[string]EnemyState {
    RemoteEnemiesMutex.Lock()
    defer RemoteEnemiesMutex.Unlock()
    result := make(map[string]EnemyState, len(RemoteEnemies))
    for k, v := range RemoteEnemies {
        result[k] = v
    }
    return result
}

// UpdateEnemyState updates or adds an enemy in the shared state
func UpdateEnemyState(e EnemyState) {
    RemoteEnemiesMutex.Lock()
    defer RemoteEnemiesMutex.Unlock()
    if RemoteEnemies == nil {
        RemoteEnemies = make(map[string]EnemyState)
    }
    RemoteEnemies[e.EnemyID] = e
}

// RemoveEnemyState removes an enemy from the shared state
func RemoveEnemyState(enemyID string) {
    RemoteEnemiesMutex.Lock()
    defer RemoteEnemiesMutex.Unlock()
    delete(RemoteEnemies, enemyID)
}

// GetAllProjectiles returns a copy of all known projectiles (safe for reading)
func GetAllProjectiles() map[string]ProjectileState {
    RemoteProjectilesMutex.Lock()
    defer RemoteProjectilesMutex.Unlock()
    result := make(map[string]ProjectileState, len(RemoteProjectiles))
    for k, v := range RemoteProjectiles {
        result[k] = v
    }
    return result
}

// UpdateProjectileState updates or adds a projectile in the shared state
func UpdateProjectileState(p ProjectileState) {
    RemoteProjectilesMutex.Lock()
    defer RemoteProjectilesMutex.Unlock()
    if RemoteProjectiles == nil {
        RemoteProjectiles = make(map[string]ProjectileState)
    }
    RemoteProjectiles[p.ProjectileID] = p
}

// RemoveProjectileState removes a projectile from the shared state
func RemoveProjectileState(projectileID string) {
    RemoteProjectilesMutex.Lock()
    defer RemoteProjectilesMutex.Unlock()
    delete(RemoteProjectiles, projectileID)
}

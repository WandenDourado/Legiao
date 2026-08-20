package network

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Client struct {
	// mu guards conn/writer/reader across a reconnect swap (dropConn/rebind,
	// reconnect.go) AND serializes every write. One lock for both because a
	// reconnect replacing the connection out from under an in-flight Send is
	// exactly the corruption ClientConn.writeMu already prevents on the host
	// side, and reconnecting is what makes a second writer/swapper a real
	// possibility instead of a hypothetical one.
	mu     sync.Mutex
	conn   net.Conn
	writer *bufio.Writer
	reader *bufio.Reader
}

// clientIdentity recompoe os campos que o host para de mandar depois do
// primeiro snapshot de cada entidade. Ver wire.go.
//
// E variavel de pacote e nao campo do Client porque quem le o resultado
// (RemotePlayers, RemoteEnemies) tambem e, e porque so existe um cliente por
// processo.
var clientIdentity = newIdentityCache()

// ConnectClient connects to a host at the given address (ip:port).
func ConnectClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	setClientKeepAlive(conn)
	c := &Client{conn: conn, writer: bufio.NewWriter(conn), reader: bufio.NewReader(conn)}
	// Starts the silence clock fresh for THIS connection, not whenever a
	// previous session's last message happened to arrive (client_liveness.go).
	resetLiveness()
	go c.readLoop(conn)
	return c, nil
}

// readLoop reads from one specific connection until it fails. conn is passed
// explicitly (not read back off c.conn) so that when it errors, this loop can
// tell whether IT is still the active connection or whether a reconnect
// (reconnect.go's rebind) already replaced it out from under a silence
// timeout — in the latter case the error is stale news and must not trigger
// a second reconnect attempt on top of the one already running.
func (c *Client) readLoop(conn net.Conn) {
	reader := bufio.NewReader(conn)
	c.mu.Lock()
	c.reader = reader
	c.mu.Unlock()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			c.mu.Lock()
			isCurrent := c.conn == conn
			c.mu.Unlock()
			if !isCurrent {
				return
			}
			log.Printf("client decode error: %v", err)
			triggerReconnect("leitura falhou: " + err.Error())
			return
		}
		// Any successfully decoded line is proof of life, not only
		// state_update — client_liveness.go times out on total silence.
		markMessageReceived()
		msg, ok := parseMessage(line)
		if !ok {
			continue
		}
		// Coalesce whatever snapshot backlog is already sitting in the
		// buffer before handling — see client_backlog.go.
		c.handleMessage(c.drainStaleSnapshots(msg))
	}
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MsgStateUpdate:
		var state StateUpdatePayload
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			log.Printf("client failed to unmarshal state update: %v", err)
			return
		}
		// Sem log por jogador: isto chega a SnapshotHz. Ver o comentario em
		// Host.BroadcastStateUpdate.
		//
		// A identidade que o host omitiu volta aqui (wire.go), entao
		// RemotePlayers continua com todos os campos preenchidos e nada acima
		// da camada de rede percebe que o fio ficou mais magro.
		players := clientIdentity.restorePlayers(state.Players)
		snap := make(map[string]PlayerState, len(players))
		for _, p := range players {
			snap[p.PlayerID] = p
		}
		RemotePlayersMutex.Lock()
		RemotePlayers = snap
		RemotePlayersMutex.Unlock()
		// Guarda o instante para a interpolacao. Passa nil de inimigos: este
		// snapshot nao fala deles, e sobrescrever com vazio faria a horda
		// piscar entre as duas mensagens.
		recordSnapshot(snap, nil)

	case MsgEnemyUpdate:
		var enemyUpdate EnemyUpdatePayload
		if err := json.Unmarshal(msg.Payload, &enemyUpdate); err != nil {
			log.Printf("client failed to unmarshal enemy update: %v", err)
			return
		}
		// Tipo, cor e vida maxima chegam so no primeiro snapshot de cada
		// inimigo; o cache devolve o resto (wire.go).
		enemies := clientIdentity.restoreEnemies(enemyUpdate.Enemies)
		snap := make(map[string]EnemyState, len(enemies))
		for _, e := range enemies {
			snap[e.EnemyID] = e
		}
		RemoteEnemiesMutex.Lock()
		RemoteEnemies = snap
		RemoteEnemiesMutex.Unlock()
		SetWaveState(enemyUpdate.Wave)
		recordSnapshot(nil, snap)

	case MsgProjectileUpdate:
		c.handleProjectileUpdate(msg.Payload)

	case MsgCombatEvent:
		var combat CombatEventPayload
		if err := json.Unmarshal(msg.Payload, &combat); err != nil {
			log.Printf("client failed to unmarshal combat event: %v", err)
			return
		}
		// Sem log por evento de combate: e um tique de luta, nao um evento
		// raro, e cresce com o numero de inimigos em campo (doc/performance.md,
		// item N1). So a morte do jogador LOCAL e logada, abaixo.

		// Handle player death/respawn
		if combat.EntityType == "player" {
			if combat.EventType == "death" {
				// Mark player as dead
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.IsDead = true
					p.Health = 0
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
				if combat.EntityID == LocalPlayerID {
					LocalPlayerDead = true
					log.Printf("[Client] jogador local morreu")
				}
			} else if combat.EventType == "damage" {
				// Apply damage to player locally
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.Health -= combat.Value
					if p.Health < 0 {
						p.Health = 0
					}
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
			} else if combat.EventType == "respawn" {
				// Handle respawn
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.IsDead = false
					p.Health = combat.Value
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
				if combat.EntityID == LocalPlayerID {
					LocalPlayerDead = false
				}
			} else if combat.EventType == "shield" {
				// Paladina shield sync: value > 0 activates/updates the aura,
				// value <= 0 breaks it (shatter animation plays client-side).
				if ClientSkills == nil {
					ClientSkills = skill.NewManager()
				}
				if combat.Value > 0 {
					if ClientSkills.HasShield(combat.EntityID) {
						ClientSkills.SetShieldHP(combat.EntityID, combat.Value)
					} else {
						pos := rl.NewVector2(0, 0)
						RemotePlayersMutex.Lock()
						if p, ok := RemotePlayers[combat.EntityID]; ok {
							pos = rl.NewVector2(float32(p.X), float32(p.Y))
						}
						RemotePlayersMutex.Unlock()
						skill.ActivateShield(ClientSkills, combat.EntityID, pos)
						ClientSkills.SetShieldHP(combat.EntityID, combat.Value)
					}
				} else {
					ClientSkills.SetShieldHP(combat.EntityID, 0)
				}
			} else if combat.EventType == "heal" {
				// Apply continuous heal from a sanctuary (synced from host).
				RemotePlayersMutex.Lock()
				if p, ok := RemotePlayers[combat.EntityID]; ok {
					p.Health = combat.Value
					if p.Health > p.MaxHealth {
						p.Health = p.MaxHealth
					}
					RemotePlayers[combat.EntityID] = p
				}
				RemotePlayersMutex.Unlock()
			}
		}

	case MsgDialogue:
		// Display only: the host decides which line is on screen and when it
		// ends. A client never advances or skips a dialogue.
		var d DialogueState
		if err := json.Unmarshal(msg.Payload, &d); err != nil {
			log.Printf("client failed to unmarshal dialogue: %v", err)
			return
		}
		SetDialogueState(d)

	case MsgGameOver:
		var gameOver GameOverPayload
		if err := json.Unmarshal(msg.Payload, &gameOver); err != nil {
			log.Printf("client failed to unmarshal game over: %v", err)
			return
		}
		log.Printf("[Client] GAME OVER: %s", gameOver.Message)
		GameOver = true

	case MsgCooldown:
		// Display only: the host owns the timers, this is the snapshot the
		// HUD counts down from.
		var cd CooldownPayload
		if err := json.Unmarshal(msg.Payload, &cd); err != nil {
			log.Printf("client failed to unmarshal cooldowns: %v", err)
			return
		}
		SetCooldowns(cd.Players)

	case MsgReset:
		var rp ResetPayload
		if err := json.Unmarshal(msg.Payload, &rp); err != nil {
			log.Printf("client failed to unmarshal reset: %v", err)
			return
		}
		log.Printf("[Client] host reiniciou a fase")
		c.applyStageReset(rp)

	case MsgTravel:
		var t TravelPayload
		if err := json.Unmarshal(msg.Payload, &t); err != nil {
			log.Printf("client failed to unmarshal travel: %v", err)
			return
		}
		log.Printf("[Client] o grupo atravessou o portal para %s", t.TargetMap)
		c.applyTravel(t)

	case MsgUltimateGrant:
		// Display/gate only, exactly like MsgCooldown: the host decided the
		// rescue happened, this client just needs to stop refusing the button
		// and the key. See progression.go.
		var g UltimateGrantPayload
		if err := json.Unmarshal(msg.Payload, &g); err != nil {
			log.Printf("client failed to unmarshal ultimate grant: %v", err)
			return
		}
		GrantUltimateForRun(entity.CharacterType(g.Character))

	case MsgRespawn:
		var respawn RespawnPayload
		if err := json.Unmarshal(msg.Payload, &respawn); err != nil {
			log.Printf("client failed to unmarshal respawn: %v", err)
			return
		}
		log.Printf("[Client] Respawn: player %s at (%d,%d) with health %.0f", respawn.PlayerID, respawn.X, respawn.Y, respawn.Health)
		// Update the existing snapshot in place: replacing the whole struct
		// would drop the player's colour and character and leave them drawn
		// as a default Mago until the next full state update.
		RemotePlayersMutex.Lock()
		if RemotePlayers == nil {
			RemotePlayers = make(map[string]PlayerState)
		}
		p := RemotePlayers[respawn.PlayerID]
		p.PlayerID = respawn.PlayerID
		p.X = respawn.X
		p.Y = respawn.Y
		p.Health = respawn.Health
		p.IsDead = false
		p.RespawnIn = 0
		RemotePlayers[respawn.PlayerID] = p
		RemotePlayersMutex.Unlock()
		if respawn.PlayerID == LocalPlayerID {
			LocalPlayerDead = false
			LocalRespawnIn = 0
		}

	case MsgFireEvent:
		var fe FireEventPayload
		if err := json.Unmarshal(msg.Payload, &fe); err != nil {
			log.Printf("client failed to unmarshal fire event: %v", err)
			return
		}
		if ClientSkills == nil {
			ClientSkills = skill.NewManager()
		}
		pos := rl.NewVector2(float32(fe.X), float32(fe.Y))
		if fe.EventType == "cast" {
			// Launch: spawn the traveling fireball visual (no explosion).
			skill.SpawnFireball(ClientSkills, fe.OwnerID, pos,
				rl.NewVector2(fe.DirX, fe.DirY))
		} else {
			// Impact: explosion + ground fire, and remove the fireball
			// visual that just landed there.
			ClientSkills.RemoveFireballsNear(pos, skill.FireballRadius*2)
			ClientSkills.AddExplosion(skill.NewExplosion(pos, fe.Radius))
			ClientSkills.AddFireGround(skill.NewFireGround(pos))
		}

	case MsgArrowVolley:
		var av ArrowVolleyPayload
		if err := json.Unmarshal(msg.Payload, &av); err != nil {
			log.Printf("client failed to unmarshal arrow volley: %v", err)
			return
		}
		if ClientSkills == nil {
			ClientSkills = skill.NewManager()
		}
		skill.SpawnArrowVolley(ClientSkills, av.OwnerID,
			rl.NewVector2(float32(av.X), float32(av.Y)),
			rl.NewVector2(av.DirX, av.DirY))

	case MsgUltimate:
		var ue UltimateEventPayload
		if err := json.Unmarshal(msg.Payload, &ue); err != nil {
			log.Printf("client failed to unmarshal ultimate event: %v", err)
			return
		}
		if ClientSkills == nil {
			ClientSkills = skill.NewManager()
		}
		pos := rl.NewVector2(float32(ue.X), float32(ue.Y))
		switch ue.Skill {
		case "meteor_rain":
			// One event per meteor: replicate the fall toward this target.
			ClientSkills.AddMeteor(skill.NewMeteor(pos))
		case "angelic_area":
			skill.ActivateAngelic(ClientSkills, ue.OwnerID, pos)
			noteLastStandNPC(ue.OwnerID, pos)
		case "celestial_arrows":
			skill.SpawnCelestialArrow(ClientSkills, ue.OwnerID, pos,
				rl.NewVector2(ue.DirX, ue.DirY))
		case "divine_avatar":
			skill.ActivateAvatar(ClientSkills, ue.OwnerID, pos)
			// So importa quando o dono e o NPC do ultimo suspiro do mapa 6 —
			// para uma jogadora de verdade lancando a propria suprema,
			// isLastStandNPC(ue.OwnerID) e falso e isto nao faz nada.
			noteLastStandNPC(ue.OwnerID, pos)
		case "graveyard":
			skill.SpawnGraveyard(ClientSkills, ue.OwnerID, pos,
				rl.NewVector2(ue.DirX, ue.DirY))
		case "spectral_legion":
			skill.ActivateLegion(ClientSkills, ue.OwnerID, pos)
			// A magia do heroi invocado chega pela mesma mensagem que a de um
			// jogador; o DONO e que diz qual e qual. E assim que o cliente
			// aprende que ha um NPC em campo, sem mensagem nova de protocolo
			// para uma coisa que acontece uma vez por fase.
			noteLastStandNPC(ue.OwnerID, pos)
		case "specter_die":
			// One specter perished on the host: mirror it locally.
			ClientSkills.KillLegionSpecterNear(ue.OwnerID, pos)
		case "legion_end":
			ClientSkills.DissolveLegion(ue.OwnerID)
			if isLastStandNPC(ue.OwnerID) {
				setLastStandNPC(rl.Vector2{}, false)
			}
		case "angelic_end":
			// O altar do cliente normalmente expira sozinho pelo tempo dele;
			// ClearOwner cobre o caso em que o host encerrou mais cedo (um
			// bot tomado por um humano no meio da ultimate — host_bot_takeover.go).
			// Sem ela a Sacerdotisa invocada ficaria desenhada sobre um
			// altar que ja apagou.
			ClientSkills.ClearOwner(ue.OwnerID)
			if isLastStandNPC(ue.OwnerID) {
				setLastStandNPC(rl.Vector2{}, false)
			}
		case "avatar_end":
			// Mesma ideia: o avatar do cliente normalmente expira sozinho
			// pelo tempo dele (ClientSkills.UpdateAvatars); ClearOwner cobre
			// o encerramento antecipado por uma tomada de posse.
			ClientSkills.ClearOwner(ue.OwnerID)
			if isLastStandNPC(ue.OwnerID) {
				setLastStandNPC(rl.Vector2{}, false)
			}
		}

	case MsgMelee:
		var mp MeleePayload
		if err := json.Unmarshal(msg.Payload, &mp); err != nil {
			log.Printf("client failed to unmarshal melee event: %v", err)
			return
		}
		if ClientSkills == nil {
			ClientSkills = skill.NewManager()
		}
		// Anchor at the owner's current known position (it will keep following
		// the owner every frame via AdvanceClientSkills).
		pos := rl.NewVector2(float32(mp.X), float32(mp.Y))
		RemotePlayersMutex.Lock()
		if p, ok := RemotePlayers[mp.OwnerID]; ok {
			pos = rl.NewVector2(float32(p.X), float32(p.Y))
		}
		RemotePlayersMutex.Unlock()
		sweep := skill.NewSwordSweep(mp.OwnerID, pos,
			rl.NewVector2(mp.DirX, mp.DirY))
		sweep.Empowered = mp.Empowered
		ClientSkills.AddSword(sweep)

	case MsgSanctuary:
		var sp SanctuaryPayload
		if err := json.Unmarshal(msg.Payload, &sp); err != nil {
			log.Printf("client failed to unmarshal sanctuary: %v", err)
			return
		}
		if ClientSkills == nil {
			ClientSkills = skill.NewManager()
		}
		center := rl.NewVector2(float32(sp.X), float32(sp.Y))
		// One sanctuary per caster (owner == id). Skip if already present.
		if _, ok := ClientSkills.Sanctuaries[sp.SanctuaryID]; ok {
			return
		}
		skill.SpawnSanctuary(ClientSkills, sp.OwnerID, center)

	case MsgSentryOrb:
		var so SentryOrbPayload
		if err := json.Unmarshal(msg.Payload, &so); err != nil {
			log.Printf("client failed to unmarshal sentry orb: %v", err)
			return
		}
		pos := rl.NewVector2(float32(so.X), float32(so.Y))
		switch so.EventType {
		case "cast":
			// O ID vem do host, e nao e gerado aqui: e ele que liga esta
			// esfera ao "impact" que vai chegar depois.
			//
			// O alvo entra pela posicao JA replicada do jogador, e nao por pos:
			// sem isso a esfera nasceria com rumo padrao e daria um solavanco
			// no primeiro quadro, ao corrigir de uma vez para o alvo real.
			target := pos
			if p, ok := sentryOrbTarget(so.TargetID); ok {
				target = p
			}
			skill.SpawnSentryOrb(false, so.OrbID, "", so.TargetID, pos, target, so.TTL)
		case "impact":
			// AddSentryBurstAt antes do Remove: o estouro tem que aparecer
			// mesmo se o "cast" desta esfera se perdeu no caminho.
			skill.RemoveSentryOrb(false, so.OrbID, false)
			skill.AddSentryBurstAt(false, pos)
		case "expire":
			skill.RemoveSentryOrb(false, so.OrbID, false)
		}

	case MsgCannonBall:
		var cb CannonBallPayload
		if err := json.Unmarshal(msg.Payload, &cb); err != nil {
			log.Printf("client failed to unmarshal cannon ball: %v", err)
			return
		}
		pos := rl.NewVector2(float32(cb.X), float32(cb.Y))
		switch cb.EventType {
		case "cast":
			dir := rl.NewVector2(cb.DirX, cb.DirY)
			skill.SpawnCannonBall(false, cb.BallID, "", pos, dir)
		case "impact":
			// AddCannonBurstAt antes do Remove, pela mesma razao da esfera da
			// sentinela: a explosao tem que aparecer mesmo se o "cast" desta
			// bola se perdeu no caminho.
			skill.RemoveCannonBall(false, cb.BallID, false)
			skill.AddCannonBurstAt(false, pos)
		case "expire":
			skill.RemoveCannonBall(false, cb.BallID, false)
		}

	default:
		log.Printf("[Client] Received message type: %s", msg.Type)
	}
}

// errNoConnection is returned by Send while the client is between
// connections (dropConn already ran, rebind has not run yet). The game loop
// calls Send every frame for MsgInput regardless of connection state, so
// this has to fail without logging — see the no-log note below.
var errNoConnection = errors.New("network: no active connection")

func (c *Client) Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	// Sem log aqui: o laco do jogo chama Send a cada quadro publicado de
	// MsgInput (client_input_rate.go), e logar o JSON inteiro por linha (ou
	// o erro de "sem conexao" enquanto reconecta) era o maior custo de
	// CPU/calor do cliente Android (doc/performance.md, item N1/mobile).
	//
	// O mesmo lock que guarda a troca de conexao em dropConn/rebind serializa
	// a escrita: sem isso, uma reconexao trocando o writer NO MEIO de um
	// Send corromperia o fluxo de JSON do outro lado, exatamente como
	// ClientConn.writeMu ja evita do lado do host.
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writer == nil {
		return errNoConnection
	}
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	return c.writer.Flush()
}

// Close tears down the connection for good. Unlike dropConn (reconnect.go),
// which drops the socket but keeps the Client alive for a retry, Close is
// for ending the session outright.
func (c *Client) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn, c.writer = nil, nil
	c.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

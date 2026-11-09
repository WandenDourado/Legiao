package network

// Client-side per-frame advancement for skill visuals that need player
// positions (the Paladina shield aura follows its owner).

import (
	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// livingPlayerPositionsClient monta player_id -> posicao a partir do snapshot,
// que e a unica verdade que o cliente tem sobre onde os jogadores estao.
//
// Os mortos ficam de fora de proposito: um alvo ausente do mapa faz a esfera
// da sentinela seguir reto em vez de perseguir um corpo, que e exatamente o
// que o host faz do outro lado.
func livingPlayerPositionsClient() map[string]rl.Vector2 {
	RemotePlayersMutex.Lock()
	defer RemotePlayersMutex.Unlock()
	out := make(map[string]rl.Vector2, len(RemotePlayers))
	for id, p := range RemotePlayers {
		if p.IsDead {
			continue
		}
		out[id] = rl.NewVector2(float32(p.X), float32(p.Y))
	}
	return out
}

// sentryOrbTarget devolve a posicao replicada de um jogador, se o cliente ja
// souber dela.
func sentryOrbTarget(playerID string) (rl.Vector2, bool) {
	RemotePlayersMutex.Lock()
	defer RemotePlayersMutex.Unlock()
	p, ok := RemotePlayers[playerID]
	if !ok {
		return rl.Vector2{}, false
	}
	return rl.NewVector2(float32(p.X), float32(p.Y)), true
}

// AdvanceClientSkills animates client-side skill visuals for one frame:
// arrows fly out, shield auras / sanctuary auras / angelic areas / avatars
// follow their owners,
// meteors fall, spectral legions hunt (respecting the map's blocked space in
// solid). Fireball and sanctuary visuals keep their dedicated calls.
//
// solid e o CollisionGrid do mapa carregado, o mesmo objeto que o host recebe
// em SetSolid. Ate 22/08/2026 isto recebia a lista plana de retangulos,
// remontada a cada quadro pelo laco do jogo — duas vezes errado de uma vez,
// e o segundo erro era o caro: ver skill/legion.go, funcao `blocked`.
func AdvanceClientSkills(dt float32, solid collision.Solid) {
	if ClientSkills == nil {
		return
	}
	ClientSkills.AdvanceFireballs(dt)
	ClientSkills.AdvanceArrows(dt)
	ClientSkills.AdvanceMeteors(dt)
	ClientSkills.AdvanceAngelics(dt)
	ClientSkills.AdvanceCelestials(dt)
	ClientSkills.AdvanceGraveyards(dt)
	// Espinhoes e nevoa da chefe (map 7) sao so desenho aqui — o dano e
	// resolvido no host (host_boss.go) e chega pelos eventos de combate de
	// sempre. Mas o desenho de cada um DEPENDE de idade (Thorn.Age,
	// DarkFog.age/TTL: telegrafo, ondulacao, fade), entao sem envelhecer os
	// dois aqui a marca do espinho ficaria congelada no primeiro quadro e a
	// nevoa nunca apareceria (DarkFog.Draw usa `age` para o fade de entrada).
	// O retorno de AdvanceFog (contagem/valor de dano) e ignorado de
	// proposito: aplicar dano e decisao do host, nao deste cliente.
	ClientSkills.AdvanceThorns(dt)
	ClientSkills.AdvanceFog(dt)

	RemotePlayersMutex.Lock()
	for id, p := range RemotePlayers {
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		ClientSkills.SetShieldAnchor(id, pos)
		ClientSkills.SetAvatarAnchor(id, pos)
		// O selo da Chuva de Meteoros segue o Mago aqui pelo mesmo caminho do
		// avatar. Ele fica ANTES do desvio dos mortos logo abaixo de
		// proposito: a chuva nao para quando quem a lancou cai, entao a marca
		// que diz de onde ela veio tambem nao para.
		ClientSkills.SetMeteorSigilAnchor(id, pos)
		ClientSkills.SetSwordAnchor(id, pos)
		ClientSkills.SetLegionAnchor(id, pos)
		// A aura do Santuario segue a Sacerdotisa como o escudo segue a
		// Paladina — e some com o dono pelo mesmo motivo: uma aura de cura
		// desenhada em cima de um corpo caido anuncia uma cura que o host
		// nao esta dando (ele so cura quem esta vivo).
		if p.IsDead {
			ClientSkills.RemoveSanctuary(id)
			continue
		}
		ClientSkills.SetSanctuaryAnchor(id, pos)
	}
	RemotePlayersMutex.Unlock()
	ClientSkills.UpdateShields(dt)
	ClientSkills.UpdateAvatars(dt)
	ClientSkills.UpdateMeteorSigils(dt)
	ClientSkills.AdvanceSwords(dt)

	// Spectral legions chase the latest enemy snapshots (visuals only).
	RemoteEnemiesMutex.Lock()
	enemyPos := make([]rl.Vector2, 0, len(RemoteEnemies))
	for _, e := range RemoteEnemies {
		enemyPos = append(enemyPos, rl.NewVector2(float32(e.X), float32(e.Y)))
	}
	RemoteEnemiesMutex.Unlock()
	ClientSkills.AdvanceLegions(dt, enemyPos, solid)

	// As esferas da sentinela nao vivem no Manager (sao de um monstro), entao
	// avancam por fora dele - mas no mesmo quadro e com o mesmo dt.
	skill.AdvanceSentryOrbs(dt, livingPlayerPositionsClient())
	skill.UpdateSentryBursts(false, dt)

	// As bolas de canhao, pela mesma razao. Nao precisam de posicoes de
	// jogador: a trajetoria e reta, nao perseguidora.
	skill.AdvanceCannonBalls(dt)
	skill.UpdateCannonBursts(false, dt)
}

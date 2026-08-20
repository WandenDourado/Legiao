package network

// Os tres relogios da Senhora das Trevas.
//
// Eles nao cabiam em `AttackCooldown`, que e UM numero: espinhao a cada 15 s e
// nevoa a cada 60 s sao independentes, e a horda tem o proprio ritmo na maquina
// de hordas. Cada um vive aqui com o seu contador.
//
// UM ALINHAMENTO A CADA 7 MINUTOS, E ELE E DE PROPOSITO. 60 e 70 tem MMC 420, e
// o 15 do espinhao divide os dois: a cada 420 s nevoa, horda e golpe caem
// juntos. Nao e defeito do desenho — e o que da a uma luta longa uma onda em
// vez de uma parede constante.
//
// Tudo aqui e do host, e por enquanto SO do host: nada disto e replicado. Num
// cliente a chefe fica em idle e nem a marca do espinhao nem a nevoa aparecem —
// o dano chega pelos eventos de combate, mas sem aviso nenhum. Fechar isso
// exige o `Anim` da chefe e dois eventos novos no protocolo, e esta anotado
// como a lacuna que falta antes de a fase ser jogavel em rede.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// bossThornEvery e o periodo do espinhao.
	bossThornEvery float32 = 15.0
	// bossFogEvery e o periodo da conjuracao.
	bossFogEvery float32 = 60.0
	// bossFirstBeat atrasa o primeiro de cada um. O grupo acabou de atravessar
	// um portal: levar um golpe antes de a camera assentar nao ensina nada.
	bossFirstBeat float32 = 6.0
	// bossSentryEvery e o periodo com que uma gargula nova sobe a cada lado.
	//
	// Ela NAO vem de `WaveDef.Sentries`, e o motivo e a forma da fase: aquele
	// campo e por HORDA, e a arena tem uma horda so, infinita — declarar
	// `Sentries: 2` arma duas no comeco e para para sempre. O acumulo tinha de
	// virar relogio, e o periodo e o mesmo da leva para que cada cerco novo
	// chegue com uma torre nova.
	bossSentryEvery float32 = 70.0
	// bossSentryStep e quantas sobem por ciclo: uma a oeste e uma a leste. A
	// ordem dos postos em `sentryPosts` ja alterna os lados, entao dois por
	// ciclo e exatamente "uma em cada portao".
	bossSentryStep = 2
	// bossCastLead e o aviso da nevoa: a danca (`cast_loop`) roda este tempo
	// antes de a nevoa entrar no ar. E o que torna a coordenacao possivel em
	// vez de sorte — o grupo ve e corre para o altar.
	bossCastLead float32 = 2.4
)

// bossClocks sao os contadores da fase. Zerados a cada `ResetStage`.
type bossClocks struct {
	thorn   float32
	fog     float32
	sentry  float32
	casting float32 // > 0 enquanto a danca roda e a nevoa ainda nao entrou
}

// ResetBossClocks rearma os relogios. Uma tentativa nova e uma luta nova.
func (h *Host) ResetBossClocks() {
	// O relogio da sentinela comeca em um ciclo cheio: a primeira dupla ja veio
	// do `Sentries: 2` da propria horda, e armar mais duas nos primeiros
	// segundos poria quatro torres em campo antes do grupo ter visto uma.
	h.boss = bossClocks{
		thorn:  bossFirstBeat,
		fog:    bossFirstBeat + bossFogEvery,
		sentry: bossSentryEvery,
	}
	if h.Skills != nil {
		h.Skills.ClearBossEffects()
	}
}

// updateBoss faz a chefe agir. Chamado uma vez por quadro, depois das hordas.
func (h *Host) updateBoss(dt float32) {
	if !h.bossPresent {
		return
	}
	boss, alive := h.livingBoss()

	// Espinhoes e nevoa ja no ar continuam resolvendo mesmo com a chefe morta:
	// o que ela lancou antes de cair ainda esta acontecendo, e apagar isso no
	// quadro da morte devolveria ao grupo uma vitoria que ele ainda nao teve.
	h.resolveThorns(dt)
	h.resolveFog(dt)

	if !alive {
		return
	}

	if h.boss.casting > 0 {
		if h.boss.casting -= dt; h.boss.casting <= 0 {
			skill.ActivateDarkFog(h.Skills, h.arenaBounds())
		}
	}

	if h.boss.thorn -= dt; h.boss.thorn <= 0 {
		h.boss.thorn = bossThornEvery
		h.castThorns(boss)
	}

	if h.boss.sentry -= dt; h.boss.sentry <= 0 {
		h.boss.sentry = bossSentryEvery
		// `armSentries` recebe o TOTAL acumulado, nao um acrescimo, e ele
		// mesmo para quando os dez postos acabam. O cursor conta postos
		// ocupados e nao gargulas vivas: uma torre derrubada fica derrubada.
		h.armSentries(h.stageMap, h.stageSentries, h.sentriesArmed+bossSentryStep)
	}

	if h.boss.fog -= dt; h.boss.fog <= 0 {
		h.boss.fog = bossFogEvery
		h.boss.casting = bossCastLead
		boss.TriggerCast()
	}
}

// castThorns tira a FOTO das posicoes e marca o chao.
//
// A foto e o contrato da habilidade: os espinhoes nascem onde os jogadores
// estavam quando ela levantou os bracos, e nao seguem ninguem. Um espinho que
// perseguisse tiraria do desvio a condicao de decisao.
func (h *Host) castThorns(boss *entity.Enemy) {
	positions := h.getAlivePlayerPositions()
	if len(positions) == 0 {
		return
	}
	boss.TriggerStrike()
	skill.SpawnThorns(h.Skills, positions)
}

// resolveThorns envelhece os espinhoes e cobra o dano dos que irromperam.
func (h *Host) resolveThorns(dt float32) {
	h.Skills.AdvanceThorns(dt)
	for _, t := range h.Skills.ThornsErupting() {
		for _, id := range h.playersInsideThorn(t) {
			h.applyBossDamage(id, skill.ThornDamage, false)
		}
	}
}

// playersInsideThorn devolve quem esta no alcance de um espinho no instante em
// que ele crava.
func (h *Host) playersInsideThorn(t *skill.Thorn) []string {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	var out []string
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		if t.Contains(rl.NewVector2(float32(p.X), float32(p.Y))) {
			out = append(out, id)
		}
	}
	return out
}

// resolveFog cobra o dano continuo da nevoa.
func (h *Host) resolveFog(dt float32) {
	times, dmg := h.Skills.AdvanceFog(dt)
	if times <= 0 {
		return
	}
	for _, id := range h.playersUnderFog() {
		for i := 0; i < times; i++ {
			h.applyBossDamage(id, dmg, true)
		}
	}
}

// playersUnderFog devolve quem esta na nevoa E desprotegido.
//
// AS DUAS SAIDAS SAO CONSULTADAS AQUI, e nao no dano: `applyBossDamage` ja
// respeita Avatar e invulnerabilidade para qualquer fonte, mas a Area Angelical
// nao e uma propriedade do jogador — e um lugar. Quem se salva e quem esta
// EM CIMA do altar, nao quem o conjurou, e e por isso que a pergunta e
// `AngelicContains(posicao)` e nao `HasAngelic(id)`.
func (h *Host) playersUnderFog() []string {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	var out []string
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if !h.Skills.FogContains(pos) {
			continue
		}
		if h.Skills.AngelicContains(pos) {
			continue
		}
		out = append(out, id)
	}
	return out
}

// applyBossDamage tira vida de um jogador, respeitando escudo, Avatar e
// invulnerabilidade. Mesma forma de `applySentryOrbDamage`.
//
// `silent` existe para a nevoa: ela cobra duas vezes por segundo, e um evento
// de combate por cobranca por jogador entulharia a rede com o que o cliente ja
// consegue inferir. A morte, essa, e sempre anunciada.
func (h *Host) applyBossDamage(playerID string, dmg float32, silent bool) {
	if h.Skills.HasAvatar(playerID) || h.IsInvulnerable(playerID) {
		return
	}
	if leftover, hpAfter, had := h.Skills.AbsorbShieldDamage(playerID, dmg); had {
		h.broadcastShieldEvent(playerID, hpAfter)
		if leftover <= 0 {
			return
		}
		dmg = leftover
	}

	h.playersMutex.Lock()
	p, ok := h.players[playerID]
	if !ok || p.IsDead {
		h.playersMutex.Unlock()
		return
	}
	p.Health -= dmg
	dead := p.Health <= 0
	if dead {
		h.markPlayerDead(p)
	}
	h.playersMutex.Unlock()

	switch {
	case dead:
		h.broadcastCombatEvent("death", playerID, "player", 0, "")
	case !silent:
		h.broadcastCombatEvent("damage", playerID, "player", dmg, "")
	}
}

// livingBoss acha a chefe viva em campo.
func (h *Host) livingBoss() (*entity.Enemy, bool) {
	for _, e := range h.EntityManager.GetAllEnemies() {
		if e != nil && e.IsActive && e.Type == h.bossType {
			return e, true
		}
	}
	return nil, false
}

// arenaBounds e a regiao que a nevoa cobre: o mundo inteiro do mapa.
//
// A nevoa nao usa a zona `arena` do mapa de proposito. A zona existe para a
// janela do climax e para o layout; se a nevoa dependesse dela, um mapa novo com
// chefe e sem a zona teria uma conjuracao que nao machuca ninguem, em silencio.
// Os limites do mundo sempre existem.
func (h *Host) arenaBounds() rl.Rectangle {
	b := h.WorldBounds
	return rl.NewRectangle(0, 0, b.Width, b.Height)
}

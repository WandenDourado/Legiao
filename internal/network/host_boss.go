package network

// Os tres relogios da Senhora das Trevas.
//
// Eles nao cabiam em `AttackCooldown`, que e UM numero: espinhao, nevoa e a
// horda (esta ultima BossDriven, ver host_waves.go) sao eventos com relogio
// proprio. Cada um vive aqui com o seu contador.
//
// Ate 30/08/2026 o alinhamento era A CADA 7 MINUTOS, de proposito: 15 s de
// espinhao, 60 s de nevoa e 70 s de horda tem MMC 420, entao a cada 420 s os
// tres caiam juntos. Esse alinhamento MORREU em 31/08/2026
// (doc/plan_world07_arena_tuning.md): espinhao passou para 10 s e nevoa para
// 30 s, e a horda deixou de ter relogio proprio — ela dispara junto da nevoa
// (bossFogEvery), entao os dois passam a bater no MESMO instante, a cada 30 s.
// A gargula (bossSentryEvery, 70 s) e o UNICO relogio que fica fora dessa
// fase de proposito: se ela tambem batesse a cada 30 s, o reforco das torres
// viraria so mais um item da mesma batida em vez de uma pressao a parte.
//
// Ate 31/08/2026 tudo aqui era do host, e SO do host: nada disto era
// replicado, e um cliente via a chefe em idle sem marca de espinhao nem
// nevoa — o dano chegava pelos eventos de combate, mas sem aviso nenhum
// (doc/plan_correcoes_mobile_e_rede.md, item 6). MsgBossFog e MsgBossThorn
// (protocol.go) fecham essa lacuna: broadcastBossFog e broadcastBossThorn
// abaixo sao a metade do host desse conserto, irmas de broadcastSentryOrb e
// broadcastCannonBall — magia de MONSTRO, entao nao viaja em MsgSkill.

import (
	"encoding/json"
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// bossThornEvery e o periodo do espinhao.
	//
	// Afinado de 15 para 10 em 31/08/2026 (doc/plan_world07_arena_tuning.md):
	// o ciclo da fase passou a bater a cada bossFogEvery (30 s), e o espinhao
	// mais frequente e o que enche esse intervalo com pressao continua em vez
	// de deixa-lo vazio entre uma nevoa e outra.
	bossThornEvery float32 = 10.0
	// bossFogEvery e o periodo da conjuracao. Afinado de 60 para 30 em
	// 31/08/2026: e o relogio de fase agora — a horda (BossDriven, ver
	// host_waves.go) dispara junto dela, no mesmo quadro.
	bossFogEvery float32 = 30.0
	// bossFirstBeat atrasa o primeiro de cada um. O grupo acabou de atravessar
	// um portal: levar um golpe antes de a camera assentar nao ensina nada.
	bossFirstBeat float32 = 6.0
	// bossSentryEvery e o periodo com que gargulas novas sobem. Fica em 70 s
	// DE PROPOSITO, fora do compasso de 30 s do resto da fase: e o unico
	// reforco que nao bate junto com nevoa/horda/espinhao, para nao virar so
	// mais um item da mesma batida.
	//
	// Ela NAO vem de `WaveDef.Sentries`, e o motivo e a forma da fase: aquele
	// campo e por HORDA, e a arena tem uma horda so, infinita — declarar
	// `Sentries: 2` arma duas no comeco e para para sempre. O acumulo tinha de
	// virar relogio, e o periodo e o mesmo da leva para que cada cerco novo
	// chegue com uma torre nova.
	bossSentryEvery float32 = 70.0
	// bossSentryStep e quantas sobem por ciclo. Dobrado de 2 para 4 em
	// 31/08/2026: duas a oeste e duas a leste. A ordem dos postos em
	// `sentryPosts` ja alterna os lados par a impar, entao tomar 4 nomes
	// seguidos continua dando 2+2 por portao.
	bossSentryStep = 4
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
	// contactTick acumula o tempo para o anel de contato (host_boss_contact.go).
	// Comeca em 0 pelo mesmo motivo dos outros: bossClocks{} zerado por
	// ResetBossClocks e um cronometro novo, e o primeiro tick chega
	// BossContactTickEvery segundos depois da chefe nascer.
	contactTick float32
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

	// Espinhoes e nevoa resolvem AQUI, antes do `if !alive`, so para o
	// quadro em que a chefe cai: o que ja estava para cravar neste tique
	// ainda crava. Depois disso nao sobra nada para resolver —
	// clearThreatsOnVictory (host_boss_victory.go) apaga os dois no mesmo
	// quadro da queda. Ate 27/09/2026 eles continuavam no ar depois da morte
	// ("apagar isso devolveria ao grupo uma vitoria que ele ainda nao
	// teve"), e foi exatamente isso que matou o grupo depois do dialogo
	// final — ver plans/plan_playtest_27_09.md §1.
	h.resolveThorns(dt)
	h.resolveFog(dt)

	if !alive {
		return
	}

	// O anel de contato (item 5, doc/plan_world07_arena_tuning.md) e uma
	// propriedade do CORPO: ao contrario do espinhao e da nevoa, ele nao
	// continua depois que a chefe cai, entao mora AQUI, depois do `if !alive`,
	// e nao ao lado das duas chamadas acima.
	h.resolveBossContact(dt, boss)

	if h.boss.casting > 0 {
		if h.boss.casting -= dt; h.boss.casting <= 0 {
			bounds := h.arenaBounds()
			skill.ActivateDarkFog(h.Skills, bounds)
			h.broadcastBossFog("cast", bounds)
			// A horda da arena e BossDriven (host_waves.go): ela nao tem
			// relogio proprio, ela espera a nevoa. Pedir aqui, no MESMO
			// quadro em que a nevoa entra no ar, e o que faz o compasso da
			// fase bater junto — updateWaves chama Waves.Update antes de
			// updateBoss, entao o pedido so e atendido no quadro seguinte
			// (16 ms, invisivel).
			if h.Waves != nil {
				h.Waves.ReleaseBatchNow()
			}
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
//
// A foto e do PE, nao do centro do sprite (27/09/2026,
// plans/plan_playtest_27_09.md §2). O espinhao e um decalque de CHAO, e o
// jogador le "estou dentro?" pelo pe. Com a foto no centro do quadro, o pe
// ficava ~103 px abaixo do centro da marca e a borda de baixo do circulo
// passava rente a sola: quem fugia para baixo tirava o pe do circulo em
// poucos pixels, mas o centro seguia dentro por mais cem — e levava o golpe
// achando que tinha saido. playersInsideThorn mede pelo mesmo ponto.
func (h *Host) castThorns(boss *entity.Enemy) {
	positions := h.alivePlayerFeet()
	if len(positions) == 0 {
		return
	}
	boss.TriggerStrike()
	skill.SpawnThorns(h.Skills, positions)
	h.broadcastBossThorn(positions)
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
		if t.Contains(playerFeet(p)) {
			out = append(out, id)
		}
	}
	return out
}

// playerFeet e onde o pe de um jogador pisa no mundo: PlayerState.X/Y e o
// centro do quadro do sprite, e entity.GroundPoint desce ate a sola pela
// FootLine do personagem — a mesma regra que a colisao de movimento e o
// portal ja usam. Caller holds playersMutex (or owns p).
func playerFeet(p *PlayerState) rl.Vector2 {
	pos := rl.NewVector2(float32(p.X), float32(p.Y))
	return entity.GroundPoint(pos, entity.GetCharacter(entity.CharacterType(p.Character)))
}

// alivePlayerFeet e a foto do espinhao: o pe de cada jogador vivo.
func (h *Host) alivePlayerFeet() []rl.Vector2 {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	out := make([]rl.Vector2, 0, len(h.players))
	for _, p := range h.players {
		if !p.IsDead {
			out = append(out, playerFeet(p))
		}
	}
	return out
}

// resolveFog cobra o dano continuo da nevoa.
func (h *Host) resolveFog(dt float32) {
	wasActive := h.Skills.FogActive()
	times, dmg := h.Skills.AdvanceFog(dt)
	if wasActive && !h.Skills.FogActive() {
		// A propria contagem de AdvanceFog zerou o ponteiro (TTL esgotado, e
		// nao um F5/portal — esses ja limpam via ClearBossEffects e nao
		// precisam de aviso, os clientes tambem limpam sozinhos nesses
		// casos). O cliente tem o mesmo relogio e chegaria ao mesmo lugar
		// sozinho, mas um evento explicito evita a nevoa sumir um quadro
		// antes ou depois em cada maquina por causa de jitter de rede.
		h.broadcastBossFog("expire", rl.Rectangle{})
	}
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
		// hpAfter zerado com `had` verdadeiro so pode ser ESTE golpe:
		// AbsorbShieldDamage recusa escudo ja quebrado.
		if hpAfter <= 0 {
			h.burstBrokenShield(playerID)
		}
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

// broadcastBossFog replica um evento de nevoa da chefe aos clientes. Mesmo
// padrao de broadcastSentryOrb/broadcastCannonBall: so o "cast" carrega dado
// (os limites), o "expire" e so o aviso.
func (h *Host) broadcastBossFog(event string, bounds rl.Rectangle) {
	payload := BossFogPayload{
		EventType: event,
		X:         int(bounds.X),
		Y:         int(bounds.Y),
		W:         bounds.Width,
		H:         bounds.Height,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] falha ao serializar evento de nevoa da chefe: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgBossFog, Payload: data})
}

// broadcastBossThorn replica a rajada de espinhos da chefe aos clientes: os
// mesmos pontos que castThorns acabou de marcar no host, numa unica
// mensagem — a foto tem que chegar inteira, ou a rajada nasceria espalhada
// por varios quadros de rede no cliente.
func (h *Host) broadcastBossThorn(centers []rl.Vector2) {
	points := make([]BossThornPoint, len(centers))
	for i, c := range centers {
		points[i] = BossThornPoint{X: int(c.X), Y: int(c.Y)}
	}
	payload := BossThornPayload{Points: points}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] falha ao serializar evento de espinhos da chefe: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgBossThorn, Payload: data})
}

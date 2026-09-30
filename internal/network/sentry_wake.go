package network

// QUANDO as sentinelas de um mapa podem abrir fogo.
//
// `sentries.go` decide QUANTAS gargulas cada fase arma e por qual porta elas
// entram. Isto e a outra metade da pergunta, e ela so apareceu quando o mapa 4
// passou a instalar as suas na CHEGADA: com alcance 1900 e o mapa inteiro
// dentro dele, as duas ilhas comecavam a atirar no quadro em que o portao se
// fechava atras do grupo — do vestibulo, antes de a fase ter mostrado o que
// ela e, e de um lugar que o grupo ainda nao consegue nem ver.
//
// A regra e a mesma dos guardas de territorio: o mapa se divide em degraus
// (`tilemap.Zone.Tier`), e a fase declara a partir de QUAL degrau a torre
// acorda. Antes disso ela esta em campo, e desenhada e pode ser morta — ela
// so nao atira.
//
// UMA VEZ ACORDADA, NAO DORME MAIS. E o mesmo principio de `Enemy.chasing`: o
// degrau e uma pergunta de AQUISICAO. Um grupo que recua para o vestibulo
// depois de ter chegado ao saguao nao apaga o que ja viu — recuar nao pode ser
// uma forma de desligar a fase.
//
// E QUEM ACORDA E O GRUPO, NAO O BATEDOR. A pergunta era "algum jogador vivo
// alcancou o degrau?", e em jogo isso significava um corpo so: um bot avanca
// lutando ate 700 px a frente do centro da festa (bot/tuning.go,
// `followRadius`), entao a boca do saguao era cruzada por ele enquanto o resto
// do grupo ainda subia o corredor — e as duas ilhas abriam fogo "antes de os
// jogadores chegarem", que foi exatamente o relato do Gui (27/08/2026). Um
// unico humano correndo na frente fazia o mesmo.
//
// Agora e a MAIORIA dos vivos. Nao "todos", de proposito: com "todos", um bot
// preso no vestibulo — ou um jogador que ficou para tras — calaria as gargulas
// pela fase inteira, e sem elas a emboscada do saguao nao derruba o grupo
// abaixo dos 30% e o climax do mapa 4 nao fecha (ver climaxRuns/world04Climax
// e doc/combat_rules.md). Maioria e o unico ponto em que nem o batedor sozinho
// abre a porta nem o retardatario sozinho a tranca.

import (
	"log"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// sentriesAwakeState e o veredicto da porta de despertar PUBLICADO para quem
// desenha — a seta de borda de tela das gargulas (game/sentry_markers.go) so
// aparece com ele verdadeiro. O host o publica a cada quadro em
// handleSentryOrbTick; o cliente o recebe no pacote de inimigos
// (EnemyUpdatePayload.SentriesAwake), porque a porta e decidida so no host
// (maioria dos vivos no degrau, com trava) e nao da para derivar do lado de
// la sem arriscar as duas maquinas discordarem.
//
// Atomico porque o cliente escreve da goroutine de rede e o desenho le da
// goroutine do laco.
var sentriesAwakeState atomic.Bool

// SentriesAwake reporta se as gargulas da fase ja podem atirar — o momento
// "As sentinelas despertaram" no mapa 4; desde o primeiro quadro nos mapas
// sem porta (5 e 7). Antes disso elas estao caladas e nao ganham seta.
func SentriesAwake() bool { return sentriesAwakeState.Load() }

func setSentriesAwake(awake bool) { sentriesAwakeState.Store(awake) }

// sentryWakeTier e o degrau de territorio a partir do qual as sentinelas de um
// mapa abrem fogo.
//
// Mapa fora desta tabela tem as gargulas acordadas desde o primeiro quadro,
// que e o certo para os mapas 5 e 7: la elas entram POR HORDA
// (`WaveDef.Sentries`), ou seja a fase ja escolheu o momento delas — armar uma
// segunda porta em cima disso seria uma torre que nasce e nao atira.
//
// So o mapa 4 entra, e o numero e a leitura da fase: o degrau 3 e a boca do
// saguao (ver world04Garrison), que e onde a emboscada arma e onde o anuncio
// "As sentinelas despertaram" acontece. E como a porta agora pede a MAIORIA do
// grupo, ela cai praticamente junto com a porta do climax, que pede o grupo
// inteiro dentro do mesmo retangulo (`castle_climax`) — as torres e a
// emboscada passam a acontecer no mesmo momento da fase, que e o que o anuncio
// sempre prometeu. O vestibulo e o corredor processional
// sao atravessados sem fogo de longe; do saguao em diante a fase inteira
// acontece debaixo dele. Se a TRAVESSIA (vestibulo ate o saguao) ficar facil
// demais, este e o numero a baixar — nao o dano da gargula. Mas dentro do
// climax do salao, com as sentinelas ja acordadas, este degrau nao muda nada:
// se a EMBOSCADA nao derrubar o grupo, o dial e o dano por acerto
// (`entity/enemy_sprite.go`, EnemyTypeCastleSentry).
var sentryWakeTier = map[string]int{
	"assets/maps/world_04.json": 3,
}

// sentriesMayFire reports whether this stage's sentries are allowed to launch
// an orb right now, latching the answer the first time it is yes.
//
// Called once per frame from handleSentryOrbTick, from the simulation
// goroutine — like `sentriesArmed`, `h.sentriesAwake` has no lock of its own
// because nothing else writes it.
func (h *Host) sentriesMayFire() bool {
	tier, gated := sentryWakeTier[h.stageMap]
	if !gated || h.sentriesAwake {
		return true
	}
	if !h.partyReachedTier(tier) {
		return false
	}
	h.sentriesAwake = true
	log.Printf("[Sentinela] %s: a maioria do grupo alcancou o degrau %d; as gargulas abriram fogo",
		h.stageMap, tier)
	return true
}

// partyReachedTier reports whether MOST of the LIVING players stand in a
// territory of at least `tier`.
//
// Vivo, e nao "qualquer corpo": um jogador que morreu la na frente e foi
// deixado para tras nao conta — a fase reage a quem esta avancando. E o
// contrario do checkpoint do mapa 6 (`anyPlayerInZone`), onde a pergunta e "o
// grupo chegou ate aqui?" e um cadaver responde que sim.
//
// Maioria estrita (`> metade`), e o porque esta no cabecalho do arquivo: com
// "qualquer um" o batedor do grupo abria fogo sobre quem ainda estava dois
// degraus atras; com "todos", um retardatario calava a fase inteira.
//
// Um grupo VAZIO nao chegou a lugar nenhum. Sem isto, `0 > 0/2` seria falso
// por acidente e nao por regra — e a regra e a mesma de `partyArrived` no
// portao do climax.
//
// Um mapa que declare um degrau e nao tenha territorio nenhum acorda as
// sentinelas na hora: uma porta que nunca abre e pior que uma porta que nao
// existe, e o log alto fica com quem monta o mapa.
func (h *Host) partyReachedTier(tier int) bool {
	if len(h.stageZones) == 0 {
		return true
	}
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	alive, ahead := 0, 0
	for _, p := range h.players {
		if p.IsDead {
			continue
		}
		alive++
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		for _, z := range h.stageZones {
			if z.Tier >= tier && z.Contains(pos) {
				ahead++
				break
			}
		}
	}
	return alive > 0 && ahead*2 > alive
}

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

import (
	"log"

	rl "github.com/gen2brain/raylib-go/raylib"
)

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
// "As sentinelas despertaram" acontece. O vestibulo e o corredor processional
// sao atravessados sem fogo de longe; do saguao em diante a fase inteira
// acontece debaixo dele. Se a travessia ficar facil demais, este e o numero a
// baixar — nao o dano.
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
	log.Printf("[Sentinela] %s: o grupo alcancou o degrau %d; as gargulas abriram fogo",
		h.stageMap, tier)
	return true
}

// partyReachedTier reports whether any LIVING player stands in a territory of
// at least `tier`.
//
// Vivo, e nao "qualquer corpo": um jogador que morreu la na frente e foi
// deixado para tras nao acorda as torres sozinho — a fase reage a quem esta
// avancando. E o contrario do checkpoint do mapa 6 (`anyPlayerInZone`), onde a
// pergunta e "o grupo chegou ate aqui?" e um cadaver responde que sim.
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
	for _, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		for _, z := range h.stageZones {
			if z.Tier >= tier && z.Contains(pos) {
				return true
			}
		}
	}
	return false
}

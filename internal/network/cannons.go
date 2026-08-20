package network

// Os canhoes do corredor final (mapa 6): postos fixos, QUANTOS e QUAO fortes.
//
// Irmao de sentries.go, mas nao a mesma coisa. A gargula sentinela e um
// `entity.Enemy` — tem sprite, entra no EntityManager, pode ser atacada por
// espada e por espectro (so nao por projetil comum). O canhao NAO: ele e
// decoracao do Tiled (uma estatua ja existente no manifesto do castelo, sem
// arte nova — ver o comentario de `enemyCannonPrefix`) com uma arma de host
// por tras. Ele nao pode ser ferido por combate normal, e so sai de campo
// quando o julgamento roteirizado da Paladina o destroi
// (host_last_stand.go:castCannonJudgment). Um jogador que tentasse "matar" o
// canhao a espada estaria martelando pedra.
//
// A composicao pertence ao MAPA, como toda tabela deste arquivo-irmao: uma
// fase nova declara os postos dela e nenhuma constante global e editada.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// CannonDamage e o dano de um acerto direto. Quase metade da vida cheia
	// de um jogador (100): "causam muito dano" do briefing precisa doer o
	// bastante para que o Escudo Sagrado (absorve 50) e a cura da Sacerdotisa
	// sejam a resposta certa, e nao dodge puro.
	CannonDamage float32 = 45
	// CannonRange e o alcance de cada canhao, em pixels. O corredor do mapa 6
	// tem ~48 celulas (6144 px) do spawn ate a sala dos canhoes; 3200 px
	// (25 celulas) alcanca um pouco alem do meio do corredor, que e
	// exatamente onde o briefing poe "ate proximo da metade" — o grupo so
	// comeca a sangrar quando entra nesse raio, nunca antes.
	CannonRange float32 = 3200
	// CannonCooldown é a cadência de cada canhão. Os dois disparam juntos para
	// a salva formar uma parede de fogo de ponta a ponta do corredor.
	CannonCooldown float32 = 2.2
)

// cannonPost e um canhao em campo: posicao fixa e o estado que muda.
type cannonPost struct {
	ID          string
	Name        string
	Position    rl.Vector2
	Destroyed   bool
	attackTimer float32
}

// InstallArrivalCannons poe em campo os canhoes que o mapa ja tem quando o
// grupo chega. Mapa sem `enemy_cannon_*` fica sem nenhum — o comportamento
// certo para qualquer mapa que nao seja o sexto.
//
// Chamado no carregamento do mapa, ao lado de InstallArrivalSentries: os
// postos ficam guardados no host porque o reinicio de fase precisa repo-los.
func (h *Host) InstallArrivalCannons(mapPath string, posts []tilemap.SpawnPoint) {
	h.stageCannonPosts = posts
	h.liveCannons = make([]*cannonPost, 0, len(posts))
	for _, p := range posts {
		h.liveCannons = append(h.liveCannons, &cannonPost{
			ID:       "cannon_" + p.Name,
			Name:     p.Name,
			Position: p.Position,
			// Ambos começam prontos, para lançar uma bola por canhão na mesma
			// salva e fechar a faixa de passagem.
			attackTimer: 0,
		})
	}
	if len(h.liveCannons) > 0 {
		log.Printf("[Canhao] %s: %d canhao(oes) armado(s)", mapPath, len(h.liveCannons))
	}
}

// RestoreCannons repoe os canhoes depois de um reinicio de fase. Irma de
// RestoreSentries: ResetStage esvazia o campo, e um canhao destruido pelo
// julgamento da Paladina numa tentativa anterior nao pode continuar
// destruido numa tentativa nova.
func (h *Host) RestoreCannons() {
	h.InstallArrivalCannons(h.stageMap, h.stageCannonPosts)
}

// LiveCannons devolve os canhoes ainda nao destruidos.
func (h *Host) LiveCannons() []*cannonPost {
	out := make([]*cannonPost, 0, len(h.liveCannons))
	for _, c := range h.liveCannons {
		if !c.Destroyed {
			out = append(out, c)
		}
	}
	return out
}

// DestroyCannons desarma todos os canhoes do mapa. Chamado pelo julgamento do
// ultimo suspiro (host_last_stand.go) — a unica forma de silencia-los.
// Devolve as posicoes dos que ainda estavam de pe, para a explosao visual.
func (h *Host) DestroyCannons() []rl.Vector2 {
	var hit []rl.Vector2
	for _, c := range h.liveCannons {
		if c.Destroyed {
			continue
		}
		c.Destroyed = true
		hit = append(hit, c.Position)
	}
	return hit
}

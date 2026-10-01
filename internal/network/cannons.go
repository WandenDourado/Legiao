package network

// Os canhoes do corredor final (mapa 6): postos fixos, QUANTOS e QUAO fortes.
//
// Irmao de sentries.go, mas nao a mesma coisa. A gargula sentinela e um
// `entity.Enemy` — tem sprite, entra no EntityManager, e a IA do jogo a
// persegue como qualquer bicho. O canhao NAO: ele e decoracao do Tiled
// (`assets/castle_cannons_manifest.json`) com uma arma de host por tras, e nao
// tem corpo no EntityManager. Nao anda, nao caca ninguem e nao entra na conta
// de "inimigos em campo".
//
// COMO ELE CAI (mudanca de 27/08/2026, a pedido do Gui).
//
// Ate aqui os dois canhoes eram destruidos por ROTEIRO, no fim da cena do
// ultimo suspiro. Isso resolvia o corredor por conta propria: o grupo caia, a
// cena rodava, os canhoes explodiam sozinhos e ninguem precisava fazer nada.
//
// Agora o corredor e um problema de jogo. O canhao tem vida (CannonMaxHP) e so
// o ARCO DE ESPADA a machuca — projetil de flecha ou de bola de fogo passa
// sem efeito. Isso e o desenho, e nao uma limitacao: com alcance de 3200 px de
// um lado e 140 px do outro, deixar o Arqueiro resolver o corredor da entrada
// apagaria a travessia inteira. Alguem precisa CHEGAR ate o canhao, e chegar
// custa 45 de dano a cada 2,2 s por canhao — o que so o Avatar dos Deuses da
// Paladina (imunidade total, 15 s) paga.
//
// A composicao pertence ao MAPA, como toda tabela deste arquivo-irmao: uma
// fase nova declara os postos dela e nenhuma constante global e editada.

import (
	"encoding/json"
	"log"

	"github.com/WandenDourado/Legiao/internal/skill"
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
	// tem ~48 celulas (6144 px) do spawn ate a sala dos canhoes.
	//
	// Eram 3200 px (25 celulas), "um pouco alem do meio". Em jogo o corredor
	// lia como calmo demais na primeira metade: o grupo andava um bom tempo
	// sem nada acontecer e a fase so comecava quando ja estava perto (Gui,
	// 27/08/2026). 4000 px (31 celulas) faz o bombardeio comecar a dois tercos
	// do caminho ate os postos, e nao na metade — antes o bastante para a
	// travessia ser a fase inteira, longe o bastante para o spawn continuar
	// sendo um lugar seguro de onde olhar o corredor.
	CannonRange float32 = 4000
	// CannonCooldown é a cadência de cada canhão. Os dois disparam juntos para
	// a salva formar uma parede de fogo de ponta a ponta do corredor.
	CannonCooldown float32 = 2.2
	// CannonMaxHP e quanto o canhao aguenta de espada.
	//
	// O numero NAO e escolhido, ele e derivado: DOIS golpes supremos por
	// canhao. Com skill.SwordAvatarDamage em 300, isso da 600 — quatro golpes
	// pelos dois postos, pouco mais de 2,5 s na cadencia dela (1,5 golpe/s),
	// deixando a maior parte dos 15 s da suprema para a caminhada, que e a
	// parte dificil. Sem a suprema sao cinco golpes por canhao sob
	// bombardeio: o mesmo que impossivel, e e para ser.
	//
	// Era 150, quando a espada dava 30/75. Quando o dano base subiu para 120
	// (orc em cinco golpes, pedido do Gui em 27/08/2026) este numero teve de
	// subir junto ou a Paladina derrubaria os dois canhoes com quatro golpes
	// NORMAIS e o Avatar deixaria de ser o pedagio da travessia.
	// TestCannonFallsToTwoEmpoweredSwings guarda os dois lados disso.
	CannonMaxHP float32 = 600
	// A caixa do posto, tirada da arte (assets/castle_cannons_manifest.json):
	// `cannon_west` mede 219x285 px ancorada na linha do chao, e o jogo ja usa
	// a meia-largura do plinto (92) para a barra de vida. E contra ELA que o
	// arco decide, e nao contra um ponto flutuando no meio dela — de pe ao
	// lado do plinto o ponto antigo (um unico ponto 70px acima do chao com
	// 80 de folga) saia do cone de 120° e o golpe nao contava, por mais
	// perto que o jogador estivesse da peca.
	//
	// cannonHitHalfWidth casa com CannonBarHalfWidth de proposito: e a mesma
	// medida da mesma peca.
	cannonHitHalfWidth float32 = CannonBarHalfWidth
	// cannonHitHeight e a altura da peca, ancorada no chao (Position.Y e a
	// base, entao a caixa sobe DAQUI para cima).
	cannonHitHeight float32 = 285
	// cannonHitSlack e a folga de conforto por cima da caixa. Existe porque o
	// arco e resolvido num instante so e o jogador esta sob bombardeio; sem
	// ela, um jogador colado na peca mas levemente fora do retangulo medido
	// ainda erraria por um triz.
	cannonHitSlack float32 = 40
	// CannonBarAbove e a altura da barra de vida acima da linha do chao do
	// posto. A peca `cannon_west` mede 285 px de altura ancorada na base
	// (assets/castle_cannons_manifest.json), entao 300 poe a barra logo acima
	// dos chifres em vez de dentro da cabeca.
	CannonBarAbove float32 = 300
	// CannonBarHalfWidth casa com o plinto (184 px de largura na arte), e nao
	// com o alcance da espada: a barra e a leitura do OBJETO.
	CannonBarHalfWidth float32 = 92
)

// cannonPost e um canhao em campo: posicao fixa e o estado que muda.
type cannonPost struct {
	ID          string
	Name        string
	Position    rl.Vector2
	HP          float32
	Destroyed   bool
	attackTimer float32
}

// hitBox e a caixa que a espada precisa alcancar: a peca inteira, ancorada na
// linha do chao onde c.Position ja fica — nao um ponto flutuando no meio
// dela.
func (c *cannonPost) hitBox() rl.Rectangle {
	return rl.NewRectangle(c.Position.X-cannonHitHalfWidth, c.Position.Y-cannonHitHeight,
		cannonHitHalfWidth*2, cannonHitHeight)
}

// hitAnchor e um ponto estavel perto do meio da caixa, usado so pela
// navegacao do bot (para onde andar e onde mirar o ataque) — o bot nao
// precisa da geometria exata do ponto-mais-proximo que hitBox() da ao golpe
// real; so precisa de um lugar razoavel para ir e mirar.
func (c *cannonPost) hitAnchor() rl.Vector2 {
	return rl.NewVector2(c.Position.X, c.Position.Y-cannonHitHeight/2)
}

// InstallArrivalCannons poe em campo os canhoes que o mapa ja tem quando o
// grupo chega. Mapa sem `enemy_cannon_*` fica sem nenhum — o comportamento
// certo para qualquer mapa que nao seja o sexto.
//
// Chamado no carregamento do mapa, ao lado de InstallArrivalSentries: os
// postos ficam guardados no host porque o reinicio de fase precisa repo-los.
func (h *Host) InstallArrivalCannons(mapPath string, posts []tilemap.SpawnPoint) {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	h.stageCannonPosts = posts
	h.liveCannons = make([]*cannonPost, 0, len(posts))
	for _, p := range posts {
		h.liveCannons = append(h.liveCannons, &cannonPost{
			ID:       "cannon_" + p.Name,
			Name:     p.Name,
			Position: p.Position,
			HP:       CannonMaxHP,
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
	// A arte volta inteira junto com a arma: a peca em ruinas nao pode
	// sobreviver ao reinicio, pelo mesmo motivo que o posto destruido nao
	// sobrevive (ver o comentario acima e internal/tilemap/piece_swap.go).
	tilemap.ClearWreckedPieces()
	h.cannonsMutex.Lock()
	posts := h.stageCannonPosts
	h.cannonsMutex.Unlock()
	// E a barra volta cheia junto com a vida. O cliente faz o mesmo em
	// applyStageReset, ao receber o MsgReset que este reinicio dispara.
	ResetCannonBars()
	h.InstallArrivalCannons(h.stageMap, posts)
}

// LiveCannons devolve os canhoes ainda nao destruidos.
func (h *Host) LiveCannons() []*cannonPost {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	out := make([]*cannonPost, 0, len(h.liveCannons))
	for _, c := range h.liveCannons {
		if !c.Destroyed {
			out = append(out, c)
		}
	}
	return out
}

// damageCannonsInSweep cobra um arco de espada nos canhoes que ele alcanca, e
// devolve as posicoes dos que cairam com este golpe.
//
// Chamado de performSwordAttack, no mesmo golpe que resolve os inimigos: o
// jogador nao tem de saber que o canhao vive num sistema separado do bicho, e
// e o arco que decide os dois. Mapa sem canhao (todos menos o 6) percorre uma
// lista vazia e nao paga nada.
func (h *Host) damageCannonsInSweep(sweep *skill.SwordSweep, dmg float32) (hit []CannonPostState, fallen []rl.Vector2) {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	for _, c := range h.liveCannons {
		if c.Destroyed {
			continue
		}
		// O arco testa o ponto da CAIXA mais perto do jogador — a superficie
		// da peca, e nao o centro dela — com cannonHitSlack de folga. Isso
		// acerta as duas perguntas de InArc de uma vez: a distancia passa a
		// ser medida ate a superficie, e o angulo passa a ser medido contra o
		// ponto da peca que esta mais na frente do jogador, que e o ponto que
		// ele esta olhando.
		if !sweep.InArc(skill.NearestPointInRect(c.hitBox(), sweep.Position), cannonHitSlack) {
			continue
		}
		c.HP -= dmg
		if c.HP <= 0 {
			c.HP = 0
			c.Destroyed = true
			fallen = append(fallen, c.Position)
			log.Printf("[Canhao] %s caiu; o corredor ficou mais silencioso", c.Name)
		} else {
			log.Printf("[Canhao] %s levou %.0f de espada (%.0f/%.0f)", c.Name, dmg, c.HP, CannonMaxHP)
		}
		hit = append(hit, CannonPostState{X: int(c.Position.X), Y: int(c.Position.Y), HP: c.HP})
	}
	return hit, fallen
}

// publishCannonState aplica a vida nova na maquina do host e replica aos
// clientes. Uma mensagem so para as duas coisas que mudam quando a espada
// acerta: a barra encolhe, e no zero a arte troca pela versao em ruinas.
//
// applyCannonState e o mesmo caminho nas duas pontas — o host nao pode ter uma
// regra de "quando a peca troca" e o cliente outra.
func (h *Host) publishCannonState(posts []CannonPostState) {
	if len(posts) == 0 {
		return
	}
	applyCannonState(posts)
	data, err := json.Marshal(CannonStatePayload{Posts: posts})
	if err != nil {
		log.Printf("[Host] falha ao serializar estado dos canhoes: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgCannonState, Payload: data})
}

// sendCannonStateTo manda o estado ATUAL de todos os postos a um cliente so.
//
// Chamado quando alguem entra ou reconecta, e por um motivo que deixou de ser
// cosmetico: o cliente monta os postos sozinho do mapa (SetCannonPosts), todos
// de pe, e so recebe MsgCannonState quando alguem BATE num canhao. Quem chega
// depois da queda ficaria com os dois em pe na tela — e, desde que o portal
// passou a depender disso (game.PortalsUnlocked), com a saida da fase fechada
// enquanto o resto do grupo a atravessa.
//
// Irmao de sendDialogueTo, pela mesma razao: um cliente que entra no meio tem
// de chegar no mesmo ponto da fase em que todo mundo esta.
func (h *Host) sendCannonStateTo(c *ClientConn) {
	posts := h.cannonStateSnapshot()
	if len(posts) == 0 {
		return
	}
	h.sendTo(c, Message{Type: MsgCannonState, Payload: MustMarshal(CannonStatePayload{Posts: posts})})
}

// cannonStateSnapshot le a vida de cada posto da verdade do host.
func (h *Host) cannonStateSnapshot() []CannonPostState {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	posts := make([]CannonPostState, 0, len(h.liveCannons))
	for _, c := range h.liveCannons {
		posts = append(posts, CannonPostState{X: int(c.Position.X), Y: int(c.Position.Y), HP: c.HP})
	}
	return posts
}

// applyCannonState leva o estado recebido para a barra e para a arte. Roda no
// host (direto) e no cliente (do MsgCannonState).
func applyCannonState(posts []CannonPostState) {
	for _, p := range posts {
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		SetCannonHealth(pos, p.HP)
		if p.HP <= 0 {
			tilemap.WreckPieceAt(pos.X, pos.Y)
		}
	}
}

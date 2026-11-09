package network

// Estes testes cobrem o defeito do item 5 (C1): o arco de espada testava um
// unico ponto (cannonHitOffsetY/cannonHitRadius, no meio do plinto) em vez da
// caixa da peca inteira (assets/castle_cannons_manifest.json: cannon_west mede
// 219x285 px ancorada na linha do chao). De pe perto da peca mas fora do
// centro dela, o ponto antigo saia da distancia OU do cone de 120 graus, e o
// golpe passava batendo no vazio.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// newTestCannon poe um unico posto de pe, na posicao do primeiro canhao real
// do mapa 6 (896, 768) — nao importa para a geometria testada, mas mantem os
// numeros ligados a arte de verdade.
func newTestCannon() (*Host, *cannonPost) {
	post := &cannonPost{ID: "cannon_west", Name: "cannon_west", Position: rl.NewVector2(896, 768), HP: CannonMaxHP}
	h := &Host{liveCannons: []*cannonPost{post}}
	return h, post
}

// sweepFacing cria um SwordSweep em pos, mirado exatamente na direcao de pos
// para target — util para testes que precisam do golpe centrado no alvo, sem
// depender de qual angulo cru isso vira.
func sweepFacing(pos, target rl.Vector2) *skill.SwordSweep {
	return skill.NewSwordSweep("p1", pos, rl.Vector2Subtract(target, pos))
}

// TestSwordHitsCannonWhenStandingBesideThePlinth e o caso exato do defeito:
// jogador colado na LATERAL da peca (nao no meio dela), perto do topo da
// caixa (285 px de altura — bem acima da linha do chao em que Position fica).
// O ponto antigo (meio do plinto, so 70px acima do chao) ficava distante e
// fora do cone deste angulo; a caixa inteira captura a mesma superficie que o
// jogador esta de fato encostado nela.
func TestSwordHitsCannonWhenStandingBesideThePlinth(t *testing.T) {
	h, post := newTestCannon()
	// A oeste da caixa (que vai de x=804 a x=988), perto do topo dela
	// (y=483 a y=768) — bem fora do "meio do plinto" que o codigo antigo
	// mirava.
	playerPos := rl.NewVector2(798, 500)
	sweep := skill.NewSwordSweep("p1", playerPos, rl.NewVector2(1, 0)) // de frente para o corredor, mirando leste
	hit, _ := h.damageCannonsInSweep(sweep, CannonDamage)
	if len(hit) != 1 {
		t.Fatalf("golpe colado na lateral do plinto nao acertou o canhao; hit=%v", hit)
	}
	if post.HP != CannonMaxHP-CannonDamage {
		t.Errorf("HP do canhao = %.0f, esperado %.0f", post.HP, CannonMaxHP-CannonDamage)
	}
}

// TestSwordHitsCannonDiagonally garante que um golpe mirado na diagonal — o
// jogador chegando pelo canto sudeste da caixa, e nao de frente por um dos
// eixos — acerta quando mira o ponto certo da caixa.
func TestSwordHitsCannonDiagonally(t *testing.T) {
	h, _ := newTestCannon()
	playerPos := rl.NewVector2(1050, 900) // sudeste do canto (988, 768) da caixa
	nearest := rl.NewVector2(988, 768)
	sweep := sweepFacing(playerPos, nearest)
	hit, _ := h.damageCannonsInSweep(sweep, CannonDamage)
	if len(hit) != 1 {
		t.Fatalf("golpe diagonal mirado no canto da caixa nao acertou; hit=%v", hit)
	}
}

// TestSwordMissesCannonWhenFacingAway prova que a caixa nao virou um alvo
// automatico: o mesmo jogador colado na lateral do plinto do primeiro teste,
// mas de COSTAS para o canhao, continua errando — a caixa muda a DISTANCIA
// medida, nao apaga o cone de 120 graus.
func TestSwordMissesCannonWhenFacingAway(t *testing.T) {
	h, post := newTestCannon()
	playerPos := rl.NewVector2(798, 500)
	sweep := skill.NewSwordSweep("p1", playerPos, rl.NewVector2(-1, 0)) // de costas, mirando para longe do canhao
	hit, _ := h.damageCannonsInSweep(sweep, CannonDamage)
	if len(hit) != 0 {
		t.Errorf("golpe de costas para o canhao acertou mesmo assim; hit=%v", hit)
	}
	if post.HP != CannonMaxHP {
		t.Errorf("HP do canhao mudou de costas para o alvo: %.0f", post.HP)
	}
}

// TestSwordMissesCannonTooFarFromTheBox garante que a folga (cannonHitSlack)
// nao virou um alcance disfarcado: parado bem alem da caixa mais a folga, o
// golpe continua errando.
func TestSwordMissesCannonTooFarFromTheBox(t *testing.T) {
	h, _ := newTestCannon()
	// A caixa vai de x=804 a x=988. cannonHitSlack e 40 e SwordRange e 140:
	// 988 + 140 + 40 = 1168 e o limite. 1400 esta bem alem disso.
	playerPos := rl.NewVector2(1400, 768)
	sweep := skill.NewSwordSweep("p1", playerPos, rl.NewVector2(-1, 0)) // mirando o canhao
	hit, _ := h.damageCannonsInSweep(sweep, CannonDamage)
	if len(hit) != 0 {
		t.Errorf("golpe disparado bem alem do alcance acertou o canhao; hit=%v", hit)
	}
}

// TestSwordDoesNotReachTheOtherCannonPost e a garantia de risco do plano: com
// dois postos a 768 px um do outro (as posicoes reais do world_06), o Avatar
// dos Deuses (o maior alcance do jogo) nao pode acertar os dois com um golpe
// so — a caixa so torna o alvo mais facil de alcancar NA SUPERFICIE, nao
// aumenta o alcance da espada.
func TestSwordDoesNotReachTheOtherCannonPost(t *testing.T) {
	west := &cannonPost{ID: "cannon_west", Name: "cannon_west", Position: rl.NewVector2(896, 768), HP: CannonMaxHP}
	east := &cannonPost{ID: "cannon_east", Name: "cannon_east", Position: rl.NewVector2(1664, 768), HP: CannonMaxHP}
	h := &Host{liveCannons: []*cannonPost{west, east}}

	// Colado no canhao oeste, com a suprema (alcance dobrado), mirando reto
	// para leste — direcao do outro posto.
	playerPos := rl.NewVector2(804-10, 768)
	sweep := skill.NewSwordSweep("p1", playerPos, rl.NewVector2(1, 0))
	sweep.Empowered = true
	hit, _ := h.damageCannonsInSweep(sweep, skill.SwordAvatarDamage)
	if len(hit) != 1 {
		t.Fatalf("esperado acertar exatamente o canhao oeste, acertou %d", len(hit))
	}
	if hit[0].X != int(west.Position.X) {
		t.Errorf("o golpe acertou o posto errado: x=%d, esperado %d", hit[0].X, int(west.Position.X))
	}
}


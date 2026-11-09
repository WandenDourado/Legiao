package network

// Os consertos do playtest de 27/09/2026 (plans/plan_playtest_27_09.md):
// dano invisivel da Imperatriz, Game Over depois da vitoria e o folego do
// renascer. Cada teste trava uma das regras que o playtest mostrou faltando.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// newPlaytestHost monta um Host minimo com um jogador vivo em (x, y).
func newPlaytestHost(t *testing.T, x, y int, health float32) (*Host, *PlayerState) {
	t.Helper()
	clearInvulnerability()
	clearRespawnGrace()
	t.Cleanup(func() {
		clearInvulnerability()
		clearRespawnGrace()
		SetBossState(BossState{})
		GameOver = false
	})
	p := &PlayerState{PlayerID: "p1", X: x, Y: y, Character: string(entity.CharMago),
		Health: health, MaxHealth: 200}
	h := &Host{
		players:       map[string]*PlayerState{"p1": p},
		Skills:        skill.NewManager(),
		EntityManager: entity.NewEntityManager(),
	}
	return h, p
}

// §2: a chefe nao tem golpe corpo a corpo contra jogador. O caminho generico
// de ataque cobrava 22 de quem estivesse a ate 1530 px dela, sem nada na tela.
func TestBossDoesNotMeleePlayers(t *testing.T) {
	h, p := newPlaytestHost(t, 0, 400, 200)
	boss := entity.NewEnemy(entity.EnemyTypeDarkLady, 0, 0)
	h.EntityManager.AddEnemy(boss)

	h.checkEnemyPlayerCollisions(map[string]bool{boss.ID: true})

	if p.Health != 200 {
		t.Errorf("a chefe tirou %.0f de vida por proximidade; ela so fere por espinhao e nevoa",
			200-p.Health)
	}
}

// §2: o anel de contato so fere espectro.
func TestBossContactSparesPlayers(t *testing.T) {
	boss := entity.NewEnemy(entity.EnemyTypeDarkLady, 0, 0)
	c := boss.HitCenter()
	h, p := newPlaytestHost(t, int(c.X), int(c.Y), 200)
	h.EntityManager.AddEnemy(boss)

	for i := 0; i < 10; i++ {
		h.resolveBossContact(skill.BossContactTickEvery, boss)
	}

	if p.Health != 200 {
		t.Errorf("jogador colado no corpo perdeu %.0f; o contato so vale para espectro",
			200-p.Health)
	}
}

// §2: o espinhao fotografa e mede pelo PE. Um jogador cujo pe esta dentro do
// circulo e atingido; um cujo pe saiu por baixo, nao — mesmo com o centro do
// sprite ainda dentro do raio.
func TestThornMeasuresTheFeet(t *testing.T) {
	h, p := newPlaytestHost(t, 1000, 1000, 200)
	feet := h.alivePlayerFeet()
	if len(feet) != 1 {
		t.Fatalf("esperava 1 pe na foto, veio %d", len(feet))
	}
	ground := entity.GroundOffset(entity.GetCharacter(entity.CharMago))
	if feet[0].Y != float32(p.Y)+ground {
		t.Fatalf("a foto foi em y=%.0f; o pe esta em y=%.0f", feet[0].Y, float32(p.Y)+ground)
	}
	thorn := &skill.Thorn{Center: feet[0]}

	if got := h.playersInsideThorn(thorn); len(got) != 1 {
		t.Fatal("parado na propria foto, o jogador tinha de estar dentro")
	}
	// Desce ate o pe sair do circulo por 10 px. O centro do sprite ainda
	// esta a menos de ThornRadius do centro da marca.
	p.Y += int(skill.ThornRadius) + 10
	if center := rl.NewVector2(float32(p.X), float32(p.Y)); rl.Vector2Distance(center, thorn.Center) > skill.ThornRadius {
		t.Skip("o caso so e interessante com o centro ainda dentro do raio")
	}
	if got := h.playersInsideThorn(thorn); len(got) != 0 {
		t.Error("com o pe fora do circulo o jogador foi atingido pelo espinho")
	}
}

// §1: a queda da chefe apaga nevoa e espinhoes e, dali em diante, nada fere
// ninguem e a corrida nao pode mais ser perdida.
func TestVictoryClearsBossThreats(t *testing.T) {
	h, p := newPlaytestHost(t, 500, 500, 10)
	h.bossPresent = true
	h.bossType = entity.EnemyTypeDarkLady
	SetBossState(BossState{Present: true, Health: 1, MaxHealth: 45000})
	skill.ActivateDarkFog(h.Skills, rl.NewRectangle(0, 0, 5000, 5000))
	skill.SpawnThorns(h.Skills, []rl.Vector2{rl.NewVector2(500, 600)})

	// Nenhuma chefe viva no EntityManager: este e o quadro da queda.
	h.updateBossState()

	if h.Skills.FogActive() {
		t.Error("a nevoa sobreviveu a queda da chefe")
	}
	if len(h.Skills.Thorns) != 0 {
		t.Errorf("%d espinho(s) sobreviveram a queda da chefe", len(h.Skills.Thorns))
	}
	// O dialogo final congela tudo; quando ele fecha, a simulacao volta.
	h.resolveFog(skill.DarkFogDuration)
	h.resolveThorns(5)
	h.applyBossDamage("p1", 999, false)
	if p.Health != 10 || p.IsDead {
		t.Errorf("depois da queda o jogador ainda tomou dano (vida %.0f, morto %v)", p.Health, p.IsDead)
	}
	if !h.victoryPeace() {
		t.Error("victoryPeace falso com a chefe caida")
	}
}

// §4: renascer da RespawnGrace segundos de imunidade, e ela aparece no
// snapshot (PlayerState.Invuln) para o corpo piscar em toda maquina.
func TestRespawnGrantsGrace(t *testing.T) {
	h, p := newPlaytestHost(t, 0, 0, 0)
	h.markPlayerDead(p)
	p.RespawnIn = 0.01

	h.tickRespawns(0.02)
	h.publishInvulnerability()

	if p.IsDead {
		t.Fatal("o jogador nao ressuscitou")
	}
	if !h.IsInvulnerable("p1") {
		t.Error("recem-levantado sem imunidade")
	}
	if p.Invuln <= 0 {
		t.Error("PlayerState.Invuln nao subiu com a janela")
	}
	if respawnGraceMultiplier("p1") != entity.RespawnGraceSpeed {
		t.Error("o bot recem-levantado nao ganhou o passo extra")
	}

	h.tickInvulnerability(entity.RespawnGrace + 0.1)
	tickRespawnGrace(entity.RespawnGrace + 0.1)
	h.publishInvulnerability()
	if h.IsInvulnerable("p1") || p.Invuln != 0 || respawnGraceMultiplier("p1") != 1 {
		t.Error("a janela do renascer nao expirou")
	}
}

// §4: a gargula mira quem tem menos vida — que, logo depois de renascer, e o
// recem-levantado. Imune nao e alvo.
func TestSentrySkipsInvulnerable(t *testing.T) {
	h, weak := newPlaytestHost(t, 100, 0, 60)
	h.players["p2"] = &PlayerState{PlayerID: "p2", X: 200, Y: 0, Health: 150, MaxHealth: 200}
	sentry := entity.NewEnemy(entity.EnemyTypeCastleSentry, 0, 0)

	if id, _, ok := h.weakestPlayerInRange(sentry); !ok || id != weak.PlayerID {
		t.Fatalf("sem imunidade a gargula devia mirar no mais fraco, mirou em %q", id)
	}
	h.GrantInvulnerability(weak.PlayerID, entity.RespawnGrace)
	if id, _, ok := h.weakestPlayerInRange(sentry); !ok || id != "p2" {
		t.Errorf("a gargula mirou em %q, que esta imune", id)
	}
}

// §1, lado do cliente: a borda viva -> caida e derivada do BossState.
func TestBossJustFellEdge(t *testing.T) {
	alive := BossState{Present: true, Health: 10}
	down := BossState{Present: true, Health: 0}
	if !bossJustFell(alive, down) {
		t.Error("a queda nao foi percebida")
	}
	if bossJustFell(down, down) {
		t.Error("a queda disparou de novo num snapshot seguinte")
	}
	if bossJustFell(BossState{}, down) {
		t.Error("a queda disparou num primeiro snapshot, sem nada espelhado")
	}
}

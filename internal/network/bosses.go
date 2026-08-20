package network

// QUAL mapa tem chefe, e onde ele nasce.
//
// Mesma forma de `waveRuns`, `garrisons`, `sentryPosts` e `lastStandHeroes`: a
// FASE declara, e o resto do sistema so le a declaracao. Um mapa que nao entra
// nesta tabela nao tem chefe e nao paga nada por isso.
//
// O chefe NAO passa por `spawnEnemyAt` nem entra em `Composition`, pelo mesmo
// motivo da gargula: ele tem `Speed 0` e vive numa ancora que o mapa declarou.
// Diferente da gargula, ele nasce UMA vez, no carregamento — ele nao e horda,
// ele e a fase.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// bossOfMap e o chefe de cada mapa, pelo mesmo caminho que World.Path usa.
var bossOfMap = map[string]entity.EnemyType{
	"assets/maps/world_07.json": entity.EnemyTypeDarkLady,
}

// BossOfMap devolve o chefe declarado por um mapa, e se ele declarou algum.
func BossOfMap(mapPath string) (entity.EnemyType, bool) {
	t, ok := bossOfMap[mapPath]
	return t, ok
}

// InstallBoss poe o chefe da fase em campo, na ancora `boss_anchor` do mapa.
//
// Chamado no carregamento, ao lado de `InstallArrivalSentries`. Guarda a ancora
// no host porque o reinicio de fase precisa repo-lo — sem isso a segunda
// tentativa do mapa 7 seria uma arena sem chefe, e a fase ficaria impossivel de
// terminar em vez de mais facil.
func (h *Host) InstallBoss(mapPath string, anchor rl.Vector2, hasAnchor bool) {
	h.bossType, h.bossPresent = "", false
	h.bossAnchor = anchor

	t, ok := bossOfMap[mapPath]
	if !ok {
		SetBossState(BossState{})
		return
	}
	if !hasAnchor {
		// Falha em silencio de um jeito ruim: a fase carregaria sem chefe e a
		// corrida infinita nunca terminaria. Log alto, como o posto de
		// sentinela sem marcador.
		log.Printf("[Chefe] %s declara %s e o mapa nao tem boss_anchor", mapPath, t)
		SetBossState(BossState{})
		return
	}

	h.bossType, h.bossPresent = t, true
	h.ResetBossClocks()
	h.EntityManager.AddEnemy(entity.NewEnemy(t, anchor.X, anchor.Y))
	def := entity.GetEnemyDef(t)
	log.Printf("[Chefe] %s: %s em (%.0f, %.0f), %.0f de vida",
		mapPath, def.Name, anchor.X, anchor.Y, def.Health)
	SetBossState(BossState{Present: true, Name: def.Name,
		Health: def.Health, MaxHealth: def.Health})
}

// RestoreBoss repoe o chefe depois de um reinicio de fase. Irma de
// `RestoreSentries`, e pelo mesmo motivo: `ResetStage` esvazia o EntityManager.
func (h *Host) RestoreBoss(mapPath string) {
	// Le a TABELA e nao `h.bossPresent`, porque InstallBoss zera essa flag na
	// primeira linha: consultá-la aqui daria falso na segunda chamada e a arena
	// reiniciaria sem chefe.
	if _, ok := bossOfMap[mapPath]; !ok {
		return
	}
	h.InstallBoss(mapPath, h.bossAnchor, true)
}

// updateBossState publica a vida do chefe. Uma vez por quadro.
//
// Procura pelo TIPO e nao guarda o ponteiro: o EntityManager e quem manda no
// ciclo de vida da criatura, e um ponteiro guardado aqui sobreviveria a morte
// dela — a barra ficaria congelada em 1 de vida para sempre.
func (h *Host) updateBossState() {
	if !h.bossPresent {
		return
	}
	for _, e := range h.EntityManager.GetAllEnemies() {
		if e == nil || !e.IsActive || e.Type != h.bossType {
			continue
		}
		SetBossState(BossState{Present: true, Name: entity.GetEnemyDef(e.Type).Name,
			Health: e.Health, MaxHealth: e.MaxHealth,
			Casting: h.boss.casting > 0, CastLeft: h.boss.casting})
		return
	}
	// Nenhum vivo: o chefe caiu. `Present` continua verdadeiro por um motivo -
	// a barra tem de ficar na tela, vazia, ate a fase virar. Uma barra que some
	// no quadro da morte tira do jogador a confirmacao do que ele acabou de
	// fazer.
	b := GetBossState()
	b.Health = 0
	SetBossState(b)
}

// BossDown reporta se a fase tem chefe e ele ja caiu. E o que a corrida
// infinita consulta para parar de repor.
func BossDown() bool {
	b := GetBossState()
	return b.Present && b.Health <= 0
}

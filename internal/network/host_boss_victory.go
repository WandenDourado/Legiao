package network

// O que acontece no exato quadro em que a chefe cai: a horda que sobrou
// desaparece e quem morreu na luta se ergue. As duas coisas rodam juntas, uma
// vez por queda (bossDeathHandled, em updateBossState), antes de qualquer
// fala da cena final.
//
// A HORDA ACABA COM A CHEFE, e nao so para de repor.
//
// A primeira versao so parava a REPOSICAO (`EndsWithBoss`, host_waves.go) e
// deixava o que estava em campo para o grupo terminar — a ideia era nao
// cortar a limpeza final, o momento em que a vitoria assenta. Na pratica isso
// prendeu a fase: reportado depois de um playtest em que a chefe caiu com uma
// leva inteira (28) ainda viva e circulando a arena, e nada nunca terminava de
// morrer sozinho — a corrida ficava presa em "restam 28" para sempre, e com
// ela o dialogo final, os fogos e o "Missao Concluida", que dependem da horda
// chegar a `WavePhaseCleared`. `endHordeOnVictory` fecha a lacuna: mata todo
// mundo que nao e a chefe (a horda da composicao E as gargulas dos portoes —
// a guarnicao nao tem mais o que defender) no instante da queda, e a corrida
// chega a `Cleared` sozinha no quadro seguinte.
//
// O reerguimento e irmao do ultimo suspiro (host_last_stand.go) mas mais
// simples: aqui nao ha resgate roteirizado nem heroi, a ameaca ja acabou, e
// um corpo no chao so atrapalharia quem esta prestes a assistir a cena de
// encerramento. So quem estava MORTO se ergue — um sobrevivente ferido
// continua ferido, a vitoria apaga a derrota de quem caiu, ela nao cura a
// fase inteira.
//
// A VITORIA TAMBEM APAGA O QUE A CHEFE DEIXOU NO AR (27/09/2026,
// plans/plan_playtest_27_09.md §1). Ate entao nevoa e espinhoes ja lancados
// continuavam resolvendo depois da queda — de proposito, para a queda nao
// apagar uma troca de golpes. No playtest daquele dia a chefe caiu com a
// nevoa no ar: o dialogo final congelou a simulacao (e o TTL da nevoa junto),
// e quando ele fechou a nevoa voltou a morder o mapa inteiro — 30/s por ate
// 8 s contra personagens de 200 de vida. O grupo inteiro morreu e o Game Over
// caiu em cima do "Missao Concluida". Agora a queda e o fim da luta:
// clearThreatsOnVictory limpa nevoa, espinhoes e esferas de gargula no mesmo
// quadro, e victoryPeace (IsInvulnerable + a guarda do Game Over em host.go)
// garante que nada que ainda esteja em campo machuque alguem depois disso.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// BossVictoryInvulnerability e so VISUAL desde 27/09/2026: a paz de
	// vitoria (victoryPeace) ja protege todo mundo ate o fim da fase. A janela
	// continua sendo concedida porque e ela que faz quem se ergueu PISCAR
	// (PlayerState.Invuln, host_invulnerability.go) — o mesmo sinal de "voltei
	// e estou protegido" do renascer comum.
	BossVictoryInvulnerability float32 = 2.0
)

// victoryPeace diz se a chefe da fase ja caiu. Dai em diante nenhum caminho
// de dano toca jogador (IsInvulnerable responde verdadeiro para todos) e a
// corrida nao pode mais ser perdida (a guarda de checkGameOver em host.go).
// Defesa em profundidade atras de clearThreatsOnVictory: se algum golpe novo
// surgir depois deste conserto sem passar por aqui, ele ainda assim nao mata
// ninguem na cena de encerramento.
func (h *Host) victoryPeace() bool {
	return h.bossPresent && h.bossDeathHandled
}

// clearThreatsOnVictory apaga, no quadro da queda, o que a chefe e as
// gargulas deixaram no ar: nevoa, espinhoes em telegrafo e esferas em voo.
// Chamado uma unica vez por queda, junto de endHordeOnVictory. O cliente faz
// o mesmo sozinho ao ver a vida da chefe zerar (client_boss_victory.go); o
// "expire" da nevoa vai mesmo assim, pelo mesmo motivo de resolveFog — para a
// nevoa sumir no mesmo quadro em toda maquina.
func (h *Host) clearThreatsOnVictory() {
	fog := h.Skills.FogActive()
	h.Skills.ClearBossEffects()
	h.boss.casting = 0
	skill.ResetSentryOrbs()
	if fog {
		h.broadcastBossFog("expire", rl.Rectangle{})
	}
}

// reviveFallenForVictory ergue, com vida cheia, todo jogador que estava morto
// quando a chefe caiu. Chamado uma unica vez por queda, por bossDeathHandled
// em updateBossState.
func (h *Host) reviveFallenForVictory() {
	var revived []PlayerState
	h.playersMutex.Lock()
	for _, p := range h.players {
		if !p.IsDead {
			continue
		}
		p.IsDead = false
		p.RespawnIn = 0
		p.Health = p.MaxHealth
		revived = append(revived, *p)
	}
	h.playersMutex.Unlock()

	for _, p := range revived {
		h.GrantInvulnerability(p.PlayerID, BossVictoryInvulnerability)
		h.broadcastRespawn(p)
	}
	if len(revived) > 0 {
		log.Printf("[Chefe] %d jogador(es) caido(s) reerguido(s) com a queda da chefe",
			len(revived))
	}
}

// endHordeOnVictory apaga tudo que restou em campo quando a chefe cai —
// a horda da composicao e as gargulas dos portoes, tudo que nao e a propria
// chefe. Chamado uma unica vez por queda, junto de reviveFallenForVictory.
//
// So `IsActive = false`: e o MESMO jeito que `Enemy.TakeDamage` mata um
// inimigo normal (enemy.go), entao o resto do sistema nem sabe que esta e uma
// morte fora do combate — some da proxima leitura de `GetAllEnemies`, some da
// contagem de `aliveCount` (host_spawn.go) e some do proximo pacote de
// posicoes para os clientes, sem exigir um evento de rede novo.
//
// Zera `Waves.pending` tambem: sem isso uma leva que ainda esperava a proxima
// nevoa (que nunca mais vai vir, porque a chefe morta para de conjurar)
// ficaria contando como "restam N" para sempre, e a corrida nunca chegaria a
// `WavePhaseCleared`.
func (h *Host) endHordeOnVictory() {
	killed := 0
	for _, e := range h.EntityManager.GetAllEnemies() {
		if e == nil || !e.IsActive || e.Type == h.bossType {
			continue
		}
		e.IsActive = false
		killed++
	}
	if h.Waves != nil {
		h.Waves.clearPending()
	}
	if killed > 0 {
		log.Printf("[Chefe] %d inimigo(s) da horda removido(s) com a queda da chefe", killed)
	}
}

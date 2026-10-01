package network

// O anel de contato da Senhora das Trevas (item 5, 31/08/2026,
// doc/plan_world07_arena_tuning.md).
//
// Nao e um ataque: nao usa AttackCooldown, nao dispara animacao, nao pede
// alvo. E uma propriedade do CORPO dela, como o fogo de chao e uma
// propriedade do LUGAR — por isso vive num arquivo a parte de host_boss.go,
// que ja tem os tres relogios da fase e nao precisa de mais um assunto.
//
// DESDE 27/09/2026 ELE SO MACHUCA ESPECTRO (plans/plan_playtest_27_09.md §2,
// pedido do Gui depois do playtest). Contra jogador a Imperatriz tem dois
// golpes e so dois — espinhao e nevoa (host_boss.go) —, ambos com aviso na
// tela. O anel continua sendo o que impede a Legiao Espectral de derreter a
// chefe colada no corpo dela (a conta esta em skill.BossContactDPS), e por
// isso so aparece na tela enquanto ha legiao em campo (game/renderer.go).

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// resolveBossContact cobra dano continuo dos espectros da Legiao dentro de
// BossContactRadius do corpo da chefe, agrupado em pedacos de
// BossContactTickEvery segundos, como a nevoa, em vez de duas dezenas de
// cobrancas por segundo por alvo. Jogador nao entra: ver o topo do arquivo.
//
// Chamado por updateBoss so enquanto a chefe esta viva (o `if !alive` de la
// ja cobre isso), com o *entity.Enemy que updateBoss ja buscou — nao ha
// motivo para procurar a chefe de novo no EntityManager.
func (h *Host) resolveBossContact(dt float32, boss *entity.Enemy) {
	h.boss.contactTick += dt
	if h.boss.contactTick < skill.BossContactTickEvery {
		return
	}
	h.boss.contactTick -= skill.BossContactTickEvery

	// HitCenter(), NAO Position: Position e o pe (FootLine da folha), e o
	// corpo visivel fica HitOffsetY (180 px) acima dele. Cobrar do pe fazia o
	// anel nascer baixo demais e sem ligacao nenhuma com a sprite — a mesma
	// correcao que a caixa de acerto de espada ja usa.
	center := boss.HitCenter()
	dmg := skill.BossContactDPS * skill.BossContactTickEvery

	// Espectros pelo MESMO caminho que StepLegions usa para o revide dos
	// inimigos comuns (host_necro.go): DamageSpectersInRadius so marca Dying
	// e devolve o evento, e "specter_die" e o mesmo evento que StepLegions ja
	// manda quando um espectro morre para um inimigo comum — o cliente ja
	// sabe espelhar (KillLegionSpecterNear), entao nao ha mensagem de
	// protocolo nova.
	for _, ev := range skill.DamageSpectersInRadius(h.Skills, center, skill.BossContactRadius, dmg) {
		h.broadcastUltimate("specter_die", ev.OwnerID, ev.SpecterPos, rl.Vector2{})
	}
}

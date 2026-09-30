package network

// O que o Escudo Sagrado passou a fazer ALEM de absorver (Gui, 27/08/2026): a
// cura do lancamento e o estouro que cobra dos inimigos encostados.
//
// As duas mudam a habilidade da mesma forma. Antes o escudo so adiava dano, e
// tanto lanca-lo quanto perde-lo eram eventos neutros: lancar com a vida cheia
// ou no fim dava no mesmo, e quebrar era so uma perda. Agora os dois momentos
// da magia valem alguma coisa.
//
// Arquivo separado de host_shield.go de proposito: aquele cuida do escudo como
// OBJETO (ancora, animacao, remocao na morte) e este das duas consequencias
// que atravessam para fora dele — a vida do jogador e o dano nos monstros.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// HealPlayerFraction devolve a `fraction` da vida MAXIMA de playerID, sem
// passar do teto, e anuncia como qualquer outra cura.
//
// Implementa ability.HostLike: quem chama e a magia (ability/shield.go), que
// nao conhece o pacote network. A conta e a mesma de handleSanctuaryTick, de
// proposito — cura e cura, venha do Santuario ou do Escudo.
func (h *Host) HealPlayerFraction(playerID string, fraction float32) {
	if fraction <= 0 {
		return
	}
	h.playersMutex.Lock()
	p, ok := h.players[playerID]
	if !ok || p.IsDead || p.Health >= p.MaxHealth {
		h.playersMutex.Unlock()
		return
	}
	p.Health += p.MaxHealth * fraction
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
	healed := p.Health
	h.playersMutex.Unlock()

	h.broadcastCombatEvent("heal", playerID, "player", healed, "")
}

// burstBrokenShield cobra o estouro do escudo de playerID: ShieldBreakDamage em
// todo inimigo dentro de ShieldBreakRadius da aura.
//
// Chamado nos QUATRO caminhos de dano a jogador (mordida, esfera da sentinela,
// bola de canhao, chefe), no ponto em que cada um ja descobriu que a absorcao
// zerou o escudo. Continuam sendo quatro copias e nao uma abstracao pela razao
// escrita em applySentryOrbDamage: as quatro tem que concordar, e concordar por
// leitura e mais seguro do que por uma abstracao apressada.
//
// O centro vem do ESCUDO e nao do jogador: a aura ja e sincronizada com o dono
// todo quadro (handleShieldTick), e ler dali dispensa o playersMutex — tres dos
// quatro chamadores nao o seguram neste ponto, e um segura.
//
// Sem visual proprio: a animacao de estilhaco do escudo ja e o efeito, e ela
// toca em todas as maquinas porque o host anuncia o escudo zerado
// (broadcastShieldEvent) antes de chegar aqui.
func (h *Host) burstBrokenShield(playerID string) {
	center, ok := h.Skills.ShieldPosition(playerID)
	if !ok {
		return
	}
	killed := 0
	for _, e := range h.EntityManager.GetAllEnemies() {
		if !e.IsActive {
			continue
		}
		// HitCenter e HitRadius, e nao Position e Radius: o estouro tem de
		// alcancar o CORPO do bicho, a mesma regra que o arco da espada usa.
		if rl.Vector2Distance(center, e.HitCenter()) > skill.ShieldBreakRadius+e.HitRadius() {
			continue
		}
		if e.TakeDamage(skill.ShieldBreakDamage) {
			h.EntityManager.RemoveEnemy(e.ID)
			h.broadcastCombatEvent("death", e.ID, "enemy", 0, playerID)
			killed++
			continue
		}
		// E quem sobreviveu SAI DE CIMA DELA. E a metade do estouro que
		// funciona mesmo sem matar ninguem: 32 de dano nao tira um orc de
		// cima da Paladina, 150 px tiram, e e esse passo que ela precisa para
		// relancar o escudo. Posto fixo (gargula, chefe) nao anda — a regra
		// mora no proprio Knockback.
		e.Knockback(center, skill.ShieldBreakKnockback, h.EntityManager.Solid)
		h.broadcastCombatEvent("damage", e.ID, "enemy", skill.ShieldBreakDamage, playerID)
	}
	if killed > 0 {
		log.Printf("[Escudo] o estouro do escudo de %s derrubou %d inimigo(s)", playerID, killed)
	}
}

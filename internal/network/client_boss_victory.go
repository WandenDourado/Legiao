package network

// A metade do cliente de clearThreatsOnVictory (host_boss_victory.go).
//
// O host apaga nevoa, espinhoes e esferas de gargula no quadro em que a chefe
// cai, e para de cobrar dano. O cliente tem os MESMOS efeitos espelhados em
// ClientSkills e no pool de esferas, e sem esta limpeza continuaria vendo
// espinho cravar e esfera voar numa arena que ja esta em paz. Nao ha
// mensagem nova para isso: a vida da chefe ja chega em todo snapshot de
// inimigos (BossState), entao a borda "viva -> caida" e derivada aqui
// (doc/skill_performance.md, Regra 6 — nao transmita o que da para derivar).

import "github.com/WandenDourado/Legiao/internal/skill"

// applyBossState publica o estado da chefe recebido do host e, na borda em que
// ela cai, limpa as ameacas que o cliente ainda espelha.
func applyBossState(next BossState) {
	prev := GetBossState()
	SetBossState(next)
	if !bossJustFell(prev, next) {
		return
	}
	if ClientSkills != nil {
		ClientSkills.ClearBossEffects()
	}
	skill.ResetSentryOrbs()
}

// bossJustFell e a borda viva -> caida. `prev.Present` evita disparar no
// primeiro snapshot de uma reconexao com a chefe ja morta: ali nao ha nada
// espelhado para limpar, e a limpeza e inofensiva de qualquer forma.
func bossJustFell(prev, next BossState) bool {
	return next.Present && next.Health <= 0 && prev.Present && prev.Health > 0
}

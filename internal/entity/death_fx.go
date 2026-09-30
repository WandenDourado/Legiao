package entity

// Morte por desintegracao: o monstro que morre vira cinzas levadas pelo vento
// ("estalo do Thanos"). Pedido e aprovado pelo Gui em 28/09/2026, a partir do
// preview desintegracao_preview.html. Ver doc/combat_rules.md, secao "Morte
// dos monstros".
//
// SO VISUAL, E SEM REDE. Nenhuma mensagem nova: cada maquina percebe a morte
// sozinha, comparando o que desenhou no quadro anterior com a lista
// autoritativa deste quadro (host: EntityManager; cliente: o snapshot
// completo). Isso cobre todo caminho de morte que existe e os que vierem —
// flecha, bola de fogo, suprema, e a horda inteira que some com a queda da
// chefe sem mandar evento nenhum.
//
// Ciclo por quadro, na goroutine de desenho (renderer):
//
//	SyncDeathFXWorld(mapa, geracao) // troca de mundo nao e morte
//	BeginDeathFXFrame(vivo)         // quem sumiu desde o quadro passado
//	... desenhar inimigos ...       // cada um registra a propria pose
//	UpdateDeathFX(dt); DrawDeathFX(view)
//
// Desligavel em Configuracoes > Video e desempenho (settings.DeathEffect):
// no celular cada grao e uma chamada de desenho a mais.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	deathFXEnabled = true
	deathBursts    []*ashBurst
	// deathNow: quem foi desenhado desde o ultimo BeginDeathFXFrame, e em que
	// pose. No inicio do quadro seguinte ele e "quem estava na tela".
	deathNow   = map[string]DeathPose{}
	deathPath  string
	deathGen   = -1
	deathGrace float64

	// Pontos de troca para os testes: o relogio do raylib precisa de janela,
	// e disparar uma explosao de verdade precisa de textura na GPU.
	deathClock = rl.GetTime
	deathSpawn = spawnDeathBurst
)

// SetDeathFXEnabled liga ou desliga o efeito. Desligar apaga o que estiver
// em curso: quem desliga porque o celular esquentou quer o alivio agora.
func SetDeathFXEnabled(on bool) {
	deathFXEnabled = on
	if !on {
		ResetDeathFX()
	}
}

// DeathFXEnabled reporta a chave atual.
func DeathFXEnabled() bool { return deathFXEnabled }

// ResetDeathFX esquece todas as mortes em curso e o registro de poses.
func ResetDeathFX() {
	deathBursts = deathBursts[:0]
	clear(deathNow)
}

// SyncDeathFXWorld informa o mundo carregado. Quando o mapa ou a geracao da
// fase mudam, tudo some ao mesmo tempo — e nada disso morreu: o efeito zera e
// fica deathFXWorldGrace segundos sem disparar.
func SyncDeathFXWorld(mapPath string, stageGen int) {
	if mapPath == deathPath && stageGen == deathGen {
		return
	}
	deathPath, deathGen = mapPath, stageGen
	ResetDeathFX()
	deathGrace = deathClock() + deathFXWorldGrace
}

// BeginDeathFXFrame dispara a desintegracao de todo inimigo que foi desenhado
// no quadro anterior e que `alive` diz nao existir mais. Chamar uma vez por
// quadro, ANTES de desenhar os inimigos.
//
// So quem estava NA TELA no quadro anterior pode explodir: um monstro que
// morreu fora da camera nao tem quem o veja, e a pose dele seria velha.
func BeginDeathFXFrame(alive func(id string) bool) {
	if !deathFXEnabled {
		return
	}
	startDeathMaskBuild()
	if deathClock() >= deathGrace {
		for id, pose := range deathNow {
			if !alive(id) {
				deathSpawn(pose)
			}
		}
	}
	clear(deathNow)
}

// watchEnemyPose registra a pose de um inimigo desenhado neste quadro.
func watchEnemyPose(id string, pose DeathPose) {
	if deathFXEnabled {
		deathNow[id] = pose
	}
}

// UpdateDeathFX avanca todas as mortes em curso.
func UpdateDeathFX(dt float32) {
	if len(deathBursts) == 0 {
		return
	}
	// Teto de passo: uma aba em segundo plano ou um quadro travado nao pode
	// teletransportar a poeira.
	dt = min(max(dt, 0), 0.05)
	kept := deathBursts[:0]
	for _, b := range deathBursts {
		b.step(dt)
		if !b.done() {
			kept = append(kept, b)
		}
	}
	clear(deathBursts[len(kept):])
	deathBursts = kept
}

// DrawDeathFX desenha as mortes em curso que caem na janela. Chamar dentro do
// BeginMode2D, depois dos inimigos.
func DrawDeathFX(view rl.Rectangle) {
	if len(deathBursts) == 0 {
		return
	}
	for _, b := range deathBursts {
		if b.visible(view) {
			b.drawBody()
		}
	}
	for _, b := range deathBursts {
		if b.visible(view) {
			b.drawDust()
		}
	}
	rl.BeginBlendMode(rl.BlendAdditive)
	for _, b := range deathBursts {
		if b.visible(view) {
			b.drawGlow()
		}
	}
	rl.EndBlendMode()
}

// BossDeathFXPlaying reporta se o corpo de uma chefe ainda esta se desfazendo.
// O dialogo de fim de fase espera por isto: a queda dela e o momento da
// campanha, e a caixa de dialogo abrindo por cima esconderia metade dele.
func BossDeathFXPlaying() bool {
	for _, b := range deathBursts {
		if b.boss && !b.bodyGone() {
			return true
		}
	}
	return false
}

// DeathFXFragments conta os fragmentos vivos (painel F3 e orcamento).
func DeathFXFragments() int {
	n := 0
	for _, b := range deathBursts {
		n += b.live
	}
	return n
}

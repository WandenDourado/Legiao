package entity

// Máquina de animação do chefe fixo.
//
// Ela existe separada de `enemy_anim_state.go` porque a pergunta que aquele
// arquivo responde — "andando ou parado?" — não se aplica: a Senhora das
// Trevas tem `Speed 0` e `enemyAnimFor` devolveria `AnimIdle` para sempre. O
// que decide o estado dela é TEMPO e INTENÇÃO, não velocidade.
//
// O arquivo é só apresentação. Nada aqui aplica dano, move a criatura ou mexe
// na hitbox — ver `bossVisualAdvanceY`, que é o exemplo mais claro da regra.

// bossAnimState é o estado de animação de um chefe fixo.
//
// Vive dentro de Enemy em vez de num mapa global porque a alternativa já foi
// tentada em outro lugar do projeto e envelhece mal: um mapa indexado por
// ponteiro sobrevive à morte do inimigo e vaza.
type bossAnimState struct {
	scanTimer  float32 // segundos até a próxima varredura
	phaseTimer float32 // quanto tempo o laço atual ainda dura
	done       bool    // o one-shot em curso terminou
	pending    EnemyAnim
}

const (
	// Intervalo entre varreduras do idle. Não é aleatório de propósito: o
	// gerador de números aleatórios não é sincronizado entre host e cliente, e
	// duas máquinas veriam a chefe olhar para lados diferentes.
	bossScanEvery = 8.0
	// Quanto tempo ela dança antes de soltar o feitiço.
	bossCastChannel = 2.4
	// A janela de desvio: quanto tempo os braços ficam cruzados tremendo antes
	// do golpe. É o número que o jogador sente como "deu tempo de sair".
	bossDodgeWindow = 1.8
	// Avanço visual do idle_scan, em unidades de mundo.
	bossAdvancePixels = 46.0
)

// IsBoss diz se esta definição usa a máquina de estados de chefe.
//
// Testado pela presença da folha de varredura, e não por um booleano novo na
// EnemyDef: quem tem `idle_scan` precisa de alguém que a dispare, e quem não
// tem continua no caminho antigo sem saber que este arquivo existe.
func (d EnemyDef) IsBoss() bool { return d.HasAnim(AnimIdleScan) }

// TriggerCast pede a conjuração. Ignorado se ela já está no meio de algo.
func (e *Enemy) TriggerCast() { e.requestBossAction(AnimCastLoop) }

// TriggerStrike pede o ataque. Ignorado se ela já está no meio de algo.
func (e *Enemy) TriggerStrike() { e.requestBossAction(AnimAttackWindup) }

func (e *Enemy) requestBossAction(a EnemyAnim) {
	switch e.Anim {
	case "", AnimIdle, AnimIdleScan:
		e.boss.pending = a
	}
}

// updateBossAnimation escolhe a folha e avança o quadro.
//
// A ordem importa: primeiro decide-se QUAL animação toca, depois avança-se o
// contador. Invertido, o contador andaria contra a folha anterior e cairia num
// quadro que a nova não tem — é o mesmo motivo comentado em `updateAnimation`.
func (e *Enemy) updateBossAnimation(def EnemyDef, dt float32) {
	if e.Anim == "" {
		e.Anim = AnimIdle
	}

	next := e.Anim
	switch e.Anim {
	case AnimIdle:
		e.boss.scanTimer -= dt
		switch {
		case e.boss.pending != "":
			next = e.boss.pending
			e.boss.pending = ""
		case e.boss.scanTimer <= 0:
			next = AnimIdleScan
		}
	case AnimIdleScan:
		if e.boss.done {
			next = AnimIdle
		}
	case AnimCastLoop:
		// Um laço não termina sozinho; quem o encerra é o relógio. Sem isto ela
		// dançaria para sempre e o feitiço nunca sairia.
		if e.boss.phaseTimer -= dt; e.boss.phaseTimer <= 0 {
			next = AnimCastRelease
		}
	case AnimAttackWindup:
		if e.boss.phaseTimer -= dt; e.boss.phaseTimer <= 0 {
			next = AnimAttackStrike
		}
	case AnimCastRelease, AnimAttackStrike:
		if e.boss.done {
			next = AnimIdle
		}
	}

	if next != e.Anim {
		e.Anim = next
		e.AnimFrame, e.animTimer, e.boss.done = 0, 0, false
		switch next {
		case AnimIdle:
			e.boss.scanTimer = bossScanEvery
		case AnimCastLoop:
			e.boss.phaseTimer = bossCastChannel
		case AnimAttackWindup:
			e.boss.phaseTimer = bossDodgeWindow
		}
	}

	ad := def.AnimDef(e.Anim)
	if ad.Looping() {
		e.AnimFrame, e.animTimer = advanceEnemyAnim(e.AnimFrame, e.animTimer, dt, ad.FrameTime, ad.Steps())
		return
	}
	e.AnimFrame, e.animTimer, e.boss.done = advanceEnemyAnimOnce(
		e.AnimFrame, e.animTimer, dt, ad.FrameTime, ad.Steps())
}

// bossVisualAdvanceY é o quanto o DESENHO da chefe avança durante a varredura.
//
// Ela dá dois passos para frente, olha, e recua — mas o ciclo de patas foi
// desenhado NO LUGAR, seguindo a regra do `enemy_creation_guide.md` §7 ("o
// deslocamento vem do código"). Este é o deslocamento.
//
// **É visual e só visual: não toca em Position nem na hitbox.** Se o corpo
// físico andasse junto, o jogador erraria golpe mirando onde ela aparece — e
// ela é um chefe fixo, o lugar dela no mapa é uma constante do combate. O preço
// aceito é que, no pico do avanço, o desenho fica ~46 px à frente da caixa.
// Contra um raio de 130 isso é um terço, e some na leitura.
//
// A curva é um seno de meia volta: zero nas pontas, máximo no meio, sem degrau
// na entrada nem na saída da animação.
func (e *Enemy) bossVisualAdvanceY(def EnemyDef) float32 {
	if e.Anim != AnimIdleScan {
		return 0
	}
	ad := def.AnimDef(AnimIdleScan)
	steps := ad.Steps()
	if steps <= 1 {
		return 0
	}
	t := float32(e.AnimFrame) / float32(steps-1) // 0..1
	return bossAdvancePixels * 4 * t * (1 - t)   // parábola: 0 -> 1 -> 0
}

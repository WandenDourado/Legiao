package entity

// Reprodução de animação: ordem de quadros e ciclos que tocam uma vez só.
//
// Este arquivo existe porque duas perguntas que antes não existiam passaram a
// existir com a Senhora das Trevas, e as duas são de REPRODUÇÃO, não de arte:
//
//  1. **A ordem dos quadros pode não ser 0,1,2,...** O `idle_scan` dela gira o
//     torso para um lado e volta; a volta é a ida tocada ao contrário
//     (0,1,2,3,2,1). Gerar os quadros da volta seria pagar duas vezes pelo
//     mesmo desenho.
//  2. **Nem toda animação é laço.** Um golpe tem que terminar e devolver o
//     controle. Antes disso, todo inimigo do jogo tocava um ciclo infinito, e
//     `advanceEnemyAnim` só sabia dar a volta com `%`.
//
// A distinção que organiza tudo aqui é entre **passo** e **coluna**. O passo é
// onde a animação está no tempo (0..Steps-1); a coluna é qual desenho da folha
// isso significa. Com `PlayOrder` vazio os dois coincidem, que é o caso de
// todos os inimigos anteriores — por isso nada deles muda.

// Steps é quantos passos a animação tem no tempo.
//
// Sem PlayOrder, um passo por coluna. É o que mantém slime, lobo, orc e gárgula
// atravessando este código sem alteração nenhuma.
func (d EnemyAnimDef) Steps() int {
	if len(d.PlayOrder) > 0 {
		return len(d.PlayOrder)
	}
	return d.Columns
}

// Column traduz um passo na coluna da folha que deve ser desenhada.
//
// Fora de faixa é normalizado em vez de estourar: um passo vem de um contador
// que outro código incrementa, e um índice inválido tem que virar um desenho
// errado, nunca um panic no meio do laço de render.
func (d EnemyAnimDef) Column(step int) int {
	n := d.Steps()
	if n <= 0 {
		return 0
	}
	step = ((step % n) + n) % n
	if len(d.PlayOrder) > 0 {
		col := d.PlayOrder[step]
		if d.Columns > 0 {
			col = ((col % d.Columns) + d.Columns) % d.Columns
		}
		return col
	}
	return step
}

// Looping diz se a animação recomeça sozinha.
//
// O zero value é `false`, e isso seria a resposta errada para todo inimigo que
// já existia — por isso o campo é `OneShot` e não `Loop`. Quem não declara nada
// continua em laço, como sempre esteve.
func (d EnemyAnimDef) Looping() bool { return !d.OneShot }

// advanceEnemyAnimOnce avança um ciclo que toca UMA vez e trava no último
// quadro, devolvendo também se ele acabou.
//
// Travar no último em vez de voltar ao primeiro é o que faz o golpe funcionar:
// entre o quadro de impacto e a máquina de estados escolher o próximo estado
// passam alguns frames de jogo, e nesse intervalo a criatura tem que ficar na
// pose final, não piscar de volta para o começo.
func advanceEnemyAnimOnce(step int, timer, dt, frameTime float32, steps int) (int, float32, bool) {
	if steps <= 0 {
		return 0, 0, true
	}
	if frameTime <= 0 {
		return steps - 1, 0, true
	}
	timer += dt
	for timer >= frameTime {
		timer -= frameTime
		step++
	}
	if step >= steps-1 {
		return steps - 1, 0, true
	}
	return step, timer, false
}

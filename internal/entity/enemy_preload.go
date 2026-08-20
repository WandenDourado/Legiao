package entity

// Carregar as folhas de inimigo ANTES de precisar delas.
//
// enemyTexture carrega sob demanda, e a demanda acontece dentro do passe de
// desenho: a primeira vez que um orc aparece, o quadro para para decodificar um
// PNG de 1232x1206 e subir 5,7 MB para a GPU. O medidor do F3 registrou
// exatamente isso — 60 fps estaveis com um quadro isolado de 43 ms num mapa
// onde cinco monstros tinham acabado de entrar em cena.
//
// O culling de camera PIOROU esse caso especifico, e vale registrar por
// honestidade: antes todos os inimigos eram desenhados, entao a folha subia no
// primeiro quadro depois do spawn, longe da tela e no meio de outras coisas
// caras. Agora a folha sobe no quadro em que o monstro fica VISIVEL, que e o
// pior momento possivel — o jogador esta olhando, e a camera esta andando.
//
// Precarregar remove os dois casos, e o custo e o que o jogo ja gastava em
// regime: as folhas de todos os inimigos registrados somam ~13,5 MB de VRAM
// (orc idle e walk, slime, lobo), e todas acabam carregadas de qualquer forma.

import "log"

// PreloadEnemyTextures sobe a folha de toda animacao de todo inimigo
// registrado. Chamar uma vez, depois de InitWindow e antes do laco do jogo.
//
// Vale para os DOIS papeis de rede: o registro de inimigos e estatico, entao o
// cliente sabe de que folhas vai precisar tanto quanto o host, e e no cliente
// que a travada e pior — ele nao controla quando o monstro aparece.
func PreloadEnemyTextures() {
	loaded, missing := 0, 0
	for _, def := range enemyRegistry {
		for _, anim := range animationsOf(def) {
			if _, _, ok := enemyTexture(def, anim); ok {
				loaded++
			} else {
				missing++
			}
		}
	}
	log.Printf("[Entity] folhas de inimigo precarregadas: %d ok, %d ausentes", loaded, missing)
}

// animationsOf lista as animacoes que a def declara, ou apenas idle quando ela
// usa os campos planos (slime e lobo, folha unica). AnimDef ja sintetiza a
// resposta para esse caso, entao pedir idle basta e nao carrega nada duas
// vezes.
func animationsOf(def EnemyDef) []EnemyAnim {
	if len(def.Anims) == 0 {
		return []EnemyAnim{AnimIdle}
	}
	anims := make([]EnemyAnim, 0, len(def.Anims))
	for anim := range def.Anims {
		anims = append(anims, anim)
	}
	return anims
}

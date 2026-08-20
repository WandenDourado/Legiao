package network

// A corrida de hordas de cada mapa.
//
// Isto morava numa variavel de pacote chamada waveDefs, com as tres hordas do
// world_01, e QUALQUER mapa carregado herdava aquela corrida — o world_02
// rodava tres hordas com a proporcao de slime e lobo do mapa 1 e anunciava
// "os lobos chegaram" na fase errada. Uma fase nova nao pode depender de
// alguem lembrar de editar uma tabela global.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// waveRuns e a corrida de cada mapa, pelo caminho a partir da raiz do repo —
// a mesma chave que World.Path e campaignMaps usam.
var waveRuns = map[string][]WaveDef{
	"assets/maps/world_01.json": world01Waves,
	"assets/maps/world_02.json": world02Waves,
	"assets/maps/world_05.json": world05Waves,
	"assets/maps/world_07.json": world07Waves,
	// O world_03 NAO entra aqui: ele nao tem corrida ao carregar. A guarnicao
	// dele ja esta em campo (network/garrisons.go) e a unica horda da fase e a
	// EMBOSCADA, que so comeca quando o grupo inteiro alcanca a fortaleza. Ver
	// world03Climax abaixo e game/climax_gate.go.
}

// world03Climax e a emboscada da fortaleza: UMA horda, e ela nao e iniciada
// pelo carregamento do mapa.
//
// Ela existe como WaveDef, e nao como um sistema novo, porque isso devolve de
// graca tudo o que o climax precisa e que a maquina de hordas ja faz: o HUD, o
// controle de quantos ficam vivos ao mesmo tempo, o anuncio na tela e —
// principalmente — `partyIsFalling`, que exige `WaveState.Total > 0` e fase de
// luta. Com a emboscada sendo uma horda de verdade, o gatilho `on_last_stand`
// volta a valer sem uma linha de condicao nova, e a Sacerdotisa se ergue pela
// mesma porta que o Necromante usou no mapa 2.
//
// MaxConcurrent e o que separa emboscada de parede: a reposicao chega em
// levas, e o grupo encurralado contra o portao tem onde respirar entre elas.
// Sem o teto, trinta corpos ao mesmo tempo num retangulo de 44x11 nao e uma
// luta, e um empurrao.
var world03Climax = []WaveDef{
	{
		Name: "A emboscada do portao",
		// A composicao e a MISTURA de reposicao, nao um total: com `Endless`
		// ela volta inteira para a fila cada vez que esvazia. Orc como massa
		// porque e a criatura mais forte que existe, e lobo para o cerco ter
		// velocidade — so orc a 130 de velocidade daria ao grupo espaco para
		// circular indefinidamente, e circular nao pode ser uma saida.
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeGarrison: 8,
			entity.EnemyTypeFast:     8,
		},
		// Nao para ate a Sacerdotisa erguer o altar.
		Endless: true,
		// E nasce EM CIMA DA MURALHA e atras do grupo, nos marcadores
		// `climax_spawn_*`, e nao a mil pixels de distancia. Uma emboscada que
		// o jogador tem de ir procurar nao e uma emboscada.
		Ambush: true,
		// Dezoito simultaneos contra CINCO jogadores. E o numero que faz o
		// grupo perder terreno sem morrer no primeiro minuto: eles aguentam,
		// recuam, e vao caindo — que e exatamente a curva que o ultimo suspiro
		// espera, porque ele dispara com TODOS abaixo de um quarto da vida.
		//
		// Eram doze, calibrados contra quatro jogadores e contra um grupo SEM
		// suprema nenhuma. Aqui o Necromante ja tem a Legiao, e e o `Endless`
		// que resolve isso sem precisar de mais corpos por quadro: os trinta
		// espectros nao ENCERRAM a emboscada, so compram sessenta segundos.
		// Passados eles a pressao volta inteira, e o grupo cai.
		MaxConcurrent: 18,
		// Uma leva a cada 1.0 s, cinco por leva. Eram 2 s e tres, e em jogo o
		// cerco chegava devagar demais para parecer emboscada: o grupo tinha
		// tempo de matar cada leva antes da seguinte, e a pressao — que e o que
		// leva o grupo aos 25% de vida e abre a cena — nunca se acumulava.
		SpawnInterval: 1.0,
		BatchSize:     5,
		Announcement:  "Eles vieram de cima da muralha",
	},
}

// world04Climax: o salao se fecha, e e a cena do ARQUEIRO.
//
// Ela era a horda mais fraca do jogo inteiro — 10 slimes e 8 lobos com teto 8 —
// numa fase em que o grupo ja tem DUAS supremas e cujo clima precisa derrubar
// cinco jogadores para o resgate acontecer. Nao derrubava cinco; nao derrubaria
// um.
//
// `Endless` pelo mesmo contrato do mapa 3: quem a encerra e `LastStandDone()`.
// Sem isso a Area Angelical simplesmente venceria a horda — doze segundos de
// cura a 20/s sobre um numero finito de monstros e uma questao de esperar —, e
// uma cena que o grupo consegue evitar nao e uma cena.
//
// Quem torna a queda inevitavel apesar do altar nao esta nesta composicao: sao
// as quatro gargulas, batendo a 1900 de fora do raio de 720. E e exatamente por
// isso que a suprema entregue aqui e a do Arqueiro — 40 de dano perfurante
// contra os 40 de vida delas.
var world04Climax = []WaveDef{{
	Name: "O salao se fecha",
	// Composicao de REPOSICAO, nao total: com `Endless` ela volta inteira para
	// a fila cada vez que esvazia.
	Composition: map[entity.EnemyType]int{
		entity.EnemyTypeGarrison: 8,
		entity.EnemyTypeFast:     12,
		entity.EnemyTypeBasic:    8,
	},
	Endless:       true,
	MaxConcurrent: 22,
	SpawnInterval: 1.1,
	BatchSize:     5,
	Announcement:  "As sentinelas despertaram",
}}

// world05Waves: o salao da senhora, a ultima fase. Cinco hordas, e o grupo
// chega com AS TRES supremas — Legiao, Area Angelical e Flechas Celestiais.
//
// A COMPOSICAO INVERTE AO LONGO DA CORRIDA, e essa e a forma da fase: comeca em
// slime e termina em orc. A horda 1 e o momento do Necromante, que apaga trinta
// slimes com um lancamento e paga por isso com sessenta segundos de recarga —
// ou seja, com a horda 2 inteira. A horda 5 e quase toda orc porque o orc e a
// unica coisa que a Legiao nao compra (cinco deles a limpam; ver
// entity/orc_legion_test.go).
//
// AS GARGULAS ENTRAM NA HORDA 3, NAO ANTES. Uma torre de alcance 1900 na
// primeira horda apareceria antes de a fase ter dito o que ela e; entrando no
// terceiro degrau, ela e a virada — e a partir dali o salao atira de longe
// enquanto a matilha fecha por perto. `Sentries` e o total acumulado em campo,
// nao um acrescimo, e o posto que ja foi usado nao volta.
// world07Waves: o cerco da arena final. UMA horda, e ela nao acaba.
//
// A forma vem do relogio que o Gui pediu: uma leva a cada 70 s, do tamanho
// exato da composicao. Por isso `BatchSize` e o total (28) e `SpawnInterval` e
// 70 — a leva inteira sai de uma vez pelos dois portoes e a proxima so vem no
// ciclo seguinte. Uma horda que gotejasse continuamente tiraria da fase o
// respiro entre levas, que e onde o grupo se reposiciona para a nevoa.
//
// `Endless` porque o unico fim e a morte da chefe. Sem isso a fase acabaria por
// contagem, e o jogador venceria a arena sem enfrentar quem a comanda.
//
// `Ambush` porque a horda tem de sair dos PORTOES, nos marcadores
// `climax_spawn_gate_*`. Sem ele ela nasceria no anel de 1000-3200 px em volta
// do jogador — ou seja, no meio da sala, do lado de dentro do cerco.
//
// A composicao e par em tudo (10/10/8) porque ela se divide entre os dois
// portoes: 5 slime, 5 lobo, 4 orc de cada lado.
//
// `MaxConcurrent` 34 e o teto: 28 da leva mais folga. Sem teto, duas levas nao
// limpas viram parede, e a fase deixa de ser uma luta contra a chefe para ser
// uma contra o acumulo.
//
// As gargulas NAO entram na composicao — elas tem Speed 0 e vivem em posto
// declarado pelo mapa (ver sentryPosts). `Sentries` e total acumulado.
var world07Waves = []WaveDef{
	{
		Name: "O cerco da senhora",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic:     10,
			entity.EnemyTypeFast:      10,
			entity.EnemyTypeGarrison:  8,
		},
		Endless:       true,
		EndsWithBoss:  true,
		Ambush:        true,
		MaxConcurrent: 34,
		SpawnInterval: 70.0,
		BatchSize:     28,
		Sentries:      2,
		Announcement:  "A Senhora das Trevas desperta",
	},
}

var world05Waves = []WaveDef{
	{
		Name: "Horda 1",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 30,
			entity.EnemyTypeFast:  10,
		},
		MaxConcurrent: 18,
		SpawnInterval: 2.2,
		BatchSize:     4,
		Announcement:  "Horda 1 - Os servos da senhora",
	},
	{
		Name: "Horda 2",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 30,
			entity.EnemyTypeFast:  26,
		},
		MaxConcurrent: 22,
		SpawnInterval: 1.8,
		BatchSize:     5,
		Announcement:  "Horda 2",
	},
	{
		Name: "Horda 3",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic:    20,
			entity.EnemyTypeFast:     32,
			entity.EnemyTypeGarrison: 8,
		},
		MaxConcurrent: 26,
		SpawnInterval: 1.5,
		BatchSize:     5,
		// As duas primeiras gargulas, nos postos mais afastados do centro: o
		// salao abre fogo de longe antes de fechar por perto.
		Sentries:     2,
		Announcement: "Horda 3 - A matilha avanca",
	},
	{
		Name: "Horda 4",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic:    16,
			entity.EnemyTypeFast:     30,
			entity.EnemyTypeGarrison: 16,
		},
		MaxConcurrent: 30,
		SpawnInterval: 1.2,
		BatchSize:     6,
		Sentries:      4,
		Announcement:  "Horda 4 - A guarda de elite",
	},
	{
		Name: "Horda 5",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic:    12,
			entity.EnemyTypeFast:     24,
			entity.EnemyTypeGarrison: 20,
		},
		// A horda se repoe ate o ultimo suspiro. Depois do climax,
		// LastStandDone impede a reposicao; derrotar o que restou conclui a
		// corrida e libera o portao norte da arena para o mapa 6.
		Endless: true,
		//
		// 38 e o numero a MEDIR, como o 42 da matilha do mapa 2: e aqui que a
		// Chuva de Meteoros do Mago cai sobre o campo cheio. Se o quadro cair,
		// baixar isto primeiro.
		MaxConcurrent: 38,
		SpawnInterval: 1.0,
		BatchSize:     6,
		Sentries:      6,
		Announcement:  "Horda 5 - Impossivel vencer",
	},
}

// WaveDefsFor devolve a corrida do mapa, ou nil quando o mapa nao declara uma.
//
// Mapa sem corrida fica quieto e com o portal liberado (WaveState.Total 0),
// que e o que um mapa de validacao de terreno quer. Mas mapa COM marcadores
// enemy_spawn_* e sem corrida quase sempre e esquecimento, nao intencao, entao
// quem chama registra o aviso.
func WaveDefsFor(mapPath string) []WaveDef {
	return waveRuns[mapPath]
}

// world01Waves: a vila. Tres hordas, depois o mapa esta limpo.
//
// A forma da rampa: a horda 1 ensina o slime sozinho; a 2 apresenta o lobo
// como acento mantendo o slime como massa; a 3 inverte a proporcao e sobe a
// concorrencia, para o jogador ser pressionado de varios lados de uma vez.
//
// AS TABELAS SAO ESCRITAS PARA CINCO JOGADORES. Nao ha nenhum lugar no codigo
// onde a quantidade de monstros dependa de quantos estao na sala, e enquanto
// estes numeros foram escritos para um, a fase inteira somava 4.280 de vida —
// contra os ~220 de dano por segundo que cinco personagens fazem juntos, isso e
// VINTE SEGUNDOS de combate real do primeiro anuncio ao mapa limpo.
//
// A rampa nao mudou de forma; mudou de escala. O que sobe primeiro e
// `MaxConcurrent`, porque e ele que decide a DIFICULDADE: a composicao sozinha
// decide quanto tempo a horda dura, e uma horda maior com o mesmo teto e uma
// fase mais longa, nao mais dificil.
//
// Nao entra orc aqui. Ele e a criatura da fortaleza e antecipa-lo ao mapa 1
// gasta a apresentacao dele sem devolver nada — o cerco basta.
var world01Waves = []WaveDef{
	{
		Name:          "Horda 1",
		Composition:   map[entity.EnemyType]int{entity.EnemyTypeBasic: 24},
		MaxConcurrent: 12,
		SpawnInterval: 2.4,
		BatchSize:     4,
		Announcement:  "Horda 1",
	},
	{
		Name: "Horda 2",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 30,
			entity.EnemyTypeFast:  18,
		},
		MaxConcurrent: 18,
		SpawnInterval: 1.9,
		BatchSize:     5,
		Announcement:  "Horda 2 - os lobos chegaram",
	},
	{
		Name: "Horda 3",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 36,
			entity.EnemyTypeFast:  30,
		},
		// Vinte e seis simultaneos sao cinco por jogador se todos engajarem —
		// o que nunca acontece, porque o grupo se espalha e a matilha escolhe o
		// mais perto. E o numero que faz a vila ficar cercada em vez de
		// visitada.
		MaxConcurrent: 26,
		SpawnInterval: 1.5,
		BatchSize:     6,
		Announcement:  "Horda 3 - a ultima",
	},
}

// world02Waves: a mata sombria. TRES hordas, e a ultima e o climax do mapa —
// nenhuma horda vem depois dela, porque uma horda depois do resgate esvaziaria
// a cena.
//
// A matilha e desenhada para NAO ser vencida por um grupo comum: e ela que
// derruba o time e faz o Necromante se levantar. Enquanto a cena nao existe
// (dialogo no meio da horda, intercepcao do Game Over, revive com
// invencibilidade), ela e simplesmente uma horda muito dura — o que se ganha
// depois e o resgate, nao o desafio. Para atravessar o mapa sem isso, F8.
//
// A HORDA 2 E NOVA, E A FUNCAO DELA E DESGASTE.
//
// Com cinco jogadores a matilha ficou facil de sobreviver, e a resposta obvia —
// subir o teto de 35 para 60 — esbarra no limite de desempenho que o proprio
// comentario abaixo documenta. Entao a queda passa a vir por ACUMULO: a horda 2
// existe para o grupo chegar a matilha sem recarga e sem vida cheia. Isso e
// dificuldade que nao custa quadro nenhum, e e a unica forma de o clima da cena
// sobreviver ao grupo de cinco.
var world02Waves = []WaveDef{
	{
		Name: "Horda 1",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 28,
			entity.EnemyTypeFast:  10,
		},
		MaxConcurrent: 16,
		SpawnInterval: 2.2,
		BatchSize:     4,
		Announcement:  "Horda 1",
	},
	{
		Name: "Horda 2",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 20,
			entity.EnemyTypeFast:  28,
		},
		MaxConcurrent: 22,
		SpawnInterval: 1.7,
		BatchSize:     5,
		Announcement:  "Horda 2 - a mata se fecha",
	},
	{
		Name: "Horda 3",
		Composition: map[entity.EnemyType]int{
			entity.EnemyTypeBasic: 12,
			entity.EnemyTypeFast:  84,
		},
		// A MATILHA SE REPOE ATE O CLIMAX, pelo mesmo contrato do mapa 3 e da
		// horda 5 do mapa 5: enquanto `LastStandDone()` for falso, a
		// composicao volta inteira para a fila cada vez que esvazia.
		//
		// Sem isto o grupo de cinco conseguia LIMPAR a matilha antes de cair a
		// 30% de vida, e a corrida terminava com o mapa limpo e a cena do
		// Necromante nunca tocada — a fase entregava o portal sem entregar a
		// suprema. Uma horda finita, por maior que fosse, e sempre uma questao
		// de aguentar; e a reposicao que faz do resgate a unica saida.
		//
		// A composicao acima deixa de ser um total e passa a ser a MISTURA de
		// reposicao. Depois do resgate a reposicao para, o que sobrou em campo
		// e finito, e matar o ultimo conclui a fase e libera o portal.
		Endless: true,
		// 42 simultaneos e o numero a MEDIR, nao um numero fechado: com 30
		// espectros da Legiao em campo ao mesmo tempo, cada um varrendo a lista
		// de inimigos por quadro, este e o ponto onde o quadro pode cair. Se
		// cair, baixar AQUI antes de mexer em qualquer outra coisa — 38,
		// depois 35, que foi o valor medido — e compensar com BatchSize e
		// intervalo. Nunca compensar cortando a composicao: o tamanho da
		// matilha e o que a cena exige.
		MaxConcurrent: 42,
		SpawnInterval: 1.0,
		BatchSize:     6,
		Announcement:  "Horda 3 - a matilha",
	},
}

// logMissingRun avisa quando um mapa tem onde nascer inimigo mas nao tem
// corrida declarada. Silencio aqui vira "o mapa nao spawna nada e ninguem sabe
// por que".
func logMissingRun(mapPath string, markers int) {
	log.Printf("[Waves] %s tem %d marcadores enemy_spawn_* mas nenhuma corrida em "+
		"waveRuns; nada vai nascer e o portal ja nasce liberado", mapPath, markers)
}

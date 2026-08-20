package network

// A guarnicao de cada mapa: QUEM defende cada posto, e quantos.
//
// Isto e para a defesa de territorio o que `waveRuns` e para a horda, e existe
// pelo mesmo motivo: a composicao pertence ao MAPA. Uma fase nova nao pode
// depender de alguem lembrar de editar uma constante global.
//
// O comportamento (perseguir dentro do setor, voltar ao posto sem curar) e do
// sistema de monstros; aqui esta so a lista de quem nasce onde.

import "github.com/WandenDourado/Legiao/internal/entity"

// GarrisonSquad e um posto e o que nasce nele.
type GarrisonSquad struct {
	// Post e o sufixo do marcador `enemy_post_*` na camada `spawn` do mapa.
	Post string
	// Count por tipo. Um posto pode misturar.
	Orcs, Wolves, Slimes int
}

// Total e quantos monstros o posto poe em campo.
func (s GarrisonSquad) Total() int { return s.Orcs + s.Wolves + s.Slimes }

// Types devolve a lista expandida, na ordem em que devem nascer: o orc
// primeiro, porque e ele que ocupa o vao e os outros se arranjam em volta.
func (s GarrisonSquad) Types() []entity.EnemyType {
	out := make([]entity.EnemyType, 0, s.Total())
	for i := 0; i < s.Orcs; i++ {
		out = append(out, entity.EnemyTypeGarrison)
	}
	for i := 0; i < s.Wolves; i++ {
		out = append(out, entity.EnemyTypeFast)
	}
	for i := 0; i < s.Slimes; i++ {
		out = append(out, entity.EnemyTypeBasic)
	}
	return out
}

// O ORC GUARDA O VAO; LOBO E SLIME PATRULHAM O CAMPO.
//
// A divisao e a leitura da fase. O campo entre duas barricadas e travessia:
// coisa rapida e fragil, que o grupo atravessa lutando em movimento. O vao e
// decisao: ali esta o orc, e enfrentar um custa mais do que dar a volta pelo
// mapa ate a outra passagem — quando ha outra.
//
// A DIFICULDADE SOBE A CADA BARRICADA, e dentro da linha C ela se divide: o
// vao oeste tem SETE orcs, acima dos cinco que derrotam a Legiao Espectral (ver
// entity/orc_legion_test.go), e o leste tem quatro. A escolha entre atravessar
// o mapa e pagar o preco e o que a linha de dois vaos existe para propor.
//
// ESTA E A PRIMEIRA FASE EM QUE O NECROMANTE TEM A SUPREMA, E ISSO DECIDE A
// COMPOSICAO.
//
// A Legiao apaga massa fraca: trinta espectros valem cerca de quatro lobos cada
// um, e a recarga e de 60 s. Se o preco do mapa fosse slime e lobo, um
// lancamento por minuto compraria a fase inteira — e isso nao seria culpa da
// ultimate, que esta fazendo exatamente o que foi desenhada para fazer. Entao o
// preco esta no ORC, que e o antidoto declarado e testado, e a massa fraca
// passa a ser o combustivel dela em vez do obstaculo.
//
// Por isso os vaos subiram mais que os setores: 22 orcs viraram 36, enquanto
// lobo e slime subiram cerca de 1,6x, para o grupo de cinco. O vao C1 e o caso
// limite de proposito — sete orcs continuam custando caro DEPOIS da suprema, e
// e isso que mantem a escolha entre os dois vaos viva em vez de decorativa.
var world03Garrison = []GarrisonSquad{
	// --- setor 1, a mata: ja em campo quando o grupo entra ---
	{Post: "mata_oeste", Slimes: 4},
	{Post: "mata_leste", Slimes: 3},
	{Post: "mata_centro", Wolves: 5},

	// --- barricada A (linha 50), vao unico a leste: a mordida inicial ---
	{Post: "vao_a_oeste", Orcs: 2, Wolves: 2},
	{Post: "vao_a_leste", Orcs: 2, Wolves: 2},

	// --- setor 2, a trilha vigiada ---
	{Post: "trilha_oeste", Wolves: 5, Slimes: 2},
	{Post: "trilha_leste", Wolves: 5, Slimes: 2},

	// --- barricada B (linha 40), vao unico a oeste ---
	{Post: "vao_b_oeste", Orcs: 3, Wolves: 2},
	{Post: "vao_b_leste", Orcs: 2, Wolves: 3},

	// --- setor 3, o corte: campo aberto, matilha ---
	{Post: "corte_oeste", Wolves: 6},
	{Post: "corte_leste", Wolves: 6},
	{Post: "corte_centro", Slimes: 6},
	{Post: "corte_norte", Wolves: 3, Slimes: 3},

	// --- barricada C (linha 30), DOIS vaos com precos diferentes ---
	// Oeste: sete orcs. Cinco sao o que limpa a Legiao Espectral, e passar
	// desse numero e o ponto — este vao existe para ser caro, e continuar caro
	// numa fase em que a suprema ja existe.
	{Post: "vao_c1_oeste", Orcs: 4, Wolves: 3},
	{Post: "vao_c1_leste", Orcs: 3, Wolves: 3},
	// Leste: pouco mais da metade. O grupo que atravessar o mapa paga em tempo
	// e em patrulha, e economiza a briga — mas nao a economiza de graca.
	{Post: "vao_c2_oeste", Orcs: 2, Wolves: 3},
	{Post: "vao_c2_leste", Orcs: 2, Wolves: 2},

	// --- setor 4, o patio: a defesa pesada ---
	{Post: "patio_oeste", Orcs: 2, Wolves: 4},
	{Post: "patio_leste", Orcs: 2, Wolves: 4},
	{Post: "patio_centro_oeste", Wolves: 3, Slimes: 3},
	{Post: "patio_centro_leste", Wolves: 3, Slimes: 3},
	{Post: "patio_norte", Orcs: 2, Slimes: 4},

	// --- barricada D (linha 20), a boca da fortaleza: a mais cara do caminho ---
	{Post: "vao_d_oeste", Orcs: 5, Wolves: 3},
	{Post: "vao_d_leste", Orcs: 5, Wolves: 3},

	// --- A RETAGUARDA: o preco de RECUAR ---
	//
	// Todo posto acima olha para o norte, porque a fase e uma subida. Depois
	// que a perseguicao virou permanente, isso deixou um buraco: o jogador anda
	// a 200 contra os 130 do orc, entao recuar era saida garantida — atras dele
	// nao havia ninguem. A linha da frente era a unica linha.
	//
	// Estes oito nao adensam vao nenhum; eles cobrem os VAZIOS por onde se
	// recua. Composicao pequena e quase toda de lobo, de proposito: retaguarda
	// existe para tirar a fuga do cardapio, nao para virar uma segunda batalha,
	// e lobo a 240 e o unico que alcanca quem esta correndo. Vinte monstros ao
	// todo, contra os 136 que ja estavam em campo.
	{Post: "mata_sul_oeste", Wolves: 2},
	{Post: "mata_sul_leste", Wolves: 2},
	{Post: "mata_norte", Wolves: 2, Slimes: 1},
	{Post: "trilha_sul", Wolves: 2, Slimes: 1},
	{Post: "corte_sul_oeste", Wolves: 2},
	{Post: "corte_sul_leste", Wolves: 2},
	{Post: "patio_sul_oeste", Wolves: 2, Slimes: 1},
	// O unico com orc, e ele nao e retaguarda: o flanco nordeste e rota de
	// APROXIMACAO da barricada D, e quem sobe por fora tem de pagar tambem.
	{Post: "patio_nordeste", Orcs: 1, Wolves: 2},
}

// O SAGUAO NAO TEM VAO PARA ESCOLHER: A PRESSAO E QUE MUDA.
//
// O mapa 3 propoe uma escolha — atravessar o mapa ate a passagem barata ou
// pagar caro na que esta na frente. O mapa 4 nao tem essa escolha: e um
// corredor unico entre dois corregos, e o grupo so pode ir para a frente. Entao
// a guarnicao aqui nao guarda vaos, ela DENSIFICA: cada faixa e mais pesada que
// a anterior, e o jogador mede o avanco pelo que aguenta.
//
// A curva sobe ate o saguao, onde a emboscada arma — e NAO alivia depois. Sair
// do climax com o mapa vazio faria a subida ate a escadaria virar caminhada, e
// a fase acabaria antes do fim. As duas faixas de cima sao as mais caras do
// trajeto, com o orc aparecendo so quando o castelo ja se mostrou tomado.
//
// A guarnicao e a emboscada convivem aqui de proposito, e isso NAO e o caso
// que `GarrisonFor` desaconselha: o que nao pode conviver e guarnicao com
// `waveRuns`, porque sao dois relogios concorrentes. `climaxRuns` nao tem
// relogio — quem a dispara e a chegada do grupo ao objetivo.
// A REGUA DESTA FASE E A SACERDOTISA, E ELA MUDA O QUE "DIFICIL" QUER DIZER.
//
// Aqui o grupo ja tem a Area Angelical: raio 720, 20 de cura por segundo, doze
// segundos, e os caidos voltam. Enfileirar mais orcs contra isso e enfileirar
// mais do que ela foi desenhada para segurar — a fase ficaria mais longa e nao
// mais dificil.
//
// O que ela NAO responde e o que vem de fora do raio, e e por isso que a
// mudanca de verdade desta fase nao esta nesta tabela: sao as quatro gargulas
// de `sentries.go`, alcance 1900, em campo desde a chegada. A guarnicao aqui
// sobe de 37 para 67 para o corredor nao ser atravessado em passo de caminhada
// enquanto elas atiram, mas quem faz a fase doer sao elas.
var world04Garrison = []GarrisonSquad{
	// --- vestibulo (tier 1): ja em campo quando o portao fecha atras ---
	{Post: "vestibulo_oeste", Slimes: 4},
	{Post: "vestibulo_leste", Slimes: 4},

	// --- corredor processional (tier 2): a matilha entra ---
	{Post: "corredor_oeste", Slimes: 3, Wolves: 2},
	{Post: "corredor_leste", Slimes: 3, Wolves: 2},
	{Post: "corredor_centro", Wolves: 4},

	// --- boca do saguao (tier 3): o preco de entrar na arena ---
	// Nos CANTOS de entrada, nao no miolo: o miolo e onde os simultaneos da
	// emboscada vao circular, e enche-lo de guarnicao tira o chao dela.
	{Post: "saguao_oeste", Orcs: 2, Wolves: 5},
	{Post: "saguao_leste", Orcs: 2, Wolves: 5},

	// --- antessala (tier 4): depois do climax, e mais pesado, nao menos ---
	{Post: "antessala_oeste", Orcs: 3, Wolves: 4},
	{Post: "antessala_leste", Orcs: 3, Wolves: 4},
	{Post: "antessala_centro", Slimes: 5},

	// --- pe da escadaria (tier 5): a ultima porta antes do portal ---
	{Post: "escadaria_oeste", Orcs: 4, Wolves: 2},
	{Post: "escadaria_leste", Orcs: 4, Wolves: 2},

	// --- A RETAGUARDA: o preco de RECUAR (mesma razao do mapa 3) ---
	//
	// O corredor do castelo e estreito e so vai para a frente, entao aqui o
	// buraco era ainda mais simples: recuar pelo tapete nao encontrava nada ate
	// o portao fechado. Treze monstros, nenhum no MIOLO do saguao — aquele
	// espaco pertence aos 22 simultaneos da emboscada, e enche-lo tira o chao
	// da cena do Arqueiro.
	{Post: "vestibulo_sul_oeste", Slimes: 2},
	{Post: "vestibulo_sul_leste", Slimes: 2},
	{Post: "corredor_sul", Wolves: 2},
	{Post: "corredor_norte", Wolves: 2},
	{Post: "antessala_sul", Orcs: 1, Wolves: 2},
	{Post: "escadaria_sul", Wolves: 2},
}

// garrisons e a guarnicao de cada mapa, pelo caminho a partir da raiz do repo
// — a mesma chave que World.Path, campaignMaps, waveRuns e lastStandHeroes.
var garrisons = map[string][]GarrisonSquad{
	"assets/maps/world_03.json": world03Garrison,
	"assets/maps/world_04.json": world04Garrison,
}

// GarrisonFor devolve a guarnicao do mapa, ou nil quando ele nao declara uma.
//
// Mapa sem guarnicao e um mapa de horda (ou de teste), e fica quieto — a mesma
// forma de WaveDefsFor. Os dois nunca convivem no mesmo mapa de proposito: uma
// fase ou e travessia de territorio ou e corrida de hordas, e misturar as duas
// daria ao jogador dois relogios concorrentes sem lhe dizer.
func GarrisonFor(mapPath string) []GarrisonSquad {
	return garrisons[mapPath]
}

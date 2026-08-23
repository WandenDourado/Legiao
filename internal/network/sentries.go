package network

// QUANTAS sentinelas cada mapa arma, e QUANDO.
//
// A gargula (`entity.EnemyTypeCastleSentry`) e a unica criatura do jogo que nao
// se move e a unica que `checkProjectileCollisions` recusa machucar com projetil
// comum. Alcance 1900, contra os 720 de raio da Area Angelical da Sacerdotisa:
// ela e, literalmente, o monstro desenhado para bater de FORA da resposta que o
// grupo tem. E por isso que ela e o instrumento de dificuldade do mapa 4, e por
// isso que a fase seguinte a ela e a que entrega as Flechas Celestiais — 40 de
// dano perfurante contra os 40 de vida dela, uma flecha por gargula.
//
// AS DUAS PORTAS DE ENTRADA SAO DIFERENTES PORQUE OS MAPAS SAO.
//
// O mapa 4 e travessia de territorio: nada "chega", tudo ja esta em campo
// quando o grupo entra. Entao as sentinelas dele sao GUARNICAO — nascem com o
// mapa, em `sentriesOnArrival`.
//
// O mapa 5 e corrida de hordas: a fase se declara em degraus, e uma gargula na
// primeira horda apareceria antes de a fase ter dito o que ela e. Entao as
// sentinelas dele entram POR HORDA, em `WaveDef.Sentries`.
//
// Os dois leem a mesma lista de postos, e a lista e do MAPA — mesma forma de
// `waveRuns`, `garrisons` e `lastStandHeroes`. Uma fase nova declara os postos
// dela e nao depende de ninguem lembrar de editar uma constante.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// sentryPosts e a ORDEM em que os postos de sentinela de cada mapa sao
// ocupados, pelo sufixo do marcador `enemy_sentry_*`.
//
// A ordem importa no mapa 5, onde as gargulas entram aos poucos: os primeiros
// nomes sao os primeiros a serem armados, entao eles sao os postos mais
// distantes do centro do salao — a fase abre o fogo de longe e vai fechando.
var sentryPosts = map[string][]string{
	// A arena final: dez postos nas EXTREMIDADES leste e oeste.
	"assets/maps/world_07.json": {
		// A gargula tem Speed 0: onde ela nasce e onde fica pela luta inteira.
		// Por isso os dez postos estao colados nas paredes LESTE e OESTE,
		// acima e abaixo de cada portao — elas guardam a boca por onde a horda
		// entra, e quem for segurar um portao tem de resolver a torre antes.
		//
		// A ordem ALTERNA os lados, e isso e o ponto: cada horda arma uma a
		// oeste e uma a leste, nunca duas do mesmo lado. Duas seguidas num lado
		// so faria o grupo abandonar aquele portao.
		"oeste_norte", "leste_norte",
		"oeste_sul", "leste_sul",
		"oeste_meio_n", "leste_meio_n",
		"oeste_meio_s", "leste_meio_s",
		"oeste_alto", "leste_alto",
	},
	"assets/maps/world_04.json": {
		"ilha_oeste", "ilha_leste",
	},
	// A ordem do mapa 5 e a coreografia da fase, e le-se de tras para a frente:
	// o grupo entra pelo SUL, entao as duas primeiras gargulas ficam no fundo
	// do salao (norte) e atiram de longe; as do meio pegam os flancos; e so as
	// duas ultimas, na horda final, ficam nas COSTAS de quem entrou. O fogo
	// comeca vindo de onde o grupo esta indo e termina vindo de toda parte.
	"assets/maps/world_05.json": {
		"oval_norte_oeste", "oval_norte_leste",
		"oval_oeste", "oval_leste",
		"oval_sul_oeste", "oval_sul_leste",
	},
}

// sentriesOnArrival e quantas sentinelas ja estao em campo quando o mapa
// carrega.
//
// So o mapa 4 tem uma entrada, e ela e a mudanca de leitura da fase: as duas
// gargulas nasciam dentro de `StartClimax`, ou seja, o corredor inteiro ate o
// saguao era corpo a corpo puro e o fogo de longa distancia era uma surpresa do
// ultimo minuto. Agora sao DUAS, uma por ilha, em campo desde o portao se
// fechar atras do grupo, e a fase inteira e uma travessia sob fogo — que e a
// unica pressao que a Area Angelical nao responde, e portanto a unica que
// ainda significa alguma coisa numa fase em que a Sacerdotisa ja tem a
// suprema.
var sentriesOnArrival = map[string]int{
	"assets/maps/world_04.json": 2,
}

// SentryPostsFor devolve os postos declarados pelo mapa.
func SentryPostsFor(mapPath string) []string { return sentryPosts[mapPath] }

// armSentries ocupa postos ate `want` deles terem sido usados nesta corrida.
//
// O CURSOR CONTA POSTOS OCUPADOS, NAO GARGULAS VIVAS, e a diferenca e a regra
// inteira: contando as vivas, uma gargula abatida na horda 3 do mapa 5 seria
// reposta quando a horda 4 pedisse quatro, e o Arqueiro estaria gastando a
// suprema num alvo que renasce. Cada posto e ocupado no maximo uma vez por
// corrida, e o que o jogador derruba fica derrubado.
//
// O cursor volta a zero em `ResetStage`, porque uma tentativa nova e uma
// corrida nova.
//
// Devolve quantas nasceram agora.
func (h *Host) armSentries(mapPath string, posts []tilemap.SpawnPoint, want int) int {
	names := sentryPosts[mapPath]
	if len(names) == 0 || want <= h.sentriesArmed {
		return 0
	}
	if want > len(names) {
		log.Printf("[Sentinela] %s pede %d sentinelas e so declara %d postos; "+
			"as que sobram nao vao existir", mapPath, want, len(names))
		want = len(names)
	}

	at := make(map[string]rl.Vector2, len(posts))
	for _, p := range posts {
		at[p.Name] = p.Position
	}

	born := 0
	for ; h.sentriesArmed < want; h.sentriesArmed++ {
		name := names[h.sentriesArmed]
		pos, ok := at[name]
		if !ok {
			// Posto declarado na tabela e ausente do mapa. Falha em silencio de
			// um jeito ruim — a fase so fica mais facil do que devia —, entao
			// vira log alto, como o posto de guarnicao sem marcador.
			log.Printf("[Sentinela] %s: o mapa nao tem enemy_sentry_%s", mapPath, name)
			continue
		}
		h.EntityManager.AddEnemy(entity.NewEnemy(entity.EnemyTypeCastleSentry, pos.X, pos.Y))
		born++
	}
	if born > 0 {
		log.Printf("[Sentinela] %s: %d gargula(s) armada(s); %d posto(s) ocupado(s) de %d",
			mapPath, born, h.sentriesArmed, len(names))
	}
	return born
}

// InstallArrivalSentries poe em campo as sentinelas que o mapa ja tem quando o
// grupo chega. Mapa sem entrada em `sentriesOnArrival` nao faz nada, que e o
// caso de todos menos o quarto.
//
// Chamado no carregamento do mapa, ao lado de `StartGarrison`: os postos ficam
// guardados no host porque o reinicio de fase precisa repo-los e porque o
// WaveRunner do mapa 5 pede sentinelas sem ter, ele proprio, acesso ao mapa.
func (h *Host) InstallArrivalSentries(mapPath string, posts []tilemap.SpawnPoint) {
	h.stageSentries = posts
	h.sentriesArmed = 0
	// Mapa novo, torres caladas de novo: quem decide quando elas abrem fogo e
	// o degrau de territorio que a fase declara (sentry_wake.go).
	h.sentriesAwake = false
	h.armSentries(mapPath, posts, sentriesOnArrival[mapPath])
}

// RestoreSentries repoe as gargulas depois de um reinicio de fase.
//
// Irma de `RestoreGarrison`, e pelo mesmo motivo: `ResetStage` esvazia o
// EntityManager inteiro, e sem isto a segunda tentativa do mapa 4 seria
// atravessada sem um unico tiro de longe — a fase ficaria MAIS FACIL a cada
// derrota, que e o contrario do que um reinicio deve fazer.
func (h *Host) RestoreSentries() {
	h.sentriesArmed = 0
	// E a porta de despertar volta a fechar: uma tentativa nova comeca com o
	// grupo no vestibulo, e as torres nao podem lembrar do avanco da anterior.
	h.sentriesAwake = false
	h.armSentries(h.stageMap, h.stageSentries, sentriesOnArrival[h.stageMap])
}

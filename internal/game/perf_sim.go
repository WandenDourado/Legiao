package game

// Quanto do quadro foi SIMULACAO, e nao desenho.
//
// O painel do F3 media `mapa` (camadas do tilemap) e `entidades` (o passe de
// desenho de monstros e projeteis), e chamava de `resto` tudo o que sobrava.
// Numa captura do world_02 esse `resto` era 90 dos 96,7 ms do quadro, e a
// palavra "resto" nao diz se aquilo e GPU, vsync, HUD ou simulacao — as quatro
// pedem correcoes diferentes.
//
// Este arquivo separa a fatia que o codigo controla diretamente: no host,
// `UpdateSimulation`; no cliente, o avanco dos visuais de magia. As duas rodam
// no laco principal, fora do bloco da camera, e por isso caiam inteiras dentro
// do `resto`.
//
// Foi exatamente aqui que estava o defeito da Legiao Espectral: trinta
// espectros testando obstaculo contra a lista inteira de solidos do mapa. Com
// esta linha no painel, o mesmo defeito na proxima magia aparece na primeira
// captura em vez de virar "o jogo esta lento".

import "time"

// simMS e quanto a simulacao custou no ultimo quadro, em milissegundos.
//
// Um valor unico e nao uma janela: ele e lido junto do medidor de quadro, que
// ja publica media e pior caso da janela dele. Duas janelas diferentes na
// mesma captura confundiriam mais do que informam.
var simMS float32

// recordSimTime guarda o custo da simulacao deste quadro.
func recordSimTime(d time.Duration) {
	simMS = float32(d.Seconds() * 1000)
}

// simMilliseconds e o custo da simulacao do ultimo quadro.
func simMilliseconds() float32 { return simMS }

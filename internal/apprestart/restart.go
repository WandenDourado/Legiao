// Package apprestart reabre o jogo do zero — processo novo, janela nova.
//
// Existe por causa de Configuracoes > Graficos > Resolucao (29/09/2026): no
// celular a resolucao de desenho so vale quando a janela NASCE (o scaler de
// hardware, ver game/render_scale.go), e "fechar e abrir" no Android nao e o
// que o jogador acha que e. Sair pelo Home e voltar reaproveita o MESMO
// processo e a MESMA janela, entao a troca nunca aparecia — foi o relato do
// Gui num Galaxy S25 Ultra. Reiniciar de verdade tira essa duvida do jogador.
//
// Supported diz se a plataforma sabe se reabrir. No desktop nao ha o que
// reiniciar: a resolucao e sempre a do monitor.
package apprestart

import "errors"

// ErrUnsupported e o erro de Restart numa plataforma sem reinicio.
var ErrUnsupported = errors.New("apprestart: plataforma nao reinicia sozinha")

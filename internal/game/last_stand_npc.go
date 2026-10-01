//go:build ignore

// ESTE ARQUIVO PODE SER APAGADO.
//
// Ele desenhava o "heroi invocado" do ultimo suspiro — o NPC que aparecia
// quando ninguem no grupo jogava com a classe da fase. Esse caso deixou de
// existir em 23/08/2026: toda classe vaga e preenchida por um BOT
// (internal/network/host_bots.go), entao a cena sempre encontra um corpo de
// verdade para reerguer, e quem lanca a suprema e um jogador.
//
// A tag `//go:build ignore` acima tira o arquivo da compilacao. Ele so continua
// aqui porque a ferramenta que gravou esta mudanca no disco sabe escrever
// arquivos e nao sabe apaga-los. Apague-o.
//
// Ver doc/combat_rules.md, "Nao ha mais NPC".

package game

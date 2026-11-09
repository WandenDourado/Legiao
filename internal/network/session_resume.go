package network

// O que a tela de Configuracoes precisa saber da sessao para reabrir o jogo e
// VOLTAR a ela (ui/resume_session.go, 29/09/2026).
//
// Reiniciar o app para aplicar a resolucao do celular matava a partida sem
// volta. Do lado do CLIENTE a volta ja existia inteira: o host guarda o slot
// ausente por ReconnectGrace e um MsgJoin com o mesmo PlayerID devolve
// posicao, vida, recargas e o mapa do grupo (host_rejoin.go) — so faltava o
// processo novo lembrar o ID e o endereco. Do lado do HOST a partida vive no
// processo; o que da para devolver e a FASE: ele reabre no mesmo mapa, com o
// mesmo personagem, e os convidados reconectam sozinhos: na rede local ao
// mesmo IP, online ao mesmo CODIGO de sala, que o host pede de volta ao
// relay com o segredo dele (OpenOnlineRoomAs).

import "sync/atomic"

// A identidade da sala online deste host, alem do codigo (online_rooms.go):
// o nome que aparece na lista e o segredo que prova ao relay que o codigo e
// dele. Os tres juntos reabrem a MESMA sala depois de um reinicio.
var onlineRoomName, onlineRoomSecret atomic.Value

// OnlineRoomName devolve o nome da sala online deste host, ou "".
func OnlineRoomName() string {
	s, _ := onlineRoomName.Load().(string)
	return s
}

// OnlineRoomSecret devolve o segredo da sala online deste host, ou "".
func OnlineRoomSecret() string {
	s, _ := onlineRoomSecret.Load().(string)
	return s
}

// StageMap e o caminho do mapa que o grupo esta jogando ("" antes do
// primeiro). Lido pelo laco do jogo, a mesma thread que o escreve.
func (h *Host) StageMap() string { return h.stageMap }

// OtherHumansPresent conta quem joga nesta partida alem desta maquina: sem
// bots e sem ausentes. E quem cai quando o host reinicia.
func OtherHumansPresent() int {
	n := 0
	for id := range PresentPlayers() {
		if id != LocalPlayerID && !isBotID(id) {
			n++
		}
	}
	return n
}

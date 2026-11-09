//go:build android

package entity

// No celular o grao e maior e o teto e a metade: houve relato de lentidao e
// aquecimento (28/09/2026), e cada fragmento e uma chamada de desenho a mais.
// Grao de 5 px em vez de 4 corta ~36% dos fragmentos sem mudar a leitura do
// efeito numa tela pequena. Quem ainda sentir pode desligar em Configuracoes >
// Video e desempenho.
const (
	deathFXCellWorld float32 = 5
	deathFXBudget            = 6000
)

//go:build !android

package entity

// Grao e orcamento da desintegracao no desktop. Ver death_fx_tuning_android.go
// para por que o celular usa outros numeros.
const (
	// deathFXCellWorld e o lado de um fragmento em px de mundo.
	deathFXCellWorld float32 = 4
	// deathFXBudget e o teto de fragmentos vivos somando todas as mortes em
	// curso. A queda da chefe mata a horda inteira no mesmo quadro
	// (endHordeOnVictory): e esse o pico que o teto segura.
	deathFXBudget = 12000
)

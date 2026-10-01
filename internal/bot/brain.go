package bot

import "github.com/WandenDourado/Legiao/internal/entity"

// Brain decides what a bot does on one simulation tick. Implementations are
// stateful (they remember their current target and the reaction/decision
// clocks from tuning.go) because the host creates exactly one Brain per bot
// and calls Think on it every tick for the life of that bot.
type Brain interface {
	Think(View) Intent
}

// For returns a fresh Brain for the given character. Unknown types fall
// back to the Mago brain, mirroring entity.GetCharacter's own fallback.
func For(char entity.CharacterType) Brain {
	switch char {
	case entity.CharPaladina:
		return &paladinaBrain{}
	case entity.CharSacerdotisa:
		return &sacerdotisaBrain{}
	case entity.CharArqueiro:
		return &arqueiroBrain{}
	case entity.CharNecromante:
		return &necromanteBrain{}
	default:
		return &magoBrain{}
	}
}

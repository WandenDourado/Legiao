package ability

import "github.com/WandenDourado/Legiao/internal/entity"

// init registers every skill and binds it to its character. Adding a new
// character only requires a RegisterCharacter call (entity) plus a BindAbility
// line here — no edits to input, host, or renderer.
func init() {
	NewFireballSkill()
	NewSanctuarySkill()
	NewArrowVolleySkill()
	NewShieldSkill()
	NewMeteorRainSkill()
	NewAngelicAreaSkill()
	NewCelestialArrowsSkill()
	NewDivineAvatarSkill()
	NewGraveyardSkill()
	NewSpectralLegionSkill()

	// Slot 0: primary skill (Q / orange button).
	BindAbility(entity.CharMago, "fireball")
	BindAbility(entity.CharSacerdotisa, "sanctuary")
	BindAbility(entity.CharArqueiro, "arrow_volley")
	BindAbility(entity.CharPaladina, "shield")
	BindAbility(entity.CharNecromante, "graveyard")

	// Slot 1: ultimate skill (R / golden button).
	BindAbility(entity.CharMago, "meteor_rain")
	BindAbility(entity.CharSacerdotisa, "angelic_area")
	BindAbility(entity.CharArqueiro, "celestial_arrows")
	BindAbility(entity.CharPaladina, "divine_avatar")
	BindAbility(entity.CharNecromante, "spectral_legion")
}

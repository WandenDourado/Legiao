package entity

// CharacterType identifies a playable character kind.
type CharacterType string

const (
	// CharMago is the default mage character and casts Bola de Fogo.
	CharMago CharacterType = "mago"
	// CharSacerdotisa is the priestess character.
	CharSacerdotisa CharacterType = "sacerdotisa"
	// CharPaladina is the paladin character.
	CharPaladina CharacterType = "paladina"
	// CharArqueiro is the elven archer character.
	CharArqueiro CharacterType = "arqueiro"
	// CharNecromante is the necromancer character.
	CharNecromante CharacterType = "necromante"
)

// CharacterDef describes the visual properties of a playable character.
type CharacterDef struct {
	Type               CharacterType
	Name               string  // Display name shown in character select.
	SpritePath         string  // Relative path to the sprite sheet (used with assets.Path).
	ReferenceImagePath string  // Optional full-character reference image for character select (used with assets.Path).
	FrameWidth         int     // Width in pixels of a single frame.
	FrameHeight        int     // Height in pixels of a single frame.
	Columns            int     // Number of columns in the sprite sheet.
	Rows               int     // Number of rows in the sprite sheet.
	RenderScale        float32 // Visual scale applied without changing world collision size.
	FrameTime          float32 // Seconds per frame during normal walk.
	SprintTime         float32 // Seconds per frame during sprint.
	// FootLine is where the character's sole sits inside a frame, measured in
	// frame pixels from the top. It is what turns the sprite into a figure
	// standing on the ground: the collision box is placed there instead of at
	// the sprite's centre, so what blocks the character is what its feet
	// touch. Zero falls back to FrameHeight.
	//
	// It is not a new measurement — it is the `anchor.y` the sprite sheet's
	// own metadata already declares (186 for the current 128x192 rig). Copy
	// that number here when installing a character; measuring the PNG again
	// would let the two drift.
	FootLine int
	// AttackSpeed is how many basic attacks per second the character may
	// land. The host converts it into a per-player cooldown
	// (1/AttackSpeed seconds), so a higher value means a faster fighter.
	// Zero means "no limit" and is only a safety fallback for characters
	// registered before this field existed.
	AttackSpeed float32
}

// AttackInterval returns the minimum time between two basic attacks, derived
// from AttackSpeed. Returns 0 when the character has no cadence limit.
func (d CharacterDef) AttackInterval() float32 {
	if d.AttackSpeed <= 0 {
		return 0
	}
	return 1.0 / d.AttackSpeed
}

// registry holds all registered character definitions keyed by type.
var registry = map[CharacterType]CharacterDef{}

// registrationOrder preserves the order characters were registered (for UI listing).
var registrationOrder []CharacterType

func init() {
	RegisterCharacter(CharacterDef{
		Type:               CharMago,
		Name:               "Mago",
		SpritePath:         "assets/sprites/mago/mago.png",
		ReferenceImagePath: "assets/sprites/mago/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Bola de fogo pesada: poucos golpes, cada um vale muito.
		AttackSpeed:        1.2,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharSacerdotisa,
		Name:               "Sacerdotisa",
		SpritePath:         "assets/sprites/sacerdotisa/sacerdotisa.png",
		ReferenceImagePath: "assets/sprites/sacerdotisa/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Ritmo medio: o projetil tambem cura aliados no caminho.
		AttackSpeed:        1.4,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharPaladina,
		Name:               "Paladina",
		SpritePath:         "assets/sprites/paladina/paladina.png",
		ReferenceImagePath: "assets/sprites/paladina/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Corpo a corpo em arco: rapido o bastante para segurar a linha.
		AttackSpeed:        1.5,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharNecromante,
		Name:               "Necromante",
		SpritePath:         "assets/sprites/necromante/necromante.png",
		ReferenceImagePath: "assets/sprites/necromante/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Cranio sombrio com roubo de vida: cadencia contida.
		AttackSpeed:        1.3,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharArqueiro,
		Name:               "Arqueiro",
		SpritePath:         "assets/sprites/arqueiro/arqueiro.png",
		ReferenceImagePath: "assets/sprites/arqueiro/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// O mais rapido do elenco: dano baixo por flecha, muitas flechas.
		AttackSpeed:        2.5,
	})
}

// RegisterCharacter adds a character definition to the global registry.
// It can be called from init() in any package that provides a new character.
func RegisterCharacter(def CharacterDef) {
	registry[def.Type] = def
	registrationOrder = append(registrationOrder, def.Type)
}

// GetCharacter returns the definition for the given type.
// Returns the Mago as fallback if the type is unknown.
func GetCharacter(ct CharacterType) CharacterDef {
	if def, ok := registry[ct]; ok {
		return def
	}
	return registry[CharMago]
}

// IsRegistered reports whether the type exists in the registry. Callers that
// must not silently fall back to the Mago (a dialogue portrait, for instance)
// check this before calling GetCharacter.
func IsRegistered(ct CharacterType) bool {
	_, ok := registry[ct]
	return ok
}

// AllCharacters returns all registered character definitions in registration order.
func AllCharacters() []CharacterDef {
	result := make([]CharacterDef, 0, len(registrationOrder))
	for _, ct := range registrationOrder {
		result = append(result, registry[ct])
	}
	return result
}

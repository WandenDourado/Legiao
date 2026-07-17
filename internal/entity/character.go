package entity

// CharacterType identifies a playable character kind.
type CharacterType string

const (
	// CharWizard is the default wizard character.
	CharWizard CharacterType = "wizard"
	// CharSacerdotisa is the priestess character.
	CharSacerdotisa CharacterType = "sacerdotisa"
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
}

// registry holds all registered character definitions keyed by type.
var registry = map[CharacterType]CharacterDef{}

// registrationOrder preserves the order characters were registered (for UI listing).
var registrationOrder []CharacterType

func init() {
	RegisterCharacter(CharacterDef{
		Type:               CharWizard,
		Name:               "Wizard",
		SpritePath:         "assets/sprites/wizard/wizard.png",
		ReferenceImagePath: "assets/sprites/wizard/reference.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.0,
		FrameTime:          0.12,
		SprintTime:         0.08,
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
	})
}

// RegisterCharacter adds a character definition to the global registry.
// It can be called from init() in any package that provides a new character.
func RegisterCharacter(def CharacterDef) {
	registry[def.Type] = def
	registrationOrder = append(registrationOrder, def.Type)
}

// GetCharacter returns the definition for the given type.
// Returns the wizard as fallback if the type is unknown.
func GetCharacter(ct CharacterType) CharacterDef {
	if def, ok := registry[ct]; ok {
		return def
	}
	return registry[CharWizard]
}

// AllCharacters returns all registered character definitions in registration order.
func AllCharacters() []CharacterDef {
	result := make([]CharacterDef, 0, len(registrationOrder))
	for _, ct := range registrationOrder {
		result = append(result, registry[ct])
	}
	return result
}

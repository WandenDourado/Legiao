package entity

import (
	"github.com/WandenDourado/Legiao/internal/sound"

	rl "github.com/gen2brain/raylib-go/raylib"
)

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
	Type       CharacterType
	Name       string // Display name shown in character select.
	SpritePath string // Relative path to the sprite sheet (used with assets.Path).
	// ReferenceImagePath is the full-body art shown in the character-select
	// card: 1024x1536 RGBA, feet on a fixed baseline. It answers "what class
	// is this?", so the reading is the silhouette.
	ReferenceImagePath string
	// PortraitImagePath is the half-body art the dialogue box shows while
	// this character speaks: 1152x1280 RGBA, cut at mid-thigh. It answers
	// "who is talking, and with what face?", so the reading is the FACE —
	// which is why it is a second image and not a crop of the one above.
	//
	// Empty, or a file that is not there, falls back to the reference art.
	// That fallback is what let the dialogue code land before the art did;
	// see internal/ui/portrait_cache.go.
	PortraitImagePath string
	FrameWidth        int     // Width in pixels of a single frame.
	FrameHeight       int     // Height in pixels of a single frame.
	Columns           int     // Number of columns in the sprite sheet.
	Rows              int     // Number of rows in the sprite sheet.
	RenderScale       float32 // Visual scale applied without changing world collision size.
	FrameTime         float32 // Seconds per frame during normal walk.
	SprintTime        float32 // Seconds per frame during sprint.
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
	// MaxHealth e a vida cheia do personagem. Zero significa
	// DefaultPlayerMaxHealth, e nao "sem vida": todo personagem registrado
	// antes deste campo existir continua valendo 100 sem precisar declarar
	// nada.
	//
	// Existe porque a vida deixou de ser igual para todo mundo: a Paladina
	// tem 300 (pedido do Gui, 27/08/2026) porque ela e a unica classe cujo
	// trabalho e ficar no alcance da mordida. Antes disso o 100 era um
	// literal repetido em quatro lugares do pacote network — quem criava o
	// host, quem criava um bot, quem reconectava e o proprio NewPlayer — e
	// dar vida diferente a uma classe teria de acertar os quatro.
	MaxHealth float32
	// AttackSound e o efeito do ataque basico deste personagem. Fica na
	// definicao, ao lado de AttackSpeed, pelo mesmo motivo que o sprite fica:
	// registrar um personagem novo tem de bastar. Vazio = ataque mudo.
	AttackSound sound.SFX
}

// DefaultPlayerMaxHealth e a vida de quem nao declara a propria.
const DefaultPlayerMaxHealth float32 = 100

// MaxHealthFor devolve a vida cheia de um personagem. E a UNICA fonte dessa
// resposta: host, bot, reconexao e NewPlayer perguntam aqui.
func MaxHealthFor(charType CharacterType) float32 {
	if hp := GetCharacter(charType).MaxHealth; hp > 0 {
		return hp
	}
	return DefaultPlayerMaxHealth
}

// AttackInterval returns the minimum time between two basic attacks, derived
// from AttackSpeed. Returns 0 when the character has no cadence limit.
func (d CharacterDef) AttackInterval() float32 {
	if d.AttackSpeed <= 0 {
		return 0
	}
	return 1.0 / d.AttackSpeed
}

// PlayAttackSound toca o ataque basico de ct em pos, para quem estiver perto
// (internal/sound). E a UNICA fonte dessa resposta: o host ao resolver o
// ataque, o golpe da Paladina e o cliente ao ver um projetil novo no
// instantaneo perguntam todos aqui.
//
// O PRAZO E A CADENCIA DO PROPRIO PERSONAGEM. Todos os cinco arquivos sao
// mais longos que o intervalo entre dois golpes — o do Necromante tem 8 s
// para uma cadencia de 0,77 s —, entao sem prazo cada personagem acumularia
// dez ataques soando ao mesmo tempo. Com AttackInterval, um golpe desvanece
// exatamente quando o proximo comeca, e o ritmo que se ouve e o ritmo que se
// joga. Nao e um numero novo: e o mesmo que ja governa a recarga no host.
func PlayAttackSound(ct CharacterType, pos rl.Vector2) {
	d := GetCharacter(ct)
	if d.AttackSound == "" {
		return
	}
	sound.PlayAtFor(d.AttackSound, pos, d.AttackInterval())
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
		PortraitImagePath:  "assets/sprites/mago/portrait.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Bola de fogo pesada: poucos golpes, cada um vale muito.
		AttackSpeed: 1.2,
		AttackSound: sound.SFXAttackMago,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharSacerdotisa,
		Name:               "Sacerdotisa",
		SpritePath:         "assets/sprites/sacerdotisa/sacerdotisa.png",
		ReferenceImagePath: "assets/sprites/sacerdotisa/reference.png",
		PortraitImagePath:  "assets/sprites/sacerdotisa/portrait.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Ritmo medio: o projetil tambem cura aliados no caminho.
		AttackSpeed: 1.4,
		AttackSound: sound.SFXAttackSacerdotisa,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharPaladina,
		Name:               "Paladina",
		SpritePath:         "assets/sprites/paladina/paladina.png",
		ReferenceImagePath: "assets/sprites/paladina/reference.png",
		PortraitImagePath:  "assets/sprites/paladina/portrait.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Corpo a corpo em arco: rapido o bastante para segurar a linha.
		AttackSpeed: 1.5,
		AttackSound: sound.SFXAttackPaladina,
		// DUAS VEZES a vida de todo mundo, e e a unica classe que sai do
		// padrao. Ela e a unica que tem de estar a 140 px do inimigo para
		// fazer o trabalho dela, enquanto as outras quatro atiram de 800 a
		// 1120 px; com 100 de vida, "segurar a linha" era um jeito mais lento
		// de morrer. Ver skill.SwordDamage, que e o outro lado da mesma
		// decisao.
		MaxHealth: 200,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharNecromante,
		Name:               "Necromante",
		SpritePath:         "assets/sprites/necromante/necromante.png",
		ReferenceImagePath: "assets/sprites/necromante/reference.png",
		PortraitImagePath:  "assets/sprites/necromante/portrait.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// Cranio sombrio com roubo de vida: cadencia contida.
		AttackSpeed: 1.3,
		AttackSound: sound.SFXAttackNecromante,
	})
	RegisterCharacter(CharacterDef{
		Type:               CharArqueiro,
		Name:               "Arqueiro",
		SpritePath:         "assets/sprites/arqueiro/arqueiro.png",
		ReferenceImagePath: "assets/sprites/arqueiro/reference.png",
		PortraitImagePath:  "assets/sprites/arqueiro/portrait.png",
		FrameWidth:         128,
		FrameHeight:        192,
		Columns:            8,
		Rows:               5,
		RenderScale:        1.15,
		FrameTime:          0.12,
		SprintTime:         0.08,
		FootLine:           186,
		// O mais rapido do elenco: dano baixo por flecha, muitas flechas.
		AttackSpeed: 2.5,
		AttackSound: sound.SFXAttackArqueiro,
	})
}

// RegisterCharacter adds a character definition to the global registry.
// It can be called from init() in any package that provides a new character.
// RegisterCharacter adds a character definition to the registry.
//
// FrameTime e SprintTime sao encurtados por GameSpeed (entity/gamespeed.go)
// pela mesma razao que o AnimWalk dos monstros: o personagem passou a andar
// mais rapido, e um ciclo de passos com a duracao antiga vira pe patinando.
// Os numeros escritos nos registros continuam sendo os autorais, medidos
// contra os 200 px/s originais.
func RegisterCharacter(def CharacterDef) {
	def.FrameTime = scaledFrameTime(def.FrameTime)
	def.SprintTime = scaledFrameTime(def.SprintTime)
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

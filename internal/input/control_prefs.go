package input

// Preferências de controle vindas da tela de Configurações
// (internal/settings, internal/ui/settings_section_controls_android.go).
// Empurradas via setter — o mesmo padrão que TouchState.SetMaxRadius já
// usa — em vez deste pacote importar internal/settings direto: a geometria
// (ComputeAndroidControlGeom, touch.go) roda TODO quadro, e um multiplicador
// já resolvido é mais barato que reconsultar o pacote de configurações a
// cada chamada, além de manter input sem depender de mais um pacote.

var (
	actionButtonScale   float32 = 1.0
	joystickRadiusScale float32 = 1.0
	leftHanded          bool
)

// SetActionButtonScale multiplica attackRadiusFrac (e, por extensão, todo o
// cluster de ações que deriva dele — ver ComputeAndroidControlGeom). Espera
// já vir clampado (settings.Settings.ActionButtonScale, 0.8..1.3).
func SetActionButtonScale(v float32) { actionButtonScale = v }

// SetJoystickRadiusScale multiplica o raio máximo do joystick de movimento
// (TouchState.MaxRadius é ajustado por quadro em ProcessInput multiplicando
// por este valor). Espera vir clampado (0.7..1.5).
func SetJoystickRadiusScale(v float32) { joystickRadiusScale = v }

// JoystickRadiusScale devolve o multiplicador atual, para quem monta o raio
// base (internal/game/input_handler.go).
func JoystickRadiusScale() float32 { return joystickRadiusScale }

// SetLeftHanded espelha o cluster de ações e o joystick: joystick à direita,
// ações à esquerda. Lido tanto por ComputeAndroidControlGeom (o cluster de
// ações) quanto pelo laço que posiciona o joystick (input_handler.go).
func SetLeftHanded(v bool) { leftHanded = v }

// LeftHanded devolve a preferência atual.
func LeftHanded() bool { return leftHanded }

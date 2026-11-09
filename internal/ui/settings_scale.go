package ui

// scaleToUnit/unitToScale deixam o Slider (que só entende 0..1, em degraus
// de 5%) representar faixas fora de 0..1 — a escala do botão de ação
// (0.8..1.3), o raio do joystick (0.7..1.5) e a escala do HUD
// (0.8..settings.HUDScaleMax).
// Compartilhado entre a seção Controles (build-tagged por plataforma) e a
// seção Interface (a mesma em toda plataforma).
func scaleToUnit(v, lo, hi float32) float32 { return (v - lo) / (hi - lo) }
func unitToScale(u, lo, hi float32) float32 { return lo + u*(hi-lo) }

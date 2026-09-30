package ui

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func testTrack() rl.Rectangle {
	return rl.NewRectangle(100, 200, 300, 20) // X=100..400
}

func TestSliderValueAtMapsPositionToValue(t *testing.T) {
	track := testTrack()
	cases := []struct {
		x    float32
		want float32
	}{
		{100, 0},    // left edge
		{400, 1},    // right edge
		{250, 0.50}, // exact middle
		{280, 0.60}, // 60% along, already on a step
	}
	for _, c := range cases {
		if got := SliderValueAt(track, c.x); got != c.want {
			t.Errorf("SliderValueAt(x=%.0f) = %v, want %v", c.x, got, c.want)
		}
	}
}

func TestSliderValueAtClampsBothEnds(t *testing.T) {
	track := testTrack()
	if got := SliderValueAt(track, track.X-500); got != 0 {
		t.Errorf("far left of the track = %v, want clamped to 0", got)
	}
	if got := SliderValueAt(track, track.X+track.Width+500); got != 1 {
		t.Errorf("far right of the track = %v, want clamped to 1", got)
	}
}

func TestSliderValueAtSnapsToFivePercentSteps(t *testing.T) {
	track := testTrack()
	// 253/300 = 84.33% -> should snap to 85%, not sit at an odd number
	// nobody could aim back to.
	got := SliderValueAt(track, track.X+253)
	want := float32(0.85)
	if diff := got - want; diff > 0.001 || diff < -0.001 {
		t.Errorf("SliderValueAt did not snap to a 5%% step: got %v, want %v", got, want)
	}
	// Every reachable value across the track must be a multiple of the step.
	for x := track.X; x <= track.X+track.Width; x += 3 {
		v := SliderValueAt(track, x)
		steps := v / SliderStep
		if rounded := float32(int(steps + 0.5)); steps-rounded > 0.001 || steps-rounded < -0.001 {
			t.Fatalf("value %v at x=%.0f is not a multiple of the %v step", v, x, SliderStep)
		}
	}
}

func TestSliderStepUpDownClampAtBounds(t *testing.T) {
	if got := SliderStepUp(1.0); got != 1.0 {
		t.Errorf("SliderStepUp(1.0) = %v, want clamped at 1.0", got)
	}
	if got := SliderStepDown(0.0); got != 0.0 {
		t.Errorf("SliderStepDown(0.0) = %v, want clamped at 0.0", got)
	}
	if got := SliderStepUp(0.5); got != 0.55 {
		t.Errorf("SliderStepUp(0.5) = %v, want 0.55", got)
	}
	if got := SliderStepDown(0.5); got != 0.45 {
		t.Errorf("SliderStepDown(0.5) = %v, want 0.45", got)
	}
}

// TestSliderTapJumpsToTouchedValue is Passo 4 item 1: tapping anywhere on
// the track has to move the value straight there, not require dragging from
// the current thumb position.
func TestSliderTapJumpsToTouchedValue(t *testing.T) {
	s := NewSlider()
	track := testTrack()
	// Directly exercising the pure mapping a tap resolves through, since
	// Update() itself needs live touch/mouse state that only exists inside
	// a raylib window (see slider_input.go) — not available in this test
	// binary. SliderValueAt is what Update calls on the first frame a touch
	// lands inside the track, so this is the tap's actual math.
	got := SliderValueAt(track, track.X+track.Width*0.7)
	if got != 0.70 {
		t.Errorf("tap at 70%% of the track = %v, want 0.70", got)
	}
	_ = s // constructed to document that Update() shares this struct's state; not callable here without a window.
}

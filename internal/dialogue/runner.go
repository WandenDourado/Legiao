package dialogue

// Runner is the cursor over one running script. It knows nothing about who is
// allowed to advance it; whoever owns the narrative (the host) calls Advance.
type Runner struct {
	script Script
	index  int
	active bool
}

// Start begins a script from its first line, replacing whatever was running.
// It reports false for an empty script, so the caller can skip publishing a
// state nobody can read.
func (r *Runner) Start(s Script) bool {
	if len(s.Lines) == 0 {
		return false
	}
	r.script = s
	r.index = 0
	r.active = true
	return true
}

// Active reports whether a line is currently on screen.
func (r *Runner) Active() bool { return r != nil && r.active }

// ScriptID returns the running script's ID, empty when idle.
func (r *Runner) ScriptID() string {
	if !r.Active() {
		return ""
	}
	return r.script.ID
}

// Current returns the line on screen plus its 1-based position in the script.
func (r *Runner) Current() (line Line, index, total int) {
	if !r.Active() {
		return Line{}, 0, 0
	}
	return r.script.Lines[r.index], r.index + 1, len(r.script.Lines)
}

// Advance moves to the next line and reports whether the script is still
// running. Advancing past the last line ends it, which is what makes Enter on
// the final line close the box instead of needing a separate dismiss.
func (r *Runner) Advance() bool {
	if !r.Active() {
		return false
	}
	r.index++
	if r.index >= len(r.script.Lines) {
		r.Stop()
		return false
	}
	return true
}

// Stop ends the running script immediately.
func (r *Runner) Stop() {
	r.active = false
	r.index = 0
	r.script = Script{}
}

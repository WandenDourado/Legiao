package dialogue

import (
	"encoding/json"
	"log"
	"path"
	"strings"

	"github.com/WandenDourado/Legiao/internal/assets"
)

// dialoguesDir is where a map's script file lives. One file per map, named
// after the map, so finding a map's narrative never needs an index.
const dialoguesDir = "assets/dialogues"

// LoadForMap returns the scripts written for the given map path
// ("assets/maps/world_01.json" -> "assets/dialogues/world_01.json").
//
// A map with no dialogue file is normal, not an error: most maps are silent.
// It returns an empty File in that case, so callers never branch on nil.
func LoadForMap(mapPath string) File {
	scriptPath := path.Join(dialoguesDir, strings.TrimSuffix(path.Base(mapPath), path.Ext(mapPath))+".json")
	f, err := Load(scriptPath)
	if err != nil {
		log.Printf("[Dialogo] %s sem roteiro (%v)", mapPath, err)
		return File{}
	}
	log.Printf("[Dialogo] %s: %d roteiros carregados de %s", mapPath, len(f.Scripts), scriptPath)
	return f
}

// Load reads and validates one dialogue file.
func Load(scriptPath string) (File, error) {
	data, err := assets.ReadFile(assets.Path(scriptPath))
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, err
	}
	f.Scripts = validScripts(f.Scripts, scriptPath)
	return f, nil
}

// validScripts drops what cannot be shown. A malformed entry is written out to
// the log and skipped rather than played half-broken, because a script that
// silently shows an empty box mid-cutscene is much harder to diagnose than a
// line in the log at load time.
// knownTrigger keeps the accepted list in one place: a trigger the director
// cannot answer would produce a script that silently never plays.
func knownTrigger(t Trigger) bool {
	switch t {
	case TriggerMapStart, TriggerWavesCleared, TriggerLastStand:
		return true
	}
	return false
}

func validScripts(in []Script, scriptPath string) []Script {
	out := make([]Script, 0, len(in))
	for _, s := range in {
		if s.ID == "" {
			log.Printf("[Dialogo] %s: roteiro sem id, ignorado", scriptPath)
			continue
		}
		if !knownTrigger(s.Trigger) {
			log.Printf("[Dialogo] %s: roteiro %s com gatilho desconhecido %q, ignorado", scriptPath, s.ID, s.Trigger)
			continue
		}
		lines := make([]Line, 0, len(s.Lines))
		for _, l := range s.Lines {
			if strings.TrimSpace(l.Text) == "" {
				log.Printf("[Dialogo] %s: linha sem texto em %s, ignorada", scriptPath, s.ID)
				continue
			}
			lines = append(lines, l)
		}
		if len(lines) == 0 {
			log.Printf("[Dialogo] %s: roteiro %s sem linhas, ignorado", scriptPath, s.ID)
			continue
		}
		s.Lines = lines
		out = append(out, s)
	}
	return out
}

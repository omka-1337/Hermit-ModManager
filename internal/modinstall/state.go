package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/thunderstore"
)

// disabledDir holds files of inactive mods, one subdirectory per mod.
const disabledDir = "disabled"

// filesLocation returns where a mod's files currently are, relative to the profile.
func filesLocation(m library.Mod, file string) string {
	if m.Active {
		return file
	}
	return path.Join(disabledDir, m.ID, file)
}

// Sync recomputes which mods can be active and moves files of mods whose state
// changed. A mod is active when it is enabled and every dependency is installed
// and active, which cascades: disabling a library deactivates its dependants.
func Sync(profileDir string, p *library.Profile) error {
	byID := map[string]int{}
	for i, m := range p.Mods {
		byID[m.ID] = i
	}
	depIDs := make([][]string, len(p.Mods))
	for i, m := range p.Mods {
		for _, d := range m.Dependencies {
			if ref, err := thunderstore.ParseDependency(d); err == nil {
				depIDs[i] = append(depIDs[i], ref.ID())
			}
		}
	}

	// Start from the enabled set and drop mods with inactive dependencies until
	// nothing changes; dependency cycles among enabled mods stay active.
	active := make([]bool, len(p.Mods))
	for i, m := range p.Mods {
		active[i] = m.Enabled
	}
	for changed := true; changed; {
		changed = false
		for i := range p.Mods {
			if !active[i] {
				continue
			}
			for _, id := range depIDs[i] {
				if j, ok := byID[id]; !ok || !active[j] {
					active[i], changed = false, true
					break
				}
			}
		}
	}

	var errs []error
	for i := range p.Mods {
		m := &p.Mods[i]
		m.UnmetDependencies = []string{}
		for k, id := range depIDs[i] {
			if j, ok := byID[id]; !ok || !active[j] {
				m.UnmetDependencies = append(m.UnmetDependencies, m.Dependencies[k])
			}
		}
		if m.Active == active[i] {
			continue
		}
		if err := moveFiles(profileDir, *m, active[i]); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.ID, err))
			continue
		}
		m.Active = active[i]
	}
	return errors.Join(errs...)
}

// moveFiles moves a mod's files between their active place and disabled/<mod-id>/.
func moveFiles(profileDir string, m library.Mod, activate bool) error {
	target := m
	target.Active = activate
	var moved []string
	for _, f := range m.Files {
		from := filepath.Join(profileDir, filepath.FromSlash(filesLocation(m, f)))
		to := filepath.Join(profileDir, filepath.FromSlash(filesLocation(target, f)))
		if _, err := os.Stat(from); errors.Is(err, os.ErrNotExist) {
			continue // already gone; nothing to move
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := os.Rename(from, to); err != nil {
			return err
		}
		moved = append(moved, filesLocation(m, f))
	}
	pruneDirs(profileDir, moved)
	return nil
}

// removeModFiles deletes a mod's files from wherever they currently are.
// Files that another active mod also lists are kept, so profiles created
// before conflict checks cannot lose files of a different mod.
func removeModFiles(profileDir string, m library.Mod, all []library.Mod) error {
	var files []string
	for _, f := range m.Files {
		shared := m.Active && slices.ContainsFunc(all, func(o library.Mod) bool {
			return o.ID != m.ID && o.Active && slices.Contains(o.Files, f)
		})
		if !shared {
			files = append(files, filesLocation(m, f))
		}
	}
	return Remove(profileDir, files)
}

// pruneDirs removes directories left empty after files were removed or moved.
func pruneDirs(profileDir string, files []string) {
	dirs := map[string]bool{}
	for _, f := range files {
		for d := path.Dir(f); d != "."; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	sorted := make([]string, 0, len(dirs))
	for d := range dirs {
		if !slices.Contains(keptDirs, d) {
			sorted = append(sorted, d)
		}
	}
	// Deepest first; os.Remove fails harmlessly on non-empty directories.
	slices.SortFunc(sorted, func(a, b string) int { return strings.Count(b, "/") - strings.Count(a, "/") })
	for _, d := range sorted {
		_ = os.Remove(filepath.Join(profileDir, filepath.FromSlash(d)))
	}
}

// Package modinstall installs and removes Thunderstore packages in a profile.
package modinstall

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"bepinexmodmanager/internal/thunderstore"
)

type entry struct {
	file   *zip.File
	dest   string
	config bool
}

// planEntries maps archive entries to their destination in the profile.
//
// A mod loader package (listed in rules.LoaderPackages, or recognised by a
// BepInEx/core folder) is unpacked into the profile root, since it carries the
// doorstop files that belong next to the game executable. Other packages are
// placed by the install rules the way r2modman does it:
//
//   - a folder named like the last segment of a route (plugins, config, ...),
//     at any depth, is installed into that route with its structure kept;
//     when several routes share the name, the one whose path matches the
//     folder's path best wins;
//   - other folders are descended into;
//   - loose files go to the route whose extension matches (the longest match
//     wins, so .mm.dll beats .dll), else to the default route, flattened to
//     their file name.
func planEntries(zr *zip.Reader, modID string, rules Rules) ([]entry, error) {
	files := map[string]*zip.File{}
	var names []string
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name, err := cleanEntryName(f.Name)
		if err != nil {
			return nil, err
		}
		if _, dup := files[name]; !dup {
			names = append(names, name)
		}
		files[name] = f
	}
	slices.Sort(names)

	if root, ok := loaderRoot(names, modID, rules); ok {
		var entries []entry
		for _, name := range names {
			rel, ok := strings.CutPrefix(name, root)
			if !ok || (root == "" && slices.Contains(loaderMetadata, strings.ToLower(rel))) {
				continue // manifest, icon, readme next to the loader folder
			}
			config := strings.HasPrefix(strings.ToLower(rel), "bepinex/config/")
			entries = append(entries, entry{file: files[name], dest: rel, config: config})
		}
		return entries, nil
	}

	var entries []entry
	add := func(name string, rule thunderstore.InstallRule, rel string) {
		e := entry{file: files[name]}
		switch rule.TrackingMethod {
		case thunderstore.TrackingNone:
			e.dest, e.config = path.Join(rule.Route, rel), true
		case thunderstore.TrackingState:
			if slices.ContainsFunc(rules.RelativeFileExclusions, func(x string) bool { return strings.EqualFold(x, rel) }) {
				return
			}
			e.dest = path.Join(rule.Route, rel)
		default: // subdir and anything unknown
			e.dest = path.Join(rule.Route, modID, rel)
		}
		entries = append(entries, e)
	}

	var walk func(dir string)
	walk = func(dir string) {
		subdirs := map[string]bool{}
		for _, name := range names {
			rest, ok := strings.CutPrefix(name, dir)
			if !ok {
				continue
			}
			if sub, _, nested := strings.Cut(rest, "/"); nested {
				subdirs[sub] = true
				continue
			}
			if rule, ok := ruleForFile(rules.Routes, rest); ok {
				add(name, rule, rest)
			}
		}
		for _, sub := range slices.Sorted(maps.Keys(subdirs)) {
			full := dir + sub + "/"
			rule, ok := ruleForDir(rules.Routes, strings.TrimSuffix(full, "/"))
			if !ok {
				walk(full)
				continue
			}
			for _, name := range names {
				if rel, ok := strings.CutPrefix(name, full); ok {
					add(name, rule, rel)
				}
			}
		}
	}
	walk("")

	// Later entries overwrite earlier ones on disk; keep the last per destination.
	seen := map[string]bool{}
	var unique []entry
	for i := len(entries) - 1; i >= 0; i-- {
		if !seen[entries[i].dest] {
			seen[entries[i].dest] = true
			unique = append(unique, entries[i])
		}
	}
	slices.Reverse(unique)
	return unique, nil
}

var loaderMetadata = []string{"manifest.json", "icon.png", "readme.md"}

// loaderRoot reports whether the package is a mod loader and returns the
// archive prefix that maps to the profile root.
func loaderRoot(names []string, modID string, rules Rules) (string, bool) {
	if folder, ok := rules.LoaderPackages[strings.ToLower(modID)]; ok {
		if folder == "" {
			return "", true
		}
		return folder + "/", true
	}
	for _, name := range names {
		i := strings.Index(strings.ToLower(name), "bepinex/core/")
		if i < 0 {
			continue
		}
		prefix := name[:i]
		if prefix == "" || (strings.Count(prefix, "/") == 1 && strings.HasSuffix(prefix, "/")) {
			return prefix, true
		}
	}
	return "", false
}

func ruleForFile(routes []thunderstore.InstallRule, name string) (thunderstore.InstallRule, bool) {
	lower := strings.ToLower(name)
	best, bestLen := thunderstore.InstallRule{}, 0
	for _, r := range routes {
		for _, ext := range r.DefaultFileExtensions {
			if strings.HasSuffix(lower, strings.ToLower(ext)) && len(ext) > bestLen {
				best, bestLen = r, len(ext)
			}
		}
	}
	if bestLen > 0 {
		return best, true
	}
	for _, r := range routes {
		if r.IsDefaultLocation {
			return r, true
		}
	}
	return thunderstore.InstallRule{}, false
}

// ruleForDir matches a folder by its name against the last route segment.
func ruleForDir(routes []thunderstore.InstallRule, dir string) (thunderstore.InstallRule, bool) {
	dirParts := strings.Split(dir, "/")
	name := dirParts[len(dirParts)-1]
	best, bestScore, found := thunderstore.InstallRule{}, -1, false
	for _, r := range routes {
		if !strings.EqualFold(path.Base(r.Route), name) {
			continue
		}
		routeParts := strings.Split(r.Route, "/")
		score := 0
		for i := 0; i < len(dirParts) && i < len(routeParts); i++ {
			if dirParts[len(dirParts)-1-i] == routeParts[len(routeParts)-1-i] {
				score++
			}
		}
		if score > bestScore {
			best, bestScore, found = r, score, true
		}
	}
	return best, found
}

// PlanFiles returns the tracked files Extract would install, without writing anything.
func PlanFiles(zipPath, modID string, rules Rules) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	entries, err := planEntries(&zr.Reader, modID, rules)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.config && !slices.Contains(files, e.dest) {
			files = append(files, e.dest)
		}
	}
	slices.Sort(files)
	return files, nil
}

// IsLoader reports whether installed files belong to a mod loader package.
// Only loader packages place files in the profile root (winhttp.dll and
// friends); install rules always put other files under a route folder.
func IsLoader(files []string) bool {
	return slices.ContainsFunc(files, func(f string) bool { return !strings.Contains(f, "/") })
}

// Extract unpacks a package archive into profileDir and returns the installed
// files as slash-separated paths relative to profileDir. Config files are
// written only if absent and are not returned, so user edits survive
// reinstalls and uninstalls.
func Extract(zipPath, profileDir, modID string, rules Rules) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	entries, err := planEntries(&zr.Reader, modID, rules)
	if err != nil {
		return nil, err
	}

	var installed []string
	for _, e := range entries {
		target := filepath.Join(profileDir, filepath.FromSlash(e.dest))
		if e.config {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		if err := writeEntry(e.file, target); err != nil {
			_ = Remove(profileDir, installed)
			return nil, fmt.Errorf("extract %s: %w", e.file.Name, err)
		}
		if !e.config && !slices.Contains(installed, e.dest) {
			installed = append(installed, e.dest)
		}
	}
	slices.Sort(installed)
	return installed, nil
}

func cleanEntryName(name string) (string, error) {
	name = strings.ReplaceAll(name, `\`, "/")
	clean := path.Clean("/" + name)[1:]
	if clean == "" || clean != strings.TrimPrefix(name, "./") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe path in archive: %q", name)
	}
	return clean, nil
}

func writeEntry(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	mode := fs.FileMode(0o644)
	if f.Mode()&0o111 != 0 {
		mode = 0o755
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// keptDirs are never pruned, so a profile always has the basic BepInEx layout.
var keptDirs = []string{"BepInEx", "BepInEx/plugins", "BepInEx/patchers", "BepInEx/config"}

// Remove deletes installed files and prunes directories left empty.
func Remove(profileDir string, files []string) error {
	var errs []error
	var removed []string
	for _, rel := range files {
		clean, err := cleanEntryName(rel)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.Remove(filepath.Join(profileDir, filepath.FromSlash(clean))); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
		removed = append(removed, clean)
	}
	pruneDirs(profileDir, removed)
	return errors.Join(errs...)
}

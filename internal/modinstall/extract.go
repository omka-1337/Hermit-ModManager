// Package modinstall installs and removes Thunderstore packages in a profile.
package modinstall

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// BepInEx folders a package can target. Files in tracked folders go into a
// per-mod subdirectory (e.g. BepInEx/plugins/<mod-id>/) so they can be removed
// cleanly; config is shared and never removed on uninstall.
var trackedRoutes = []string{"plugins", "patchers", "core", "monomod"}

const configRoute = "config"

var metadataFiles = []string{"manifest.json", "icon.png", "readme.md", "changelog.md", "license", "license.md", "license.txt"}

// Extract unpacks a package archive into profileDir and returns the installed
// files as slash-separated paths relative to profileDir. Config files are
// written only if absent and are not returned, so user edits survive
// reinstalls and uninstalls.
//
// A BepInEx loader pack (an archive with BepInEx/core inside, optionally under
// one wrapper folder such as "BepInExPack/") is unpacked as-is into the profile
// root, since it also carries the doorstop files that belong next to the game
// executable.
func Extract(zipPath, profileDir, modID string) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	var files []*zip.File
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			files = append(files, f)
		}
	}
	loaderRoot, isLoader := findLoaderRoot(files)

	var installed []string
	cleanup := func() {
		_ = Remove(profileDir, installed)
	}
	for _, f := range files {
		name, err := cleanEntryName(f.Name)
		if err != nil {
			cleanup()
			return nil, err
		}
		var dest string
		var isConfig bool
		if isLoader {
			rel, ok := strings.CutPrefix(name, loaderRoot)
			if !ok {
				continue // manifest, icon, readme next to the wrapper folder
			}
			dest = rel
			isConfig = strings.HasPrefix(strings.ToLower(rel), "bepinex/config/")
		} else {
			dest, isConfig = modDestination(name, modID)
		}
		if dest == "" {
			continue
		}

		target := filepath.Join(profileDir, filepath.FromSlash(dest))
		if isConfig {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		if err := writeEntry(f, target); err != nil {
			cleanup()
			return nil, fmt.Errorf("extract %s: %w", name, err)
		}
		if !isConfig && !slices.Contains(installed, dest) {
			installed = append(installed, dest)
		}
	}
	slices.Sort(installed)
	return installed, nil
}

// modDestination maps an archive path of a regular mod to its place in the profile.
func modDestination(name, modID string) (dest string, isConfig bool) {
	lower := strings.ToLower(name)
	if slices.Contains(metadataFiles, lower) {
		return "", false
	}
	if strings.HasPrefix(lower, "bepinex/") {
		name, lower = name[len("bepinex/"):], lower[len("bepinex/"):]
	}
	first, rest, nested := strings.Cut(name, "/")
	if nested {
		route := strings.ToLower(first)
		if route == configRoute {
			return path.Join("BepInEx", configRoute, rest), true
		}
		if slices.Contains(trackedRoutes, route) {
			return path.Join("BepInEx", route, modID, rest), false
		}
	}
	if strings.HasSuffix(lower, ".mm.dll") {
		return path.Join("BepInEx", "monomod", modID, name), false
	}
	return path.Join("BepInEx", "plugins", modID, name), false
}

// findLoaderRoot detects a BepInEx loader pack and returns the archive prefix
// that maps to the profile root ("" or "<Folder>/").
func findLoaderRoot(files []*zip.File) (string, bool) {
	for _, f := range files {
		name := strings.ReplaceAll(f.Name, `\`, "/")
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

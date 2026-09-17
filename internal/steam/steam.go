// Package steam locates Steam installations, their libraries and installed apps.
package steam

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type App struct {
	AppID string `json:"appId"`
	Name  string `json:"name"`
	// InstallPath is the absolute game directory (steamapps/common/<installdir>).
	InstallPath string `json:"installPath"`
	// LibraryPath is the Steam library the app is installed in.
	LibraryPath string `json:"libraryPath"`
}

// CompatDataPath is where Proton keeps the app's prefix (compatdata/<appid>).
func (a App) CompatDataPath() string {
	return filepath.Join(a.LibraryPath, "steamapps", "compatdata", a.AppID)
}

// DefaultRoots returns existing Steam root directories for the current user:
// native, Flatpak and Snap installations, with symlinked duplicates removed.
func DefaultRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".local/share/Steam"),
		filepath.Join(home, ".steam/steam"),
		filepath.Join(home, ".steam/root"),
		filepath.Join(home, ".var/app/com.valvesoftware.Steam/.local/share/Steam"),
		filepath.Join(home, "snap/steam/common/.local/share/Steam"),
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		candidates = append([]string{filepath.Join(xdg, "Steam")}, candidates...)
	}
	var roots []string
	for _, c := range candidates {
		if !isDir(filepath.Join(c, "steamapps")) {
			continue
		}
		roots = appendUnique(roots, canonical(c))
	}
	return roots
}

// Libraries returns all library folders registered in the given Steam roots.
// Each root is itself a library even if libraryfolders.vdf is missing.
func Libraries(roots []string) []string {
	var libs []string
	for _, root := range roots {
		if isDir(filepath.Join(root, "steamapps")) {
			libs = appendUnique(libs, canonical(root))
		}
		data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		kv, err := ParseVDF(string(data))
		if err != nil {
			continue
		}
		for _, entry := range kv.Get("libraryfolders").Children {
			// New format: "0" { "path" "..." }; old format: "1" "/path".
			path := entry.Value.String("path")
			if path == "" && isNumeric(entry.Key) {
				path = entry.Value.Value
			}
			if path != "" && isDir(filepath.Join(path, "steamapps")) {
				libs = appendUnique(libs, canonical(path))
			}
		}
	}
	return libs
}

// InstalledApps lists apps installed in all libraries of the given roots.
// Apps whose install directory is missing are skipped.
func InstalledApps(roots []string) []App {
	var apps []App
	for _, lib := range Libraries(roots) {
		manifests, _ := filepath.Glob(filepath.Join(lib, "steamapps", "appmanifest_*.acf"))
		for _, m := range manifests {
			app, ok := readManifest(lib, m)
			if ok && !slices.ContainsFunc(apps, func(a App) bool { return a.AppID == app.AppID }) {
				apps = append(apps, app)
			}
		}
	}
	return apps
}

// FindAppByPath returns the installed app whose install directory is path.
func FindAppByPath(roots []string, path string) (App, bool) {
	path = canonical(path)
	for _, app := range InstalledApps(roots) {
		if app.InstallPath == path {
			return app, true
		}
	}
	return App{}, false
}

func readManifest(lib, path string) (App, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return App{}, false
	}
	kv, err := ParseVDF(string(data))
	if err != nil {
		return App{}, false
	}
	state := kv.Get("AppState")
	app := App{
		AppID:       state.String("appid"),
		Name:        state.String("name"),
		LibraryPath: lib,
	}
	installDir := state.String("installdir")
	if app.AppID == "" || installDir == "" {
		return App{}, false
	}
	app.InstallPath = canonical(filepath.Join(lib, "steamapps", "common", installDir))
	if !isDir(app.InstallPath) {
		return App{}, false
	}
	if app.Name == "" {
		app.Name = installDir
	}
	return app, true
}

// canonical resolves symlinks so the same directory reached via ~/.steam/steam
// and ~/.local/share/Steam compares equal.
func canonical(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func appendUnique(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func isNumeric(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

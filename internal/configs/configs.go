// Package configs lists, parses and edits mod config files of a profile.
//
// BepInEx .cfg files are INI-like; BepInEx writes each entry as
//
//	## Description (possibly several lines)
//	# Setting type: Int32
//	# Default value: 32
//	# Acceptable value range: From 4 to 50
//	Player Count = 32
//
// Saving only rewrites the value part of changed lines, so comments, headers
// and formatting stay exactly as they were.
package configs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Extensions are the config files r2modman shows in its editor.
var Extensions = []string{".cfg", ".txt", ".json", ".yml", ".yaml", ".ini"}

var ErrNotConfig = errors.New("not a config file of this profile")

type FileInfo struct {
	// Path is slash-separated, relative to the profile.
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	// Plugin is the plugin name from the header BepInEx writes, if any.
	Plugin string `json:"plugin"`
}

// skippedDirs hold manager data, disabled mods and caches, not configs.
var skippedDirs = []string{"disabled", "_state", "dotnet", "BepInEx/cache"}

// List returns the config files of a profile, sorted by path.
func List(profileDir string) ([]FileInfo, error) {
	files := []FileInfo{}
	err := filepath.WalkDir(profileDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(profileDir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if slices.Contains(skippedDirs, rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isConfigPath(rel) || !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		info := FileInfo{Path: rel, Size: fi.Size(), ModTime: fi.ModTime()}
		if strings.HasSuffix(strings.ToLower(rel), ".cfg") {
			info.Plugin = pluginName(p)
		}
		files = append(files, info)
		return nil
	})
	return files, err
}

func isConfigPath(rel string) bool {
	lower := strings.ToLower(rel)
	if rel == "profile.json" || path.Base(lower) == "manifest.json" && strings.HasPrefix(rel, "BepInEx/plugins/") {
		return false
	}
	return slices.ContainsFunc(Extensions, func(ext string) bool { return strings.HasSuffix(lower, ext) })
}

var createdBy = regexp.MustCompile(`^##\s*Settings file was created by plugin (.+?)(?: v\S+)?\s*$`)

func pluginName(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		if m := createdBy.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return m[1]
		}
	}
	return ""
}

// Resolve returns the absolute path of a config file inside profileDir,
// rejecting anything outside it or not listed by List.
func Resolve(profileDir, rel string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(rel, `\`, "/"))[1:]
	if clean == "" || clean != rel || !isConfigPath(clean) {
		return "", ErrNotConfig
	}
	top, _, _ := strings.Cut(clean, "/")
	if slices.ContainsFunc(skippedDirs, func(d string) bool { return clean == d || strings.HasPrefix(clean, d+"/") || top == d }) {
		return "", ErrNotConfig
	}
	full := filepath.Join(profileDir, filepath.FromSlash(clean))
	// Symlinks could point outside the profile.
	if resolved, err := filepath.EvalSymlinks(full); err == nil {
		base, _ := filepath.EvalSymlinks(profileDir)
		if !strings.HasPrefix(resolved, base+string(filepath.Separator)) {
			return "", ErrNotConfig
		}
	}
	return full, nil
}

// ReadText returns the raw contents of a config file.
func ReadText(profileDir, rel string) (string, error) {
	full, err := Resolve(profileDir, rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	if len(data) > 5<<20 {
		return "", fmt.Errorf("%s is too large to edit", rel)
	}
	return string(data), nil
}

// WriteText replaces a config file's contents atomically.
func WriteText(profileDir, rel, text string) error {
	full, err := Resolve(profileDir, rel)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(full); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), mode); err != nil {
		return err
	}
	return os.Rename(tmp, full)
}

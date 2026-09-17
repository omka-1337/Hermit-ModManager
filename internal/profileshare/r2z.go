// Package profileshare exports and imports profiles in r2modman's format, so
// profiles can be shared with r2modman users in both directions.
//
// An .r2z file is a zip with:
//
//	export.r2x   YAML: profile name and mods ("<author>-<name>", version, enabled)
//	config/...   contents of BepInEx/config
//	<path>       other text config files from the profile (see configExtensions)
package profileshare

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
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"hermit/internal/library"
	"hermit/internal/thunderstore"
)

const manifestName = "export.r2x"

// configExtensions are the files r2modman includes in exports, and the only
// ones accepted on import: profiles come from strangers, so nothing
// executable may be written into a profile.
var configExtensions = []string{".cfg", ".txt", ".json", ".yml", ".yaml", ".ini"}

type Version struct {
	Major int `yaml:"major" json:"major"`
	Minor int `yaml:"minor" json:"minor"`
	Patch int `yaml:"patch" json:"patch"`
}

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

func parseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q", s)
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q", s)
		}
		nums[i] = n
	}
	return Version{nums[0], nums[1], nums[2]}, nil
}

type ExportMod struct {
	// Name is "<author>-<name>".
	Name    string  `yaml:"name" json:"name"`
	Version Version `yaml:"version" json:"version"`
	Enabled bool    `yaml:"enabled" json:"enabled"`
}

// Ref returns the package reference of the mod.
func (m ExportMod) Ref() (thunderstore.PackageRef, error) {
	return thunderstore.ParseDependency(m.Name + "-" + m.Version.String())
}

type ExportFormat struct {
	ProfileName string      `yaml:"profileName" json:"profileName"`
	Mods        []ExportMod `yaml:"mods" json:"mods"`
}

// WriteR2Z writes a profile as an .r2z archive. Only Thunderstore mods are
// listed, since other sources cannot be installed from a name and version.
func WriteR2Z(w io.Writer, profile library.Profile, profileDir string) error {
	export := ExportFormat{ProfileName: profile.Name, Mods: []ExportMod{}}
	for _, m := range profile.Mods {
		if m.Source.Type != library.SourceThunderstore {
			continue
		}
		v, err := parseVersion(m.Version)
		if err != nil {
			return fmt.Errorf("%s: %w", m.ID, err)
		}
		export.Mods = append(export.Mods, ExportMod{Name: m.ID, Version: v, Enabled: m.Enabled})
	}
	manifest, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	if err := addBytes(zw, manifestName, manifest); err != nil {
		return err
	}
	err = filepath.WalkDir(profileDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(profileDir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if slices.Contains([]string{"disabled", "_state", "dotnet", "BepInEx/cache"}, rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if name, ok := strings.CutPrefix(rel, "BepInEx/config/"); ok {
			return addFile(zw, "config/"+name, p)
		}
		if !hasConfigExtension(rel) || rel == "profile.json" || isPluginManifest(rel) {
			return nil
		}
		return addFile(zw, rel, p)
	})
	if err != nil {
		return err
	}
	return zw.Close()
}

// isPluginManifest matches BepInEx/plugins/<mod>/manifest.json, which r2modman leaves out.
func isPluginManifest(rel string) bool {
	parts := strings.Split(rel, "/")
	return len(parts) == 4 && parts[0] == "BepInEx" && parts[1] == "plugins" && parts[3] == "manifest.json"
}

func hasConfigExtension(name string) bool {
	lower := strings.ToLower(name)
	return slices.ContainsFunc(configExtensions, func(ext string) bool { return strings.HasSuffix(lower, ext) })
}

func addBytes(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func addFile(zw *zip.Writer, name, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// ReadExport reads export.r2x from an .r2z archive.
func ReadExport(zipPath string) (ExportFormat, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return ExportFormat{}, fmt.Errorf("not a profile archive: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !strings.EqualFold(f.Name, manifestName) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ExportFormat{}, err
		}
		defer rc.Close()
		return parseExport(rc)
	}
	return ExportFormat{}, errors.New("not a profile archive: export.r2x is missing")
}

func parseExport(r io.Reader) (ExportFormat, error) {
	var raw struct {
		ProfileName *string `yaml:"profileName"`
		Mods        *[]struct {
			Name    string  `yaml:"name"`
			Version Version `yaml:"version"`
			// r2modman treats a missing flag as enabled.
			Enabled *bool `yaml:"enabled"`
		} `yaml:"mods"`
	}
	if err := yaml.NewDecoder(io.LimitReader(r, 10<<20)).Decode(&raw); err != nil {
		return ExportFormat{}, fmt.Errorf("invalid export.r2x: %w", err)
	}
	if raw.ProfileName == nil || raw.Mods == nil {
		return ExportFormat{}, errors.New("invalid export.r2x: profileName or mods missing")
	}
	export := ExportFormat{ProfileName: *raw.ProfileName, Mods: []ExportMod{}}
	for _, m := range *raw.Mods {
		export.Mods = append(export.Mods, ExportMod{Name: m.Name, Version: m.Version, Enabled: m.Enabled == nil || *m.Enabled})
	}
	return export, nil
}

// ExtractConfigs writes the config files of an .r2z archive into a profile,
// overwriting existing ones. Anything that is not a text config file, or
// that would land outside the profile or in manager-owned files, is skipped.
func ExtractConfigs(zipPath, profileDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || strings.EqualFold(f.Name, manifestName) || strings.EqualFold(f.Name, "mods.yml") {
			continue
		}
		dest, ok := configDestination(f.Name)
		if !ok {
			continue
		}
		if err := writeZipFile(f, filepath.Join(profileDir, filepath.FromSlash(dest))); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}
	return nil
}

func configDestination(name string) (string, bool) {
	name = strings.ReplaceAll(name, `\`, "/")
	clean := path.Clean("/" + name)[1:]
	if clean == "" || clean != strings.TrimPrefix(name, "./") || !hasConfigExtension(clean) {
		return "", false
	}
	if rest, ok := strings.CutPrefix(clean, "config/"); ok {
		clean = "BepInEx/config/" + rest
	}
	top, _, _ := strings.Cut(clean, "/")
	if clean == "profile.json" || slices.Contains([]string{"disabled", "_state"}, top) {
		return "", false
	}
	return clean, true
}

func writeZipFile(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(rc, 100<<20)); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

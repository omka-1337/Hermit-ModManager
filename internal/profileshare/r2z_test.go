package profileshare

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"bepinexmodmanager/internal/library"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	return names
}

func TestExportMatchesR2modmanLayout(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "profile")
	for _, f := range []string{
		"profile.json", "doorstop_config.ini", "winhttp.dll",
		"BepInEx/config/BepInEx.cfg", "BepInEx/config/Mod/settings.dat",
		"BepInEx/plugins/A-Mod/Mod.dll", "BepInEx/plugins/A-Mod/manifest.json", "BepInEx/plugins/A-Mod/data/items.json",
		"BepInEx/cache/chainloader.json", "BepInEx/LogOutput.log",
		"disabled/B-Off/BepInEx/plugins/B-Off/off.cfg",
	} {
		write(t, filepath.Join(profileDir, f), "x")
	}
	profile := library.Profile{Name: "My Profile", Mods: []library.Mod{
		{ID: "BepInEx-BepInExPack", Version: "5.4.2100", Enabled: true, Source: library.ModSource{Type: library.SourceThunderstore}},
		{ID: "B-Off", Version: "1.0.0", Enabled: false, Source: library.ModSource{Type: library.SourceThunderstore}},
		{ID: "local-Thing", Version: "1.0.0", Enabled: true, Source: library.ModSource{Type: library.SourceLocal}},
	}}

	archive := filepath.Join(dir, "out.r2z")
	var buf bytes.Buffer
	if err := WriteR2Z(&buf, profile, profileDir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(archive, buf.Bytes(), 0o644)

	want := []string{"BepInEx/plugins/A-Mod/data/items.json", "config/BepInEx.cfg", "config/Mod/settings.dat", "doorstop_config.ini", "export.r2x"}
	if got := zipNames(t, archive); !slices.Equal(got, want) {
		t.Errorf("archive:\n got  %v\n want %v", got, want)
	}

	export, err := ReadExport(archive)
	if err != nil {
		t.Fatal(err)
	}
	if export.ProfileName != "My Profile" || len(export.Mods) != 2 || export.Mods[0].Version != (Version{5, 4, 2100}) || export.Mods[1].Enabled {
		t.Errorf("export: %+v", export)
	}
}

// r2modmanExport is export.r2x as written by r2modman (js-yaml).
const r2modmanExport = `profileName: Friends
mods:
  - name: BepInEx-BepInExPack
    version:
      major: 5
      minor: 4
      patch: 2100
    enabled: true
  - name: notnotnotswipez-MoreCompany
    version:
      major: 1
      minor: 14
      patch: 0
    enabled: false
  - name: x753-More_Suits
    version:
      major: 1
      minor: 5
      patch: 2
`

func TestReadR2modmanExport(t *testing.T) {
	export, err := parseExport(strings.NewReader(r2modmanExport))
	if err != nil {
		t.Fatal(err)
	}
	if export.ProfileName != "Friends" || len(export.Mods) != 3 {
		t.Fatalf("export: %+v", export)
	}
	ref, err := export.Mods[1].Ref()
	if err != nil || ref.String() != "notnotnotswipez-MoreCompany-1.14.0" || export.Mods[1].Enabled {
		t.Errorf("mod: %+v %v", export.Mods[1], err)
	}
	if !export.Mods[2].Enabled {
		t.Error("missing enabled flag means enabled")
	}
	if _, err := parseExport(strings.NewReader("foo: bar\n")); err == nil {
		t.Error("expected error for yaml without profile fields")
	}
}

func TestExtractConfigsIsSafe(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "in.r2z")
	f, _ := os.Create(archive)
	zw := zip.NewWriter(f)
	for name, content := range map[string]string{
		"export.r2x":                      r2modmanExport,
		"config/BepInEx.cfg":              "imported",
		"BepInEx/plugins/A-Mod/data.json": "{}",
		"BepInEx/plugins/A-Mod/Evil.dll":  "MZ",
		"run_bepinex.sh":                  "#!/bin/sh",
		"../escape.cfg":                   "x",
		"profile.json":                    "{}",
		"disabled/A-Mod/x.cfg":            "x",
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()
	f.Close()

	profileDir := filepath.Join(dir, "profile")
	write(t, filepath.Join(profileDir, "BepInEx/config/BepInEx.cfg"), "default")
	write(t, filepath.Join(profileDir, "profile.json"), "mine")
	if err := ExtractConfigs(archive, profileDir); err != nil {
		t.Fatal(err)
	}

	if data, _ := os.ReadFile(filepath.Join(profileDir, "BepInEx/config/BepInEx.cfg")); string(data) != "imported" {
		t.Errorf("config not overwritten: %q", data)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "BepInEx/plugins/A-Mod/data.json")); err != nil {
		t.Error("plugin config missing")
	}
	for _, bad := range []string{"BepInEx/plugins/A-Mod/Evil.dll", "run_bepinex.sh", "disabled", "../escape.cfg"} {
		if _, err := os.Stat(filepath.Join(profileDir, bad)); err == nil {
			t.Errorf("%s must not be extracted", bad)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(profileDir, "profile.json")); string(data) != "mine" {
		t.Error("profile.json overwritten")
	}
}

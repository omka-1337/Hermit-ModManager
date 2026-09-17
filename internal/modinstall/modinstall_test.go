package modinstall

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/thunderstore"
)

func makeZip(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	f, err := os.CreateTemp(dir, "*.zip")
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExtractLayouts(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			"bepinex folder",
			map[string]string{"BepInEx/plugins/MoreCompany.dll": "", "icon.png": "", "manifest.json": "{}", "README.md": ""},
			[]string{"BepInEx/plugins/A-Mod/MoreCompany.dll"},
		},
		{
			"route at root",
			map[string]string{"plugins/LethalLib/LethalLib.dll": "", "CHANGELOG.md": "", "LICENSE": ""},
			[]string{"BepInEx/plugins/A-Mod/LethalLib/LethalLib.dll"},
		},
		{
			"loose files go to plugins, monomod by extension",
			map[string]string{"Mod.dll": "", "assets/x.bundle": "", "Assembly-CSharp.Mod.mm.dll": ""},
			[]string{"BepInEx/monomod/A-Mod/Assembly-CSharp.Mod.mm.dll", "BepInEx/plugins/A-Mod/Mod.dll", "BepInEx/plugins/A-Mod/assets/x.bundle"},
		},
		{
			"patchers and case-insensitive routes",
			map[string]string{"BepInEx/Patchers/P.dll": "", "Plugins/Q.dll": ""},
			[]string{"BepInEx/patchers/A-Mod/P.dll", "BepInEx/plugins/A-Mod/Q.dll"},
		},
		{
			"loader pack with wrapper folder",
			map[string]string{
				"BepInExPack/BepInEx/core/BepInEx.dll":   "",
				"BepInExPack/BepInEx/config/BepInEx.cfg": "",
				"BepInExPack/winhttp.dll":                "",
				"BepInExPack/doorstop_config.ini":        "",
				"manifest.json":                          "{}",
			},
			[]string{"BepInEx/core/BepInEx.dll", "doorstop_config.ini", "winhttp.dll"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			profile := filepath.Join(tmp, "profile")
			got, err := Extract(makeZip(t, tmp, tc.files), profile, "A-Mod")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("files:\n got  %v\n want %v", got, tc.want)
			}
			for _, f := range got {
				if _, err := os.Stat(filepath.Join(profile, f)); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestExtractKeepsExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "profile")
	cfg := filepath.Join(profile, "BepInEx", "config", "mod.cfg")
	os.MkdirAll(filepath.Dir(cfg), 0o755)
	os.WriteFile(cfg, []byte("user"), 0o644)

	archive := makeZip(t, tmp, map[string]string{"BepInEx/config/mod.cfg": "default", "BepInEx/config/new.cfg": "new", "Mod.dll": ""})
	files, err := Extract(archive, profile, "A-Mod")
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, cfg) != "user" || readFile(t, filepath.Join(profile, "BepInEx/config/new.cfg")) != "new" {
		t.Error("config handling")
	}
	if len(files) != 1 {
		t.Errorf("config must not be tracked: %v", files)
	}
}

func TestExtractRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../evil.dll", "plugins/../../evil.dll", "/abs.dll"} {
		tmp := t.TempDir()
		profile := filepath.Join(tmp, "profile")
		archive := makeZip(t, tmp, map[string]string{"ok.dll": "", name: ""})
		if _, err := Extract(archive, profile, "A-Mod"); err == nil {
			t.Errorf("%q: expected error", name)
		}
		if _, err := os.Stat(filepath.Join(tmp, "evil.dll")); err == nil {
			t.Errorf("%q escaped the profile", name)
		}
	}
}

func TestRemovePrunesEmptyDirs(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "profile")
	files, err := Extract(makeZip(t, tmp, map[string]string{"plugins/Deep/Dir/a.dll": ""}), profile, "A-Mod")
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(profile, files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(profile, "BepInEx/plugins/A-Mod")); !os.IsNotExist(err) {
		t.Error("mod dir not pruned")
	}
	if _, err := os.Stat(filepath.Join(profile, "BepInEx/plugins")); err != nil {
		t.Error("plugins dir must be kept")
	}
}

// fakeDownloader serves generated packages keyed by "<ns>-<name>-<version>".
type fakeDownloader struct {
	t        *testing.T
	dir      string
	packages map[string][]string // package -> dependencies
	calls    []string
}

func (f *fakeDownloader) DownloadPackage(_ context.Context, ref thunderstore.PackageRef, _ func(int64, int64)) (thunderstore.Archive, error) {
	f.calls = append(f.calls, ref.String())
	deps, ok := f.packages[ref.String()]
	if !ok {
		return thunderstore.Archive{}, thunderstore.ErrNotFound
	}
	manifest, _ := json.Marshal(map[string]any{"name": ref.Name, "version_number": ref.Version, "dependencies": deps})
	files := map[string]string{"manifest.json": "\xef\xbb\xbf" + string(manifest)}
	if strings.HasPrefix(ref.Name, "BepInExPack") {
		files["BepInExPack/BepInEx/core/BepInEx.dll"] = ref.Version
		files["BepInExPack/winhttp.dll"] = ""
	} else {
		files["BepInEx/plugins/"+ref.Name+".dll"] = ref.Version
	}
	return thunderstore.Archive{Path: makeZip(f.t, f.dir, files), SHA256: "sha-" + ref.String(), URL: "https://example/" + ref.String()}, nil
}

func TestInstallerResolvesDependencies(t *testing.T) {
	tmp := t.TempDir()
	lib, err := library.New(filepath.Join(tmp, "lib"), nil)
	if err != nil {
		t.Fatal(err)
	}
	gamePath := filepath.Join(tmp, "game")
	os.MkdirAll(gamePath, 0o755)
	game, err := lib.AddGame("Game", gamePath)
	if err != nil {
		t.Fatal(err)
	}
	dl := &fakeDownloader{t: t, dir: tmp, packages: map[string][]string{
		"BepInEx-BepInExPack-5.4.2100": {},
		"BepInEx-BepInExPack-5.4.2200": {},
		"Evaisa-LethalLib-1.1.1":       {"BepInEx-BepInExPack-5.4.2100"},
		"Evaisa-LethalLib-1.0.0":       {"BepInEx-BepInExPack-5.4.2100"},
		"x753-More_Suits-1.5.2":        {"BepInEx-BepInExPack-5.4.2200", "Evaisa-LethalLib-1.0.0"},
	}}
	in := NewInstaller(lib, dl)
	ctx := context.Background()
	pid := game.ActiveProfile

	var progress []Progress
	profile, err := in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "Evaisa", Name: "LethalLib", Version: "1.1.1"}, func(p Progress) {
		progress = append(progress, p)
	})
	if err != nil {
		t.Fatal(err)
	}
	if ids := modVersions(profile); !slices.Equal(ids, []string{"BepInEx-BepInExPack@5.4.2100", "Evaisa-LethalLib@1.1.1"}) {
		t.Fatalf("mods: %v", ids)
	}
	if len(progress) == 0 || progress[0].Target != "Evaisa-LethalLib" {
		t.Errorf("progress: %+v", progress)
	}
	dir, _ := lib.ProfileDir(game.ID, pid)
	if readFile(t, filepath.Join(dir, "BepInEx/core/BepInEx.dll")) != "5.4.2100" {
		t.Error("loader pack not installed at profile root")
	}
	if m := profile.Mods[1]; m.Source.SHA256 != "sha-Evaisa-LethalLib-1.1.1" || m.Author != "Evaisa" || len(m.Dependencies) != 1 {
		t.Errorf("mod entry: %+v", m)
	}

	// More_Suits needs a newer pack (upgrade) and an older LethalLib (kept).
	profile, err = in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "x753", Name: "More_Suits", Version: "1.5.2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"BepInEx-BepInExPack@5.4.2200", "Evaisa-LethalLib@1.1.1", "x753-More_Suits@1.5.2"}
	if ids := modVersions(profile); !slices.Equal(ids, want) {
		t.Fatalf("mods: %v", ids)
	}
	if slices.Contains(dl.calls, "Evaisa-LethalLib-1.0.0") {
		t.Error("older dependency must not be downloaded when a newer one is installed")
	}
	if readFile(t, filepath.Join(dir, "BepInEx/core/BepInEx.dll")) != "5.4.2200" {
		t.Error("loader pack not upgraded")
	}

	profile, err = in.Uninstall(game.ID, pid, "x753-More_Suits")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Mods) != 2 {
		t.Errorf("after uninstall: %v", modVersions(profile))
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx/plugins/x753-More_Suits")); !os.IsNotExist(err) {
		t.Error("uninstalled files remain")
	}

	if _, err := in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "No", Name: "Such", Version: "1.0.0"}, nil); err == nil {
		t.Error("expected error for missing package")
	}
}

func modVersions(p library.Profile) []string {
	var out []string
	for _, m := range p.Mods {
		out = append(out, m.ID+"@"+m.Version)
	}
	slices.Sort(out)
	return out
}

func TestCompareVersions(t *testing.T) {
	if CompareVersions("5.4.2100", "5.4.21") <= 0 || CompareVersions("1.0.0", "1.0.0") != 0 || CompareVersions("1.2.0", "1.10.0") >= 0 {
		t.Error("CompareVersions")
	}
}

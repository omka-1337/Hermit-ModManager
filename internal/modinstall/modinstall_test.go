package modinstall

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
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
	sideloader := DefaultRules()
	sideloader.Routes = append(sideloader.Routes,
		thunderstore.InstallRule{Route: "BepInEx/Sideloader", TrackingMethod: thunderstore.TrackingState, DefaultFileExtensions: []string{".hotmod"}},
		thunderstore.InstallRule{Route: "QMods", TrackingMethod: thunderstore.TrackingState},
	)
	sideloader.RelativeFileExclusions = []string{"manifest.json", "icon.png", "README.md"}
	fork := DefaultRules()
	fork.LoaderPackages["denikson-bepinexpack_valheim"] = "BepInExPack_Valheim"

	cases := []struct {
		name  string
		rules Rules
		modID string
		files map[string]string
		want  []string
	}{
		{
			"bepinex folder, metadata goes to the default route like in r2modman",
			DefaultRules(), "A-Mod",
			map[string]string{"BepInEx/plugins/MoreCompany.dll": "", "icon.png": "", "manifest.json": "{}", "README.md": ""},
			[]string{"BepInEx/plugins/A-Mod/MoreCompany.dll", "BepInEx/plugins/A-Mod/README.md", "BepInEx/plugins/A-Mod/icon.png", "BepInEx/plugins/A-Mod/manifest.json"},
		},
		{
			"route folder at root keeps its structure",
			DefaultRules(), "A-Mod",
			map[string]string{"plugins/LethalLib/LethalLib.dll": ""},
			[]string{"BepInEx/plugins/A-Mod/LethalLib/LethalLib.dll"},
		},
		{
			"route folder nested in another folder",
			DefaultRules(), "A-Mod",
			map[string]string{"Wrapper/patchers/P.dll": "", "Wrapper/Plugins/Q.dll": ""},
			[]string{"BepInEx/patchers/A-Mod/P.dll", "BepInEx/plugins/A-Mod/Q.dll"},
		},
		{
			"loose files are flattened; longest extension wins",
			DefaultRules(), "A-Mod",
			map[string]string{"Mod.dll": "", "assets/x.bundle": "", "Assembly-CSharp.Mod.mm.dll": ""},
			[]string{"BepInEx/monomod/A-Mod/Assembly-CSharp.Mod.mm.dll", "BepInEx/plugins/A-Mod/Mod.dll", "BepInEx/plugins/A-Mod/x.bundle"},
		},
		{
			"state routes install without a mod folder and honour exclusions",
			sideloader, "A-Mod",
			map[string]string{"Cool.hotmod": "", "QMods/Thing/mod.json": "", "manifest.json": "{}", "BepInEx/plugins/P.dll": ""},
			[]string{"BepInEx/Sideloader/Cool.hotmod", "BepInEx/plugins/A-Mod/P.dll", "BepInEx/plugins/A-Mod/manifest.json", "QMods/Thing/mod.json"},
		},
		{
			"loader pack by heuristic",
			DefaultRules(), "BepInEx-BepInExPack",
			map[string]string{
				"BepInExPack/BepInEx/core/BepInEx.dll":   "",
				"BepInExPack/BepInEx/config/BepInEx.cfg": "",
				"BepInExPack/winhttp.dll":                "",
				"BepInExPack/doorstop_config.ini":        "",
				"manifest.json":                          "{}",
			},
			[]string{"BepInEx/core/BepInEx.dll", "doorstop_config.ini", "winhttp.dll"},
		},
		{
			"loader pack from schema with a custom root folder",
			fork, "denikson-BepInExPack_Valheim",
			map[string]string{"BepInExPack_Valheim/BepInEx/core/BepInEx.dll": "", "BepInExPack_Valheim/unstripped_corlib/mscorlib.dll": "", "icon.png": ""},
			[]string{"BepInEx/core/BepInEx.dll", "unstripped_corlib/mscorlib.dll"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			profile := filepath.Join(tmp, "profile")
			archive := makeZip(t, tmp, tc.files)
			planned, err := PlanFiles(archive, tc.modID, tc.rules)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Extract(archive, profile, tc.modID, tc.rules)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) || !slices.Equal(planned, tc.want) {
				t.Fatalf("files:\n got  %v\n plan %v\n want %v", got, planned, tc.want)
			}
			for _, f := range got {
				if _, err := os.Stat(filepath.Join(profile, f)); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestRulesFromSchema(t *testing.T) {
	schema := &thunderstore.Ecosystem{
		Games: map[string]thunderstore.EcosystemGame{
			"valheim": {
				Thunderstore: &struct {
					DisplayName string `json:"displayName"`
				}{"Valheim"},
				R2modman: []thunderstore.GameSettings{{
					PackageLoader: "bepinex",
					Distributions: []thunderstore.Distribution{{Platform: "steam", Identifier: "892970"}},
					InstallRules:  []thunderstore.InstallRule{{Route: "BepInEx/plugins", TrackingMethod: thunderstore.TrackingSubdir, IsDefaultLocation: true}},
				}},
			},
		},
		ModloaderPackages: []thunderstore.ModloaderPackage{
			{PackageID: "denikson-BepInExPack_Valheim", RootFolder: "BepInExPack_Valheim", Loader: "bepinex"},
			{PackageID: "LavaGang-MelonLoader", Loader: "melonloader"},
		},
	}
	r := RulesFromSchema(schema, "892970", "")
	if len(r.Routes) != 1 || r.LoaderPackages["denikson-bepinexpack_valheim"] != "BepInExPack_Valheim" || len(r.LoaderPackages) != 1 {
		t.Errorf("known game: %+v", r)
	}
	if r := RulesFromSchema(schema, "1", ""); len(r.Routes) != len(DefaultRules().Routes) {
		t.Errorf("unknown game should use default routes: %+v", r.Routes)
	}
	if r := RulesFromSchema(nil, "892970", ""); len(r.Routes) != len(DefaultRules().Routes) {
		t.Error("nil schema should use defaults")
	}
}

func TestExtractKeepsExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "profile")
	cfg := filepath.Join(profile, "BepInEx", "config", "mod.cfg")
	os.MkdirAll(filepath.Dir(cfg), 0o755)
	os.WriteFile(cfg, []byte("user"), 0o644)

	archive := makeZip(t, tmp, map[string]string{"BepInEx/config/mod.cfg": "default", "BepInEx/config/new.cfg": "new", "Mod.dll": ""})
	files, err := Extract(archive, profile, "A-Mod", DefaultRules())
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
		if _, err := Extract(archive, profile, "A-Mod", DefaultRules()); err == nil {
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
	files, err := Extract(makeZip(t, tmp, map[string]string{"plugins/Deep/Dir/a.dll": ""}), profile, "A-Mod", DefaultRules())
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
	in := NewInstaller(lib, dl, nil)
	ctx := context.Background()
	pid := game.ActiveProfile

	var progress []Progress
	profile, err := in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "Evaisa", Name: "LethalLib", Version: "1.1.1"}, false, func(p Progress) {
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
	profile, err = in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "x753", Name: "More_Suits", Version: "1.5.2"}, false, nil)
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

	if _, err := in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "No", Name: "Such", Version: "1.0.0"}, false, nil); err == nil {
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

func findMod(t *testing.T, p library.Profile, id string) library.Mod {
	t.Helper()
	for _, m := range p.Mods {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("mod %s not in profile", id)
	return library.Mod{}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestEnableDisableCascade(t *testing.T) {
	tmp := t.TempDir()
	lib, _ := library.New(filepath.Join(tmp, "lib"), nil)
	gamePath := filepath.Join(tmp, "game")
	os.MkdirAll(gamePath, 0o755)
	game, _ := lib.AddGame("Game", gamePath)
	pid := game.ActiveProfile
	dir, _ := lib.ProfileDir(game.ID, pid)
	dl := &fakeDownloader{t: t, dir: tmp, packages: map[string][]string{
		"BepInEx-BepInExPack-5.4.2100": {},
		"Evaisa-LethalLib-1.1.1":       {"BepInEx-BepInExPack-5.4.2100"},
		"Yuppie-YuppieMod-1.0.0":       {"Evaisa-LethalLib-1.1.1"},
	}}
	in := NewInstaller(lib, dl, nil)
	ctx := context.Background()

	if _, err := in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "Yuppie", Name: "YuppieMod", Version: "1.0.0"}, false, nil); err != nil {
		t.Fatal(err)
	}
	pack := "BepInEx/core/BepInEx.dll"
	yuppie := "BepInEx/plugins/Yuppie-YuppieMod/YuppieMod.dll"

	// Disabling the loader cascades through LethalLib to YuppieMod.
	p, err := in.SetEnabled(game.ID, pid, "BepInEx-BepInExPack", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"BepInEx-BepInExPack", "Evaisa-LethalLib", "Yuppie-YuppieMod"} {
		if findMod(t, p, id).Active {
			t.Errorf("%s should be inactive", id)
		}
	}
	if y := findMod(t, p, "Yuppie-YuppieMod"); !y.Enabled || len(y.UnmetDependencies) != 1 {
		t.Errorf("YuppieMod keeps user choice and reports unmet deps: %+v", y)
	}
	if exists(filepath.Join(dir, pack)) || !exists(filepath.Join(dir, "disabled/BepInEx-BepInExPack", pack)) {
		t.Error("loader files not moved to disabled/")
	}
	if exists(filepath.Join(dir, yuppie)) || !exists(filepath.Join(dir, "disabled/Yuppie-YuppieMod", yuppie)) {
		t.Error("dependant files not moved to disabled/")
	}

	// A mod with unmet dependencies cannot be enabled.
	if _, err := in.SetEnabled(game.ID, pid, "Yuppie-YuppieMod", true); !errors.Is(err, ErrUnmetDependencies) {
		t.Errorf("enable with unmet deps: %v", err)
	}

	// Re-enabling the loader restores everything.
	p, _ = in.SetEnabled(game.ID, pid, "BepInEx-BepInExPack", true)
	if y := findMod(t, p, "Yuppie-YuppieMod"); !y.Active || len(y.UnmetDependencies) != 0 {
		t.Errorf("YuppieMod not restored: %+v", y)
	}
	if !exists(filepath.Join(dir, yuppie)) || exists(filepath.Join(dir, "disabled")) {
		t.Error("files not moved back or disabled/ not pruned")
	}

	// Uninstalling a dependency deactivates dependants; reinstalling reactivates them.
	p, _ = in.Uninstall(game.ID, pid, "Evaisa-LethalLib")
	if y := findMod(t, p, "Yuppie-YuppieMod"); y.Active || y.UnmetDependencies[0] != "Evaisa-LethalLib-1.1.1" {
		t.Errorf("after uninstalling dependency: %+v", y)
	}
	p, _ = in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "Evaisa", Name: "LethalLib", Version: "1.1.1"}, false, nil)
	if !findMod(t, p, "Yuppie-YuppieMod").Active {
		t.Error("dependant not reactivated after reinstalling dependency")
	}

	// A disabled mod can be uninstalled and updates keep it disabled.
	p, _ = in.SetEnabled(game.ID, pid, "Yuppie-YuppieMod", false)
	p, err = in.Install(ctx, game.ID, pid, thunderstore.PackageRef{Namespace: "Yuppie", Name: "YuppieMod", Version: "1.0.0"}, false, nil)
	if err != nil || findMod(t, p, "Yuppie-YuppieMod").Enabled || exists(filepath.Join(dir, yuppie)) {
		t.Errorf("reinstall of disabled mod: err=%v", err)
	}
	if _, err := in.Uninstall(game.ID, pid, "Yuppie-YuppieMod"); err != nil || exists(filepath.Join(dir, "disabled")) {
		t.Errorf("uninstall disabled mod: %v", err)
	}
}

func TestRefreshFixesStaleState(t *testing.T) {
	tmp := t.TempDir()
	lib, _ := library.New(filepath.Join(tmp, "lib"), nil)
	os.MkdirAll(filepath.Join(tmp, "game"), 0o755)
	game, _ := lib.AddGame("Game", filepath.Join(tmp, "game"))
	dir, _ := lib.ProfileDir(game.ID, game.ActiveProfile)
	dl := &fakeDownloader{t: t, dir: tmp, packages: map[string][]string{"A-Lib-1.0.0": {}, "A-Mod-1.0.0": {"A-Lib-1.0.0"}}}
	in := NewInstaller(lib, dl, nil)
	if _, err := in.Install(context.Background(), game.ID, game.ActiveProfile, thunderstore.PackageRef{Namespace: "A", Name: "Mod", Version: "1.0.0"}, false, nil); err != nil {
		t.Fatal(err)
	}
	// Simulate an old manager version that removed the dependency without syncing.
	lib.UpdateProfile(game.ID, game.ActiveProfile, func(p *library.Profile) error {
		p.Mods = slices.DeleteFunc(p.Mods, func(m library.Mod) bool { return m.ID == "A-Lib" })
		return nil
	})
	p, err := in.Refresh(game.ID, game.ActiveProfile)
	if err != nil || findMod(t, p, "A-Mod").Active || !exists(filepath.Join(dir, "disabled/A-Mod")) {
		t.Errorf("refresh: %+v %v", p.Mods, err)
	}
}

func TestLoaderConflicts(t *testing.T) {
	tmp := t.TempDir()
	lib, _ := library.New(filepath.Join(tmp, "lib"), nil)
	os.MkdirAll(filepath.Join(tmp, "game"), 0o755)
	game, _ := lib.AddGame("Game", filepath.Join(tmp, "game"))
	pid := game.ActiveProfile
	dir, _ := lib.ProfileDir(game.ID, pid)
	dl := &fakeDownloader{t: t, dir: tmp, packages: map[string][]string{
		"BepInEx-BepInExPack-5.4.2100": {},
		"Fork-BepInExPack_Fork-1.0.0":  {},
		"A-UsesPack-1.0.0":             {"BepInEx-BepInExPack-5.4.2100"},
		"B-UsesFork-1.0.0":             {"Fork-BepInExPack_Fork-1.0.0"},
		"C-UsesBoth-1.0.0":             {"BepInEx-BepInExPack-5.4.2100", "Fork-BepInExPack_Fork-1.0.0"},
	}}
	in := NewInstaller(lib, dl, nil)
	ctx := context.Background()
	ref := func(s string) thunderstore.PackageRef {
		r, err := thunderstore.ParseDependency(s)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	if _, err := in.Install(ctx, game.ID, pid, ref("A-UsesPack-1.0.0"), false, nil); err != nil {
		t.Fatal(err)
	}

	plan, err := in.PlanInstall(ctx, game.ID, pid, ref("B-UsesFork-1.0.0"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Packages) != 2 || plan.Packages[0].Action != ActionInstall {
		t.Errorf("plan packages: %+v", plan.Packages)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("plan conflicts: %+v", plan.Conflicts)
	}
	if c := plan.Conflicts[0]; c.ModID != "BepInEx-BepInExPack" || c.Reason != ConflictLoader || !c.Installed || len(c.Files) == 0 {
		t.Errorf("conflict: %+v", c)
	}

	if _, err := in.Install(ctx, game.ID, pid, ref("B-UsesFork-1.0.0"), false, nil); !errors.Is(err, ErrConflicts) {
		t.Fatalf("install without replace: %v", err)
	}
	if p, _ := lib.GetProfile(game.ID, pid); len(p.Mods) != 2 {
		t.Errorf("profile changed by a rejected install: %v", modVersions(p))
	}

	p, err := in.Install(ctx, game.ID, pid, ref("B-UsesFork-1.0.0"), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ids := modVersions(p); !slices.Equal(ids, []string{"A-UsesPack@1.0.0", "B-UsesFork@1.0.0", "Fork-BepInExPack_Fork@1.0.0"}) {
		t.Fatalf("after replace: %v", ids)
	}
	if findMod(t, p, "A-UsesPack").Active {
		t.Error("dependant of the replaced loader must become inactive")
	}
	if readFile(t, filepath.Join(dir, "BepInEx/core/BepInEx.dll")) != "1.0.0" {
		t.Error("fork loader files not in place")
	}

	// Two loaders required by one package can never be installed together.
	_, err = in.Install(ctx, game.ID, pid, ref("C-UsesBoth-1.0.0"), true, nil)
	if err == nil || errors.Is(err, ErrConflicts) {
		t.Errorf("conflicting dependencies: %v", err)
	}
}

func TestRemoveKeepsFilesOfOtherMods(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "p")
	mkShared := filepath.Join(profile, "winhttp.dll")
	os.MkdirAll(profile, 0o755)
	os.WriteFile(mkShared, nil, 0o644)
	a := library.Mod{ID: "a", Active: true, Files: []string{"winhttp.dll"}}
	b := library.Mod{ID: "b", Active: true, Files: []string{"winhttp.dll"}}
	if err := removeModFiles(profile, a, []library.Mod{a, b}); err != nil {
		t.Fatal(err)
	}
	if !exists(mkShared) {
		t.Error("file shared with another mod was removed")
	}
}

package profileshare

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/modinstall"
	"bepinexmodmanager/internal/thunderstore"
)

type fakeRepo struct {
	t        *testing.T
	dir      string
	packages map[string][]string
	codes    map[string][]byte
}

func (f *fakeRepo) Versions(context.Context, string, string) ([]thunderstore.Version, error) {
	return nil, nil
}

func (f *fakeRepo) DownloadPackage(_ context.Context, ref thunderstore.PackageRef, _ func(int64, int64)) (thunderstore.Archive, error) {
	deps, ok := f.packages[ref.String()]
	if !ok {
		return thunderstore.Archive{}, thunderstore.ErrNotFound
	}
	path := filepath.Join(f.dir, ref.String()+".zip")
	out, _ := os.Create(path)
	zw := zip.NewWriter(out)
	manifest, _ := json.Marshal(map[string]any{"dependencies": deps})
	w, _ := zw.Create("manifest.json")
	w.Write(manifest)
	w, _ = zw.Create("BepInEx/plugins/" + ref.Name + ".dll")
	w.Write([]byte(ref.Version))
	w, _ = zw.Create("BepInEx/config/" + ref.Name + ".cfg")
	w.Write([]byte("default"))
	zw.Close()
	out.Close()
	return thunderstore.Archive{Path: path, URL: "https://example/" + ref.String()}, nil
}

func (f *fakeRepo) UploadProfile(_ context.Context, archive []byte) (string, error) {
	code := "11111111-2222-3333-4444-555555555555"
	f.codes[code] = archive
	return code, nil
}

func (f *fakeRepo) DownloadProfile(_ context.Context, code string) ([]byte, error) {
	return f.codes[code], nil
}

func TestExportImportRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	lib, _ := library.New(filepath.Join(tmp, "lib"), nil)
	os.MkdirAll(filepath.Join(tmp, "game"), 0o755)
	game, _ := lib.AddGame("Game", filepath.Join(tmp, "game"))
	repo := &fakeRepo{t: t, dir: tmp, codes: map[string][]byte{}, packages: map[string][]string{
		"A-Lib-1.0.0": {},
		"A-Mod-2.0.0": {"A-Lib-1.0.0"},
		"B-Off-1.0.0": {},
	}}
	in := modinstall.NewInstaller(lib, repo, nil)
	sharer := NewSharer(lib, in, repo, filepath.Join(tmp, "imports"))
	ctx := context.Background()
	src := game.ActiveProfile
	for _, pkg := range []string{"A-Mod-2.0.0", "B-Off-1.0.0"} {
		ref, _ := thunderstore.ParseDependency(pkg)
		if _, err := in.Install(ctx, game.ID, src, ref, modinstall.Options{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	in.SetEnabled(game.ID, src, "B-Off", false)
	srcDir, _ := lib.ProfileDir(game.ID, src)
	os.WriteFile(filepath.Join(srcDir, "BepInEx/config/Mod.cfg"), []byte("tuned"), 0o644)

	code, err := sharer.ExportCode(ctx, game.ID, src)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := sharer.Preview(ctx, Source{Code: code})
	if err != nil || preview.ProfileName != "Default" || len(preview.Mods) != 3 {
		t.Fatalf("preview: %+v %v", preview, err)
	}

	var progress []ImportProgress
	result, err := sharer.Import(ctx, game.ID, preview.Archive, "Imported", func(p ImportProgress) { progress = append(progress, p) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := result.Profile
	if p.Name != "Imported" || len(p.Mods) != 3 || len(result.Failed) != 0 {
		t.Fatalf("imported: %+v failed=%v", p.Mods, result.Failed)
	}
	for _, m := range p.Mods {
		if (m.ID == "B-Off") == m.Enabled {
			t.Errorf("%s enabled=%v", m.ID, m.Enabled)
		}
	}
	dstDir, _ := lib.ProfileDir(game.ID, p.ID)
	if data, _ := os.ReadFile(filepath.Join(dstDir, "BepInEx/config/Mod.cfg")); string(data) != "tuned" {
		t.Errorf("config not imported: %q", data)
	}
	if len(progress) != 4 || progress[3].Done != 3 {
		t.Errorf("progress: %+v", progress)
	}

	// Mods that are no longer available are reported, the rest is imported.
	delete(repo.packages, "A-Lib-1.0.0")
	delete(repo.packages, "A-Mod-2.0.0")
	os.RemoveAll(filepath.Join(tmp, "A-Lib-1.0.0.zip"))
	result, err = sharer.Import(ctx, game.ID, preview.Archive, "Partial", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Failed) != 2 || len(result.Profile.Mods) != 1 {
		t.Errorf("partial: mods=%v failed=%v", result.Profile.Mods, result.Failed)
	}
}

func TestImportKeepsListedVersions(t *testing.T) {
	tmp := t.TempDir()
	lib, _ := library.New(filepath.Join(tmp, "lib"), nil)
	os.MkdirAll(filepath.Join(tmp, "game"), 0o755)
	game, _ := lib.AddGame("Game", filepath.Join(tmp, "game"))
	repo := &fakeRepo{t: t, dir: tmp, codes: map[string][]byte{}, packages: map[string][]string{
		"BepInEx-BepInExPack-5.4.2100": {},
		"BepInEx-BepInExPack-5.4.2305": {},
		"A-Mod-1.0.0":                  {"BepInEx-BepInExPack-5.4.2100"},
	}}
	sharer := NewSharer(lib, modinstall.NewInstaller(lib, repo, nil), repo, filepath.Join(tmp, "imports"))

	// The profile was exported with a newer pack than the mod's manifest asks for.
	archive := filepath.Join(tmp, "p.r2z")
	f, _ := os.Create(archive)
	zw := zip.NewWriter(f)
	w, _ := zw.Create("export.r2x")
	w.Write([]byte("profileName: P\nmods:\n" +
		"  - {name: BepInEx-BepInExPack, version: {major: 5, minor: 4, patch: 2305}, enabled: true}\n" +
		"  - {name: A-Mod, version: {major: 1, minor: 0, patch: 0}, enabled: true}\n"))
	zw.Close()
	f.Close()

	result, err := sharer.Import(context.Background(), game.ID, archive, "P", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range result.Profile.Mods {
		if m.ID == "BepInEx-BepInExPack" && m.Version != "5.4.2305" {
			t.Errorf("pack downgraded to %s by a dependency", m.Version)
		}
	}
}

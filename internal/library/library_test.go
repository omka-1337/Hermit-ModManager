package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestLibrary(t *testing.T) *Library {
	t.Helper()
	lib, err := New(filepath.Join(t.TempDir(), "lib"))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func fakeGame(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		p := filepath.Join(dir, f)
		if strings.HasSuffix(f, "/") {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestAddGameCreatesDefaultProfile(t *testing.T) {
	lib := newTestLibrary(t)
	path := fakeGame(t, "Valheim_Data/", "Valheim.exe")

	g, err := lib.AddGame("Valheim", path)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID != "valheim" || g.Runtime != RuntimeProton {
		t.Fatalf("unexpected game: %+v", g)
	}

	profiles, err := lib.ListProfiles(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].ID != g.ActiveProfile || profiles[0].Name != DefaultProfileName {
		t.Fatalf("unexpected profiles: %+v (active %q)", profiles, g.ActiveProfile)
	}
	for _, sub := range []string{"plugins", "patchers", "config"} {
		if _, err := os.Stat(filepath.Join(lib.profileDir(g.ID, g.ActiveProfile), "BepInEx", sub)); err != nil {
			t.Errorf("missing BepInEx/%s: %v", sub, err)
		}
	}

	entries, _ := os.ReadDir(path)
	if len(entries) != 2 {
		t.Errorf("game directory was modified: %v", entries)
	}
}

func TestAddGameValidation(t *testing.T) {
	lib := newTestLibrary(t)
	path := fakeGame(t)

	if _, err := lib.AddGame("", path); !errors.Is(err, ErrEmptyName) {
		t.Errorf("empty name: got %v", err)
	}
	if _, err := lib.AddGame("X", filepath.Join(path, "missing")); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("missing path: got %v", err)
	}
	if _, err := lib.AddGame("X", path); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.AddGame("Y", path); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("duplicate path: got %v", err)
	}
}

func TestIDsAreUniqueAndStableAcrossRename(t *testing.T) {
	lib := newTestLibrary(t)
	a, _ := lib.AddGame("My Game", fakeGame(t))
	b, _ := lib.AddGame("My Game", fakeGame(t))
	c, _ := lib.AddGame("Гра", fakeGame(t))
	if a.ID != "my-game" || b.ID != "my-game-2" || c.ID != "game" {
		t.Fatalf("ids: %q %q %q", a.ID, b.ID, c.ID)
	}

	renamed, err := lib.RenameGame(a.ID, "Other")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := lib.GetGame(a.ID)
	if renamed.ID != a.ID || got.Name != "Other" {
		t.Fatalf("rename: %+v", got)
	}

	games, _ := lib.ListGames()
	if len(games) != 3 || games[0].Name != "My Game" || games[2].Name != "Гра" {
		t.Fatalf("list: %+v", games)
	}
}

func TestRemoveProfile(t *testing.T) {
	lib := newTestLibrary(t)
	g, _ := lib.AddGame("Game", fakeGame(t))

	if err := lib.RemoveProfile(g.ID, g.ActiveProfile); !errors.Is(err, ErrLastProfile) {
		t.Fatalf("last profile: got %v", err)
	}

	second, err := lib.CreateProfile(g.ID, "Modded")
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.RemoveProfile(g.ID, g.ActiveProfile); err != nil {
		t.Fatal(err)
	}
	g, _ = lib.GetGame(g.ID)
	if g.ActiveProfile != second.ID {
		t.Fatalf("active profile not switched: %q", g.ActiveProfile)
	}
}

func TestSetActiveProfileAndRemoveGame(t *testing.T) {
	lib := newTestLibrary(t)
	g, _ := lib.AddGame("Game", fakeGame(t))

	if _, err := lib.SetActiveProfile(g.ID, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown profile: got %v", err)
	}
	p, _ := lib.CreateProfile(g.ID, "Second")
	if g, _ = lib.SetActiveProfile(g.ID, p.ID); g.ActiveProfile != p.ID {
		t.Errorf("active: %q", g.ActiveProfile)
	}

	if err := lib.RemoveGame(g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.GetGame(g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after remove: got %v", err)
	}
}

func TestRejectsPathTraversalIDs(t *testing.T) {
	lib := newTestLibrary(t)
	for _, id := range []string{"", ".", "..", "../x", `a\b`} {
		if _, err := lib.GetGame(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetGame(%q): got %v", id, err)
		}
	}
}

func TestInspectGamePath(t *testing.T) {
	lib := newTestLibrary(t)
	path := fakeGame(t, "Lethal Company_Data/", "Lethal Company.exe")

	c, err := lib.InspectGamePath(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Lethal Company" || c.Runtime != RuntimeProton || c.AlreadyAdded {
		t.Fatalf("candidate: %+v", c)
	}

	if _, err := lib.AddGame(c.Name, path); err != nil {
		t.Fatal(err)
	}
	if c, _ = lib.InspectGamePath(path); !c.AlreadyAdded {
		t.Error("expected AlreadyAdded")
	}

	other := fakeGame(t)
	if c, _ = lib.InspectGamePath(other); c.Name != filepath.Base(other) {
		t.Errorf("fallback name: %q", c.Name)
	}
}

func TestDetectRuntime(t *testing.T) {
	cases := map[Runtime][]string{
		RuntimeProton:  {"Game_Data/", "Game.exe"},
		RuntimeNative:  {"Game_Data/", "Game.x86_64"},
		RuntimeUnknown: {"readme.txt"},
	}
	for want, files := range cases {
		if got := DetectRuntime(fakeGame(t, files...)); got != want {
			t.Errorf("%v: got %q, want %q", files, got, want)
		}
	}
}

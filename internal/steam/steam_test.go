package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func manifest(appid, name, installdir string) string {
	return `"AppState"
{
	"appid"		"` + appid + `"
	"name"		"` + name + `"
	"installdir"		"` + installdir + `"
	"InstalledDepots" { "1" { "manifest" "2" } }
}`
}

func TestParseVDF(t *testing.T) {
	kv, err := ParseVDF(`// comment
"Root"
{
	"Key"	"va\"lue"
	"Nested" { "Path" "C:\\Games" }
}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := kv.String("root", "key"); got != `va"lue` {
		t.Errorf("key: %q", got)
	}
	if got := kv.String("Root", "nested", "path"); got != `C:\Games` {
		t.Errorf("nested: %q", got)
	}
	for _, bad := range []string{`"a" {`, `"a" "b" }`, `"a"`, `"unterminated`} {
		if _, err := ParseVDF(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestInstalledAppsAcrossLibraries(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "Steam")
	second := filepath.Join(tmp, "Games", "SteamLibrary")
	oldFormat := filepath.Join(tmp, "Old")

	write(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), `"libraryfolders"
{
	"0" { "path" "`+root+`" }
	"1" { "path" "`+second+`" }
	"2" { "path" "`+filepath.Join(tmp, "missing")+`" }
}`)
	write(t, filepath.Join(root, "steamapps", "appmanifest_1.acf"), manifest("1", "Root Game", "RootGame"))
	write(t, filepath.Join(root, "steamapps", "common", "RootGame", "x"), "")
	write(t, filepath.Join(root, "steamapps", "appmanifest_2.acf"), manifest("2", "Not Installed", "Gone"))
	write(t, filepath.Join(second, "steamapps", "appmanifest_3.acf"), manifest("3", "Second Game", "Second"))
	write(t, filepath.Join(second, "steamapps", "common", "Second", "x"), "")

	// Symlinked alias of the root must not produce duplicates.
	alias := filepath.Join(tmp, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}

	apps := InstalledApps([]string{root, alias})
	if len(apps) != 2 {
		t.Fatalf("apps: %+v", apps)
	}
	if apps[0].AppID != "1" || apps[1].AppID != "3" || apps[1].LibraryPath != second {
		t.Fatalf("apps: %+v", apps)
	}
	if want := filepath.Join(second, "steamapps", "compatdata", "3"); apps[1].CompatDataPath() != want {
		t.Errorf("compatdata: %q", apps[1].CompatDataPath())
	}

	app, ok := FindAppByPath([]string{alias}, filepath.Join(alias, "steamapps", "common", "RootGame"))
	if !ok || app.AppID != "1" {
		t.Errorf("FindAppByPath: %+v %v", app, ok)
	}

	write(t, filepath.Join(oldFormat, "steamapps", "libraryfolders.vdf"), `"LibraryFolders"
{
	"TimeNextStatsReport" "123"
	"1" "`+second+`"
}`)
	if libs := Libraries([]string{oldFormat}); len(libs) != 2 || libs[1] != second {
		t.Errorf("old format libraries: %v", libs)
	}
}

func TestLaunchOptions(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "userdata", "123", "config", "localconfig.vdf"), `"UserLocalConfigStore"
{
	"Software" { "Valve" { "Steam" { "apps" {
		"1966720" { "LaunchOptions" "\"/opt/bmm\" run -- %command%" }
		"10" { "LastPlayed" "1" }
	} } } }
}`)
	got := LaunchOptions([]string{root}, "1966720")
	if len(got) != 1 || got[0] != `"/opt/bmm" run -- %command%` {
		t.Errorf("got %q", got)
	}
	if got := LaunchOptions([]string{root}, "10"); len(got) != 0 {
		t.Errorf("app without options: %q", got)
	}
}

func TestIconPath(t *testing.T) {
	root := t.TempDir()
	hash := "80d2453285274e4723c884a671cdd3f8fe2f766f"
	write(t, filepath.Join(root, "appcache/librarycache/1966720", hash+".jpg"), "jpg")
	write(t, filepath.Join(root, "appcache/librarycache/1966720/header.jpg"), "")
	write(t, filepath.Join(root, "appcache/librarycache/892970/2f64c9a826e2c6cf3253fea4834c2e612db09143.jpg"), "jpg")
	write(t, filepath.Join(root, "appcache/librarycache/70_icon.jpg"), "jpg")

	if got := IconPath([]string{root}, "892970"); filepath.Base(got) != "2f64c9a826e2c6cf3253fea4834c2e612db09143.jpg" {
		t.Errorf("jpg icon: %q", got)
	}
	write(t, filepath.Join(root, "steam/games", hash+".ico"), "ico")
	if got := IconPath([]string{root}, "1966720"); got != filepath.Join(root, "steam/games", hash+".ico") {
		t.Errorf("ico preferred: %q", got)
	}
	if got := IconPath([]string{root}, "70"); filepath.Base(got) != "70_icon.jpg" {
		t.Errorf("old layout: %q", got)
	}
	if IconPath([]string{root}, "../x") != "" || IconPath([]string{root}, "10") != "" {
		t.Error("invalid or unknown app must have no icon")
	}
}

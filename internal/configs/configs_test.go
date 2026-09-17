package configs

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const sample = "## Settings file was created by plugin MoreCompany v1.14.0\r\n" +
	"## Plugin GUID: me.swipez.melonloader.morecompany\r\n" +
	"\r\n" +
	"[Cosmetics]\r\n" +
	"\r\n" +
	"## Should you be able to see cosmetics of other players?\r\n" +
	"# Setting type: Boolean\r\n" +
	"# Default value: true\r\n" +
	"Show Cosmetics = true\r\n" +
	"\r\n" +
	"## Comma separated list of cosmetics to disable\r\n" +
	"# Setting type: String\r\n" +
	"# Default value: \r\n" +
	"Disabled Cosmetics = \r\n" +
	"\r\n" +
	"[General]\r\n" +
	"\r\n" +
	"## How many players can be in your lobby?\r\n" +
	"## Second line.\r\n" +
	"# Setting type: Int32\r\n" +
	"# Default value: 32\r\n" +
	"# Acceptable value range: From 4 to 50\r\n" +
	"Player Count = 32\r\n" +
	"\r\n" +
	"# Setting type: LogLevel\r\n" +
	"# Default value: Fatal, Error, Warning\r\n" +
	"# Acceptable values: None, Fatal, Error, Warning, Message, Info, Debug, All\r\n" +
	"# Multiple values can be set at the same time by separating them with , (e.g. Debug, Warning)\r\n" +
	"LogLevels = Fatal, Error, Warning\r\n" +
	"\r\n" +
	"# Setting type: ConsoleOutRedirectType\r\n" +
	"# Default value: Auto\r\n" +
	"# Acceptable values: Auto, ConsoleOut, StandardOut\r\n" +
	"StandardOutType = Auto\r\n"

func TestParseCfg(t *testing.T) {
	doc := ParseCfg(sample)
	if !slices.Equal(doc.Header, []string{"Settings file was created by plugin MoreCompany v1.14.0", "Plugin GUID: me.swipez.melonloader.morecompany"}) {
		t.Errorf("header: %q", doc.Header)
	}
	if len(doc.Sections) != 2 || len(doc.Sections[0].Entries) != 2 || len(doc.Sections[1].Entries) != 3 {
		t.Fatalf("sections: %+v", doc.Sections)
	}
	show := doc.Sections[0].Entries[0]
	if show.Key != "Show Cosmetics" || show.Value != "true" || show.Type != "Boolean" || show.Default != "true" || show.Line != 8 {
		t.Errorf("boolean: %+v", show)
	}
	if empty := doc.Sections[0].Entries[1]; empty.Value != "" || !empty.HasDefault || empty.Default != "" {
		t.Errorf("empty string: %+v", empty)
	}
	count := doc.Sections[1].Entries[0]
	if !count.HasRange || count.Min != 4 || count.Max != 50 || count.Description != "How many players can be in your lobby?\nSecond line." {
		t.Errorf("range: %+v", count)
	}
	flags := doc.Sections[1].Entries[1]
	if !flags.Multiple || len(flags.AcceptableValues) != 8 {
		t.Errorf("flags: %+v", flags)
	}
	if enum := doc.Sections[1].Entries[2]; enum.Multiple || !slices.Equal(enum.AcceptableValues, []string{"Auto", "ConsoleOut", "StandardOut"}) {
		t.Errorf("enum: %+v", enum)
	}
}

func TestApplyChangesKeepsEverythingElse(t *testing.T) {
	doc := ParseCfg(sample)
	show, disabled, count := doc.Sections[0].Entries[0], doc.Sections[0].Entries[1], doc.Sections[1].Entries[0]
	out, err := ApplyChanges(sample, []Change{
		{Section: "Cosmetics", Key: show.Key, Line: show.Line, Value: "false"},
		{Section: "Cosmetics", Key: disabled.Key, Line: disabled.Line, Value: "hat, glasses"},
		// Stale line number: found by section and key instead.
		{Section: "General", Key: count.Key, Line: 0, Value: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(sample, "Show Cosmetics = true\r\n", "Show Cosmetics = false\r\n", 1)
	want = strings.Replace(want, "Disabled Cosmetics = \r\n", "Disabled Cosmetics = hat, glasses\r\n", 1)
	want = strings.Replace(want, "Player Count = 32\r\n", "Player Count = 8\r\n", 1)
	if out != want {
		t.Errorf("output:\n%q\nwant:\n%q", out, want)
	}

	if _, err := ApplyChanges(sample, []Change{{Section: "General", Key: "Missing", Value: "1"}}); err == nil {
		t.Error("unknown key must fail")
	}
	if _, err := ApplyChanges(sample, []Change{{Section: "General", Key: count.Key, Line: count.Line, Value: "1\n[Evil]"}}); err == nil {
		t.Error("multi-line value must fail")
	}
}

func TestListAndResolve(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"BepInEx/config/me.swipez.melonloader.morecompany.cfg": sample,
		"BepInEx/config/BepInEx.cfg":                           "[Logging]\n",
		"BepInEx/plugins/A-Mod/data/items.json":                "{}",
		"BepInEx/plugins/A-Mod/manifest.json":                  "{}",
		"BepInEx/plugins/A-Mod/Mod.dll":                        "",
		"BepInEx/cache/chainloader.json":                       "{}",
		"disabled/B-Off/x.cfg":                                 "",
		"doorstop_config.ini":                                  "",
		"profile.json":                                         "{}",
	} {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	files, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
		if f.Path == "BepInEx/config/me.swipez.melonloader.morecompany.cfg" && f.Plugin != "MoreCompany" {
			t.Errorf("plugin name: %q", f.Plugin)
		}
	}
	want := []string{"BepInEx/config/BepInEx.cfg", "BepInEx/config/me.swipez.melonloader.morecompany.cfg", "BepInEx/plugins/A-Mod/data/items.json", "doorstop_config.ini"}
	if !slices.Equal(paths, want) {
		t.Errorf("files:\n got  %v\n want %v", paths, want)
	}

	for _, bad := range []string{"../x.cfg", "profile.json", "BepInEx/plugins/A-Mod/Mod.dll", "disabled/B-Off/x.cfg", "/etc/x.cfg", "BepInEx/../BepInEx/config/BepInEx.cfg"} {
		if _, err := Resolve(dir, bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	os.Symlink("/etc/hostname", filepath.Join(dir, "BepInEx/config/link.cfg"))
	if _, err := Resolve(dir, "BepInEx/config/link.cfg"); err == nil {
		t.Error("symlink outside profile must be rejected")
	}

	if err := WriteText(dir, "doorstop_config.ini", "[General]\nenabled=true\n"); err != nil {
		t.Fatal(err)
	}
	if text, _ := ReadText(dir, "doorstop_config.ini"); text != "[General]\nenabled=true\n" {
		t.Errorf("text round trip: %q", text)
	}
}

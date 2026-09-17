package plugininfo_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"hermit/internal/plugininfo"
	"hermit/internal/plugininfo/testasm"
)

func TestParsePluginAttributes(t *testing.T) {
	b := testasm.New()
	b.TypeDefs = append(b.TypeDefs, [3]uint16{b.Str("Plugin"), b.Str("MoreCompany")})
	const class = 2
	threeStrings := []byte{0x20, 3, 0x01, 0x0E, 0x0E, 0x0E}
	b.Attribute(class, "BepInEx", "BepInPlugin", threeStrings,
		testasm.SerString("me.swipez.melonloader.morecompany"), testasm.SerString("MoreCompany"), testasm.SerString("1.14.0"))
	// BepInDependency(string, DependencyFlags) with a soft flag.
	b.Attribute(class, "BepInEx", "BepInDependency", []byte{0x20, 2, 0x01, 0x0E, 0x11, 0x0D},
		testasm.SerString("com.lib.optional"), []byte{2, 0, 0, 0})
	// BepInDependency(string, string) with a minimum version.
	b.Attribute(class, "BepInEx", "BepInDependency", []byte{0x20, 2, 0x01, 0x0E, 0x0E},
		testasm.SerString("com.lib.core"), testasm.SerString("1.2.0"))
	b.Attribute(class, "BepInEx", "BepInIncompatibility", []byte{0x20, 1, 0x01, 0x0E}, testasm.SerString("com.other.lobby"))
	b.Attribute(class, "BepInEx", "BepInProcess", []byte{0x20, 1, 0x01, 0x0E}, testasm.SerString("Lethal Company.exe"))
	// Attributes from other namespaces are ignored.
	b.Attribute(class, "System.Runtime.CompilerServices", "BepInPlugin", threeStrings,
		testasm.SerString("x"), testasm.SerString("y"), testasm.SerString("z"))

	plugins, err := plugininfo.Parse(b.Build())
	if err != nil {
		t.Fatal(err)
	}
	want := []plugininfo.Plugin{{
		GUID: "me.swipez.melonloader.morecompany", Name: "MoreCompany", Version: "1.14.0",
		Class: "MoreCompany.Plugin",
		Dependencies: []plugininfo.Dependency{
			{GUID: "com.lib.optional", Soft: true},
			{GUID: "com.lib.core", MinVersion: "1.2.0"},
		},
		Incompatibilities: []string{"com.other.lobby"},
		Processes:         []string{"Lethal Company.exe"},
	}}
	if !reflect.DeepEqual(plugins, want) {
		t.Errorf("plugins:\n got  %+v\n want %+v", plugins, want)
	}
}

func TestParseLibraryAndGarbage(t *testing.T) {
	plugins, err := plugininfo.Parse(testasm.New().Build())
	if err != nil || len(plugins) != 0 {
		t.Errorf("library: %v %v", plugins, err)
	}
	for _, data := range [][]byte{nil, []byte("MZ"), bytes.Repeat([]byte{0xFF}, 512)} {
		if _, err := plugininfo.Parse(data); !errors.Is(err, plugininfo.ErrNotAssembly) {
			t.Errorf("garbage %q: %v", data, err)
		}
	}
	// Truncated images must fail cleanly, never panic.
	img := func() []byte {
		b := testasm.New()
		b.TypeDefs = append(b.TypeDefs, [3]uint16{b.Str("P"), 0})
		b.Attribute(2, "BepInEx", "BepInPlugin", []byte{0x20, 3, 0x01, 0x0E, 0x0E, 0x0E}, testasm.SerString("a"), testasm.SerString("b"), testasm.SerString("c"))
		return b.Build()
	}()
	for n := 0; n < len(img); n += 7 {
		plugininfo.Parse(img[:n])
	}
}

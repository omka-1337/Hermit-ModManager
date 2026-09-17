//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeSysfs builds /sys with the given cards: name -> {driver, boot_vga}.
func fakeSysfs(t *testing.T, nvidiaLoaded bool, cards map[string][2]string) string {
	t.Helper()
	sys := t.TempDir()
	if nvidiaLoaded {
		os.MkdirAll(filepath.Join(sys, "module", "nvidia"), 0o755)
	}
	for name, c := range cards {
		driver, bootVGA := c[0], c[1]
		devDir := filepath.Join(sys, "devices", name)
		drvDir := filepath.Join(sys, "bus", "pci", "drivers", driver)
		os.MkdirAll(devDir, 0o755)
		os.MkdirAll(drvDir, 0o755)
		os.Symlink(drvDir, filepath.Join(devDir, "driver"))
		os.WriteFile(filepath.Join(devDir, "boot_vga"), []byte(bootVGA+"\n"), 0o644)
		cardDir := filepath.Join(sys, "class", "drm", name)
		os.MkdirAll(cardDir, 0o755)
		os.Symlink(devDir, filepath.Join(cardDir, "device"))
	}
	os.MkdirAll(filepath.Join(sys, "class", "drm", "card1-eDP-1"), 0o755)
	return sys
}

func TestIsHybridNVIDIA(t *testing.T) {
	cases := []struct {
		name   string
		nvidia bool
		cards  map[string][2]string
		want   bool
	}{
		{"intel display, nvidia offload", true, map[string][2]string{"card0": {"nvidia", "0"}, "card1": {"i915", "1"}}, true},
		{"nvidia display", true, map[string][2]string{"card0": {"nvidia", "1"}, "card1": {"i915", "0"}}, false},
		{"no nvidia module", false, map[string][2]string{"card0": {"amdgpu", "1"}}, false},
		{"nouveau hybrid", false, map[string][2]string{"card0": {"nouveau", "0"}, "card1": {"i915", "1"}}, false},
	}
	for _, tc := range cases {
		if got := isHybridNVIDIA(fakeSysfs(t, tc.nvidia, tc.cards)); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

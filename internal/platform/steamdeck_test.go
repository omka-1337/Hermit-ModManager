package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeSystem(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestIsSteamDeck(t *testing.T) {
	t.Setenv("SteamDeck", "")
	cases := map[string]struct {
		files map[string]string
		want  bool
	}{
		"deck lcd": {map[string]string{
			"sys/class/dmi/id/sys_vendor":   "Valve\n",
			"sys/class/dmi/id/product_name": "Jupiter\n",
		}, true},
		"deck oled": {map[string]string{
			"sys/class/dmi/id/sys_vendor":   "Valve\n",
			"sys/class/dmi/id/product_name": "Galileo\n",
		}, true},
		"steamos elsewhere": {map[string]string{
			"etc/os-release": "ID=steamos\nVARIANT_ID=\"steamdeck\"\n",
		}, true},
		"regular desktop": {map[string]string{
			"sys/class/dmi/id/sys_vendor":   "HP\n",
			"sys/class/dmi/id/product_name": "HP EliteBook 1050 G1\n",
			"etc/os-release":                "ID=arch\n",
		}, false},
		"holo desktop install": {map[string]string{
			"etc/os-release": "ID=steamos\nVARIANT_ID=holo\n",
		}, false},
	}
	for name, tc := range cases {
		if got := isSteamDeck(fakeSystem(t, tc.files)); got != tc.want {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}

	t.Setenv("SteamDeck", "1")
	if !isSteamDeck(fakeSystem(t, nil)) {
		t.Error("Steam's SteamDeck=1 must be honoured")
	}
}

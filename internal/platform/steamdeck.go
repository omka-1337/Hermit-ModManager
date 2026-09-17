package platform

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// deckProducts are the DMI product names of Steam Deck models: LCD and OLED.
var deckProducts = []string{"jupiter", "galileo"}

// IsSteamDeck reports whether Hermit runs on a Steam Deck. The hardware is
// checked first so it also works in desktop mode, where Steam sets no
// environment variables.
func IsSteamDeck() bool {
	return isSteamDeck("/")
}

func isSteamDeck(root string) bool {
	vendor := readTrimmed(filepath.Join(root, "sys/class/dmi/id/sys_vendor"))
	product := readTrimmed(filepath.Join(root, "sys/class/dmi/id/product_name"))
	if strings.EqualFold(vendor, "Valve") && slices.Contains(deckProducts, strings.ToLower(product)) {
		return true
	}
	// SteamOS in a VM or on other handhelds Valve supports.
	release := readTrimmed(filepath.Join(root, "etc/os-release"))
	for _, line := range strings.Split(release, "\n") {
		if key, value, ok := strings.Cut(line, "="); ok && key == "VARIANT_ID" {
			if strings.Trim(strings.TrimSpace(value), `"`) == "steamdeck" {
				return true
			}
		}
	}
	return os.Getenv("SteamDeck") == "1"
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

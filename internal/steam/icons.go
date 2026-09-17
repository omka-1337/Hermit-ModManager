package steam

import (
	"os"
	"path/filepath"
	"regexp"
)

var iconHash = regexp.MustCompile(`^[0-9a-f]{40}\.jpg$`)

// IconPath returns the best locally cached icon of an app, or "" if Steam has
// none. Steam keeps a 32×32 JPEG named by the icon hash in the library cache
// and, for apps with a desktop shortcut, a multi-size .ico with the same hash.
func IconPath(roots []string, appID string) string {
	if appID == "" || !isNumeric(appID) {
		return ""
	}
	for _, root := range roots {
		cache := filepath.Join(root, "appcache", "librarycache")
		entries, _ := os.ReadDir(filepath.Join(cache, appID))
		for _, e := range entries {
			if e.IsDir() || !iconHash.MatchString(e.Name()) {
				continue
			}
			hash := e.Name()[:40]
			if ico := filepath.Join(root, "steam", "games", hash+".ico"); isFile(ico) {
				return ico
			}
			return filepath.Join(cache, appID, e.Name())
		}
		// Older Steam versions used a flat cache layout.
		if old := filepath.Join(cache, appID+"_icon.jpg"); isFile(old) {
			return old
		}
	}
	return ""
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

package steam

import (
	"os"
	"path/filepath"
)

// LaunchOptions returns the Launch Options set for an app by each Steam user
// found in the given roots (empty values are skipped). Steam keeps them in
// userdata/<user>/config/localconfig.vdf and rewrites that file on exit, so
// it is only read, never written.
func LaunchOptions(roots []string, appID string) []string {
	var result []string
	for _, root := range roots {
		files, _ := filepath.Glob(filepath.Join(root, "userdata", "*", "config", "localconfig.vdf"))
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			kv, err := ParseVDF(string(data))
			if err != nil {
				continue
			}
			opts := kv.String("UserLocalConfigStore", "Software", "Valve", "Steam", "apps", appID, "LaunchOptions")
			if opts != "" {
				result = append(result, opts)
			}
		}
	}
	return result
}

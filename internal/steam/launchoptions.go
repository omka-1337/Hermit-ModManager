package steam

import (
	"os"
	"path/filepath"
	"sync"
)

// The UI polls launch options every few seconds to notice when Steam finally
// writes them, so parsed files are kept until they change on disk.
var (
	parsedMu sync.Mutex
	parsed   = map[string]parsedConfig{}
)

type parsedConfig struct {
	modTime int64
	size    int64
	kv      *KeyValues
}

// LaunchOptions returns the Launch Options set for an app by each Steam user
// found in the given roots (empty values are skipped). Steam keeps them in
// userdata/<user>/config/localconfig.vdf and rewrites that file on exit, so
// it is only read, never written.
func LaunchOptions(roots []string, appID string) []string {
	var result []string
	for _, root := range roots {
		files, _ := filepath.Glob(filepath.Join(root, "userdata", "*", "config", "localconfig.vdf"))
		for _, f := range files {
			kv, err := localConfig(f)
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

// localConfig parses a localconfig.vdf, reusing the last parse while the file
// is unchanged.
func localConfig(path string) (*KeyValues, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	parsedMu.Lock()
	defer parsedMu.Unlock()
	if cached, ok := parsed[path]; ok && cached.modTime == info.ModTime().UnixNano() && cached.size == info.Size() {
		return cached.kv, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	kv, err := ParseVDF(string(data))
	if err != nil {
		return nil, err
	}
	parsed[path] = parsedConfig{modTime: info.ModTime().UnixNano(), size: info.Size(), kv: kv}
	return kv, nil
}

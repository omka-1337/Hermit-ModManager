package library

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectRuntime guesses how a Unity game is run by looking for its <name>_Data
// folder and a matching executable next to it.
func DetectRuntime(gamePath string) Runtime {
	entries, err := os.ReadDir(gamePath)
	if err != nil {
		return RuntimeUnknown
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), "_Data")
		if !ok || !e.IsDir() {
			continue
		}
		if isFile(filepath.Join(gamePath, name+".exe")) {
			return RuntimeProton
		}
		for _, ext := range []string{".x86_64", ".x86", ""} {
			if isFile(filepath.Join(gamePath, name+ext)) {
				return RuntimeNative
			}
		}
	}
	return RuntimeUnknown
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

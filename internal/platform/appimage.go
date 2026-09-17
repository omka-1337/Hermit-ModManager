package platform

import (
	"bytes"
	"encoding/base64"
	"os"
)

// originalEnvVar is set by Hermit's first AppRun hook in the AppImage to the
// environment before the AppImage changed it (NUL-separated, base64).
const originalEnvVar = "HERMIT_ORIGINAL_ENV"

// OriginalEnv returns the environment Hermit was started with, before the
// AppImage's AppRun added library paths, GTK settings and such. Outside an
// AppImage it is simply the current environment.
func OriginalEnv() []string {
	encoded := os.Getenv(originalEnvVar)
	if encoded == "" {
		return os.Environ()
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return os.Environ()
	}
	var env []string
	for _, kv := range bytes.Split(data, []byte{0}) {
		if len(kv) > 0 && !bytes.HasPrefix(kv, []byte(originalEnvVar+"=")) {
			env = append(env, string(kv))
		}
	}
	return env
}

// OriginalWorkingDir is the directory Hermit was started from. The AppImage
// changes into its own directory so the bundled WebKit finds its helper
// processes; the AppImage runtime keeps the original one in $OWD.
func OriginalWorkingDir() string {
	if owd := os.Getenv("OWD"); owd != "" && os.Getenv("APPIMAGE") != "" {
		return owd
	}
	dir, _ := os.Getwd()
	return dir
}

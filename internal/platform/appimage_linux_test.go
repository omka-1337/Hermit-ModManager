package platform

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
)

func TestOriginalEnv(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "/tmp/.mount_hermit/usr/lib")
	original := strings.Join([]string{"HOME=/home/me", "SteamAppId=1966720", "WINEDLLOVERRIDES="}, "\x00") + "\x00"
	t.Setenv(originalEnvVar, base64.StdEncoding.EncodeToString([]byte(original)))

	got := OriginalEnv()
	if !slices.Equal(got, []string{"HOME=/home/me", "SteamAppId=1966720", "WINEDLLOVERRIDES="}) {
		t.Errorf("env: %q", got)
	}
	if env := initialEnv(); env["LD_LIBRARY_PATH"] != "" || env["SteamAppId"] != "1966720" {
		t.Errorf("initialEnv: %v", env)
	}

	t.Setenv("APPIMAGE", "/home/me/Hermit.AppImage")
	t.Setenv("OWD", "/games/Lethal Company")
	if dir := OriginalWorkingDir(); dir != "/games/Lethal Company" {
		t.Errorf("working dir: %q", dir)
	}
}

//go:build linux

// Package platform holds OS-specific startup tweaks.
package platform

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigureRendering fixes slow rendering on hybrid laptops where an NVIDIA
// GPU is present but the display is driven by another GPU (e.g. Intel).
//
// Wails disables WebKit's DMA-BUF renderer whenever the nvidia module is
// loaded, which makes WebKit copy every frame through the CPU, and GTK picks
// the Vulkan renderer on the NVIDIA card. Both are only needed when NVIDIA
// drives the display, so on hybrid setups DMA-BUF is re-enabled and GTK is
// switched to OpenGL on the display GPU. Values set by the user are kept.
//
// Must be called at the start of main, before the application is created.
func ConfigureRendering() {
	if !isHybridNVIDIA("/sys") {
		return
	}
	userEnv := initialEnv()
	if _, ok := userEnv["WEBKIT_DISABLE_DMABUF_RENDERER"]; !ok {
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	}
	if _, ok := userEnv["GSK_RENDERER"]; !ok {
		os.Setenv("GSK_RENDERER", "gl")
	}
}

var cardName = regexp.MustCompile(`^card\d+$`)

// isHybridNVIDIA reports whether the nvidia module is loaded while the boot
// (display) GPU uses a different driver.
func isHybridNVIDIA(sysfs string) bool {
	if _, err := os.Stat(filepath.Join(sysfs, "module", "nvidia")); err != nil {
		return false
	}
	cards, _ := os.ReadDir(filepath.Join(sysfs, "class", "drm"))
	for _, c := range cards {
		if !cardName.MatchString(c.Name()) {
			continue
		}
		device := filepath.Join(sysfs, "class", "drm", c.Name(), "device")
		bootVGA, err := os.ReadFile(filepath.Join(device, "boot_vga"))
		if err != nil || strings.TrimSpace(string(bootVGA)) != "1" {
			continue
		}
		driver, err := filepath.EvalSymlinks(filepath.Join(device, "driver"))
		return err == nil && filepath.Base(driver) != "nvidia"
	}
	return false
}

// initialEnv returns the environment the process was started with. Wails
// modifies the environment in its package init, before main runs, so
// os.Getenv cannot tell user settings from Wails defaults; in the AppImage the
// AppRun hooks change it even earlier, which OriginalEnv undoes.
func initialEnv() map[string]string {
	env := map[string]string{}
	var entries []string
	if os.Getenv(originalEnvVar) != "" {
		entries = OriginalEnv()
	} else if data, err := os.ReadFile("/proc/self/environ"); err == nil {
		for _, kv := range bytes.Split(data, []byte{0}) {
			entries = append(entries, string(kv))
		}
	}
	for _, kv := range entries {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	return env
}

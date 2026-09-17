package library

import (
	"os"
	"path/filepath"
	"strings"
)

// Detection is what can be learned about a game from its install directory.
type Detection struct {
	Unity   bool    `json:"unity"`
	Runtime Runtime `json:"runtime"`
	Backend Backend `json:"backend"`
	// Executable is the game executable file name, e.g. "Valheim.exe".
	Executable string `json:"executable"`
}

// DetectGame inspects a game directory without modifying it.
//
// Unity is recognised by UnityPlayer.dll / UnityCrashHandler*.exe (Windows
// builds), UnityPlayer.so (Linux builds), or, for old Unity versions without
// those files, by a <Name>_Data folder with managed or IL2CPP data next to a
// matching executable.
func DetectGame(dir string) Detection {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Detection{Runtime: RuntimeUnknown, Backend: BackendUnknown}
	}
	files := map[string]bool{}
	var dataDirs []string
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if e.IsDir() {
			if base, ok := strings.CutSuffix(e.Name(), "_Data"); ok {
				dataDirs = append(dataDirs, base)
			}
			continue
		}
		files[name] = true
	}

	windowsUnity := files["unityplayer.dll"] || files["unitycrashhandler64.exe"] || files["unitycrashhandler32.exe"]
	linuxUnity := files["unityplayer.so"]

	d := Detection{Runtime: RuntimeUnknown, Backend: BackendUnknown}
	for _, base := range dataDirs {
		exe, native := unityExecutable(files, base)
		if exe == "" {
			continue
		}
		dataDir := filepath.Join(dir, base+"_Data")
		if !windowsUnity && !linuxUnity && !isDir(filepath.Join(dataDir, "Managed")) && !isDir(filepath.Join(dataDir, "il2cpp_data")) {
			continue
		}
		d.Unity = true
		d.Executable = exe
		d.Runtime = RuntimeProton
		if native {
			d.Runtime = RuntimeNative
		}
		d.Backend = detectBackend(files, dataDir)
		return d
	}
	return d
}

// unityExecutable finds the executable matching a <base>_Data folder,
// preferring a native Linux binary over a Windows one.
func unityExecutable(files map[string]bool, base string) (name string, native bool) {
	lower := strings.ToLower(base)
	for _, ext := range []string{".x86_64", ".x86"} {
		if files[lower+ext] {
			return base + ext, true
		}
	}
	if files[lower+".exe"] {
		return base + ".exe", false
	}
	return "", false
}

func detectBackend(files map[string]bool, dataDir string) Backend {
	if files["gameassembly.dll"] || files["gameassembly.so"] || isDir(filepath.Join(dataDir, "il2cpp_data")) {
		return BackendIL2CPP
	}
	if isDir(filepath.Join(dataDir, "Managed")) {
		return BackendMono
	}
	return BackendUnknown
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

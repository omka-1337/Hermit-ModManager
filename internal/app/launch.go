package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hermit/internal/launch"
	"hermit/internal/library"
	"hermit/internal/steam"
)

type LaunchInfo struct {
	// Supported is false when the manager cannot launch the game with mods;
	// Reason explains why.
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
	// LaunchOptions is the value to put into the game's Steam Launch Options.
	LaunchOptions string `json:"launchOptions"`
	// Configured reports whether Steam's saved config has these launch options.
	// Steam writes its config lazily, so false may just mean "not saved yet".
	Configured bool `json:"configured"`
	Running    bool `json:"running"`
	// RunningProfile is the profile of the running session, if any.
	RunningProfile string `json:"runningProfile"`
}

type LaunchService struct {
	lib        *library.Library
	steamRoots []string
}

func NewLaunchService(lib *library.Library, steamRoots []string) *LaunchService {
	return &LaunchService{lib: lib, steamRoots: steamRoots}
}

func (s *LaunchService) GetLaunchInfo(gameID string) (LaunchInfo, error) {
	game, err := s.lib.GetGame(gameID)
	if err != nil {
		return LaunchInfo{}, err
	}
	info := LaunchInfo{LaunchOptions: RecommendedLaunchOptions()}
	switch {
	case game.SteamAppID == "":
		info.Reason = "Only Steam games can be launched for now."
	case game.Runtime == library.RuntimeUnknown:
		info.Reason = "Could not tell whether the game runs natively or through Proton."
	default:
		info.Supported = true
	}
	for _, opts := range steam.LaunchOptions(s.steamRoots, game.SteamAppID) {
		if isOurLaunchOptions(opts) {
			info.Configured = true
		}
	}
	if dataDir, err := s.lib.GameDataDir(gameID); err == nil {
		if session, err := launch.ReadSession(dataDir); err == nil && session.Running() {
			info.Running = true
			info.RunningProfile = session.ProfileID
		}
	}
	return info, nil
}

// Play makes the profile active and starts the game through Steam. Mods load
// only if the game's launch options run it through the wrapper.
func (s *LaunchService) Play(gameID, profileID string) error {
	info, err := s.GetLaunchInfo(gameID)
	if err != nil {
		return err
	}
	if !info.Supported {
		return errors.New(info.Reason)
	}
	if info.Running {
		return errors.New("the game is already running")
	}
	game, err := s.lib.SetActiveProfile(gameID, profileID)
	if err != nil {
		return err
	}
	return exec.Command("xdg-open", "steam://rungameid/"+game.SteamAppID).Start()
}

// RecommendedLaunchOptions is the Steam Launch Options value that runs games
// through this executable.
func RecommendedLaunchOptions() string {
	return `"` + executablePath() + `" run -- %command%`
}

func isOurLaunchOptions(opts string) bool {
	exe := filepath.Base(executablePath())
	return strings.Contains(opts, "%command%") &&
		(strings.Contains(opts, exe+`" run `) || strings.Contains(opts, exe+" run "))
}

func executablePath() string {
	if appImage := os.Getenv("APPIMAGE"); appImage != "" {
		return appImage
	}
	exe, err := os.Executable()
	if err != nil {
		return ID
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// GetLaunchReport returns what BepInEx loaded during the last session of a
// profile, or nil if the profile was not launched with mods yet.
func (s *LaunchService) GetLaunchReport(gameID, profileID string) (*launch.Report, error) {
	dataDir, err := s.lib.GameDataDir(gameID)
	if err != nil {
		return nil, err
	}
	return launch.ReadReport(dataDir, profileID)
}

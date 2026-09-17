package launch

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"hermit/internal/library"
	"hermit/internal/modinstall"
)

// Wrapper runs a game command (Steam's %command%) with the active profile of
// the game linked in.
type Wrapper struct {
	Lib       *library.Library
	Installer *modinstall.Installer
	// Notify shows a desktop notification; failures to set up mods are
	// reported through it because Steam hides the wrapper's output.
	Notify func(summary, body string)
}

var appIDArg = regexp.MustCompile(`^AppId=(\d+)$`)

// Run parses "[--game <id>] -- <command...>", runs the command and returns
// its exit code. Mod setup problems never prevent the game from starting: the
// game is then launched without mods.
func (w *Wrapper) Run(args []string) int {
	gameID, command, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	logOut := io.Writer(os.Stderr)
	plan := launchPlan{command: command, env: os.Environ()}
	setupErr := w.setup(gameID, &plan, &logOut)
	logger := log.New(logOut, "[hermit] ", log.LstdFlags)
	switch {
	case setupErr != nil:
		logger.Printf("launching without mods: %v", setupErr)
		if w.Notify != nil {
			w.Notify("Mods were not loaded", setupErr.Error())
		}
	case !plan.modded:
		logger.Printf("profile has no active BepInEx, launching without mods")
	default:
		logger.Printf("launching with profile %s: %q", plan.profileID, plan.command)
	}

	code := runCommand(plan.command, plan.env, logger)
	if plan.cleanup != nil {
		if err := plan.cleanup(); err != nil {
			logger.Printf("cleanup: %v", err)
		}
	}
	return code
}

func parseArgs(args []string) (gameID string, command []string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--game":
			if i+1 >= len(args) {
				return "", nil, errors.New("--game needs a value")
			}
			gameID = args[i+1]
			i++
		case "--":
			command = args[i+1:]
			if len(command) == 0 {
				return "", nil, errors.New("no command after --")
			}
			return gameID, command, nil
		default:
			return "", nil, fmt.Errorf("unexpected argument %q", args[i])
		}
	}
	return "", nil, errors.New(`usage: run [--game <id>] -- <command...> (in Steam: run -- %command%)`)
}

// launchPlan is what the wrapper runs. setup fills it in only when mods can be
// loaded; on error the original command runs unchanged.
type launchPlan struct {
	command   []string
	env       []string
	modded    bool
	profileID string
	// cleanup runs after the game exits.
	cleanup func() error
}

// nativeLaunchers are scripts that BepInEx packs for native Linux games ship in
// the profile root, in the order r2modman tries them. They take Steam's
// %command%, find the game executable in it and load BepInEx from their own
// directory, so nothing needs to be linked into the game folder.
var nativeLaunchers = []string{"run_bepinex.sh", "start_game_bepinex.sh"}

// setup prepares the active profile of the game for launching.
func (w *Wrapper) setup(gameID string, plan *launchPlan, logOut *io.Writer) error {
	if w.Lib == nil || w.Installer == nil {
		return errors.New("mod manager data could not be opened")
	}
	game, err := w.findGame(gameID, plan.command)
	if err != nil {
		return err
	}
	dataDir, err := w.Lib.GameDataDir(game.ID)
	if err != nil {
		return err
	}
	if f, err := os.Create(filepath.Join(dataDir, "launch.log")); err == nil {
		*logOut = io.MultiWriter(os.Stderr, f)
	}
	if game.Runtime != library.RuntimeProton && game.Runtime != library.RuntimeNative {
		return fmt.Errorf("%s: could not tell whether the game runs natively or through Proton", game.Name)
	}

	prev, err := ReadSession(dataDir)
	if err != nil {
		return err
	}
	if prev.Running() {
		return fmt.Errorf("%s is already running (pid %d)", game.Name, prev.PID)
	}
	if prev != nil {
		// Left over from a session that did not exit cleanly.
		Unlink(game.Path, prev.Links, w.Lib.Root())
	}

	profile, err := w.Installer.Refresh(game.ID, game.ActiveProfile)
	if err != nil {
		return err
	}
	profileDir, err := w.Lib.ProfileDir(game.ID, profile.ID)
	if err != nil {
		return err
	}

	session := Session{ProfileID: profile.ID, PID: os.Getpid(), StartedAt: time.Now(), Links: []string{}}
	command, env := plan.command, plan.env
	modded := false
	if game.Runtime == library.RuntimeNative {
		launcher, err := nativeLauncher(profileDir)
		if err != nil {
			return err
		}
		if launcher != "" {
			command = append([]string{launcher}, command...)
			modded = true
		}
	} else {
		session.Links, err = LinkProfile(game.Path, profileDir, w.Lib.Root())
		if err != nil {
			return err
		}
		if len(session.Links) > 0 {
			env = withDLLOverride(env, "winhttp", "n,b")
			modded = true
		}
	}
	if err := writeSession(dataDir, session); err != nil {
		Unlink(game.Path, session.Links, w.Lib.Root())
		return err
	}

	*plan = launchPlan{
		command:   command,
		env:       env,
		modded:    modded,
		profileID: profile.ID,
		cleanup: func() error {
			var reportErr error
			if modded {
				reportErr = SaveReport(dataDir, BuildReport(profileDir, session, profile.Mods))
			}
			return errors.Join(reportErr, Unlink(game.Path, session.Links, w.Lib.Root()), os.Remove(sessionPath(dataDir)))
		},
	}
	return nil
}

// nativeLauncher returns the BepInEx launcher script of a profile, made
// executable, or "" when the profile has no active loader.
func nativeLauncher(profileDir string) (string, error) {
	if _, err := os.Stat(filepath.Join(profileDir, "BepInEx", "core")); err != nil {
		return "", nil
	}
	for _, name := range nativeLaunchers {
		script := filepath.Join(profileDir, name)
		fi, err := os.Stat(script)
		if err != nil {
			continue
		}
		// Archives usually lose the executable bit.
		if fi.Mode()&0o111 != 0o111 {
			if err := os.Chmod(script, fi.Mode()|0o755); err != nil {
				return "", err
			}
		}
		return script, nil
	}
	return "", errors.New("the installed BepInEx pack has no Linux launcher (run_bepinex.sh or start_game_bepinex.sh); this game may only support BepInEx through Proton")
}

// findGame resolves the game from --game, Steam's SteamAppId variable, or the
// AppId=<n> argument Steam passes to its reaper process.
func (w *Wrapper) findGame(gameID string, command []string) (library.Game, error) {
	if gameID != "" {
		return w.Lib.GetGame(gameID)
	}
	appID := os.Getenv("SteamAppId")
	if appID == "" {
		for _, arg := range command {
			if m := appIDArg.FindStringSubmatch(arg); m != nil {
				appID = m[1]
				break
			}
		}
	}
	if appID == "" {
		return library.Game{}, errors.New("cannot tell which game is launched; pass --game <id>")
	}
	return w.Lib.FindGameBySteamAppID(appID)
}

// withDLLOverride adds a Wine DLL override unless the DLL is already configured.
func withDLLOverride(env []string, dll, mode string) []string {
	const key = "WINEDLLOVERRIDES="
	for i, kv := range env {
		value, ok := strings.CutPrefix(kv, key)
		if !ok {
			continue
		}
		for _, entry := range strings.Split(value, ";") {
			names, _, _ := strings.Cut(entry, "=")
			for _, name := range strings.Split(names, ",") {
				if strings.EqualFold(strings.TrimSpace(name), dll) {
					return env
				}
			}
		}
		if value != "" {
			value += ";"
		}
		out := append([]string(nil), env...)
		out[i] = key + value + dll + "=" + mode
		return out
	}
	return append(env, key+dll+"="+mode)
}

func runCommand(command, env []string, logger *log.Logger) int {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		logger.Printf("start %s: %v", command[0], err)
		return 127
	}

	// Forward termination signals so stopping the game from Steam works, while
	// the wrapper stays alive to clean up.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sigs:
				_ = cmd.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()

	err := cmd.Wait()
	signal.Stop(sigs)
	close(done)
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr):
		return exitErr.ExitCode()
	default:
		logger.Printf("wait: %v", err)
		return 1
	}
}

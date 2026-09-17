package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/app"
	"bepinexmodmanager/internal/launch"
	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/modinstall"
	"bepinexmodmanager/internal/platform"
	"bepinexmodmanager/internal/profileshare"
	"bepinexmodmanager/internal/settings"
	"bepinexmodmanager/internal/steam"
	"bepinexmodmanager/internal/thunderstore"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[modinstall.Progress](app.InstallProgressEvent)
	application.RegisterEvent[profileshare.ImportProgress](app.ImportProgressEvent)
}

type backend struct {
	root       string
	steamRoots []string
	lib        *library.Library
	ts         *thunderstore.Client
	installer  *modinstall.Installer
}

func newBackend() (*backend, error) {
	root := filepath.Join(xdg.DataHome, app.ID)
	steamRoots := steam.DefaultRoots()
	lib, err := library.New(root, steamRoots)
	if err != nil {
		return nil, err
	}
	ts := thunderstore.NewClient(
		thunderstore.DefaultBaseURL,
		app.ID+"/"+app.Version,
		filepath.Join(root, "cache", "thunderstore"),
	)
	rules := func(ctx context.Context, game library.Game) modinstall.Rules {
		schema, err := ts.Schema(ctx)
		if err != nil {
			log.Printf("install rules: %v; using defaults", err)
		}
		return modinstall.RulesFromSchema(schema, game.SteamAppID, game.Executable)
	}
	return &backend{
		root:       root,
		steamRoots: steamRoots,
		lib:        lib,
		ts:         ts,
		installer:  modinstall.NewInstaller(lib, ts, rules),
	}, nil
}

func main() {
	// "run -- <command>" is the Steam launch options wrapper; no GUI.
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(runWrapper(os.Args[2:]))
	}

	platform.ConfigureRendering()

	b, err := newBackend()
	if err != nil {
		log.Fatal(err)
	}
	settingsStore := settings.NewStore(b.root)

	wailsApp := application.New(application.Options{
		Name:        app.Name,
		Description: "Mod manager for BepInEx games",
		Services: []application.Service{
			application.NewService(app.NewInfoService()),
			application.NewService(b.lib),
			application.NewService(settingsStore),
			application.NewService(app.NewBrowseService(b.lib, b.ts, settingsStore)),
			application.NewService(app.NewInstallService(b.installer)),
			application.NewService(app.NewLaunchService(b.lib, b.steamRoots)),
			application.NewService(app.NewIconService(b.steamRoots)),
			application.NewService(app.NewShareService(
				profileshare.NewSharer(b.lib, b.installer, b.ts, filepath.Join(b.root, "cache", "imports")),
			)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            app.Name,
		Width:            1200,
		Height:           760,
		MinWidth:         900,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(15, 17, 23),
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func runWrapper(args []string) int {
	w := &launch.Wrapper{Notify: notify}
	b, err := newBackend()
	if err == nil {
		w.Lib, w.Installer = b.lib, b.installer
	}
	return w.Run(args)
}

func notify(summary, body string) {
	_ = exec.Command("notify-send", "--app-name="+app.Name, summary, body).Run()
}

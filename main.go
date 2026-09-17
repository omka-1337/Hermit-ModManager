package main

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/app"
	"bepinexmodmanager/internal/library"
	"bepinexmodmanager/internal/settings"
	"bepinexmodmanager/internal/steam"
	"bepinexmodmanager/internal/thunderstore"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	root := filepath.Join(xdg.DataHome, app.ID)
	lib, err := library.New(root, steam.DefaultRoots())
	if err != nil {
		log.Fatal(err)
	}

	settingsStore := settings.NewStore(root)
	ts := thunderstore.NewClient(
		thunderstore.DefaultBaseURL,
		app.ID+"/"+app.Version,
		filepath.Join(root, "cache", "thunderstore"),
	)

	wailsApp := application.New(application.Options{
		Name:        app.Name,
		Description: "Mod manager for BepInEx games",
		Services: []application.Service{
			application.NewService(app.NewInfoService()),
			application.NewService(lib),
			application.NewService(settingsStore),
			application.NewService(app.NewBrowseService(lib, ts, settingsStore)),
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

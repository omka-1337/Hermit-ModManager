package main

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/app"
	"bepinexmodmanager/internal/library"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	lib, err := library.New(filepath.Join(xdg.DataHome, app.ID))
	if err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        app.Name,
		Description: "Mod manager for BepInEx games",
		Services: []application.Service{
			application.NewService(app.NewInfoService()),
			application.NewService(lib),
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

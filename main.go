package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"bepinexmodmanager/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	wailsApp := application.New(application.Options{
		Name:        app.Name,
		Description: "Mod manager for BepInEx games",
		Services: []application.Service{
			application.NewService(app.NewInfoService()),
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

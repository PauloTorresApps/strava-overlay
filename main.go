package main

import (
	"embed"
	"log"
	"strava-overlay/internal/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {

	if err := config.LoadConfig(); err != nil {
		log.Fatalf("❌ Erro ao carregar configurações: %v", err)
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Strava Add Overlay",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 13, G: 17, B: 23, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

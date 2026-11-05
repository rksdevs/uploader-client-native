package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := InitDiagnosticLogger(); err != nil {
		println("Failed to init diagnostic log:", err.Error())
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "WoW Logs Uploader",
		Width:             1400,
		Height:            880,
		MinWidth:          960,
		MinHeight:         640,
		WindowStartState:  options.Normal,
		HideWindowOnClose: true,
		OnBeforeClose:     app.onBeforeClose,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 12, G: 10, B: 9, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		msg := err.Error()
		println("Error:", msg)
		WriteAppLog("ERROR", "wails", msg)
	}
	WriteAppLog("INFO", "shutdown", "main returned")
}

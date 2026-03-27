// Command desktop starts the pi-go desktop application (Wails + embedded frontend).
package main

import (
	"embed"
	"log"

	"github.com/ai-gateway/pi-go/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := app.New()

	if err := wails.Run(&options.App{
		Title:     "pi-go",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  a.OnStartup,
		OnDomReady: a.OnDomReady,
		OnShutdown: a.OnShutdown,
		Bind: []interface{}{
			a,
		},
	}); err != nil {
		log.Fatalf("desktop: %v", err)
	}
}

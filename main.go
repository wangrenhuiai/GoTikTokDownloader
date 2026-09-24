package main

import (
	"context"
	"embed"

	"gotiktokdownloader/backend/app"
	"gotiktokdownloader/backend/logging"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := logging.New(logging.DEBUG)
	a := app.New(logger, nil)
	wailsApp := a
	err := wails.Run(&options.App{
		Title:  "GoTikTokDownloader",
		Width:  1100,
		Height: 750,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		OnStartup: func(ctx context.Context) {
			wailsApp.Startup(ctx)
			wailsApp.SetEmitter(func(event string, data any) {
				wailsruntime.EventsEmit(ctx, event, data)
			})
		},
		OnShutdown: func(ctx context.Context) {
			wailsApp.Shutdown(ctx)
		},
		Bind: []interface{}{wailsApp},
	})
	if err != nil {
		logger.Errorf("wails run: %v", err)
	}
}

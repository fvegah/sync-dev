package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	appInstance := NewApp()

	// Create application with options
	app := application.New(application.Options{
		Name:        "SyncDev",
		Description: "Folder Sync for Mac",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// ActivationPolicyAccessory will be added in Plan 02-02
			// For now, keep as regular app
		},
	})

	// Create main window
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "SyncDev",
		Width:     1024,
		Height:    700,
		MinWidth:  800,
		MinHeight: 600,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				FullSizeContent:    true,
			},
			Appearance: application.NSAppearanceNameDarkAqua,
		},
	})

	// Hide instead of close - KEY for tray behavior
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	// Register app as a service for frontend bindings
	app.RegisterService(application.NewService(appInstance))

	// OnStartup equivalent
	appInstance.startup(app, window)

	// Register shutdown handler
	app.OnShutdown(func() {
		appInstance.shutdown()
	})

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

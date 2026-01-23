package main

import (
	"embed"
	"log"

	"SyncDev/internal/tray"

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
			// Run as menu bar app, no Dock icon
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Create main window (hidden by default for tray app)
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "SyncDev",
		Width:     1024,
		Height:    700,
		MinWidth:  800,
		MinHeight: 600,
		Hidden:    true, // Start hidden for menu bar app
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

	// Initialize tray manager
	trayManager := tray.NewManager(app, window, appInstance)

	// Register app as a service for frontend bindings
	app.RegisterService(application.NewService(appInstance))

	// OnStartup equivalent
	appInstance.startup(app, window, trayManager)

	// Register shutdown handler
	app.OnShutdown(func() {
		appInstance.shutdown()
	})

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

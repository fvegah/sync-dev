package tray

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// State represents the current sync state for icon display
type State int

const (
	StateIdle State = iota
	StateSyncing
	StateError
)

// Manager manages the system tray icon and menu
type Manager struct {
	app       *application.App
	systray   *application.SystemTray
	window    *application.WebviewWindow
	pauseItem *application.MenuItem
	isPaused  bool
}

// SyncActions defines the interface for sync operations
type SyncActions interface {
	SyncNow()
	IsPaused() bool
	Pause()
	Resume()
}

// NewManager creates a new tray manager
func NewManager(app *application.App, window *application.WebviewWindow, actions SyncActions) *Manager {
	m := &Manager{
		app:    app,
		window: window,
	}

	// Create system tray
	m.systray = app.SystemTray.New()
	m.systray.SetTooltip("SyncDev - Folder Sync")

	// Set initial icon (idle state)
	if runtime.GOOS == "darwin" {
		m.systray.SetTemplateIcon(IconIdle)
	}

	// Create context menu
	menu := app.NewMenu()

	menu.Add("Sync Now").OnClick(func(ctx *application.Context) {
		go actions.SyncNow()
	})

	m.pauseItem = menu.Add("Pause Sync")
	m.pauseItem.OnClick(func(ctx *application.Context) {
		if m.isPaused {
			actions.Resume()
			m.isPaused = false
			m.pauseItem.SetLabel("Pause Sync")
		} else {
			actions.Pause()
			m.isPaused = true
			m.pauseItem.SetLabel("Resume Sync")
		}
	})

	menu.AddSeparator()

	menu.Add("Open SyncDev").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})

	menu.AddSeparator()

	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	m.systray.SetMenu(menu)

	// Attach window for click-to-toggle behavior
	m.systray.AttachWindow(window).WindowOffset(5)

	return m
}

// SetState updates the tray icon based on sync state
func (m *Manager) SetState(state State) {
	if runtime.GOOS != "darwin" {
		return
	}

	var icon []byte
	switch state {
	case StateIdle:
		icon = IconIdle
	case StateSyncing:
		icon = IconSyncing
	case StateError:
		icon = IconError
	default:
		icon = IconIdle
	}
	m.systray.SetTemplateIcon(icon)
}

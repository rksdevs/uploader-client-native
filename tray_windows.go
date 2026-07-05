//go:build windows

package main

import (
	"context"
	"log"
	"sync"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type trayMenuItems struct {
	open       *systray.MenuItem
	pause      *systray.MenuItem
	resume     *systray.MenuItem
	quit       *systray.MenuItem
	statusLine *systray.MenuItem
}

var (
	trayMu          sync.Mutex
	trayMenu        trayMenuItems
	trayStatusText  = "WoW Logs Uploader"
	trayInitialized bool
	trayReady       bool
)

func (a *App) initSystemTray() {
	if trayInitialized {
		return
	}
	trayInitialized = true
	go systray.Run(func() { a.onTrayReady() }, a.onTrayExit)
}

func (a *App) onTrayReady() {
	if len(trayIconICO) > 0 {
		systray.SetIcon(trayIconICO)
	}
	systray.SetTitle("WoW Logs")

	trayMu.Lock()
	trayMenu.open = systray.AddMenuItem("Open WoW Logs Uploader", "Show the main window")
	trayMenu.statusLine = systray.AddMenuItem(trayStatusText, "Current auto-upload status")
	trayMenu.statusLine.Disable()
	systray.AddSeparator()
	trayMenu.pause = systray.AddMenuItem("Pause auto-upload", "Stop automatic combat log uploads")
	trayMenu.resume = systray.AddMenuItem("Resume auto-upload", "Resume automatic combat log uploads")
	trayMenu.resume.Disable()
	systray.AddSeparator()
	trayMenu.quit = systray.AddMenuItem("Quit", "Exit WoW Logs Uploader")
	trayReady = true
	trayMu.Unlock()

	a.updateTrayStatusLocked(trayStatusText)
	a.refreshTrayMenuState()

	go func() {
		for range trayMenu.open.ClickedCh {
			a.showMainWindow()
		}
	}()
	go func() {
		for range trayMenu.pause.ClickedCh {
			if err := a.DisableAutoUpload(); err != nil {
				log.Printf("[Tray] pause auto-upload: %v\n", err)
			}
			a.refreshTrayMenuState()
			a.updateTrayStatusFromWatcher()
		}
	}()
	go func() {
		for range trayMenu.resume.ClickedCh {
			if err := a.ResumeAutoUpload(); err != nil {
				log.Printf("[Tray] resume auto-upload: %v\n", err)
			}
			a.refreshTrayMenuState()
			a.updateTrayStatusFromWatcher()
		}
	}()
	go func() {
		for range trayMenu.quit.ClickedCh {
			a.quitFromTray()
		}
	}()

	a.updateTrayStatusFromWatcher()
}

func (a *App) onTrayExit() {
	trayMu.Lock()
	trayInitialized = false
	trayReady = false
	trayMenu = trayMenuItems{}
	trayMu.Unlock()
}

func (a *App) refreshTrayMenuState() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayMenu.pause == nil {
		return
	}
	enabled := a.config.AutoUploadEnabled
	if enabled {
		trayMenu.pause.Enable()
		trayMenu.resume.Disable()
	} else if a.hasTailBaseline() {
		trayMenu.pause.Disable()
		trayMenu.resume.Enable()
	} else {
		trayMenu.pause.Disable()
		trayMenu.resume.Disable()
	}
}

func (a *App) showMainWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

func (a *App) quitFromTray() {
	a.requestAppQuit()
}

func (a *App) shutdownSystemTray() {
	systray.Quit()
}

func (a *App) updateTrayStatus(tooltip string) {
	trayMu.Lock()
	defer trayMu.Unlock()
	a.updateTrayStatusLocked(tooltip)
}

func (a *App) updateTrayStatusLocked(tooltip string) {
	if tooltip == "" {
		tooltip = "WoW Logs Uploader"
	}
	trayStatusText = tooltip
	if !trayReady {
		return
	}
	systray.SetTooltip(tooltip)
	if trayMenu.statusLine != nil {
		trayMenu.statusLine.SetTitle(tooltip)
	}
}

func (a *App) updateTrayStatusFromWatcher() {
	status := a.GetAutoUploadWatcherStatus()
	a.updateTrayStatus(trayStatusLabel(status))
}

func trayStatusLabel(status AutoUploadWatcherStatus) string {
	switch status.Status {
	case "watching":
		return "Watching combat log"
	case "waiting_wow":
		if status.Detail != "" {
			return status.Detail
		}
		return "Waiting for WoW to close"
	case "waiting_stable":
		return "Waiting for log file to stabilize"
	case "uploading", "staging_ready":
		if status.Detail != "" {
			return status.Detail
		}
		return "Uploading combat log"
	case "awaiting_selection":
		return "Choose raid instances to upload"
	case "awaiting_server_drift":
		return "Confirm Warmane realm"
	case "paused":
		return "Auto-upload paused"
	case "error":
		if status.Detail != "" {
			return "Error: " + status.Detail
		}
		return "Auto-upload error"
	default:
		if !status.Running && status.Status == "idle" {
			return "WoW Logs Uploader"
		}
		if status.Detail != "" {
			return status.Detail
		}
		return "WoW Logs Uploader"
	}
}

func (a *App) onBeforeClose(ctx context.Context) bool {
	if a.isAppQuitting() || !a.minimizeToTrayEnabled() {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (a *App) minimizeToTrayEnabled() bool {
	return !a.config.DisableMinimizeToTray
}

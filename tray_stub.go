//go:build !windows

package main

import "context"

func (a *App) initSystemTray() {}

func (a *App) shutdownSystemTray() {}

func (a *App) updateTrayStatus(string) {}

func (a *App) updateTrayStatusFromWatcher() {}

func (a *App) showMainWindow() {}

func (a *App) onBeforeClose(ctx context.Context) bool {
	return false
}

func (a *App) minimizeToTrayEnabled() bool {
	return false
}

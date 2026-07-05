package main

import (
	"log"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type autoUploadDriftResolution struct {
	ServerName string
	Confirmed  bool
}

type autoUploadDriftWaiter struct {
	Check         *warmaneDriftCheck
	StagingPath   string
	SourceLogPath string
	SplitOffset   int64
	ResolveCh     chan autoUploadDriftResolution
}

func (a *App) setAutoUploadDriftPending(waiter *autoUploadDriftWaiter) {
	a.autoUploadDriftMu.Lock()
	a.autoUploadDriftPending = waiter
	a.autoUploadDriftMu.Unlock()
}

func (a *App) takeAutoUploadDriftPending() *autoUploadDriftWaiter {
	a.autoUploadDriftMu.Lock()
	defer a.autoUploadDriftMu.Unlock()
	waiter := a.autoUploadDriftPending
	a.autoUploadDriftPending = nil
	return waiter
}

func (a *App) peekAutoUploadDriftPending() *autoUploadDriftWaiter {
	a.autoUploadDriftMu.Lock()
	defer a.autoUploadDriftMu.Unlock()
	return a.autoUploadDriftPending
}

// awaitWarmaneServerDriftConfirmation blocks until the user confirms a server for upload.
func (a *App) awaitWarmaneServerDriftConfirmation(
	check *warmaneDriftCheck,
	stagingPath, sourceLogPath string,
	splitOffset int64,
) (serverName string, ok bool) {
	waiter := &autoUploadDriftWaiter{
		Check:         check,
		StagingPath:   stagingPath,
		SourceLogPath: sourceLogPath,
		SplitOffset:   splitOffset,
		ResolveCh:     make(chan autoUploadDriftResolution, 1),
	}
	a.setAutoUploadDriftPending(waiter)

	log.Printf("[AutoUpload] Warmane server drift: %s\n", formatWarmaneDriftLog(check))
	a.setWatcherStatus("awaiting_server_drift", "New log lines look like a different Warmane realm — confirm server.")
	if a.ctx != nil {
		a.showMainWindow()
		runtime.EventsEmit(a.ctx, "auto_upload_server_drift", map[string]interface{}{
			"defaultServer":     check.DefaultServer,
			"detectedServers":   check.DetectedServers,
			"guidPrefixByRealm": check.GuidPrefixByRealm,
			"multipleDetected":  check.MultipleDetected,
			"stagingPath":       stagingPath,
			"sourceLogPath":     sourceLogPath,
		})
	}

	resolution := <-waiter.ResolveCh
	a.takeAutoUploadDriftPending()

	if !resolution.Confirmed || strings.TrimSpace(resolution.ServerName) == "" {
		return "", false
	}
	return strings.TrimSpace(resolution.ServerName), true
}

func (a *App) resolveAutoUploadDrift(serverName string, confirmed bool) error {
	waiter := a.peekAutoUploadDriftPending()
	if waiter == nil || waiter.ResolveCh == nil {
		return nil
	}
	waiter.ResolveCh <- autoUploadDriftResolution{
		ServerName: strings.TrimSpace(serverName),
		Confirmed:  confirmed,
	}
	return nil
}

// ResolveAutoUploadServerDrift continues auto-upload with the chosen server.
func (a *App) ResolveAutoUploadServerDrift(serverName string) error {
	return a.resolveAutoUploadDrift(serverName, true)
}

// CancelAutoUploadServerDrift skips this upload cycle and keeps the staging slice for retry.
func (a *App) CancelAutoUploadServerDrift() error {
	return a.resolveAutoUploadDrift("", false)
}

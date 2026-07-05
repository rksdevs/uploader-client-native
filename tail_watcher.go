package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	wowClosedRequired      = 5 * time.Minute
	fileStableRequired     = 12 * time.Second
	autoUploadPollEvery    = 10 * time.Second
	autoUploadActivityEvery = 5 * time.Second
)

// AutoUploadWatcherStatus is exposed to the frontend for live watcher state.
type AutoUploadWatcherStatus struct {
	Running          bool                  `json:"running"`
	Status           string                `json:"status"`
	Detail           string                `json:"detail,omitempty"`
	LastStagingBytes int64                 `json:"lastStagingBytes,omitempty"`
	LastStagingAt    string                `json:"lastStagingAt,omitempty"`
	StagingPath      string                `json:"stagingPath,omitempty"`
	FileActivity     CombatLogFileActivity   `json:"fileActivity"`
}

type autoUploadWatcherRuntime struct {
	mu               sync.Mutex
	status           string
	detail           string
	lastStagingBytes int64
	lastStagingAt    time.Time
	stagingPath      string
	wowClosedAt      time.Time
	lastFileChangeAt time.Time
	lastStableSize   int64
	fileStableSince  time.Time
	lastStagedOffset     int64
	lastPremiumCheckAt   time.Time
}

func (a *App) initAutoUploadWatcherFields() {
	if a.autoUploadWatcher == nil {
		a.autoUploadWatcher = &autoUploadWatcherRuntime{
			status:           "idle",
			lastStagedOffset: -1,
		}
	}
}

func (a *App) initWowCloseTracking() {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()
	if isWowProcessRunning() {
		a.autoUploadWatcher.wowClosedAt = time.Time{}
		return
	}
	// WoW already closed when watcher starts (e.g. after reboot) — do not wait another 5 minutes.
	a.autoUploadWatcher.wowClosedAt = time.Now().Add(-wowClosedRequired - time.Second)
}

func (a *App) setWatcherStatus(status, detail string) {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	a.autoUploadWatcher.status = status
	a.autoUploadWatcher.detail = detail
	a.autoUploadWatcher.mu.Unlock()
	a.updateTrayStatusFromWatcher()
	a.emitAutoUploadStatus()
}

func (a *App) GetAutoUploadWatcherStatus() AutoUploadWatcherStatus {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()
	return AutoUploadWatcherStatus{
		Running:          a.autoUploadRunning,
		Status:           a.autoUploadWatcher.status,
		Detail:           a.autoUploadWatcher.detail,
		LastStagingBytes: a.autoUploadWatcher.lastStagingBytes,
		LastStagingAt:    formatTimeRFC3339(a.autoUploadWatcher.lastStagingAt),
		StagingPath:      a.autoUploadWatcher.stagingPath,
		FileActivity:     a.buildCombatLogActivityLocked(),
	}
}

func (a *App) buildCombatLogActivityLocked() CombatLogFileActivity {
	state, _ := a.loadTailState()
	logDir := strings.TrimSpace(a.config.LogDirectory)
	if logDir == "" {
		return CombatLogFileActivity{BaselineSize: state.SourceFileSize}
	}
	activity, err := computeCombatLogActivity(combatLogPath(logDir), state)
	if err != nil {
		return CombatLogFileActivity{BaselineSize: state.SourceFileSize}
	}

	running := isWowProcessRunning()
	activity.WowRunning = running
	if running {
		activity.WowClosedDetail = "WoW is running"
	} else if !a.autoUploadWatcher.wowClosedAt.IsZero() {
		closedFor := time.Since(a.autoUploadWatcher.wowClosedAt)
		activity.WowClosedReady = closedFor >= wowClosedRequired
		if activity.WowClosedReady {
			activity.WowClosedDetail = "WoW closed long enough"
		} else {
			remaining := (wowClosedRequired - closedFor).Round(time.Second)
			activity.WowClosedDetail = fmt.Sprintf("WoW closed — %s until staging", remaining)
		}
	} else {
		activity.WowClosedDetail = "Waiting for WoW to close"
	}

	if info, err := os.Stat(combatLogPath(logDir)); err == nil {
		now := time.Now()
		if info.Size() != a.autoUploadWatcher.lastStableSize {
			activity.FileStable = false
		} else if a.autoUploadWatcher.fileStableSince.IsZero() {
			activity.FileStable = false
		} else {
			activity.FileStable = now.Sub(a.autoUploadWatcher.fileStableSince) >= fileStableRequired
		}
	}

	return activity
}

func (a *App) emitAutoUploadStatus() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auto_upload_status", a.GetAutoUploadWatcherStatus())
	}
}

func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (a *App) startAutoUploadWatcher() {
	a.autoUploadMu.Lock()
	if a.autoUploadRunning {
		a.autoUploadMu.Unlock()
		return
	}
	stop := make(chan struct{})
	a.autoUploadStop = stop
	a.autoUploadRunning = true
	a.autoUploadMu.Unlock()

	a.setWatcherStatus("watching", "Monitoring combat log for changes.")

	go a.runAutoUploadWatcher(stop)
}

func (a *App) stopAutoUploadWatcher() {
	a.autoUploadMu.Lock()
	if !a.autoUploadRunning {
		a.autoUploadMu.Unlock()
		return
	}
	close(a.autoUploadStop)
	a.autoUploadStop = nil
	a.autoUploadRunning = false
	a.autoUploadMu.Unlock()
	a.setWatcherStatus("paused", "Auto-upload watcher stopped.")
}

func (a *App) restartAutoUploadWatcherIfNeeded() {
	if a.config.AutoUploadEnabled && a.hasTailBaseline() && a.autoUploadBlockReason(a.config.DefaultServer) == "" {
		a.stopAutoUploadWatcher()
		a.startAutoUploadWatcher()
	}
}

func (a *App) runAutoUploadWatcher(stop <-chan struct{}) {
	log.Println("[AutoUpload] Watcher started.")
	defer log.Println("[AutoUpload] Watcher stopped.")

	logDir := strings.TrimSpace(a.config.LogDirectory)
	if logDir == "" {
		a.setWatcherStatus("error", "Combat logs folder is not configured.")
		return
	}

	logPath := combatLogPath(logDir)
	state, err := a.loadTailState()
	if err != nil {
		a.setWatcherStatus("error", fmt.Sprintf("Could not load tail state: %v", err))
		return
	}
	if err := reconcileTailStateOnStartup(logPath, &state); err != nil {
		log.Printf("[AutoUpload] Startup reconcile warning: %v\n", err)
	} else if state.BaselineEstablishedAt != "" {
		_ = a.saveTailState(state)
	}

	a.initWowCloseTracking()
	a.initCombatLogStableTracking(logPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		a.setWatcherStatus("error", fmt.Sprintf("File watcher error: %v", err))
		return
	}
	defer watcher.Close()

	if err := watcher.Add(logDir); err != nil {
		a.setWatcherStatus("error", fmt.Sprintf("Could not watch logs folder: %v", err))
		return
	}

	ticker := time.NewTicker(autoUploadPollEvery)
	defer ticker.Stop()
	activityTicker := time.NewTicker(autoUploadActivityEvery)
	defer activityTicker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			a.evaluateAutoUploadCycle(logPath)
		case <-activityTicker.C:
			a.emitAutoUploadStatus()
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if a.isCombatLogEvent(ev.Name) {
				a.noteCombatLogChanged(logPath)
				a.evaluateAutoUploadCycle(logPath)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[AutoUpload] fsnotify error: %v\n", err)
		}
	}
}

func (a *App) isCombatLogEvent(name string) bool {
	return strings.EqualFold(filepath.Base(name), combatLogFileName)
}

func (a *App) initCombatLogStableTracking(logPath string) {
	a.initAutoUploadWatcherFields()
	info, err := os.Stat(logPath)
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()
	if err == nil {
		a.autoUploadWatcher.lastStableSize = info.Size()
		a.autoUploadWatcher.fileStableSince = time.Now()
	}
}

func (a *App) noteCombatLogChanged(logPath string) {
	a.initAutoUploadWatcherFields()
	info, err := os.Stat(logPath)
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()
	a.autoUploadWatcher.lastFileChangeAt = time.Now()
	a.autoUploadWatcher.fileStableSince = time.Time{}
	a.autoUploadWatcher.lastStagedOffset = -1
	if err == nil {
		a.autoUploadWatcher.lastStableSize = info.Size()
	}
}

func (a *App) evaluateAutoUploadCycle(logPath string) {
	if !a.config.AutoUploadEnabled {
		a.setWatcherStatus("paused", "Auto-upload is disabled.")
		return
	}
	if reason := a.autoUploadBlockReason(a.config.DefaultServer); reason != "" {
		a.setWatcherStatus("error", reason)
		return
	}

	if isWowProcessRunning() {
		a.initAutoUploadWatcherFields()
		a.autoUploadWatcher.mu.Lock()
		a.autoUploadWatcher.wowClosedAt = time.Time{}
		a.autoUploadWatcher.mu.Unlock()
		a.setWatcherStatus("waiting_wow", "WoW is running — waiting for client to close.")
		return
	}

	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	if a.autoUploadWatcher.wowClosedAt.IsZero() {
		a.autoUploadWatcher.wowClosedAt = time.Now()
	}
	wowClosedFor := time.Since(a.autoUploadWatcher.wowClosedAt)
	a.autoUploadWatcher.mu.Unlock()

	if wowClosedFor < wowClosedRequired {
		remaining := (wowClosedRequired - wowClosedFor).Round(time.Second)
		a.setWatcherStatus("waiting_wow", fmt.Sprintf("WoW closed — waiting %s before staging.", remaining))
		return
	}

	if !a.isCombatLogStable(logPath) {
		a.setWatcherStatus("waiting_stable", "Combat log is still changing — waiting for file to stabilize.")
		return
	}

	if a.shouldRecheckAutoUploadPremium() && !a.ensureAutoUploadPremiumActive() {
		return
	}

	if err := a.prepareAutoUploadStaging(logPath); err != nil {
		if err == errNoCombatLogDelta {
			state, _ := a.loadTailState()
			activity, _ := computeCombatLogActivity(logPath, state)
			if activity.HasPendingChanges {
				a.setWatcherStatus("watching", fmt.Sprintf("Detected %s of new log data — waiting for upload conditions.", formatBytes(activity.PendingBytes)))
			} else {
				a.setWatcherStatus("watching", "No new combat log lines since last tail.")
			}
			return
		}
		a.setWatcherStatus("error", err.Error())
		return
	}
}

func (a *App) isCombatLogStable(logPath string) bool {
	info, err := os.Stat(logPath)
	if err != nil {
		return false
	}

	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()

	now := time.Now()
	if info.Size() != a.autoUploadWatcher.lastStableSize {
		a.autoUploadWatcher.lastStableSize = info.Size()
		a.autoUploadWatcher.lastFileChangeAt = now
		a.autoUploadWatcher.fileStableSince = time.Time{}
		return false
	}

	if a.autoUploadWatcher.fileStableSince.IsZero() {
		a.autoUploadWatcher.fileStableSince = now
		return false
	}
	return now.Sub(a.autoUploadWatcher.fileStableSince) >= fileStableRequired
}

func (a *App) prepareAutoUploadStaging(logPath string) error {
	state, err := a.loadTailState()
	if err != nil {
		return err
	}

	split, err := findTailSplitOffset(logPath, state)
	if err != nil {
		return err
	}

	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	if split.SplitOffset == a.autoUploadWatcher.lastStagedOffset {
		a.autoUploadWatcher.mu.Unlock()
		return errNoCombatLogDelta
	}
	a.autoUploadWatcher.mu.Unlock()

	stagingDir, err := a.stagingDir()
	if err != nil {
		return err
	}
	stagingPath := filepath.Join(stagingDir, stagingFileName)

	written, err := copyCombatLogDelta(logPath, stagingPath, split.SplitOffset)
	if err != nil {
		return err
	}

	a.autoUploadWatcher.mu.Lock()
	a.autoUploadWatcher.lastStagingBytes = written
	a.autoUploadWatcher.lastStagingAt = time.Now()
	a.autoUploadWatcher.stagingPath = stagingPath
	a.autoUploadWatcher.lastStagedOffset = split.SplitOffset
	a.autoUploadWatcher.mu.Unlock()

	log.Printf("[AutoUpload] Staged %d bytes to %s (reanchored=%v)\n", written, stagingPath, split.Reanchored)
	a.setWatcherStatus("staging_ready", fmt.Sprintf("Staged %s — starting auto-upload.", formatBytes(written)))

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auto_upload_staging_ready", map[string]interface{}{
			"bytes":       written,
			"stagingPath": stagingPath,
			"reanchored":  split.Reanchored,
		})
	}

	a.autoUploadMu.Lock()
	if a.autoUploadUploading {
		a.autoUploadMu.Unlock()
		return nil
	}
	a.autoUploadUploading = true
	a.autoUploadMu.Unlock()

	go a.processStagedAutoUpload(stagingPath, logPath, split.SplitOffset)
	return nil
}

// PrepareAutoUploadStagingNow is a debug/manual trigger for Phase 2 verification.
func (a *App) PrepareAutoUploadStagingNow() (AutoUploadWatcherStatus, error) {
	logDir := strings.TrimSpace(a.config.LogDirectory)
	if logDir == "" {
		return a.GetAutoUploadWatcherStatus(), fmt.Errorf("combat logs folder is not configured")
	}
	logPath := combatLogPath(logDir)
	if err := a.prepareAutoUploadStaging(logPath); err != nil {
		return a.GetAutoUploadWatcherStatus(), err
	}
	return a.GetAutoUploadWatcherStatus(), nil
}

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type autoUploadJobContext struct {
	PreprocessID  int
	SourceLogPath string
	StagingPath   string
	SplitOffset   int64
}

func (a *App) clearAutoUploadInFlight() {
	a.autoUploadMu.Lock()
	a.autoUploadUploading = false
	a.autoUploadMu.Unlock()
}

func (a *App) resetStagedOffsetForRetry() {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	a.autoUploadWatcher.lastStagedOffset = -1
	a.autoUploadWatcher.mu.Unlock()
}

func (a *App) processStagedAutoUpload(stagingPath, sourceLogPath string, splitOffset int64) {
	if !a.ensureAutoUploadPremiumActive() {
		a.resetStagedOffsetForRetry()
		a.clearAutoUploadInFlight()
		return
	}

	serverName := strings.TrimSpace(a.config.DefaultServer)
	if serverName == "" {
		a.setWatcherStatus("error", "Default server is not configured for auto-upload.")
		a.resetStagedOffsetForRetry()
		a.clearAutoUploadInFlight()
		return
	}

	if scan, err := scanWarmaneRealmsFromCombatLog(stagingPath); err != nil {
		log.Printf("[AutoUpload] Warmane realm scan failed (continuing with default): %v\n", err)
	} else if drift := evaluateWarmaneServerDrift(serverName, scan); drift != nil {
		chosen, ok := a.awaitWarmaneServerDriftConfirmation(drift, stagingPath, sourceLogPath, splitOffset)
		if !ok {
			a.resetStagedOffsetForRetry()
			a.clearAutoUploadInFlight()
			a.setWatcherStatus("watching", "Server confirmation cancelled — will retry when conditions are met.")
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "auto_upload_drift_cancelled", map[string]interface{}{})
			}
			return
		}
		serverName = chosen
	}

	a.continueStagedAutoUpload(serverName, stagingPath, sourceLogPath, splitOffset)
}

func (a *App) continueStagedAutoUpload(serverName, stagingPath, sourceLogPath string, splitOffset int64) {
	a.setWatcherStatus("uploading", "Zipping staged combat log slice…")
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auto_upload_uploading", map[string]interface{}{
			"stagingPath": stagingPath,
			"serverName":  serverName,
		})
	}

	resp, err := a.uploadStagedAutoLog(serverName, stagingPath)
	if err != nil {
		code, statusCode, message := "", 0, err.Error()
		var apiErr *apiError
		if errors.As(err, &apiErr) {
			code, statusCode = apiErr.Code, apiErr.Status
		}

		log.Printf("[AutoUpload] auto-preprocess failed (status=%d code=%s): %s\n", statusCode, code, message)

		switch code {
		case "NO_BOSSES_FOUND":
			if advanceErr := a.advanceTailFingerprintFromLog(sourceLogPath); advanceErr != nil {
				log.Printf("[AutoUpload] Could not advance tail after no-bosses skip: %v\n", advanceErr)
			} else {
				removeStagingFile(stagingPath)
			}
			a.recordHistory(HistoryEntry{
				Source:     historySourceAuto,
				ServerName: serverName,
				StartedAt:  time.Now().UTC().Format(time.RFC3339),
				Status:     historySkipped,
				Message:    message,
				Raids:      []string{},
				Bosses:     []string{},
				LogIDs:     []int{},
			})
			a.setWatcherStatus("watching", "No raid bosses in new log slice — skipped.")
			a.clearAutoUploadInFlight()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "auto_upload_skipped", map[string]interface{}{
					"reason":  "no_bosses",
					"message": message,
				})
			}
			return
		case "MAINTENANCE_MODE":
			a.setWatcherStatus("error", message)
			a.resetStagedOffsetForRetry()
			a.clearAutoUploadInFlight()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "auto_upload_error", map[string]interface{}{"message": message, "code": code})
			}
			return
		case "AUTO_UPLOAD_PREMIUM_REQUIRED", "DEVICE_LIMIT_EXCEEDED", "AUTO_UPLOAD_TOKEN_REQUIRED":
			a.disableAutoUploadForPremiumLoss(message)
			a.clearAutoUploadInFlight()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "auto_upload_error", map[string]interface{}{"message": message, "code": code})
			}
			return
		case "RATE_LIMIT_EXCEEDED":
			a.setWatcherStatus("error", message)
			a.resetStagedOffsetForRetry()
			a.clearAutoUploadInFlight()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "auto_upload_error", map[string]interface{}{
					"message": message,
					"code":    code,
				})
			}
			return
		case "AUTO_UPLOAD_IN_PROGRESS":
			a.setWatcherStatus("watching", "Another auto-upload is still processing — will retry.")
			a.resetStagedOffsetForRetry()
			a.clearAutoUploadInFlight()
			return
		}

		a.setWatcherStatus("error", message)
		a.resetStagedOffsetForRetry()
		a.clearAutoUploadInFlight()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "auto_upload_error", map[string]interface{}{"message": message, "code": code})
		}
		return
	}

	autoCtx := &autoUploadJobContext{
		PreprocessID:  resp.PreprocessID,
		SourceLogPath: sourceLogPath,
		StagingPath:   stagingPath,
		SplitOffset:   splitOffset,
	}

	if resp.AutoQueued {
		log.Printf("[AutoUpload] Single instance auto-queued preprocessId=%d server=%s\n", resp.PreprocessID, serverName)
		a.setWatcherStatus("uploading", "Raid instance queued — processing in background.")
		state, _ := a.loadTailState()
		state.LastPreprocessID = resp.PreprocessID
		_ = a.saveTailState(state)
		a.startJobMonitor(resp.PreprocessID, autoCtx, "")
		return
	}

	if len(resp.Instances) == 0 {
		a.setWatcherStatus("error", "Preprocess returned no raid instances.")
		a.resetStagedOffsetForRetry()
		a.clearAutoUploadInFlight()
		return
	}

	log.Printf("[AutoUpload] Multiple instances (%d) — prompting user preprocessId=%d\n", len(resp.Instances), resp.PreprocessID)
	a.setWatcherStatus("awaiting_selection", "Multiple raid instances detected — choose which to upload.")
	if a.ctx != nil {
		a.showMainWindow()
		runtime.EventsEmit(a.ctx, "auto_upload_instances_ready", map[string]interface{}{
			"preprocessId":               resp.PreprocessID,
			"instances":                  resp.Instances,
			"hasMultipleDetectedServers": resp.HasMultipleDetectedServers,
			"defaultServer":              serverName,
			"stagingPath":                stagingPath,
			"sourceLogPath":              sourceLogPath,
		})
	}
}

func (a *App) uploadStagedAutoLog(serverName, stagingPath string) (*PreprocessResponse, error) {
	file, err := os.Open(stagingPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	a.setWatcherStatus("uploading", fmt.Sprintf("Uploading %s for preprocessing…", formatBytes(info.Size())))
	return a.uploadCombatLog(file, info.Size(), preprocessUploadOptions{
		Endpoint:   "/api/v5/uploader/auto-preprocess",
		ServerName: serverName,
		Source:     historySourceAuto,
		Prepare: func(req *http.Request) {
			req.Header.Set("X-Device-Id", a.ensureDeviceID())
		},
	})
}

func (a *App) rememberPendingAutoUpload(preprocessID int, autoCtx *autoUploadJobContext) {
	state, err := a.loadTailState()
	if err != nil {
		log.Printf("[AutoUpload] Could not load tail state: %v\n", err)
		return
	}
	state.PendingPreprocessID = preprocessID
	state.PendingStagingPath = autoCtx.StagingPath
	state.PendingSourcePath = autoCtx.SourceLogPath
	if err := a.saveTailState(state); err != nil {
		log.Printf("[AutoUpload] Could not remember pending upload: %v\n", err)
	}
}

func (a *App) clearPendingAutoUpload() {
	state, err := a.loadTailState()
	if err != nil {
		return
	}
	state.PendingPreprocessID = 0
	state.PendingStagingPath = ""
	state.PendingSourcePath = ""
	if err := a.saveTailState(state); err != nil {
		log.Printf("[AutoUpload] Could not clear pending upload: %v\n", err)
	}
}

// resumePendingAutoUpload picks up jobs left running when the app closed, so a finished upload
// still advances the tail instead of being staged and uploaded a second time.
func (a *App) resumePendingAutoUpload() {
	state, err := a.loadTailState()
	if err != nil || state.PendingPreprocessID == 0 {
		return
	}
	a.autoUploadMu.Lock()
	a.autoUploadUploading = true
	a.autoUploadMu.Unlock()
	log.Printf("[AutoUpload] Resuming job monitor for preprocessId=%d\n", state.PendingPreprocessID)
	a.startJobMonitor(state.PendingPreprocessID, &autoUploadJobContext{
		PreprocessID:  state.PendingPreprocessID,
		SourceLogPath: state.PendingSourcePath,
		StagingPath:   state.PendingStagingPath,
	}, "")
}

// keepStagingForRetry moves the slice out of the staging name the watcher reuses.
func keepStagingForRetry(stagingPath string, preprocessID int) (string, error) {
	retryPath := filepath.Join(filepath.Dir(stagingPath), fmt.Sprintf("retry-%d.txt", preprocessID))
	if err := os.Rename(stagingPath, retryPath); err != nil {
		return "", err
	}
	return retryPath, nil
}

func (a *App) handleAutoUploadJobsFinished(autoCtx *autoUploadJobContext, outcome jobOutcome) {
	if autoCtx == nil {
		return
	}
	defer a.clearAutoUploadInFlight()

	switch outcome {
	case jobOutcomeAllUploaded, jobOutcomePartial:
		a.clearPendingAutoUpload()
		if err := a.advanceTailFingerprintFromLog(autoCtx.SourceLogPath); err != nil {
			log.Printf("[AutoUpload] Could not advance tail after upload: %v\n", err)
		}
		if outcome == jobOutcomeAllUploaded {
			removeStagingFile(autoCtx.StagingPath)
			a.setWatcherStatus("watching", "Auto-upload complete — tail baseline advanced.")
		} else {
			// Retrying the whole slice is safe: the server drops raids it already has.
			retryPath, err := keepStagingForRetry(autoCtx.StagingPath, autoCtx.PreprocessID)
			if err != nil {
				log.Printf("[AutoUpload] Could not keep failed slice: %v\n", err)
			} else {
				a.setHistoryRetryPath(autoCtx.PreprocessID, retryPath)
			}
			a.setWatcherStatus("watching", "Some raids failed to process — retry them from upload history.")
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "auto_upload_complete", map[string]interface{}{})
		}
	case jobOutcomeTimedOut:
		// Not a failure: staging, tail and pending state stay so the next start resumes watching.
		a.setWatcherStatus("error", "Upload is still in the server queue — it will be checked again when the app restarts.")
	default:
		a.clearPendingAutoUpload()
		a.setWatcherStatus("error", "Auto-upload jobs failed — staging copy kept for retry.")
		a.resetStagedOffsetForRetry()
	}
}

// AbandonAutoUploadInstanceSelection releases the upload lock when the user cancels instance picking.
func (a *App) AbandonAutoUploadInstanceSelection() {
	a.resetStagedOffsetForRetry()
	a.clearAutoUploadInFlight()
	a.setWatcherStatus("watching", "Instance selection cancelled — will retry when conditions are met.")
}

// EnqueueAutoUploadJobs queues user-selected instances from the auto-upload instance picker.
func (a *App) EnqueueAutoUploadJobs(preprocessId int, selectedInstances []Instance, sourceLogPath, stagingPath string) (string, error) {
	result, err := a.EnqueueJobs(preprocessId, selectedInstances, false)
	if err != nil {
		return result, err
	}
	autoCtx := &autoUploadJobContext{
		PreprocessID:  preprocessId,
		SourceLogPath: sourceLogPath,
		StagingPath:   stagingPath,
	}
	a.startJobMonitor(preprocessId, autoCtx, "")
	return result, nil
}

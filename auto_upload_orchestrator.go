package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type autoUploadJobContext struct {
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

	zipBytes, err := zipStagingCombatLog(stagingPath)
	if err != nil {
		log.Printf("[AutoUpload] Zip failed: %v\n", err)
		a.setWatcherStatus("error", fmt.Sprintf("Could not zip staging file: %v", err))
		a.resetStagedOffsetForRetry()
		a.clearAutoUploadInFlight()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "auto_upload_error", map[string]interface{}{"message": err.Error()})
		}
		return
	}

	a.setWatcherStatus("uploading", fmt.Sprintf("Uploading %s for preprocessing…", formatBytes(int64(len(zipBytes)))))
	resp, statusCode, respBody, err := a.postAutoPreprocess(zipBytes, serverName)
	if err != nil {
		apiErr := parseAPIErrorBody(respBody)
		code := apiErr.Code
		message := apiErr.Message
		if message == "" {
			message = err.Error()
		}

		log.Printf("[AutoUpload] auto-preprocess failed (status=%d code=%s): %s\n", statusCode, code, message)

		switch code {
		case "NO_BOSSES_FOUND":
			if advanceErr := a.advanceTailFingerprintFromLog(sourceLogPath); advanceErr != nil {
				log.Printf("[AutoUpload] Could not advance tail after no-bosses skip: %v\n", advanceErr)
			} else {
				removeStagingFile(stagingPath)
			}
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

func (a *App) handleAutoUploadJobsFinished(autoCtx *autoUploadJobContext, anyUploaded bool) {
	if autoCtx == nil {
		return
	}
	defer a.clearAutoUploadInFlight()
	if anyUploaded {
		if err := a.advanceTailFingerprintFromLog(autoCtx.SourceLogPath); err != nil {
			log.Printf("[AutoUpload] Could not advance tail after upload: %v\n", err)
		}
		removeStagingFile(autoCtx.StagingPath)
		a.setWatcherStatus("watching", "Auto-upload complete — tail baseline advanced.")
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "auto_upload_complete", map[string]interface{}{})
		}
		return
	}
	a.setWatcherStatus("error", "Auto-upload jobs failed — staging copy kept for retry.")
	a.resetStagedOffsetForRetry()
}

// AbandonAutoUploadInstanceSelection releases the upload lock when the user cancels instance picking.
func (a *App) AbandonAutoUploadInstanceSelection() {
	a.resetStagedOffsetForRetry()
	a.clearAutoUploadInFlight()
	a.setWatcherStatus("watching", "Instance selection cancelled — will retry when conditions are met.")
}

// EnqueueAutoUploadJobs queues user-selected instances from the auto-upload instance picker.
func (a *App) EnqueueAutoUploadJobs(preprocessId int, selectedInstances []Instance, sourceLogPath, stagingPath string) (string, error) {
	result, err := a.EnqueueJobs(preprocessId, selectedInstances)
	if err != nil {
		return result, err
	}
	autoCtx := &autoUploadJobContext{
		SourceLogPath: sourceLogPath,
		StagingPath:   stagingPath,
	}
	a.startJobMonitor(preprocessId, autoCtx, "")
	return result, nil
}

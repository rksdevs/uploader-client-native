package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadRequest is a manual upload. Path picks a specific file; empty means WoWCombatLog.txt in
// LogDirectory. NewContentOnly sends only what was added since the last successful upload.
type UploadRequest struct {
	Path           string `json:"path"`
	LogDirectory   string `json:"logDirectory"`
	ServerName     string `json:"serverName"`
	Unlisted       bool   `json:"unlisted"`
	NewContentOnly bool   `json:"newContentOnly"`
}

// UploadPreview describes the log before upload. No line count: that would read the whole file.
type UploadPreview struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	SizeBytes   int64  `json:"sizeBytes"`
	ModifiedAt  string `json:"modifiedAt,omitempty"`
	HasPrevious bool   `json:"hasPrevious"`
	NewBytes    int64  `json:"newBytes"`
	WowRunning  bool   `json:"wowRunning"`
	TooLarge    bool   `json:"tooLarge"`
}

func resolveUploadPath(path, logDirectory string) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	return combatLogPath(logDirectory)
}

// manualTailState views the manual upload marker as a TailState so re-anchoring is shared with
// auto-upload. Falls back to the auto-upload baseline when no manual upload happened yet.
func manualTailState(state TailState) (TailState, bool) {
	if len(state.ManualFingerprint) > 0 {
		return TailState{
			TailFingerprint: state.ManualFingerprint,
			LastByteOffset:  state.ManualOffset,
			SourceFileSize:  state.ManualOffset,
		}, true
	}
	if state.BaselineEstablishedAt != "" && len(state.TailFingerprint) > 0 {
		return state, true
	}
	return TailState{}, false
}

func (a *App) GetUploadPreview(path, logDirectory string) (*UploadPreview, error) {
	logPath := resolveUploadPath(path, logDirectory)
	preview := &UploadPreview{Path: logPath, WowRunning: isWowProcessRunning()}
	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return preview, nil
		}
		return nil, err
	}
	preview.Exists = true
	preview.SizeBytes = info.Size()
	preview.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339)
	preview.NewBytes = info.Size()
	preview.TooLarge = info.Size() > maxUploadSourceBytes

	if strings.TrimSpace(path) != "" {
		return preview, nil
	}
	state, err := a.loadTailState()
	if err != nil {
		return preview, nil
	}
	if marker, ok := manualTailState(state); ok {
		preview.HasPrevious = true
		split, err := findTailSplitOffset(logPath, marker)
		switch {
		case errors.Is(err, errNoCombatLogDelta):
			preview.NewBytes = 0
		case err == nil:
			preview.NewBytes = info.Size() - split.SplitOffset
		}
	}
	return preview, nil
}

// UploadLog compresses, uploads and preprocesses a combat log.
func (a *App) UploadLog(request UploadRequest) (*PreprocessResponse, error) {
	logPath := resolveUploadPath(request.Path, request.LogDirectory)
	log.Printf("[Go Backend] UPLOAD: %s server=%s unlisted=%v newOnly=%v\n",
		logPath, request.ServerName, request.Unlisted, request.NewContentOnly)

	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", filepath.Base(logPath), err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var offset int64
	if request.NewContentOnly {
		state, err := a.loadTailState()
		if err != nil {
			return nil, err
		}
		if marker, ok := manualTailState(state); ok {
			split, err := findTailSplitOffset(logPath, marker)
			if errors.Is(err, errNoCombatLogDelta) {
				return nil, fmt.Errorf("nothing new to upload since your last upload")
			}
			if err != nil {
				return nil, err
			}
			offset = split.SplitOffset
		}
	}

	size := info.Size() - offset
	return a.uploadCombatLog(io.NewSectionReader(file, offset, size), size, preprocessUploadOptions{
		Endpoint:   "/api/v5/uploader/preprocess",
		ServerName: request.ServerName,
		Unlisted:   request.Unlisted,
		Source:     historySourceManual,
	})
}

// markUploadedThrough records the current end of the log after a successful manual upload, so the
// next "new since last upload" starts there. Keeps the auto-upload baseline in step too.
func (a *App) markUploadedThrough(logPath string) error {
	if err := a.advanceTailFingerprintFromLog(logPath); err != nil {
		return err
	}
	lines, _, fileSize, _, err := readTailFingerprint(logPath, tailFingerprintLineCount)
	if err != nil {
		return err
	}
	state, err := a.loadTailState()
	if err != nil {
		return err
	}
	state.ManualFingerprint = lines
	state.ManualOffset = fileSize
	return a.saveTailState(state)
}

// notifyUploadResult shows a desktop notification when the user enabled them.
func (a *App) notifyUploadResult(preprocessID int, outcome jobOutcome) {
	if !a.config.DesktopNotifications {
		return
	}
	title, message := uploadNotificationText(outcome)
	if err := showDesktopNotification(title, message); err != nil {
		log.Printf("[Go Backend] Notification failed for %d: %v\n", preprocessID, err)
	}
}

func uploadNotificationText(outcome jobOutcome) (string, string) {
	switch outcome {
	case jobOutcomeAllUploaded:
		return "Upload complete", "Your log is ready on wow-logs.co.in."
	case jobOutcomePartial:
		return "Upload partly complete", "Some raids failed to process. Open the app to retry."
	case jobOutcomeAllFailed:
		return "Upload failed", "The server could not process your log."
	default:
		return "Upload still queued", "The server is busy. Check View all logs later."
	}
}

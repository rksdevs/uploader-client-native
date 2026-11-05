package main

import (
	"log"
	"os"
	"strings"
	"time"
)

func (a *App) advanceTailFingerprintFromLog(logPath string) error {
	lines, _, fileSize, modTime, err := readTailFingerprint(logPath, tailFingerprintLineCount)
	if err != nil {
		return err
	}

	state, err := a.loadTailState()
	if err != nil {
		return err
	}
	if state.BaselineEstablishedAt == "" {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state.TailFingerprint = append([]string(nil), lines...)
	state.LastByteOffset = fileSize
	state.SourceFileSize = fileSize
	state.SourceFileMtimeUnix = modTime
	state.LastUploadAt = now
	return a.saveTailState(state)
}

func removeStagingFile(stagingPath string) {
	if strings.TrimSpace(stagingPath) == "" {
		return
	}
	if err := os.Remove(stagingPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[AutoUpload] Could not delete staging file %s: %v\n", stagingPath, err)
	}
}

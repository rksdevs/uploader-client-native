package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// CombatLogFileActivity summarizes whether the source combat log changed since the tail baseline.
type CombatLogFileActivity struct {
	FileExists        bool   `json:"fileExists"`
	CurrentSize       int64  `json:"currentSize"`
	BaselineSize      int64  `json:"baselineSize"`
	PendingBytes      int64  `json:"pendingBytes"`
	HasPendingChanges bool   `json:"hasPendingChanges"`
	LastModified      string `json:"lastModified,omitempty"`
	LastModifiedUnix  int64  `json:"lastModifiedUnix,omitempty"`
	LastLinePreview   string `json:"lastLinePreview,omitempty"`
	WowRunning        bool   `json:"wowRunning"`
	WowClosedReady    bool   `json:"wowClosedReady"`
	WowClosedDetail   string `json:"wowClosedDetail,omitempty"`
	FileStable        bool   `json:"fileStable"`
}

func computeCombatLogActivity(logPath string, state TailState) (CombatLogFileActivity, error) {
	activity := CombatLogFileActivity{
		BaselineSize: state.SourceFileSize,
	}

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return activity, nil
		}
		return activity, err
	}

	activity.FileExists = true
	activity.CurrentSize = info.Size()
	activity.LastModifiedUnix = info.ModTime().Unix()
	activity.LastModified = info.ModTime().Local().Format(time.RFC3339)

	syncOffset := state.LastByteOffset
	if syncOffset < 0 {
		syncOffset = 0
	}
	if activity.CurrentSize > syncOffset {
		activity.PendingBytes = activity.CurrentSize - syncOffset
		activity.HasPendingChanges = true
	}

	if preview, err := readLastNonEmptyLine(logPath); err == nil {
		activity.LastLinePreview = preview
	}

	return activity, nil
}

func readLastNonEmptyLine(logPath string) (string, error) {
	info, err := os.Stat(logPath)
	if err != nil || info.Size() == 0 {
		return "", err
	}

	const tailRead = 64 * 1024
	start := info.Size() - tailRead
	if start < 0 {
		start = 0
	}

	f, err := os.Open(logPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Seek(start, 0); err != nil {
		return "", err
	}
	buf := make([]byte, info.Size()-start)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	buf = buf[:n]

	lines := strings.Split(string(buf), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			if len(line) > 160 {
				return line[:160] + "…", nil
			}
			return line, nil
		}
	}
	return "", fmt.Errorf("no lines")
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

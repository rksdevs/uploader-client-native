package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var errNoCombatLogDelta = fmt.Errorf("no new combat log content since last tail")

// tailSplitResult describes where to start copying new log content.
type tailSplitResult struct {
	SplitOffset int64
	Reanchored  bool
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// findTailSplitOffset locates the byte offset where new content begins after the stored fingerprint.
func findTailSplitOffset(logPath string, state TailState) (tailSplitResult, error) {
	info, err := os.Stat(logPath)
	if err != nil {
		return tailSplitResult{}, err
	}
	if info.Size() == 0 {
		return tailSplitResult{}, errNoCombatLogDelta
	}

	syncOffset := state.LastByteOffset
	if syncOffset < 0 {
		syncOffset = 0
	}

	// Fast path: LastByteOffset is the synced EOF from baseline / last successful tail.
	if info.Size() > syncOffset {
		reanchored := info.Size() < state.SourceFileSize
		return tailSplitResult{SplitOffset: syncOffset, Reanchored: reanchored}, nil
	}
	if info.Size() == syncOffset {
		return tailSplitResult{}, errNoCombatLogDelta
	}

	// File shrank — try to re-anchor using the stored fingerprint.
	if len(state.TailFingerprint) == 0 {
		return tailSplitResult{}, errNoCombatLogDelta
	}

	if matchEnd, found := findFingerprintEndOffset(logPath, state.TailFingerprint, 0); found {
		if matchEnd >= info.Size() {
			return tailSplitResult{}, errNoCombatLogDelta
		}
		return tailSplitResult{SplitOffset: matchEnd, Reanchored: true}, nil
	}

	return tailSplitResult{}, fmt.Errorf("could not locate tail baseline in combat log")
}

// findFingerprintEndOffset returns the byte offset immediately after the matched fingerprint block.
func findFingerprintEndOffset(logPath string, fingerprint []string, minOffset int64) (int64, bool) {
	file, err := os.Open(logPath)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	var (
		offset      int64
		window      []string
		windowStart []int64
		lastMatch   int64 = -1
	)

	appendWindow := func(line string, lineStart int64) {
		window = append(window, line)
		windowStart = append(windowStart, lineStart)
		if len(window) > len(fingerprint) {
			window = window[1:]
			windowStart = windowStart[1:]
		}
		if len(window) == len(fingerprint) && linesEqual(window, fingerprint) {
			end := lineStart + int64(len(line)) + 1
			if lineStart >= minOffset || minOffset == 0 {
				lastMatch = end
			}
		}
	}

	for scanner.Scan() {
		lineStart := offset
		line := scanner.Text()
		offset += int64(len(line)) + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		appendWindow(line, lineStart)
	}
	if err := scanner.Err(); err != nil {
		return 0, false
	}
	if lastMatch < 0 {
		return 0, false
	}
	return lastMatch, true
}

// copyCombatLogDelta streams bytes from splitOffset to EOF into the staging file.
func copyCombatLogDelta(logPath, stagingPath string, splitOffset int64) (int64, error) {
	src, err := os.Open(logPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return 0, err
	}
	if splitOffset >= info.Size() {
		return 0, errNoCombatLogDelta
	}
	if _, err := src.Seek(splitOffset, io.SeekStart); err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(stagingPath), 0755); err != nil {
		return 0, err
	}
	dst, err := os.Create(stagingPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, err
	}
	if written == 0 {
		return 0, errNoCombatLogDelta
	}
	return written, nil
}

// reconcileTailStateOnStartup adjusts tail state when the source log changed on disk while the app was closed.
func reconcileTailStateOnStartup(logPath string, state *TailState) error {
	if state == nil || state.BaselineEstablishedAt == "" {
		return nil
	}

	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Legacy baselines stored the fingerprint block start instead of synced EOF.
	if state.LastByteOffset > 0 && state.LastByteOffset < state.SourceFileSize && info.Size() >= state.SourceFileSize {
		state.LastByteOffset = state.SourceFileSize
	}

	// Append-only growth: keep synced offset; pending bytes = current size - LastByteOffset.
	if info.Size() >= state.LastByteOffset {
		state.SourceFileSize = info.Size()
		state.SourceFileMtimeUnix = info.ModTime().Unix()
		return nil
	}

	// File shrank — re-anchor using the stored fingerprint.
	if len(state.TailFingerprint) == 0 {
		return nil
	}

	if end, found := findFingerprintEndOffset(logPath, state.TailFingerprint, 0); found {
		state.LastByteOffset = end
		state.SourceFileSize = info.Size()
		state.SourceFileMtimeUnix = info.ModTime().Unix()
		return nil
	}

	lines, _, fileSize, modTime, err := readTailFingerprint(logPath, tailFingerprintLineCount)
	if err != nil {
		return err
	}
	state.TailFingerprint = lines
	state.LastByteOffset = fileSize
	state.SourceFileSize = fileSize
	state.SourceFileMtimeUnix = modTime
	return nil
}

// parseWotLKLineTimestamp extracts the timestamp prefix from a legacy combat log line.
func parseWotLKLineTimestamp(line string) (time.Time, bool) {
	parts := strings.SplitN(line, "  ", 2)
	if len(parts) < 1 {
		return time.Time{}, false
	}
	ts := strings.TrimSpace(parts[0])
	layouts := []string{"1/2 15:04:05.000", "01/02 15:04:05.000"}
	year := time.Now().Year()
	for _, layout := range layouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local), true
		}
	}
	return time.Time{}, false
}

// fingerprintFromBytes builds the last N non-empty lines from a byte slice (used in tests).
func fingerprintFromBytes(content []byte, lineCount int) ([]string, int64, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	type lineRecord struct {
		text   string
		offset int64
	}
	var (
		records []lineRecord
		offset  int64
	)
	for scanner.Scan() {
		lineStart := offset
		line := scanner.Text()
		offset += int64(len(line)) + 1
		if strings.TrimSpace(line) == "" {
			continue
		}
		records = append(records, lineRecord{text: line, offset: lineStart})
		if len(records) > lineCount {
			records = records[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return nil, 0, fmt.Errorf("no lines")
	}
	lines := make([]string, len(records))
	for i, rec := range records {
		lines[i] = rec.text
	}
	return lines, records[0].offset, nil
}

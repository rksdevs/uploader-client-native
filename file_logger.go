package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxDiagnosticLogBytes = 1 << 20 // 1 MiB; oldest lines dropped from the tail

var (
	diagLogMu     sync.Mutex
	diagLogPath   string
	diagLogReady  bool
	diagLogStderr io.Writer = os.Stderr
)

// InitDiagnosticLogger prepares %AppData%/WoWLogsUploader/logs/uploader.log and routes
// standard library log output into it (newest lines at the top of the file).
func InitDiagnosticLogger() error {
	dir, err := diagnosticLogDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	diagLogPath = filepath.Join(dir, "uploader.log")
	diagLogReady = true

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetOutput(io.MultiWriter(diagLogStderr, &stdLogBridge{}))

	WriteAppLog("INFO", "startup", fmt.Sprintf("diagnostic log file %s (max %d bytes, newest first)", diagLogPath, maxDiagnosticLogBytes))
	return nil
}

func diagnosticLogDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "WoWLogsUploader", "logs"), nil
}

func diagnosticLogPath() string {
	return diagLogPath
}

// WriteAppLog appends a structured line to the diagnostic log (newest first).
func WriteAppLog(level, category, message string) {
	if !diagLogReady || diagLogPath == "" {
		return
	}
	if level == "" {
		level = "INFO"
	}
	if category == "" {
		category = "app"
	}
	prependDiagnosticLine(formatLogLine(level, category, message))
}

func formatLogLine(level, category, message string) string {
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	msg := strings.TrimSpace(message)
	msg = strings.ReplaceAll(msg, "\r\n", " ")
	msg = strings.ReplaceAll(msg, "\n", " ")
	return fmt.Sprintf("%s [%s] [%s] %s", ts, strings.ToUpper(level), category, msg)
}

func prependDiagnosticLine(line string) {
	entry := line
	if !strings.HasSuffix(entry, "\n") {
		entry += "\n"
	}

	diagLogMu.Lock()
	defer diagLogMu.Unlock()

	existing, err := os.ReadFile(diagLogPath)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	combined := append([]byte(entry), existing...)
	if len(combined) > maxDiagnosticLogBytes {
		combined = combined[:maxDiagnosticLogBytes]
		if idx := bytes.LastIndex(combined, []byte{'\n'}); idx > 0 {
			combined = combined[:idx+1]
		}
	}

	_ = os.WriteFile(diagLogPath, combined, 0644)
}

// stdLogBridge forwards log.Printf output into the diagnostic file.
type stdLogBridge struct{}

func (stdLogBridge) Write(p []byte) (int, error) {
	text := strings.TrimSpace(string(p))
	if text != "" {
		prependDiagnosticLine(text)
	}
	return len(p), nil
}

// LogClientMessage is exposed to the React UI for console-level diagnostics.
func (a *App) LogClientMessage(level string, message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	WriteAppLog(level, "frontend", message)
}

// GetDiagnosticLogPath returns the absolute path to uploader.log for support staff.
func (a *App) GetDiagnosticLogPath() string {
	return diagnosticLogPath()
}

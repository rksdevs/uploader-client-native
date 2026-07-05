package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	blockedAutoUploadServer  = "Whitemane_Gilneas"
	tailFingerprintLineCount = 15
	combatLogFileName        = "WoWCombatLog.txt"
	stagingFileName          = "wow-logs-combatlog.txt"
)

// AutoUploadSettingsResponse is returned to the frontend for the auto-upload panel.
type AutoUploadSettingsResponse struct {
	Enabled                 bool     `json:"enabled"`
	DefaultServer           string   `json:"defaultServer"`
	DeviceID                string   `json:"deviceId"`
	HasBaseline             bool     `json:"hasBaseline"`
	BaselineEstablishedAt   string   `json:"baselineEstablishedAt,omitempty"`
	TailFingerprint         []string `json:"tailFingerprint,omitempty"`
	LogDirectory            string   `json:"logDirectory"`
	HasAPIToken             bool     `json:"hasApiToken"`
	CanEnable               bool     `json:"canEnable"`
	ServerAllowed           bool     `json:"serverAllowed"`
	BlockReason             string   `json:"blockReason,omitempty"`
	Watcher                 AutoUploadWatcherStatus `json:"watcher"`
	MinimizeToTray          bool     `json:"minimizeToTray"`
}

// BaselinePreview is shown when the user establishes the auto-upload tail baseline.
type BaselinePreview struct {
	Lines                   []string `json:"lines"`
	BaselineEstablishedAt   string   `json:"baselineEstablishedAt"`
	SourceFileSize          int64    `json:"sourceFileSize"`
	LastByteOffset          int64    `json:"lastByteOffset"`
	Message                 string   `json:"message"`
}

func (a *App) stagingDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not resolve app directory: %w", err)
	}
	dir := filepath.Join(filepath.Dir(exe), "staging")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) ensureDeviceID() string {
	if strings.TrimSpace(a.config.DeviceID) != "" {
		return a.config.DeviceID
	}
	a.config.DeviceID = uuid.NewString()
	_ = a.saveConfig()
	return a.config.DeviceID
}

// IsServerAllowedForAutoUpload reports whether auto-upload is supported for the given server name.
func (a *App) IsServerAllowedForAutoUpload(serverName string) bool {
	return strings.TrimSpace(serverName) != "" && serverName != blockedAutoUploadServer
}

func (a *App) autoUploadBlockReason(serverName string) string {
	if strings.TrimSpace(serverName) == "" {
		return "Select a default server for auto-upload."
	}
	if serverName == blockedAutoUploadServer {
		return "Auto-upload is not available for Whitemane_Gilneas (Cataclysm Classic). Use manual upload."
	}
	if strings.TrimSpace(a.config.LogDirectory) == "" {
		return "Select your combat logs folder first."
	}
	if strings.TrimSpace(a.config.ApiToken) == "" {
		return "Add your premium API token in Premium settings."
	}
	return ""
}

// GetAutoUploadSettings returns current auto-upload configuration for the UI.
func (a *App) GetAutoUploadSettings() AutoUploadSettingsResponse {
	state, _ := a.loadTailState()
	server := a.config.DefaultServer
	resp := AutoUploadSettingsResponse{
		Enabled:               a.config.AutoUploadEnabled,
		DefaultServer:         server,
		DeviceID:              a.ensureDeviceID(),
		HasBaseline:           a.hasTailBaseline(),
		BaselineEstablishedAt: state.BaselineEstablishedAt,
		TailFingerprint:       state.TailFingerprint,
		LogDirectory:          a.config.LogDirectory,
		HasAPIToken:           strings.TrimSpace(a.config.ApiToken) != "",
		ServerAllowed:         a.IsServerAllowedForAutoUpload(server),
	}
	resp.BlockReason = a.autoUploadBlockReason(server)
	resp.CanEnable = resp.BlockReason == ""
	resp.Watcher = a.GetAutoUploadWatcherStatus()
	resp.MinimizeToTray = a.GetMinimizeToTray()
	return resp
}

// SaveDefaultServer persists the default server used by auto-upload.
func (a *App) SaveDefaultServer(serverName string) error {
	a.config.DefaultServer = strings.TrimSpace(serverName)
	return a.saveConfig()
}

// DisableAutoUpload turns off automatic upload without clearing the tail baseline.
func (a *App) DisableAutoUpload() error {
	a.config.AutoUploadEnabled = false
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.stopAutoUploadWatcher()
	a.updateTrayStatusFromWatcher()
	return nil
}

// ResumeAutoUpload re-enables auto-upload when a tail baseline already exists.
func (a *App) ResumeAutoUpload() error {
	if !a.hasTailBaseline() {
		return fmt.Errorf("enable auto-upload from settings first to establish a baseline")
	}
	if reason := a.autoUploadBlockReason(a.config.DefaultServer); reason != "" {
		return fmt.Errorf("%s", reason)
	}
	a.config.AutoUploadEnabled = true
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.startAutoUploadWatcher()
	a.updateTrayStatusFromWatcher()
	return nil
}

// GetMinimizeToTray returns whether closing the window minimizes to the system tray.
func (a *App) GetMinimizeToTray() bool {
	return a.minimizeToTrayEnabled()
}

// SetMinimizeToTray toggles close-to-tray behaviour.
func (a *App) SetMinimizeToTray(enabled bool) error {
	a.config.DisableMinimizeToTray = !enabled
	return a.saveConfig()
}

func combatLogPath(logDirectory string) string {
	return filepath.Join(logDirectory, combatLogFileName)
}

// readTailFingerprint scans a combat log and returns the last N non-empty lines plus file metadata.
func readTailFingerprint(logPath string, lineCount int) (lines []string, lastByteOffset int64, fileSize int64, modTime int64, err error) {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	fileSize = info.Size()
	modTime = info.ModTime().Unix()

	file, err := os.Open(logPath)
	if err != nil {
		return nil, 0, fileSize, modTime, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	type lineRecord struct {
		text   string
		offset int64
	}
	var records []lineRecord
	var offset int64

	for scanner.Scan() {
		lineStart := offset
		line := scanner.Text()
		offset += int64(len(line)) + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		records = append(records, lineRecord{text: line, offset: lineStart})
		if len(records) > lineCount {
			records = records[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fileSize, modTime, err
	}
	if len(records) == 0 {
		return nil, 0, fileSize, modTime, fmt.Errorf("combat log is empty")
	}

	lines = make([]string, len(records))
	for i, rec := range records {
		lines[i] = rec.text
	}
	return lines, records[0].offset, fileSize, modTime, nil
}

// EstablishAutoUploadBaseline captures the current EOF tail as the starting point for auto-upload.
func (a *App) EstablishAutoUploadBaseline(serverName string) (*BaselinePreview, error) {
	serverName = strings.TrimSpace(serverName)
	if !a.IsServerAllowedForAutoUpload(serverName) {
		if serverName == blockedAutoUploadServer {
			return nil, fmt.Errorf("auto-upload is not supported for Whitemane_Gilneas")
		}
		return nil, fmt.Errorf("default server is required")
	}
	if strings.TrimSpace(a.config.LogDirectory) == "" {
		return nil, fmt.Errorf("combat logs folder is not configured")
	}
	if strings.TrimSpace(a.config.ApiToken) == "" {
		return nil, fmt.Errorf("premium API token is required for auto-upload")
	}

	logPath := combatLogPath(a.config.LogDirectory)
	lines, _, fileSize, modTime, err := readTailFingerprint(logPath, tailFingerprintLineCount)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", combatLogFileName, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state := TailState{
		TailFingerprint:       append([]string(nil), lines...),
		LastByteOffset:        fileSize,
		SourceFileSize:        fileSize,
		SourceFileMtimeUnix:   modTime,
		BaselineEstablishedAt: now,
	}
	if err := a.saveTailState(state); err != nil {
		return nil, fmt.Errorf("could not save tail state: %w", err)
	}

	a.config.DefaultServer = serverName
	a.config.AutoUploadEnabled = true
	if err := a.saveConfig(); err != nil {
		return nil, err
	}

	a.startAutoUploadWatcher()
	a.updateTrayStatusFromWatcher()

	preview := &BaselinePreview{
		Lines:                 lines,
		BaselineEstablishedAt: now,
		SourceFileSize:        fileSize,
		LastByteOffset:        fileSize,
		Message:               "Auto-upload enabled. Tailing from the events below; only new combat after this point will be uploaded automatically.",
	}
	return preview, nil
}

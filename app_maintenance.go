package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	addonSyncFileName = "addon-sync.json"
	addonReleasesAPI  = "https://api.github.com/repos/rksdevs/WowLogsAddon/releases/latest"
	addonDownloadPage = "https://github.com/rksdevs/WowLogsAddon/releases/latest"
)

// ArchiveCombatLog moves WoWCombatLog.txt to Logs/archive so the next raid starts a small file.
// WoW keeps the file open while running, so it must be closed first.
func (a *App) ArchiveCombatLog(logDirectory string) (string, error) {
	if isWowProcessRunning() {
		return "", fmt.Errorf("close WoW before archiving the combat log")
	}
	archived, err := archiveLogFile(combatLogPath(logDirectory), time.Now())
	if err != nil {
		return "", err
	}
	state, err := a.loadTailState()
	if err != nil {
		return archived, err
	}
	return archived, a.saveTailState(tailStateAfterArchive(state))
}

func archiveLogFile(logPath string, now time.Time) (string, error) {
	if _, err := os.Stat(logPath); err != nil {
		return "", fmt.Errorf("could not find the combat log: %w", err)
	}
	dir := filepath.Join(filepath.Dir(logPath), "archive")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	archived := filepath.Join(dir, fmt.Sprintf("WoWCombatLog-%s.txt", now.Format("2006-01-02-150405")))
	if err := os.Rename(logPath, archived); err != nil {
		return "", fmt.Errorf("could not archive the combat log: %w", err)
	}
	return archived, nil
}

// tailStateAfterArchive treats everything in the next file as new.
func tailStateAfterArchive(state TailState) TailState {
	state.TailFingerprint = nil
	state.LastByteOffset = 0
	state.SourceFileSize = 0
	state.ManualFingerprint = nil
	state.ManualOffset = 0
	return state
}

// UpdateInfo is the result of an update check.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Required  bool   `json:"required"`
	Current   string `json:"current"`
	Version   string `json:"version"`
	Notes     string `json:"notes,omitempty"`
}

type releaseInfo struct {
	Version             string `json:"version"`
	MinSupportedVersion string `json:"minSupportedVersion"`
	DownloadURL         string `json:"downloadUrl"`
	SHA256              string `json:"sha256"`
	Notes               string `json:"notes"`
}

// compareVersions returns -1, 0 or 1 for dotted numeric versions.
func compareVersions(a, b string) int {
	left := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	right := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")
	for i := 0; i < len(left) || i < len(right); i++ {
		l, r := 0, 0
		if i < len(left) {
			l, _ = strconv.Atoi(left[i])
		}
		if i < len(right) {
			r, _ = strconv.Atoi(right[i])
		}
		if l != r {
			if l < r {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (a *App) fetchRelease() (*releaseInfo, error) {
	body, err := a.fetchJSONGET("/api/v5/uploader/app-version")
	if err != nil {
		return nil, err
	}
	var release releaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("could not read release info: %w", err)
	}
	return &release, nil
}

func updateInfoFor(current string, release *releaseInfo) UpdateInfo {
	info := UpdateInfo{Current: current, Version: release.Version, Notes: release.Notes}
	info.Available = release.DownloadURL != "" && compareVersions(current, release.Version) < 0
	info.Required = info.Available && release.MinSupportedVersion != "" &&
		compareVersions(current, release.MinSupportedVersion) < 0
	return info
}

// CheckForUpdate asks the server for the latest release.
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	release, err := a.fetchRelease()
	if err != nil {
		return nil, err
	}
	info := updateInfoFor(AppVersion, release)
	return &info, nil
}

// downloadVerified saves url to destination, deleting it unless the sha256 matches. A release with
// no checksum or a non-https link is refused: this file is about to be executed.
func downloadVerified(client *http.Client, url, wantSHA256, destination string, allowHTTP bool) error {
	if strings.TrimSpace(wantSHA256) == "" {
		return fmt.Errorf("the release has no checksum")
	}
	if !allowHTTP && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("the release link is not https")
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(out, hash), resp.Body)
	closeErr := out.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr == nil && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimSpace(wantSHA256)) {
		copyErr = fmt.Errorf("the download failed its checksum")
	}
	if copyErr != nil {
		os.Remove(destination)
		return copyErr
	}
	return nil
}

// swapExecutable puts the new build in place. Windows allows renaming a running exe.
func swapExecutable(current, downloaded string) error {
	old := current + ".old"
	os.Remove(old)
	if err := os.Rename(current, old); err != nil {
		return err
	}
	if err := os.Rename(downloaded, current); err != nil {
		_ = os.Rename(old, current)
		return err
	}
	return nil
}

// InstallUpdate downloads the latest build, replaces this exe and restarts into it.
func (a *App) InstallUpdate() error {
	release, err := a.fetchRelease()
	if err != nil {
		return err
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	downloaded := current + ".new"
	client := &http.Client{Timeout: 10 * time.Minute}
	if err := downloadVerified(client, release.DownloadURL, release.SHA256, downloaded, false); err != nil {
		return err
	}
	if err := swapExecutable(current, downloaded); err != nil {
		os.Remove(downloaded)
		return fmt.Errorf("could not replace the app: %w", err)
	}
	if err := exec.Command(current).Start(); err != nil {
		return fmt.Errorf("updated, but could not restart: %w", err)
	}
	a.requestAppQuit()
	return nil
}

func removeOldExecutable() {
	if current, err := os.Executable(); err == nil {
		os.Remove(current + ".old")
	}
}

// DesktopSettings are the Settings → Behaviour switches.
type DesktopSettings struct {
	MinimizeToTray       bool `json:"minimizeToTray"`
	StartWithWindows     bool `json:"startWithWindows"`
	DesktopNotifications bool `json:"desktopNotifications"`
}

func (a *App) GetDesktopSettings() DesktopSettings {
	return DesktopSettings{
		MinimizeToTray:       !a.config.DisableMinimizeToTray,
		StartWithWindows:     isStartWithWindowsEnabled(),
		DesktopNotifications: a.config.DesktopNotifications,
	}
}

func (a *App) SetStartWithWindows(enabled bool) error {
	return setStartWithWindows(enabled)
}

func (a *App) SetDesktopNotifications(enabled bool) error {
	a.config.DesktopNotifications = enabled
	return a.saveConfig()
}

// AddonDatasetSync is the last write of one dataset into the addon.
type AddonDatasetSync struct {
	Dataset  string `json:"dataset"`
	Label    string `json:"label"`
	Rows     int    `json:"rows"`
	Realm    string `json:"realm,omitempty"`
	Season   int    `json:"season,omitempty"`
	SyncedAt string `json:"syncedAt"`
}

// AddonStatus feeds the Addon sync screen.
type AddonStatus struct {
	WowDirectory     string             `json:"wowDirectory"`
	FolderFound      bool               `json:"folderFound"`
	AddonInstalled   bool               `json:"addonInstalled"`
	InstalledVersion string             `json:"installedVersion,omitempty"`
	LatestVersion    string             `json:"latestVersion,omitempty"`
	UpdateAvailable  bool               `json:"updateAvailable"`
	DownloadURL      string             `json:"downloadUrl"`
	Datasets         []AddonDatasetSync `json:"datasets"`
}

// tocVersion reads "## Version: x" from the addon's .toc.
func tocVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if value, found := strings.CutPrefix(strings.TrimSpace(line), "## Version:"); found {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var (
	addonLatestMu sync.Mutex
	addonLatest   string
	addonLatestAt time.Time
)

func latestAddonVersion() string {
	addonLatestMu.Lock()
	defer addonLatestMu.Unlock()
	if time.Since(addonLatestAt) < time.Hour {
		return addonLatest
	}
	addonLatestAt = time.Now()

	req, err := http.NewRequest(http.MethodGet, addonReleasesAPI, nil)
	if err != nil {
		return addonLatest
	}
	req.Header.Set("User-Agent", "WoWLogsUploader")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Addon] Latest version check failed: %v\n", err)
		return addonLatest
	}
	defer resp.Body.Close()
	var parsed struct {
		TagName string `json:"tag_name"`
	}
	if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&parsed) == nil {
		addonLatest = strings.TrimPrefix(parsed.TagName, "v")
	}
	return addonLatest
}

func (a *App) addonSyncPath() string {
	return filepath.Join(filepath.Dir(a.configPath), addonSyncFileName)
}

func (a *App) loadAddonSyncs() []AddonDatasetSync {
	data, err := os.ReadFile(a.addonSyncPath())
	if err != nil {
		return []AddonDatasetSync{}
	}
	var syncs []AddonDatasetSync
	if err := json.Unmarshal(data, &syncs); err != nil {
		return []AddonDatasetSync{}
	}
	return syncs
}

func mergeAddonSync(syncs []AddonDatasetSync, entry AddonDatasetSync) []AddonDatasetSync {
	out := []AddonDatasetSync{entry}
	for _, existing := range syncs {
		if existing.Dataset != entry.Dataset {
			out = append(out, existing)
		}
	}
	return out
}

// recordAddonSync remembers what was last written for each dataset.
func (a *App) recordAddonSync(entry AddonDatasetSync) {
	entry.SyncedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(mergeAddonSync(a.loadAddonSyncs(), entry), "", "  ")
	if err == nil {
		err = os.WriteFile(a.addonSyncPath(), data, 0644)
	}
	if err != nil {
		log.Printf("[Addon] Could not record sync: %v\n", err)
	}
}

// GetAddonStatus reports the game folder, addon versions and sync history.
func (a *App) GetAddonStatus() AddonStatus {
	status := AddonStatus{
		WowDirectory: a.config.WowDirectory,
		DownloadURL:  addonDownloadPage,
		Datasets:     a.loadAddonSyncs(),
	}
	if strings.TrimSpace(status.WowDirectory) == "" {
		return status
	}
	addonsDir := filepath.Join(status.WowDirectory, "Interface", "AddOns")
	if info, err := os.Stat(addonsDir); err == nil && info.IsDir() {
		status.FolderFound = true
	}
	status.InstalledVersion = tocVersion(filepath.Join(addonsDir, "WowLogsAddon", "WowLogsAddon.toc"))
	status.AddonInstalled = status.InstalledVersion != ""
	status.LatestVersion = latestAddonVersion()
	status.UpdateAvailable = status.AddonInstalled && status.LatestVersion != "" &&
		compareVersions(status.InstalledVersion, status.LatestVersion) < 0
	return status
}

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Azeroth Survivors files live in the game's own folder, not the WotLK log folder:
//
//	<game>/benilla-config/Logs/WoWCombatLog.txt
//	<game>/benilla-config/saved/WowLogsAS.lua     (WoW Logs AS)
//	<game>/benilla-config/saved/AzerothStats.lua  (Azeroth Stats, by Maidenhead)
//
// Each location is remembered on its own and can be changed on its own. These files are only ever
// read: clean copies go to a separate folder.
const (
	asConfigFolder = "benilla-config"
	asLogName      = "WoWCombatLog.txt"
	asAddonName    = "WowLogsAS.lua"
	asStatsName    = "AzerothStats.lua"
)

// AsLocations is what the Azeroth Survivors screen shows and remembers.
type AsLocations struct {
	LogPath     string `json:"logPath"`
	AddonPath   string `json:"addonPath"`
	StatsPath   string `json:"statsPath"`
	CopiesDir   string `json:"copiesDir"`
	LogFound    bool   `json:"logFound"`
	AddonFound  bool   `json:"addonFound"`
	StatsFound  bool   `json:"statsFound"`
	AddonChosen bool   `json:"addonChosen"` // set by the player rather than found beside the log
	StatsChosen bool   `json:"statsChosen"`
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// asConfigDir accepts the game folder or its benilla-config folder.
func asConfigDir(folder string) string {
	if strings.EqualFold(filepath.Base(folder), asConfigFolder) {
		return folder
	}
	return filepath.Join(folder, asConfigFolder)
}

// defaultAsCopiesDir keeps copies out of the game folder, where an update could clear them.
func defaultAsCopiesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Documents", "WoW Logs", "Azeroth Survivors")
}

// resolveAsLocations fills what the player hasn't set from the log's own game folder.
func resolveAsLocations(cfg Config) AsLocations {
	loc := AsLocations{
		LogPath: cfg.AsLogPath, AddonPath: cfg.AsAddonPath, StatsPath: cfg.AsStatsPath, CopiesDir: cfg.AsCopiesDir,
		AddonChosen: cfg.AsAddonPath != "", StatsChosen: cfg.AsStatsPath != "",
	}
	if loc.AddonPath == "" && loc.LogPath != "" {
		loc.AddonPath = filepath.Join(filepath.Dir(filepath.Dir(loc.LogPath)), "saved", asAddonName)
	}
	if loc.StatsPath == "" && loc.LogPath != "" {
		loc.StatsPath = filepath.Join(filepath.Dir(filepath.Dir(loc.LogPath)), "saved", asStatsName)
	}
	if loc.CopiesDir == "" {
		loc.CopiesDir = defaultAsCopiesDir()
	}
	loc.LogFound, loc.AddonFound, loc.StatsFound = fileExists(loc.LogPath), fileExists(loc.AddonPath), fileExists(loc.StatsPath)
	return loc
}

// GetAsLocations returns the remembered locations, with blanks filled from the game folder.
func (a *App) GetAsLocations() AsLocations {
	return resolveAsLocations(a.config)
}

// setAsLocation remembers one location. An empty path goes back to finding it from the game folder.
func setAsLocation(cfg *Config, kind, path string) error {
	path = strings.TrimSpace(path)
	switch kind {
	case "log":
		cfg.AsLogPath = path
	case "addon":
		cfg.AsAddonPath = path
	case "stats":
		cfg.AsStatsPath = path
	case "copies":
		cfg.AsCopiesDir = path
	default:
		return errors.New("unknown location")
	}
	return nil
}

func (a *App) SetAsLocation(kind, path string) (AsLocations, error) {
	if err := setAsLocation(&a.config, kind, path); err != nil {
		return a.GetAsLocations(), err
	}
	if err := a.saveConfig(); err != nil {
		return a.GetAsLocations(), err
	}
	return a.GetAsLocations(), nil
}

// useAsGameFolder points every file location at one game folder. Saves the player chose by hand
// are kept, because they may live elsewhere on purpose.
func useAsGameFolder(cfg *Config, folder string) error {
	dir := asConfigDir(folder)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return errors.New("this folder has no benilla-config folder. Choose the folder AzerothSurvivors.exe is in")
	}
	cfg.AsLogPath = filepath.Join(dir, "Logs", asLogName)
	return nil
}

// SelectAsGameFolder finds all three files from the game folder.
func (a *App) SelectAsGameFolder() (AsLocations, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select your Azeroth Survivors folder (the one with AzerothSurvivors.exe)",
	})
	if err != nil || folder == "" {
		return a.GetAsLocations(), err
	}
	if err := useAsGameFolder(&a.config, folder); err != nil {
		return a.GetAsLocations(), err
	}
	if err := a.saveConfig(); err != nil {
		return a.GetAsLocations(), err
	}
	return a.GetAsLocations(), nil
}

// SelectAsStatsFile picks the Azeroth Stats save when it isn't beside the log.
func (a *App) SelectAsStatsFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select AzerothStats.lua",
		Filters: []runtime.FileFilter{{DisplayName: "Addon save (*.lua)", Pattern: "*.lua"}},
	})
}

// SelectAsCopiesDir picks where clean copies are written.
func (a *App) SelectAsCopiesDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Choose a folder for clean copies",
		CanCreateDirectories: true,
	})
}

// OpenAsFolder shows a folder in Explorer, for a clean copy just written.
func (a *App) OpenAsFolder(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return errors.New("this folder no longer exists")
	}
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(path))
	return nil
}

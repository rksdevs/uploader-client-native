package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A clean copy is a new folder holding the chosen runs and only their addon data, so the three
// files describe the same runs. Nothing existing is edited, renamed or deleted: the game's log and
// both saves are only read, and every file here is new (a name that exists is never overwritten).
// The copy uploads like the originals, on the website or here.

// AsCopyRequest picks runs and which addon data goes with them.
type AsCopyRequest struct {
	Indexes      []int `json:"indexes"`
	IncludeAddon bool  `json:"includeAddon"`
	IncludeStats bool  `json:"includeStats"`
}

// AsCopyResult says what was written.
type AsCopyResult struct {
	Dir       string   `json:"dir"`
	Files     []string `json:"files"`
	Runs      int      `json:"runs"`
	AddonRuns int      `json:"addonRuns"`
	StatsRuns int      `json:"statsRuns"`
}

// newAsCopyDir makes a folder that didn't exist before, named after the log and the time.
func newAsCopyDir(root, logPath string, now time.Time) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("choose a folder for clean copies first")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("could not use the copies folder: %w", err)
	}
	base := strings.TrimSuffix(filepath.Base(logPath), filepath.Ext(logPath))
	name := fmt.Sprintf("%s clean %s", base, now.Format("2006-01-02 150405"))
	for n := 1; n < 100; n++ {
		dir := filepath.Join(root, name)
		if n > 1 {
			dir = filepath.Join(root, fmt.Sprintf("%s (%d)", name, n))
		}
		err := os.Mkdir(dir, 0o755)
		if err == nil {
			return dir, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("could not create the copy folder: %w", err)
		}
	}
	return "", errors.New("could not find a free folder name for this copy")
}

// writeNewFile refuses to replace anything.
func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// writeAsCopy writes the copy's files. The names match the originals, so the copy uploads anywhere
// the originals do.
func writeAsCopy(dir string, logText, addon, stats []byte) ([]string, error) {
	files := []struct {
		name string
		data []byte
	}{{asLogName, logText}, {asAddonName, addon}, {asStatsName, stats}}
	var written []string
	for _, file := range files {
		if file.data == nil {
			continue
		}
		path := filepath.Join(dir, file.name)
		if err := writeNewFile(path, file.data); err != nil {
			return written, fmt.Errorf("could not write %s: %w", file.name, err)
		}
		written = append(written, path)
	}
	return written, nil
}

// CreateAsCleanCopy writes the chosen runs, with their WoW Logs AS and Azeroth Stats data, to a new
// folder under the copies location.
func (a *App) CreateAsCleanCopy(request AsCopyRequest) (*AsCopyResult, error) {
	a.as.mu.Lock()
	scan, path, ours, stats, statsVersion := a.as.scan, a.as.path, a.as.ours, a.as.stats, a.as.statsVersion
	a.as.mu.Unlock()
	if scan == nil {
		return nil, errors.New("scan a log first")
	}
	if len(request.Indexes) == 0 {
		return nil, errors.New("choose at least one run")
	}
	logText, err := scan.Text(request.Indexes)
	if err != nil {
		return nil, err
	}
	addon, statsFile, addonRuns, statsRuns, err := chosenSaves(request.Indexes, ours, stats, statsVersion, request.IncludeAddon, request.IncludeStats)
	if err != nil {
		return nil, fmt.Errorf("could not copy the addon data: %w", err)
	}
	dir, err := newAsCopyDir(resolveAsLocations(a.config).CopiesDir, path, time.Now())
	if err != nil {
		return nil, err
	}
	files, err := writeAsCopy(dir, logText, addon, statsFile)
	if err != nil {
		return nil, err
	}
	return &AsCopyResult{Dir: dir, Files: files, Runs: len(request.Indexes), AddonRuns: addonRuns, StatsRuns: statsRuns}, nil
}

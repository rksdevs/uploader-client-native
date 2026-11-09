package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rksdevs/aslog/review"
	"github.com/rksdevs/aslog/saves"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Azeroth Survivors uploads. Opt-in by nature: nothing here runs until the player opens the screen,
// and the routes are new, so older app versions are unaffected.
const (
	asMaxLogBytes   = 500 * 1024 * 1024 // gateway AS_MAX_RAW_BYTES
	asMaxAddonBytes = 64 * 1024 * 1024  // gateway AS_MAX_ADDON_BYTES
	asMaxStatsBytes = 256 * 1024 * 1024 // gateway AS_MAX_STATS_BYTES: the file holds large backups
	asMinActiveMs   = 60_000            // the server gives shorter runs no report
	asUploadTimeout = 10 * time.Minute
)

// asSession holds the log last scanned and the addon runs matched to it, so chosen runs can be cut
// out without reading anything again.
type asSession struct {
	mu           sync.Mutex
	path         string
	scan         *review.Scan
	ours         map[int]map[string]any // log run index -> matched WoW Logs AS run
	stats        map[int]map[string]any // log run index -> matched Azeroth Stats run
	statsVersion string
}

// AsScanResult is what the review screen shows. The same parser runs on the website, so a player
// sees the same runs in both.
type AsScanResult struct {
	Path        string       `json:"path"`
	Runs        []review.Run `json:"runs"`
	Matches     []AsRunMatch `json:"matches"`
	AddonPath   string       `json:"addonPath"`
	StatsPath   string       `json:"statsPath"`
	AddonRuns   int          `json:"addonRuns"` // runs in the save, matched or not
	StatsRuns   int          `json:"statsRuns"`
	AddonError  string       `json:"addonError"`
	StatsError  string       `json:"statsError"`
	MinActiveMs int          `json:"minActiveMs"`
	GameRunning bool         `json:"gameRunning"`
}

// AsRunMatch says which addon saves hold data for a log run, using the server's own rules
// (aslog/saves), so what the player sees here is what the report will have.
type AsRunMatch struct {
	Index int  `json:"index"`
	Addon bool `json:"addon"`
	Stats bool `json:"stats"`
}

// AsUploadRequest sends whole runs only, never single lines. Addon data goes only for the chosen
// runs: the rest of each save stays on this computer.
type AsUploadRequest struct {
	Indexes   []int  `json:"indexes"`
	Private   bool   `json:"private"`
	HeroName  string `json:"heroName"`
	SendAddon bool   `json:"sendAddon"`
	SendStats bool   `json:"sendStats"`
}

// AsUploadStatus mirrors the gateway's upload response.
type AsUploadStatus struct {
	ID               int             `json:"id"`
	Status           string          `json:"status"`
	ErrorCode        string          `json:"errorCode"`
	RunsCreated      int             `json:"runsCreated"`
	RunsDuplicate    int             `json:"runsDuplicate"`
	RunsSkippedQuota int             `json:"runsSkippedQuota"`
	AddonIgnored     bool            `json:"addonIgnored"`
	Runs             []AsUploadedRun `json:"runs"`
}

// AsUploadedRun is a run the server stored.
type AsUploadedRun struct {
	PublicID string `json:"publicId"`
}

// SelectAsLogFile picks the game's log. It lives in the game folder, not a WoW Logs folder.
func (a *App) SelectAsLogFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select your Azeroth Survivors combat log",
		Filters: []runtime.FileFilter{{DisplayName: "Combat logs (*.txt;*.zip)", Pattern: "*.txt;*.zip"}},
	})
}

// SelectAsAddonFile picks the addon's save file when it wasn't found beside the log.
func (a *App) SelectAsAddonFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select WowLogsAS.lua",
		Filters: []runtime.FileFilter{{DisplayName: "Addon save (*.lua)", Pattern: "*.lua"}},
	})
}

// findAsAddonFile looks for benilla-config/saved/WowLogsAS.lua from benilla-config/Logs/<log>.
func findAsAddonFile(logPath string) string {
	return findSavedFile(logPath, asAddonName)
}

func findSavedFile(logPath, name string) string {
	candidate := filepath.Join(filepath.Dir(filepath.Dir(logPath)), "saved", name)
	if fileExists(candidate) {
		return candidate
	}
	return ""
}

// asSavePaths picks the saves for a log: the player's chosen ones, else the ones beside it.
func asSavePaths(cfg Config, logPath string) (addon, stats string) {
	addon, stats = cfg.AsAddonPath, cfg.AsStatsPath
	if addon == "" {
		addon = findSavedFile(logPath, asAddonName)
	}
	if stats == "" {
		stats = findSavedFile(logPath, asStatsName)
	}
	return addon, stats
}

// readSaveFile reads a save only up to the server's limit for it.
func readSaveFile(path string, limit int64) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > limit {
		return "", errors.New("this file is larger than the server accepts")
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// asMatchSet is the result of matching both saves to a scanned log.
type asMatchSet struct {
	matches      []AsRunMatch
	ours, stats  map[int]map[string]any
	addonRuns    int
	statsRuns    int
	statsVersion string
	addonErr     string
	statsErr     string
}

// matchAsSaves ties each log run to its addon runs. A save that can't be read costs only its own
// matches: the log still uploads, as on the server.
func matchAsSaves(scan *review.Scan, addonSrc, statsSrc string, addonErr, statsErr error) asMatchSet {
	set := asMatchSet{ours: map[int]map[string]any{}, stats: map[int]map[string]any{}}
	var ours, stats []map[string]any
	if addonErr == nil && addonSrc != "" {
		if ours, addonErr = saves.OurRuns(addonSrc); addonErr == nil {
			set.addonRuns = len(ours)
		}
	}
	if statsErr == nil && statsSrc != "" {
		if stats, set.statsVersion, statsErr = saves.StatsRuns(statsSrc); statsErr == nil {
			set.statsRuns = len(stats)
		}
	}
	if addonErr != nil {
		set.addonErr = "Couldn't read the WoW Logs AS save, so its data is left out."
	}
	if statsErr != nil {
		set.statsErr = "Couldn't read the Azeroth Stats save, so its data is left out."
	}
	for _, run := range scan.Runs {
		match := AsRunMatch{Index: run.Index}
		if len(ours) > 0 {
			if found := saves.MatchOurs(ours, scan.RunLines(run.Index)); found != nil {
				set.ours[run.Index], match.Addon = found, true
			}
		}
		if len(stats) > 0 {
			if found := saves.MatchStats(stats, run.Kills, run.ActiveMs); found != nil {
				set.stats[run.Index], match.Stats = found, true
			}
		}
		set.matches = append(set.matches, match)
	}
	return set
}

// ScanAsLog finds the runs in a log and which addon data belongs to each. Nothing is sent, and no
// file is changed.
func (a *App) ScanAsLog(path string) (*AsScanResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not open this file: %w", err)
	}
	if info.Size() > asMaxLogBytes {
		return nil, errors.New("this log is too large to upload")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read this file: %w", err)
	}
	scan, err := review.Open(data)
	if err != nil {
		return nil, errors.New("this doesn't look like an Azeroth Survivors combat log. Only English logs can be read for now")
	}

	addonPath, statsPath := asSavePaths(a.config, path)
	var addonSrc, statsSrc string
	var addonErr, statsErr error
	if addonPath != "" {
		addonSrc, addonErr = readSaveFile(addonPath, asMaxAddonBytes)
	}
	if statsPath != "" {
		statsSrc, statsErr = readSaveFile(statsPath, asMaxStatsBytes)
	}
	set := matchAsSaves(scan, addonSrc, statsSrc, addonErr, statsErr)

	a.as.mu.Lock()
	a.as.path, a.as.scan = path, scan
	a.as.ours, a.as.stats, a.as.statsVersion = set.ours, set.stats, set.statsVersion
	a.as.mu.Unlock()
	return &AsScanResult{
		Path: path, Runs: scan.Runs, Matches: set.matches, AddonPath: addonPath, StatsPath: statsPath,
		AddonRuns: set.addonRuns, StatsRuns: set.statsRuns, AddonError: set.addonErr, StatsError: set.statsErr,
		MinActiveMs: asMinActiveMs, GameRunning: isAsGameRunning(),
	}, nil
}

// chosenSaves builds the two save copies for chosen runs, in log order. A nil result means no
// chosen run had that addon's data.
func chosenSaves(indexes []int, ours, stats map[int]map[string]any, statsVersion string, wantOurs, wantStats bool) (addon, statsFile []byte, addonRuns, statsRuns int, err error) {
	sorted := append([]int(nil), indexes...)
	sort.Ints(sorted)
	var pickedOurs, pickedStats []map[string]any
	for _, index := range sorted {
		if run := ours[index]; wantOurs && run != nil {
			pickedOurs = append(pickedOurs, run)
		}
		if run := stats[index]; wantStats && run != nil {
			pickedStats = append(pickedStats, run)
		}
	}
	if len(pickedOurs) > 0 {
		if addon, err = saves.OurCopy(pickedOurs); err != nil {
			return nil, nil, 0, 0, err
		}
	}
	if len(pickedStats) > 0 {
		if statsFile, err = saves.StatsCopy(statsVersion, pickedStats); err != nil {
			return nil, nil, 0, 0, err
		}
	}
	return addon, statsFile, len(pickedOurs), len(pickedStats), nil
}

// buildAsUpload writes the multipart body. All files are small: the log is gzip of chosen runs,
// and each save holds only those runs' data.
func buildAsUpload(gz, addon, stats []byte, request AsUploadRequest, discarded int) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	fields := map[string]string{
		"private":       strconv.FormatBool(request.Private),
		"runsDiscarded": strconv.Itoa(discarded),
	}
	if name := strings.TrimSpace(request.HeroName); name != "" {
		fields["heroName"] = name
	}
	for key, value := range fields {
		if err := form.WriteField(key, value); err != nil {
			return nil, "", err
		}
	}
	logPart, err := form.CreateFormFile("logFile", "WoWCombatLog.txt.gz")
	if err != nil {
		return nil, "", err
	}
	if _, err := logPart.Write(gz); err != nil {
		return nil, "", err
	}
	if addon != nil {
		addonPart, err := form.CreateFormFile("addonFile", "WowLogsAS.lua")
		if err != nil {
			return nil, "", err
		}
		if _, err := addonPart.Write(addon); err != nil {
			return nil, "", err
		}
	}
	if stats != nil {
		statsPart, err := form.CreateFormFile("statsFile", "AzerothStats.lua")
		if err != nil {
			return nil, "", err
		}
		if _, err := statsPart.Write(stats); err != nil {
			return nil, "", err
		}
	}
	if err := form.Close(); err != nil {
		return nil, "", err
	}
	return &body, form.FormDataContentType(), nil
}

// UploadAsRuns sends the chosen runs and only their addon data.
func (a *App) UploadAsRuns(request AsUploadRequest) (*AsUploadStatus, error) {
	a.as.mu.Lock()
	scan, ours, statsRuns, statsVersion := a.as.scan, a.as.ours, a.as.stats, a.as.statsVersion
	a.as.mu.Unlock()
	if scan == nil {
		return nil, errors.New("scan a log first")
	}
	if len(request.Indexes) == 0 {
		return nil, errors.New("choose at least one run")
	}
	gz, err := scan.Extract(request.Indexes)
	if err != nil {
		return nil, err
	}

	// A log must always upload: addon data that can't be written costs the extras, not the run.
	addon, stats, _, _, err := chosenSaves(request.Indexes, ours, statsRuns, statsVersion, request.SendAddon, request.SendStats)
	if err != nil {
		addon, stats = nil, nil
	}

	body, contentType, err := buildAsUpload(gz, addon, stats, request, len(scan.Runs)-len(request.Indexes))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), asUploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiBaseURL+"/api/as/uploader/uploads", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	a.setAPIHeaders(req)
	return doAsRequest(req)
}

// AsKnownRun is a run in the scanned log that this player already uploaded.
type AsKnownRun struct {
	Index    int    `json:"index"`
	PublicID string `json:"publicId"`
}

// KnownAsRuns says which runs of the scanned log are already on the player's account, by the run
// fingerprint the server keeps (aslog segment.Run.Fingerprint). Only fingerprints are sent, never log
// lines. Signed out, or a gateway too old to know the route, marks none: the screen works as before.
func (a *App) KnownAsRuns() ([]AsKnownRun, error) {
	a.as.mu.Lock()
	scan := a.as.scan
	a.as.mu.Unlock()
	known := []AsKnownRun{}
	if scan == nil || a.authToken() == "" {
		return known, nil
	}
	byFingerprint := map[string][]int{}
	fingerprints := []string{}
	for _, run := range scan.Runs {
		if run.Fingerprint == "" {
			continue
		}
		if _, seen := byFingerprint[run.Fingerprint]; !seen {
			fingerprints = append(fingerprints, run.Fingerprint)
		}
		byFingerprint[run.Fingerprint] = append(byFingerprint[run.Fingerprint], run.Index)
	}
	if len(fingerprints) == 0 {
		return known, nil
	}
	body, err := json.Marshal(map[string]any{"fingerprints": fingerprints})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiBaseURL+"/api/as/uploader/runs/known", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	a.setAPIHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the server: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return known, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIError(resp.StatusCode, resp.Status, raw)
	}
	var parsed struct {
		Known map[string]string `json:"known"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("could not read the server response: %w", err)
	}
	for fingerprint, publicID := range parsed.Known {
		for _, index := range byFingerprint[fingerprint] {
			known = append(known, AsKnownRun{Index: index, PublicID: publicID})
		}
	}
	sort.Slice(known, func(i, j int) bool { return known[i].Index < known[j].Index })
	return known, nil
}

// GetAsUploadStatus polls an upload until the worker finishes it.
func (a *App) GetAsUploadStatus(id int) (*AsUploadStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/as/uploader/uploads/%d", a.apiBaseURL, id), nil)
	if err != nil {
		return nil, err
	}
	a.setAPIHeaders(req)
	return doAsRequest(req)
}

func doAsRequest(req *http.Request) (*AsUploadStatus, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the server: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, newAPIError(resp.StatusCode, resp.Status, raw)
	}
	var status AsUploadStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return nil, fmt.Errorf("could not read the server response: %w", err)
	}
	return &status, nil
}

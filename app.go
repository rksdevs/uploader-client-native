package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	LogDirectory            string `json:"logDirectory"`
	AllLogsURL              string `json:"allLogsURL,omitempty"`
	WowDirectory            string `json:"wowDirectory,omitempty"`
	ApiToken                string `json:"apiToken,omitempty"`
	ApiTokenType            string `json:"apiTokenType,omitempty"` // "personal" or "guild"
	FollowedPlayers         string `json:"followedPlayers,omitempty"` // comma-separated player names
	Theme                   string `json:"theme,omitempty"`           // "light" or "dark"
}

type App struct {
	ctx          context.Context
	apiBaseURL   string
	viewLogURL   string
	pendingPolls map[int]bool
	pollLock     sync.Mutex
	config       Config
	configPath   string
}

func fallbackUploaderServers() []UploaderServer {
	servers := []UploaderServer{
		{ID: 0, Value: "Whitemane_Frostmourne", Label: "Whitemane-Frostmourne"},
		{ID: 0, Value: "Warmane_Icecrown", Label: "Warmane - Icecrown"},
		{ID: 0, Value: "Warmane_Onyxia", Label: "Warmane - Onyxia"},
		{ID: 0, Value: "Sunwell", Label: "Sunwell"},
		{ID: 0, Value: "AstraWow_Wrathion", Label: "Dev-Server-Testing"},
		{ID: 0, Value: "AstraWow_Neltharion", Label: "WOTLK-PTR-Server"},
		{ID: 0, Value: "Warmane_Lordaeron", Label: "Warmane - Lordaeron"},
		{ID: 0, Value: "Stormforge_Frostmourne_S1", Label: "Stormforge - FrostmourneS1"},
		{ID: 0, Value: "Freedom_Wow", Label: "Freedom - WoW"},
		{ID: 0, Value: "Rising_Gods", Label: "Rising - Gods"},
		{ID: 0, Value: "Chromiecraft", Label: "Chromiecraft"},
	}
	return normalizeServerLabels(servers)
}

func normalizeServerLabels(servers []UploaderServer) []UploaderServer {
	order := map[string]int{
		"Warmane_Lordaeron":         0,
		"Warmane_Icecrown":          1,
		"Warmane_Onyxia":            2,
		"Stormforge_Frostmourne_S1": 3,
		"Freedom_Wow":               4,
		"Rising_Gods":               5,
		"Chromiecraft":              6,
		"AstraWow_Wrathion":         7,
		"AstraWow_Neltharion":       8,
		"Whitemane_Frostmourne":     9,
		"Sunwell":                   10,
	}

	for i := range servers {
		switch servers[i].Value {
		case "Warmane_Lordaeron":
			servers[i].Label = "Warmane - Lordaeron"
		case "Warmane_Icecrown":
			servers[i].Label = "Warmane - Icecrown"
		case "Warmane_Onyxia":
			servers[i].Label = "Warmane - Onyxia"
		case "Stormforge_Frostmourne_S1":
			servers[i].Label = "Stormforge - FrostmourneS1"
		case "Freedom_Wow":
			servers[i].Label = "Freedom - WoW"
		case "Rising_Gods":
			servers[i].Label = "Rising - Gods"
		case "AstraWow_Wrathion":
			servers[i].Label = "Dev-Server-Testing"
		case "AstraWow_Neltharion":
			servers[i].Label = "WOTLK-PTR-Server"
		}
	}

	sort.SliceStable(servers, func(i, j int) bool {
		ri, iok := order[servers[i].Value]
		rj, jok := order[servers[j].Value]
		switch {
		case iok && jok:
			return ri < rj
		case iok:
			return true
		case jok:
			return false
		default:
			return strings.ToLower(servers[i].Label) < strings.ToLower(servers[j].Label)
		}
	})

	return servers
}

func NewApp() *App {
	return &App{
		pendingPolls: make(map[int]bool),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("[Go Backend] Startup complete.")

	env := runtime.Environment(a.ctx)
	if env.BuildType == "dev" {
		log.Println("[Go Backend] Development environment detected.")
		a.apiBaseURL = "http://localhost:8000"
		a.viewLogURL = "http://localhost:3000"
	} else {
		log.Println("[Go Backend] Production environment detected.")
		a.apiBaseURL = "https://wow-logs.co.in"
		a.viewLogURL = "https://wow-logs.co.in"
	}

	if err := a.loadConfig(); err != nil {
		log.Printf("[Go Backend] Could not load config file (this is normal on first run): %v\n", err)
	}
}

func (a *App) loadConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("could not get user config directory: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "WoWLogsUploader")
	if err := os.MkdirAll(appConfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create app config directory: %w", err)
	}
	a.configPath = filepath.Join(appConfigDir, "config.json")

	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}

	if err := json.Unmarshal(data, &a.config); err != nil {
		return fmt.Errorf("could not parse config file: %w", err)
	}

	log.Printf("[Go Backend] Successfully loaded config. Log Directory: %s\n", a.config.LogDirectory)
	return nil
}

func (a *App) saveConfig() error {
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config to JSON: %w", err)
	}
	if err := os.WriteFile(a.configPath, data, 0644); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}
	log.Printf("[Go Backend] Successfully saved config to: %s\n", a.configPath)
	return nil
}

func (a *App) GetSavedDirectory() string {
	return a.config.LogDirectory
}

func (a *App) GetWowDirectory() string {
	return a.config.WowDirectory
}

func (a *App) GetTheme() string {
	if a.config.Theme == "" {
		return "light"
	}
	return a.config.Theme
}

func (a *App) SetTheme(theme string) error {
	a.config.Theme = theme
	return a.saveConfig()
}

func (a *App) SelectDirectory() (string, error) {
	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select your WoW Logs folder",
	})
	if err != nil {
		return "", err
	}
	if directory != "" {
		log.Printf("[Go Backend] User selected new directory: %s\n", directory)
		a.config.LogDirectory = directory
		if err := a.saveConfig(); err != nil {
			log.Printf("[Go Backend] ERROR: Failed to save config: %v\n", err)
		}
	}
	return directory, nil
}

func (a *App) SelectWowDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select WoW Directory (e.g. World of Warcraft 3.3.5a)",
	})
	if err != nil {
		return "", err
	}
	if dir != "" {
		log.Printf("[Go Backend] User selected WoW directory: %s\n", dir)
		a.config.WowDirectory = dir
		if err := a.saveConfig(); err != nil {
			log.Printf("[Go Backend] ERROR: Failed to save config: %v\n", err)
		}
	}
	return dir, nil
}

func (a *App) PreprocessLog(logDirectory string, serverName string) (*PreprocessResponse, error) {
	log.Printf("[Go Backend] PREPROCESS: Starting for directory '%s', Server: '%s'\n", logDirectory, serverName)
	logPath := filepath.Join(logDirectory, "WoWCombatLog.txt")

	// TEMPORARY (local slicing test): raised from 500 MB to 3 GB.
	// Revert to 500 MB after testing.
	const maxLogSizeBytes = 3 * 1024 * 1024 * 1024 // 3 GB
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return nil, fmt.Errorf("could not access WoWCombatLog.txt: %w", err)
	}
	if fileInfo.Size() > maxLogSizeBytes {
		sizeMB := fileInfo.Size() / 1024 / 1024
		return nil, fmt.Errorf(
			"WoWCombatLog.txt is too large to upload (%d MB). Maximum allowed size is 3072 MB. "+
				"Please clear your log file in-game (type /combatlog in chat to toggle it off and on) before uploading.",
			sizeMB,
		)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("could not read WoWCombatLog.txt: %w", err)
	}

	// Create a zip archive in memory.
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	fileWriter, err := zipWriter.Create("WoWCombatLog.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create zip entry: %w", err)
	}
	if _, err = fileWriter.Write(logData); err != nil {
		return nil, fmt.Errorf("failed to write data to zip entry: %w", err)
	}
	zipWriter.Close()

	// Create a multipart form request.
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	part, err := writer.CreateFormFile("logFile", "WoWCombatLog.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(part, buf); err != nil {
		return nil, fmt.Errorf("failed to copy zip data to form: %w", err)
	}
	_ = writer.WriteField("serverName", serverName)
	writer.Close()

	// Make the HTTP POST request.
	apiURL := fmt.Sprintf("%s/api/v5/uploader/preprocess", a.apiBaseURL)
	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create preprocess request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Socket-ID", "wails-native-client-polling") // Use a static ID for polling

	// TEMPORARY (local large-log slicing test): raised preprocess HTTP timeout.
	// Revert to 60s after testing.
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send preprocess request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Go Backend] PREPROCESS Response: Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("server returned error: %s", resp.Status)
	}

	var preprocessResponse PreprocessResponse
	if err := json.Unmarshal(respBody, &preprocessResponse); err != nil {
		return nil, fmt.Errorf("failed to decode preprocess response: %w", err)
	}

	preprocessResponse.ViewLogURL = a.viewLogURL
	return &preprocessResponse, nil
}

func (a *App) EnqueueJobs(preprocessId int, selectedInstances []Instance) (string, error) {
	log.Printf("[Go Backend] ENQUEUE: Queuing %d jobs for PreprocessID: %d\n", len(selectedInstances), preprocessId)

	requestData, err := json.Marshal(map[string]interface{}{
		"preprocessId":      preprocessId,
		"selectedInstances": selectedInstances,
		"socketId":          "wails-native-client-polling",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal enqueue request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v5/uploader/enqueue-jobs", a.apiBaseURL)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("failed to create enqueue request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send enqueue request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Go Backend] ENQUEUE Response: Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("server returned non-202 status: %s", resp.Status)
	}

	return "Jobs successfully queued! You will be notified upon completion.", nil
}

func (a *App) StartMonitoringJob(preprocessId int) {
	a.pollLock.Lock()
	if a.pendingPolls[preprocessId] {
		log.Printf("[Go Backend] POLLING: Monitoring for PreprocessID %d is already active.", preprocessId)
		a.pollLock.Unlock()
		return
	}
	a.pendingPolls[preprocessId] = true
	a.pollLock.Unlock()

	log.Printf("[Go Backend] POLLING: Started monitoring for PreprocessID: %d\n", preprocessId)

	go func() {
		defer func() {
			a.pollLock.Lock()
			delete(a.pendingPolls, preprocessId)
			a.pollLock.Unlock()
			log.Printf("[Go Backend] POLLING: Stopped monitoring for PreprocessID: %d\n", preprocessId)
		}()

		timeout := time.After(15 * time.Minute)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		notifiedLogs := make(map[int]bool)

		for {
			select {
			case <-timeout:
				log.Printf("[Go Backend] POLLING: Timed out for PreprocessID: %d\n", preprocessId)
				return
			case <-ticker.C:
				status, err := a.checkJobStatus(preprocessId)
				if err != nil {
					log.Printf("[Go Backend] POLLING: Error checking status for %d: %v\n", preprocessId, err)
					continue
				}

				allJobsConsideredDone := status.TotalJobs > 0 && status.JobsCompleted >= status.TotalJobs

				for _, logStatus := range status.Logs {
					if !notifiedLogs[logStatus.ID] && (logStatus.Status == "uploaded" || logStatus.Status == "failed") {
						log.Printf("[Go Backend] POLLING: Detected completed log %d with status '%s'. Notifying frontend.\n", logStatus.ID, logStatus.Status)
						runtime.EventsEmit(a.ctx, "job_notification", map[string]interface{}{
							"logId":      logStatus.ID,
							"status":     logStatus.Status,
							"viewLogURL": status.ViewLogURL,
						})
						notifiedLogs[logStatus.ID] = true
					}
				}

				if allJobsConsideredDone && len(notifiedLogs) == status.TotalJobs {
					log.Printf("[Go Backend] POLLING: All %d jobs for PreprocessID %d are complete. Stopping poll.\n", status.TotalJobs, preprocessId)
					return
				}
			}
		}
	}()
}

func (a *App) checkJobStatus(preprocessId int) (*JobStatusResponse, error) {
	apiURL := fmt.Sprintf("%s/api/v5/uploader/status/%d", a.apiBaseURL, preprocessId)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to poll for job status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("polling returned non-200 status: %s", resp.Status)
	}

	var statusResponse JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return nil, fmt.Errorf("failed to decode job status response: %w", err)
	}

	statusResponse.ViewLogURL = a.viewLogURL
	return &statusResponse, nil
}

func (a *App) GetUploaderServers() []UploaderServer {
	apiURL := fmt.Sprintf("%s/api/v5/uploader/servers", a.apiBaseURL)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("[Go Backend] Failed to fetch uploader servers, using fallback: %v\n", err)
		return fallbackUploaderServers()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Go Backend] Server list API returned %s, using fallback\n", resp.Status)
		return fallbackUploaderServers()
	}

	var payload UploaderServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		log.Printf("[Go Backend] Failed to decode uploader server list, using fallback: %v\n", err)
		return fallbackUploaderServers()
	}

	if len(payload.Servers) == 0 {
		return fallbackUploaderServers()
	}

	return normalizeServerLabels(payload.Servers)
}

func (a *App) OpenLogPage(logId int) {
	if logId <= 0 {
		return
	}
	logURL := fmt.Sprintf("%s/%d", strings.TrimRight(a.viewLogURL, "/"), logId)
	runtime.BrowserOpenURL(a.ctx, logURL)
}

func (a *App) OpenAllLogsPage() {
	allLogsURL := strings.TrimSpace(a.config.AllLogsURL)
	if allLogsURL == "" {
		allLogsURL = fmt.Sprintf("%s/all-logs", strings.TrimRight(a.viewLogURL, "/"))
	}

	runtime.BrowserOpenURL(a.ctx, allLogsURL)
}

type addonFilterRaid struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type addonFilterBoss struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	RaidID int    `json:"raidId"`
}

type addonFilters struct {
	Raids        []addonFilterRaid   `json:"raids"`
	Bosses       []addonFilterBoss   `json:"bosses"`
	Classes      []string            `json:"classes"`
	Difficulties []string            `json:"difficulties"`
	Roles        []string            `json:"roles"`
	Ladders      []string            `json:"ladders"`
	SpecsByClass map[string][]string `json:"specsByClass"`
}

type addonRankingRow struct {
	Key           string   `json:"key"`
	PlayerID      int      `json:"playerId"`
	PlayerName    string   `json:"playerName"`
	Realm         string   `json:"realm"`
	Server        string   `json:"server"`
	PlayerClass   string   `json:"playerClass"`
	PlayerSpec    string   `json:"playerSpec"`
	RaidID        *int     `json:"raidId"`
	RaidName      string   `json:"raidName"`
	BossID        int      `json:"bossId"`
	BossName      string   `json:"bossName"`
	Difficulty    string   `json:"difficulty"`
	Points        float64  `json:"points"`
	Percentile    float64  `json:"percentile"`
	CategoryRank  *int     `json:"categoryRank"`
	OverallRank   *int     `json:"overallRank"`
	OverallPoints float64  `json:"overallPoints"`
	Role          string   `json:"role"`
	Ladder        string   `json:"ladder"`
	Amount        float64  `json:"amount"`
	IsFollowed    bool     `json:"isFollowed"`
	Trend         *float64 `json:"trend"`
	LatestDate    string   `json:"latestDate"`
}

type addonRankingsResponse struct {
	SchemaVersion      int               `json:"schemaVersion"`
	ServerName         string            `json:"serverName"`
	Realm              string            `json:"realm"`
	Season             int               `json:"season"`
	GeneratedAt        string            `json:"generatedAt"`
	IsPremium          bool              `json:"isPremium"`
	Count              int               `json:"count"`
	PointsCount        int               `json:"pointsCount"`
	PerformanceCount   int               `json:"performanceCount"`
	Filters            addonFilters      `json:"filters"`
	Rows               []addonRankingRow `json:"rows"`
	PerformanceFilters addonFilters      `json:"performanceFilters"`
	PerformanceRows    []addonRankingRow `json:"performanceRows"`
}

func escapeLuaString(input string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	return replacer.Replace(input)
}

func buildSavedVariablesLua(payload *addonRankingsResponse) string {
	var sb strings.Builder
	nowUnix := time.Now().Unix()

	dictStrings := []string{}
	dictMap := map[string]int{}
	getStrId := func(s string) int {
		if s == "" {
			return 0 // Handle empty
		}
		if id, exists := dictMap[s]; exists {
			return id
		}
		dictStrings = append(dictStrings, s)
		id := len(dictStrings)
		dictMap[s] = id
		return id
	}

	sb.WriteString("WowLogsAddonDB = {\n")
	sb.WriteString("  meta = { version = 2 },\n")
	sb.WriteString(fmt.Sprintf("  rankings = { updatedAt = %d, serverName = \"%s\", realm = \"%s\", season = %d, isPremium = %v, players = {},\n",
		nowUnix,
		escapeLuaString(payload.ServerName),
		escapeLuaString(payload.Realm),
		payload.Season,
		payload.IsPremium,
	))

	sb.WriteString("    filters = {\n")
	// For filters, we leave them un-minified so the UI filter dropdowns work easily, 
	// or we can let the UI use them as-is since they are small. Actually, we'll keep
	// filter names as strings here, they are very small.
	sb.WriteString("      raids = {\n")
	for _, r := range payload.Filters.Raids {
		sb.WriteString(fmt.Sprintf("        { id = %d, name = \"%s\" },\n", r.ID, escapeLuaString(r.Name)))
	}
	sb.WriteString("      },\n")

	sb.WriteString("      bosses = {\n")
	for _, b := range payload.Filters.Bosses {
		sb.WriteString(fmt.Sprintf("        { id = %d, name = \"%s\", raidId = %d },\n", b.ID, escapeLuaString(b.Name), b.RaidID))
	}
	sb.WriteString("      },\n")

	sb.WriteString("      classes = {")
	for i, cls := range payload.Filters.Classes {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(cls)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      difficulties = {")
	for i, d := range payload.Filters.Difficulties {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(d)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      roles = {")
	for i, r := range payload.Filters.Roles {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(r)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      ladders = {")
	for i, l := range payload.Filters.Ladders {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(l)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      specsByClass = {\n")
	for cls, specs := range payload.Filters.SpecsByClass {
		sb.WriteString(fmt.Sprintf("        [\"%s\"] = {", escapeLuaString(cls)))
		for i, s := range specs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(s)))
		}
		sb.WriteString("},\n")
	}
	sb.WriteString("      },\n")
	sb.WriteString("    },\n")

	sb.WriteString("    rows = {\n")
	for _, r := range payload.Rows {
		raidID := "nil"
		if r.RaidID != nil {
			raidID = fmt.Sprintf("%d", *r.RaidID)
		}
		categoryRank := "nil"
		if r.CategoryRank != nil {
			categoryRank = fmt.Sprintf("%d", *r.CategoryRank)
		}

		isFollowedStr := "false"
		if r.IsFollowed {
			isFollowedStr = "true"
		}

		// CSV String format for rows:
		// key,playerName,classID,specID,raidId,raidNameID,bossId,bossNameID,difficultyID,points,percentile,categoryRank,isFollowed
		sb.WriteString(fmt.Sprintf("      \"%s,%s,%d,%d,%s,%d,%d,%d,%d,%.2f,%.2f,%s,%s\",\n",
			escapeLuaString(r.Key),
			escapeLuaString(r.PlayerName),
			getStrId(r.PlayerClass),
			getStrId(r.PlayerSpec),
			raidID,
			getStrId(r.RaidName),
			r.BossID,
			getStrId(r.BossName),
			getStrId(r.Difficulty),
			r.Points,
			r.Percentile,
			categoryRank,
			isFollowedStr,
		))
	}
	sb.WriteString("    },\n")

	sb.WriteString("    performanceFilters = {\n")
	sb.WriteString("      raids = {\n")
	for _, r := range payload.PerformanceFilters.Raids {
		sb.WriteString(fmt.Sprintf("        { id = %d, name = \"%s\" },\n", r.ID, escapeLuaString(r.Name)))
	}
	sb.WriteString("      },\n")

	sb.WriteString("      bosses = {\n")
	for _, b := range payload.PerformanceFilters.Bosses {
		sb.WriteString(fmt.Sprintf("        { id = %d, name = \"%s\", raidId = %d },\n", b.ID, escapeLuaString(b.Name), b.RaidID))
	}
	sb.WriteString("      },\n")

	sb.WriteString("      classes = {")
	for i, cls := range payload.PerformanceFilters.Classes {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(cls)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      difficulties = {")
	for i, d := range payload.PerformanceFilters.Difficulties {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(d)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      roles = {")
	for i, r := range payload.PerformanceFilters.Roles {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(r)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      ladders = {")
	for i, l := range payload.PerformanceFilters.Ladders {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(l)))
	}
	sb.WriteString("},\n")

	sb.WriteString("      specsByClass = {\n")
	for cls, specs := range payload.PerformanceFilters.SpecsByClass {
		sb.WriteString(fmt.Sprintf("        [\"%s\"] = {", escapeLuaString(cls)))
		for i, s := range specs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"", escapeLuaString(s)))
		}
		sb.WriteString("},\n")
	}
	sb.WriteString("      },\n")
	sb.WriteString("    },\n")

	sb.WriteString("    performanceRows = {\n")
	for _, r := range payload.PerformanceRows {
		raidID := "nil"
		if r.RaidID != nil {
			raidID = fmt.Sprintf("%d", *r.RaidID)
		}
		categoryRank := "nil"
		if r.CategoryRank != nil {
			categoryRank = fmt.Sprintf("%d", *r.CategoryRank)
		}

		trendStr := "nil"
		if r.Trend != nil {
			trendStr = fmt.Sprintf("%.2f", *r.Trend)
		}
		latestDate := "nil"
		if r.LatestDate != "" {
			latestDate = escapeLuaString(r.LatestDate)
		}
		isFollowedStr := "false"
		if r.IsFollowed {
			isFollowedStr = "true"
		}

		// CSV String format for performanceRows:
		// key,playerName,classID,specID,roleID,ladderID,raidId,raidNameID,bossId,bossNameID,difficultyID,amount,percentile,categoryRank,isFollowed,trend,latestDate
		sb.WriteString(fmt.Sprintf("      \"%s,%s,%d,%d,%d,%d,%s,%d,%d,%d,%d,%.2f,%.2f,%s,%s,%s,%s\",\n",
			escapeLuaString(r.Key),
			escapeLuaString(r.PlayerName),
			getStrId(r.PlayerClass),
			getStrId(r.PlayerSpec),
			getStrId(r.Role),
			getStrId(r.Ladder),
			raidID,
			getStrId(r.RaidName),
			r.BossID,
			getStrId(r.BossName),
			getStrId(r.Difficulty),
			r.Amount,
			r.Percentile,
			categoryRank,
			isFollowedStr,
			trendStr,
			latestDate,
		))
	}
	sb.WriteString("    },\n")

	// Emit Dictionary at the end of rankings structure, before closing WowLogsAddonDB
	sb.WriteString("    dict = {\n")
	for i, s := range dictStrings {
		sb.WriteString(fmt.Sprintf("      [%d] = \"%s\",\n", i+1, escapeLuaString(s)))
	}
	sb.WriteString("    },\n")

	sb.WriteString("  },\n")
	sb.WriteString("}\n")
	return sb.String()
}

func (a *App) UpdateAddonRankings(serverName string, season int) (string, error) {
	if strings.TrimSpace(serverName) == "" {
		return "", fmt.Errorf("server is required")
	}
	wowDir := strings.TrimSpace(a.config.WowDirectory)
	if wowDir == "" {
		return "", fmt.Errorf("wow directory is not configured")
	}

	accountDir := filepath.Join(wowDir, "WTF", "Account")
	entries, err := os.ReadDir(accountDir)
	if err != nil {
		return "", fmt.Errorf("could not read account directory (%s): %w", accountDir, err)
	}

	var accountPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			svDir := filepath.Join(accountDir, entry.Name(), "SavedVariables")
			accountPaths = append(accountPaths, filepath.Join(svDir, "WowLogsAddon.lua"))
		}
	}

	if len(accountPaths) == 0 {
		return "", fmt.Errorf("no accounts found in %s", accountDir)
	}

	q := url.Values{}
	q.Set("serverName", serverName)
	if season > 0 {
		q.Set("season", fmt.Sprintf("%d", season))
	}

	// Add followed players to query if set
	followedPlayers := strings.TrimSpace(a.config.FollowedPlayers)
	if followedPlayers != "" {
		q.Set("followedPlayers", followedPlayers)
	}

	apiURL := fmt.Sprintf("%s/api/v5/uploader/addon-rankings-full?%s", a.apiBaseURL, q.Encode())
	client := &http.Client{Timeout: 45 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create rankings request: %w", err)
	}

	// Send API token if configured (for premium features)
	apiToken := strings.TrimSpace(a.config.ApiToken)
	if apiToken != "" {
		req.Header.Set("X-API-Token", apiToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch addon rankings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("rankings endpoint error: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload addonRankingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to parse addon rankings response: %w", err)
	}

	luaData := buildSavedVariablesLua(&payload)
	
	successCount := 0
	for _, path := range accountPaths {
		svDir := filepath.Dir(path)
		if err := os.MkdirAll(svDir, os.ModePerm); err != nil {
			log.Printf("[Go Backend] Could not create directory %s: %v\n", svDir, err)
			continue
		}

		if err := os.WriteFile(path, []byte(luaData), 0644); err != nil {
			log.Printf("[Go Backend] Failed to write to %s: %v\n", path, err)
		} else {
			log.Printf("[Go Backend] Successfully updated rankings at %s\n", path)
			successCount++
		}
	}

	if successCount == 0 {
		return "", fmt.Errorf("failed to write addon file to any account directory")
	}

	premiumSuffix := ""
	if payload.IsPremium {
		premiumSuffix = " [Premium ✓]"
	}
	return fmt.Sprintf("Updated %d points rows and %d performance rows for %s (Season %d)%s.\n\nSuccessfully synced to %d account(s).\n\n⚠️ Please LOGOUT and log back in (do NOT use /reload if installing for the first time).", payload.PointsCount, payload.PerformanceCount, payload.Realm, payload.Season, premiumSuffix, successCount), nil
}

// GetPremiumConfig returns current premium settings (token type, token, and followed players)
func (a *App) GetPremiumConfig() map[string]string {
	return map[string]string{
		"apiToken":        a.config.ApiToken,
		"apiTokenType":    a.config.ApiTokenType,
		"followedPlayers": a.config.FollowedPlayers,
	}
}

// SavePremiumConfig saves the premium token and followed players settings
func (a *App) SavePremiumConfig(apiToken, apiTokenType, followedPlayers string) error {
	a.config.ApiToken = strings.TrimSpace(apiToken)
	a.config.ApiTokenType = strings.TrimSpace(apiTokenType)
	a.config.FollowedPlayers = strings.TrimSpace(followedPlayers)
	if err := a.saveConfig(); err != nil {
		return fmt.Errorf("failed to save premium config: %w", err)
	}
	log.Printf("[Go Backend] Premium config saved. Token type: %s, Followed players: %s", a.config.ApiTokenType, a.config.FollowedPlayers)
	return nil
}

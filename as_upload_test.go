package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func asTestLog(seconds int) string {
	var b strings.Builder
	for i := 0; i < seconds; i++ {
		fmt.Fprintf(&b, "1/2 10:%02d:%02d.000  Your Fireball hits Kobold for 40.\n", i/60, i%60)
		fmt.Fprintf(&b, "1/2 10:%02d:%02d.500  Your Fireball hits Kobold for 41.\n", i/60, i%60)
	}
	fmt.Fprintf(&b, "1/2 10:%02d:%02d.000  You die.\n", seconds/60, seconds%60)
	return b.String()
}

func TestFindAsAddonFile(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, "Logs")
	saved := filepath.Join(root, "saved")
	os.MkdirAll(logs, 0o755)
	os.MkdirAll(saved, 0o755)
	log := filepath.Join(logs, "WoWCombatLog.txt")

	if got := findAsAddonFile(log); got != "" {
		t.Errorf("no save yet, got %s", got)
	}
	want := filepath.Join(saved, "WowLogsAS.lua")
	os.WriteFile(want, []byte("WowLogsASDB = {}"), 0o644)
	if got := findAsAddonFile(log); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestScanAsLogRefusesWhatIsNotALog(t *testing.T) {
	a := &App{}
	path := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(path, []byte("hello\n"), 0o644)
	if _, err := a.ScanAsLog(path); err == nil {
		t.Error("should refuse")
	}
	if _, err := a.ScanAsLog(path + ".missing"); err == nil {
		t.Error("should fail on a missing file")
	}
}

// asTestSaves writes saves where only the first of two log runs has addon data: the first run kills
// nothing and lasts 70 s, the second is a different spell.
func asTestSaves(t *testing.T, saved string) {
	t.Helper()
	os.MkdirAll(saved, 0o755)
	sync := `{ ["text"] = "Your Fireball hits Kobold for 40." }`
	ours := `WowLogsASDB = { ["runs"] = {
		{ ["schema"] = 2, ["gameRunId"] = "first", ["chat"] = { ["sync"] = { ` + sync + `, ` + sync + `, ` + sync + ` } } },
		{ ["schema"] = 2, ["gameRunId"] = "other", ["chat"] = { ["sync"] = { { ["text"] = "Your Frostbolt hits Wolf for 9." } } } },
	} }`
	stats := `AzerothStatsDB = { ["version"] = "2.1.0", ["settings"] = { ["private"] = true }, ["runs"] = {
		{ ["id"] = "azstats-1", ["summary"] = { ["kills"] = 0, ["t"] = 70 } },
		{ ["id"] = "azstats-2", ["summary"] = { ["kills"] = 55, ["t"] = 300 } },
	} }`
	os.WriteFile(filepath.Join(saved, asAddonName), []byte(ours), 0o644)
	os.WriteFile(filepath.Join(saved, asStatsName), []byte(stats), 0o644)
}

func asTestGame(t *testing.T) (logPath, saved string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "benilla-config")
	logs, saved := filepath.Join(root, "Logs"), filepath.Join(root, "saved")
	os.MkdirAll(logs, 0o755)
	logPath = filepath.Join(logs, asLogName)
	other := strings.ReplaceAll(asTestLog(80), "Fireball hits Kobold", "Shadow Bolt hits Wolf")
	other = strings.ReplaceAll(other, "1/2 10:", "1/2 11:")
	// One kill, so its numbers differ from the first run's (Azeroth Stats matches on kills and time).
	other = strings.Replace(other, "  You die.", "  Wolf dies.\n1/2 11:01:20.050  You die.", 1)
	os.WriteFile(logPath, []byte(asTestLog(70)+other), 0o644)
	asTestSaves(t, saved)
	return logPath, saved
}

func TestScanMatchesBothSaves(t *testing.T) {
	logPath, _ := asTestGame(t)
	a := &App{}
	scan, err := a.ScanAsLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Runs) != 2 || scan.AddonRuns != 2 || scan.StatsRuns != 2 || scan.AddonError != "" || scan.StatsError != "" {
		t.Fatalf("scan = %+v", scan)
	}
	want := []AsRunMatch{{Index: 1, Addon: true, Stats: true}, {Index: 2}}
	if fmt.Sprint(scan.Matches) != fmt.Sprint(want) {
		t.Errorf("matches = %+v (runs %+v)", scan.Matches, scan.Runs)
	}
	t.Logf("run 1: %d kills %d ms; run 2: %d kills %d ms", scan.Runs[0].Kills, scan.Runs[0].ActiveMs, scan.Runs[1].Kills, scan.Runs[1].ActiveMs)
}

func TestScanSurvivesBrokenSaves(t *testing.T) {
	logPath, saved := asTestGame(t)
	os.WriteFile(filepath.Join(saved, asAddonName), []byte("WowLogsASDB = os.exit()"), 0o644)
	a := &App{}
	scan, err := a.ScanAsLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if scan.AddonError == "" || scan.Matches[0].Addon || !scan.Matches[0].Stats {
		t.Errorf("a bad save must cost only its own data: %+v", scan)
	}
}

func TestChosenSavesHoldOnlyChosenRuns(t *testing.T) {
	logPath, _ := asTestGame(t)
	a := &App{}
	if _, err := a.ScanAsLog(logPath); err != nil {
		t.Fatal(err)
	}
	addon, stats, addonRuns, statsRuns, err := chosenSaves([]int{1, 2}, a.as.ours, a.as.stats, a.as.statsVersion, true, true)
	if err != nil || addonRuns != 1 || statsRuns != 1 {
		t.Fatalf("runs %d %d, err %v", addonRuns, statsRuns, err)
	}
	if !strings.Contains(string(addon), `"first"`) || strings.Contains(string(addon), `"other"`) {
		t.Errorf("addon copy: %s", addon)
	}
	if !strings.Contains(string(stats), "azstats-1") || strings.Contains(string(stats), "azstats-2") || strings.Contains(string(stats), "settings") {
		t.Errorf("stats copy must hold the matched run and version only: %s", stats)
	}
	if addon, _, _, _, _ := chosenSaves([]int{2}, a.as.ours, a.as.stats, "", true, true); addon != nil {
		t.Error("a run without addon data gives no addon file")
	}
	if addon, stats, _, _, _ := chosenSaves([]int{1}, a.as.ours, a.as.stats, "", false, false); addon != nil || stats != nil {
		t.Error("switched off means not sent")
	}
}

func TestUploadAsRunsSendsChosenRunsAndAddon(t *testing.T) {
	var got struct {
		token, private, discarded, hero string
		log, addon, stats               []byte
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/as/uploader/uploads" {
			t.Errorf("path = %s", r.URL.Path)
		}
		got.token = r.Header.Get("X-API-Token")
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		form := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := form.NextPart()
			if err != nil {
				break
			}
			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "private":
				got.private = string(data)
			case "runsDiscarded":
				got.discarded = string(data)
			case "heroName":
				got.hero = string(data)
			case "logFile":
				got.log = data
			case "addonFile":
				got.addon = data
			case "statsFile":
				got.stats = data
			}
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"id": 7, "status": "QUEUED"})
	}))
	defer server.Close()

	logPath, _ := asTestGame(t)
	// sessionLoaded keeps the test away from a developer's real stored session.
	a := &App{apiBaseURL: server.URL, config: Config{ApiToken: "tok"}, sessionLoaded: true}
	scan, err := a.ScanAsLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Runs) != 2 || scan.MinActiveMs != 60_000 {
		t.Fatalf("scan = %+v", scan)
	}
	status, err := a.UploadAsRuns(AsUploadRequest{Indexes: []int{1}, Private: true, HeroName: " Inno ", SendAddon: true, SendStats: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.token != "tok" {
		t.Error("the upload must carry the app's credential")
	}
	if status.ID != 7 || got.private != "true" || got.discarded != "1" || got.hero != "Inno" {
		t.Errorf("status %d, private %s, discarded %s, hero %s", status.ID, got.private, got.discarded, got.hero)
	}
	if !bytes.HasPrefix(got.log, []byte{0x1f, 0x8b}) || !strings.Contains(string(got.addon), `"first"`) || strings.Contains(string(got.addon), `"other"`) {
		t.Errorf("files: log %d bytes, addon %q", len(got.log), got.addon)
	}
	if !strings.Contains(string(got.stats), "azstats-1") || strings.Contains(string(got.stats), "azstats-2") {
		t.Errorf("stats: %q", got.stats)
	}

	// Azeroth Stats switched off, and a run with no addon data: only the log goes.
	got.addon, got.stats = nil, nil
	if _, err := a.UploadAsRuns(AsUploadRequest{Indexes: []int{2}, SendAddon: true}); err != nil {
		t.Fatal(err)
	}
	if got.addon != nil || got.stats != nil || got.private != "false" {
		t.Errorf("addon %q stats %q private %s", got.addon, got.stats, got.private)
	}
}

func TestAsLocations(t *testing.T) {
	logPath, saved := asTestGame(t)
	game := filepath.Dir(filepath.Dir(filepath.Dir(logPath)))

	var cfg Config
	if err := useAsGameFolder(&cfg, t.TempDir()); err == nil {
		t.Error("a folder without benilla-config must be refused")
	}
	if err := useAsGameFolder(&cfg, game); err != nil || cfg.AsLogPath != logPath {
		t.Fatalf("game folder: %v %s", err, cfg.AsLogPath)
	}
	loc := resolveAsLocations(cfg)
	if !loc.LogFound || !loc.AddonFound || !loc.StatsFound || loc.AddonPath != filepath.Join(saved, asAddonName) || loc.AddonChosen {
		t.Errorf("found from the game folder: %+v", loc)
	}
	if loc.CopiesDir == "" || strings.HasPrefix(loc.CopiesDir, game) {
		t.Errorf("copies must default outside the game folder: %s", loc.CopiesDir)
	}

	// A save chosen by hand wins, and survives choosing the game folder again.
	elsewhere := filepath.Join(t.TempDir(), "AzerothStats.lua")
	if err := setAsLocation(&cfg, "stats", elsewhere); err != nil {
		t.Fatal(err)
	}
	useAsGameFolder(&cfg, filepath.Join(game, "benilla-config"))
	loc = resolveAsLocations(cfg)
	if loc.StatsPath != elsewhere || loc.StatsFound || !loc.StatsChosen {
		t.Errorf("chosen save: %+v", loc)
	}
	if addon, stats := asSavePaths(cfg, logPath); addon != filepath.Join(saved, asAddonName) || stats != elsewhere {
		t.Errorf("save paths for scan: %s %s", addon, stats)
	}
	setAsLocation(&cfg, "stats", "")
	if resolveAsLocations(cfg).StatsPath != filepath.Join(saved, asStatsName) {
		t.Error("clearing goes back to the game folder")
	}
	if err := setAsLocation(&cfg, "nope", "x"); err == nil {
		t.Error("unknown kinds are refused")
	}
}

func TestCleanCopyWritesNewFilesOnly(t *testing.T) {
	logPath, saved := asTestGame(t)
	copies := filepath.Join(t.TempDir(), "copies")
	before := map[string]string{}
	for _, path := range []string{logPath, filepath.Join(saved, asAddonName), filepath.Join(saved, asStatsName)} {
		data, _ := os.ReadFile(path)
		before[path] = string(data)
	}

	a := &App{config: Config{AsCopiesDir: copies}}
	if _, err := a.CreateAsCleanCopy(AsCopyRequest{Indexes: []int{1}}); err == nil {
		t.Error("needs a scan first")
	}
	if _, err := a.ScanAsLog(logPath); err != nil {
		t.Fatal(err)
	}
	result, err := a.CreateAsCleanCopy(AsCopyRequest{Indexes: []int{1}, IncludeAddon: true, IncludeStats: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 3 || result.Runs != 1 || result.AddonRuns != 1 || result.StatsRuns != 1 || filepath.Dir(result.Dir) != copies {
		t.Fatalf("result = %+v", result)
	}

	// The copy is a log the app reads again, with the same run and the same matches.
	again := &App{config: Config{AsAddonPath: filepath.Join(result.Dir, asAddonName), AsStatsPath: filepath.Join(result.Dir, asStatsName)}}
	scan, err := again.ScanAsLog(filepath.Join(result.Dir, asLogName))
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Runs) != 1 || !scan.Matches[0].Addon || !scan.Matches[0].Stats || scan.AddonRuns != 1 || scan.StatsRuns != 1 {
		t.Errorf("copy scan = %+v", scan)
	}

	// The originals are untouched.
	for path, data := range before {
		now, _ := os.ReadFile(path)
		if string(now) != data {
			t.Errorf("%s changed", path)
		}
	}

	// Without addon data only the log is written, and a second copy gets its own folder.
	second, err := a.CreateAsCleanCopy(AsCopyRequest{Indexes: []int{2}, IncludeAddon: true, IncludeStats: true})
	if err != nil || len(second.Files) != 1 || second.Dir == result.Dir {
		t.Errorf("second copy: %+v %v", second, err)
	}
}

func TestCopyNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	first, err := newAsCopyDir(root, "C:/x/WoWCombatLog.txt", now)
	if err != nil || filepath.Base(first) != "WoWCombatLog clean 2026-11-08 090000" {
		t.Fatalf("%s %v", first, err)
	}
	second, err := newAsCopyDir(root, "C:/x/WoWCombatLog.txt", now)
	if err != nil || second == first || filepath.Base(second) != "WoWCombatLog clean 2026-11-08 090000 (2)" {
		t.Errorf("same second: %s %v", second, err)
	}
	path := filepath.Join(first, "keep.txt")
	os.WriteFile(path, []byte("mine"), 0o644)
	if err := writeNewFile(path, []byte("new")); err == nil {
		t.Error("an existing file must not be replaced")
	}
	if data, _ := os.ReadFile(path); string(data) != "mine" {
		t.Error("the existing file changed")
	}
	if _, err := newAsCopyDir("  ", "x.txt", now); err == nil {
		t.Error("needs a folder")
	}
}

func TestUploadAsRunsNeedsAScanAndARun(t *testing.T) {
	a := &App{}
	if _, err := a.UploadAsRuns(AsUploadRequest{Indexes: []int{1}}); err == nil {
		t.Error("no scan should fail")
	}
	path := filepath.Join(t.TempDir(), "WoWCombatLog.txt")
	os.WriteFile(path, []byte(asTestLog(70)), 0o644)
	if _, err := a.ScanAsLog(path); err != nil {
		t.Fatal(err)
	}
	if _, err := a.UploadAsRuns(AsUploadRequest{}); err == nil {
		t.Error("no runs chosen should fail")
	}
}

func TestUploadErrorKeepsTheServerMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"code":"AS_QUOTA_PRIVATE","message":"You've reached your limit."}`))
	}))
	defer server.Close()
	a := &App{apiBaseURL: server.URL, sessionLoaded: true}
	_, err := a.GetAsUploadStatus(1)
	if err == nil || !strings.Contains(err.Error(), "reached your limit") {
		t.Errorf("err = %v", err)
	}
}

// Owner's case: from a log with several runs, the screen must show which ones are already uploaded.
func TestKnownAsRunsMarksUploadedRuns(t *testing.T) {
	var sent []string
	var token string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/as/uploader/runs/known" || r.Method != http.MethodPost {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		token = r.Header.Get("X-API-Token")
		var body struct {
			Fingerprints []string `json:"fingerprints"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		sent = body.Fingerprints
		// The second run is on the account.
		json.NewEncoder(w).Encode(map[string]any{"known": map[string]string{body.Fingerprints[1]: "pub2"}})
	}))
	defer server.Close()

	logPath, _ := asTestGame(t)
	a := &App{apiBaseURL: server.URL, config: Config{ApiToken: "tok"}, sessionLoaded: true}
	scan, err := a.ScanAsLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	known, err := a.KnownAsRuns()
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok" || len(sent) != 2 || sent[0] != scan.Runs[0].Fingerprint || len(sent[0]) != 64 {
		t.Fatalf("token %q, sent %v", token, sent)
	}
	if len(known) != 1 || known[0].Index != scan.Runs[1].Index || known[0].PublicID != "pub2" {
		t.Errorf("known = %+v", known)
	}
}

// Signed out, nothing scanned, or a gateway without the route: nothing is marked and nothing fails.
func TestKnownAsRunsQuietWhenItCannotTell(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	logPath, _ := asTestGame(t)

	a := &App{apiBaseURL: server.URL, config: Config{ApiToken: "tok"}, sessionLoaded: true}
	if known, err := a.KnownAsRuns(); err != nil || len(known) != 0 || calls != 0 {
		t.Errorf("before a scan: %v %v, %d calls", known, err, calls)
	}
	if _, err := a.ScanAsLog(logPath); err != nil {
		t.Fatal(err)
	}
	if known, err := a.KnownAsRuns(); err != nil || len(known) != 0 || calls != 1 {
		t.Errorf("old gateway: %v %v, %d calls", known, err, calls)
	}
	signedOut := &App{apiBaseURL: server.URL, sessionLoaded: true}
	if _, err := signedOut.ScanAsLog(logPath); err != nil {
		t.Fatal(err)
	}
	if known, err := signedOut.KnownAsRuns(); err != nil || len(known) != 0 || calls != 1 {
		t.Errorf("signed out: %v %v, %d calls", known, err, calls)
	}
}

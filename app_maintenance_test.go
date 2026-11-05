package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchiveLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, combatLogFileName)
	if err := os.WriteFile(logPath, []byte("raid"), 0644); err != nil {
		t.Fatal(err)
	}
	archived, err := archiveLogFile(logPath, time.Date(2026, 11, 5, 21, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(archived) != "WoWCombatLog-2026-11-05-213000.txt" {
		t.Fatalf("got %s", archived)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("the live log should be gone")
	}
	if data, _ := os.ReadFile(archived); string(data) != "raid" {
		t.Fatal("archive content differs")
	}
	if _, err := archiveLogFile(logPath, time.Now()); err == nil {
		t.Fatal("expected an error for a missing log")
	}
}

func TestNextFileAfterArchiveIsAllNew(t *testing.T) {
	state := tailStateAfterArchive(TailState{
		TailFingerprint:       []string{"old"},
		LastByteOffset:        900,
		SourceFileSize:        900,
		ManualFingerprint:     []string{"old"},
		ManualOffset:          900,
		BaselineEstablishedAt: "2026-01-01T00:00:00Z",
	})
	path := filepath.Join(t.TempDir(), combatLogFileName)
	if err := os.WriteFile(path, []byte("3/22 20:00:01.123  SPELL_DAMAGE,x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	split, err := findTailSplitOffset(path, state)
	if err != nil || split.SplitOffset != 0 {
		t.Fatalf("split=%+v err=%v", split, err)
	}
	if state.BaselineEstablishedAt == "" {
		t.Fatal("auto-upload should stay configured")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"3.2.4", "4.0.0", -1},
		{"4.0.0", "3.2.4", 1},
		{"3.10.0", "3.9.9", 1},
		{"v1.3.0", "1.3", 0},
		{"1.3.0", "1.3.1", -1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("%s vs %s: got %d", c.a, c.b, got)
		}
	}
}

func TestUpdateInfoFor(t *testing.T) {
	info := updateInfoFor("3.2.4", &releaseInfo{Version: "4.0.0", DownloadURL: "https://x/app.exe", MinSupportedVersion: "2.0.0"})
	if !info.Available || info.Required {
		t.Fatalf("got %+v", info)
	}
	info = updateInfoFor("1.9.0", &releaseInfo{Version: "4.0.0", DownloadURL: "https://x/app.exe", MinSupportedVersion: "2.0.0"})
	if !info.Required {
		t.Fatalf("got %+v", info)
	}
	if updateInfoFor("3.2.4", &releaseInfo{Version: "4.0.0"}).Available {
		t.Fatal("a release with no download is not available")
	}
	if updateInfoFor("4.0.0", &releaseInfo{Version: "4.0.0", DownloadURL: "https://x"}).Available {
		t.Fatal("same version is not an update")
	}
}

func TestDownloadVerified(t *testing.T) {
	payload := []byte("new build")
	sum := sha256.Sum256(payload)
	good := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "app.new")

	if err := downloadVerified(server.Client(), server.URL, strings.ToUpper(good), destination, true); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(destination); string(data) != string(payload) {
		t.Fatal("content differs")
	}

	if err := downloadVerified(server.Client(), server.URL, strings.Repeat("0", 64), destination, true); err == nil {
		t.Fatal("expected a checksum failure")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("a failed download must not be left behind")
	}
	if err := downloadVerified(server.Client(), server.URL, "", destination, true); err == nil {
		t.Fatal("a release without a checksum must be refused")
	}
	if err := downloadVerified(server.Client(), server.URL, good, destination, false); err == nil {
		t.Fatal("a non-https link must be refused")
	}
}

func TestSwapExecutable(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "app.exe")
	downloaded := current + ".new"
	if err := os.WriteFile(current, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := swapExecutable(current, filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected an error")
	}
	if data, _ := os.ReadFile(current); string(data) != "old" {
		t.Fatal("a failed swap must restore the old build")
	}
	if err := os.WriteFile(downloaded, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := swapExecutable(current, downloaded); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(current); string(data) != "new" {
		t.Fatal("swap did not install the new build")
	}
}

func TestTocVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "WowLogsAddon.toc")
	if err := os.WriteFile(path, []byte("## Interface: 30300\r\n## Version: 1.3.0\r\nsrc\\Core.lua\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := tocVersion(path); got != "1.3.0" {
		t.Fatalf("got %q", got)
	}
	if got := tocVersion(path + ".missing"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeAddonSyncKeepsOneEntryPerDataset(t *testing.T) {
	syncs := mergeAddonSync([]AddonDatasetSync{{Dataset: "guild", Rows: 1}, {Dataset: "leaderboard", Rows: 2}},
		AddonDatasetSync{Dataset: "leaderboard", Rows: 9})
	if len(syncs) != 2 || syncs[0].Dataset != "leaderboard" || syncs[0].Rows != 9 {
		t.Fatalf("got %+v", syncs)
	}
}

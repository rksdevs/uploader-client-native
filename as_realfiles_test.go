package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestAsRealFilesCleanCopy runs scan and clean copy on a real game folder, read only:
//
//	AS_REAL_GAME_DIR=D:\AzerothSurvivorsWindowsNew AS_REAL_COPIES_DIR=<scratch> go test -run RealFiles -v .
func TestAsRealFilesCleanCopy(t *testing.T) {
	game, copies := os.Getenv("AS_REAL_GAME_DIR"), os.Getenv("AS_REAL_COPIES_DIR")
	if game == "" || copies == "" {
		t.Skip("set AS_REAL_GAME_DIR and AS_REAL_COPIES_DIR")
	}
	var cfg Config
	if err := useAsGameFolder(&cfg, game); err != nil {
		t.Fatal(err)
	}
	cfg.AsCopiesDir = copies
	loc := resolveAsLocations(cfg)
	originals := []string{loc.LogPath, loc.AddonPath, loc.StatsPath}
	hash := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			return "missing"
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:8])
	}
	before := map[string]string{}
	for _, path := range originals {
		before[path] = hash(path)
	}

	a := &App{config: cfg}
	scan, err := a.ScanAsLog(loc.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	var indexes []int
	addonMatched, statsMatched := 0, 0
	for i, run := range scan.Runs {
		m := scan.Matches[i]
		t.Logf("run %2d %s %6.1fs %5d kills  addon=%-5v stats=%v", run.Index, run.Start, float64(run.ActiveMs)/1000, run.Kills, m.Addon, m.Stats)
		if run.ActiveMs >= asMinActiveMs {
			indexes = append(indexes, run.Index)
		}
		if m.Addon {
			addonMatched++
		}
		if m.Stats {
			statsMatched++
		}
	}
	t.Logf("%d runs; addon save %d runs, %d matched; stats save %d runs, %d matched; errors %q %q",
		len(scan.Runs), scan.AddonRuns, addonMatched, scan.StatsRuns, statsMatched, scan.AddonError, scan.StatsError)

	result, err := a.CreateAsCleanCopy(AsCopyRequest{Indexes: indexes, IncludeAddon: true, IncludeStats: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range result.Files {
		info, _ := os.Stat(file)
		t.Logf("wrote %s (%d bytes)", filepath.Base(file), info.Size())
	}

	// The copy reads back with the same runs and matches.
	again := &App{config: Config{AsAddonPath: filepath.Join(result.Dir, asAddonName), AsStatsPath: filepath.Join(result.Dir, asStatsName)}}
	copyScan, err := again.ScanAsLog(filepath.Join(result.Dir, asLogName))
	if err != nil {
		t.Fatal(err)
	}
	copyAddon, copyStats := 0, 0
	for _, m := range copyScan.Matches {
		if m.Addon {
			copyAddon++
		}
		if m.Stats {
			copyStats++
		}
	}
	if len(copyScan.Runs) != len(indexes) || copyAddon != result.AddonRuns || copyStats != result.StatsRuns {
		t.Errorf("copy: %d runs (want %d), addon %d (want %d), stats %d (want %d)",
			len(copyScan.Runs), len(indexes), copyAddon, result.AddonRuns, copyStats, result.StatsRuns)
	}

	for _, path := range originals {
		if now := hash(path); now != before[path] {
			t.Errorf("%s changed: %s -> %s", path, before[path], now)
		}
	}
}

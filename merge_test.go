package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeLog(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\r\n")+"\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	return path
}

func hit(stamp, target string) string {
	return fmt.Sprintf(`%s  SPELL_DAMAGE,0x0700000000000001,"Alpha",0x514,0xF130008EF5000001,"%s",0x10a48,1,"Bolt",0x4,100`, stamp, target)
}

func TestScanMergeFileFindsRangeBossesAndRealm(t *testing.T) {
	path := writeLog(t, t.TempDir(), "a.txt",
		hit("3/15 20:00:00.000", "Lord Marrowgar"),
		hit("3/15 20:30:00.000", "Prince Valanar"),
		hit("3/15 21:00:00.500", "Some Trash"),
	)
	file := scanMergeFile(path)
	if file.Error != "" {
		t.Fatal(file.Error)
	}
	if file.LastMs-file.FirstMs != 3600500 {
		t.Fatalf("range %d", file.LastMs-file.FirstMs)
	}
	if strings.Join(file.Bosses, ",") != "Blood Prince Council,Lord Marrowgar" {
		t.Fatalf("bosses %v", file.Bosses)
	}
	if len(file.Realms) != 1 || file.Realms[0] != "Warmane_Icecrown" {
		t.Fatalf("realms %v", file.Realms)
	}
}

func TestScanMergeFileWithoutCombatLines(t *testing.T) {
	path := writeLog(t, t.TempDir(), "notes.txt", "hello", "world")
	if scanMergeFile(path).Error == "" {
		t.Fatal("expected an error for a non-log file")
	}
	if scanMergeFile(filepath.Join(t.TempDir(), "missing.txt")).Error == "" {
		t.Fatal("expected an error for a missing file")
	}
}

func TestInferLogYearAcrossNewYear(t *testing.T) {
	january := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	if got := inferLogYear("12/31 20:00:00.000", january); got != 2025 {
		t.Fatalf("got %d", got)
	}
	if got := inferLogYear("1/1 20:00:00.000", january); got != 2026 {
		t.Fatalf("got %d", got)
	}
}

func TestPlanMergeOrdersFilesAndClassifiesLinks(t *testing.T) {
	hour := int64(3600000)
	plan := planMerge([]MergeFile{
		{Name: "day2", SizeBytes: 10, FirstMs: 30 * hour, LastMs: 31 * hour, Bosses: []string{"The Lich King"}},
		{Name: "part2", SizeBytes: 10, FirstMs: 2*hour - 14000, LastMs: 3 * hour, Bosses: []string{"Sindragosa"}},
		{Name: "part1", SizeBytes: 10, FirstMs: 1 * hour, LastMs: 2 * hour, Bosses: []string{"Sindragosa", "Festergut"}},
	})
	if plan.Files[0].Name != "part1" || plan.Files[2].Name != "day2" {
		t.Fatalf("order: %s %s", plan.Files[0].Name, plan.Files[2].Name)
	}
	if !plan.Links[0].Overlap || plan.Links[0].GapMs != -14000 {
		t.Fatalf("link 0: %+v", plan.Links[0])
	}
	if !plan.Links[1].NextDay || plan.Links[1].Overlap {
		t.Fatalf("link 1: %+v", plan.Links[1])
	}
	if !plan.MultiDay || !plan.CanMerge || len(plan.Bosses) != 3 || plan.TotalBytes != 30 {
		t.Fatalf("plan: %+v", plan)
	}
}

func TestPlanMergeWarnings(t *testing.T) {
	week := int64(8 * 24 * 3600000)
	plan := planMerge([]MergeFile{
		{Name: "a", FirstMs: 1000, LastMs: 2000, Realms: []string{"Warmane_Icecrown"}, Header: "COMBAT_LOG_VERSION,9"},
		{Name: "b", FirstMs: week, LastMs: week + 1000, Realms: []string{"Warmane_Onyxia"}, Header: "COMBAT_LOG_VERSION,8"},
	})
	if len(plan.Warnings) != 3 {
		t.Fatalf("warnings: %v", plan.Warnings)
	}

	single := planMerge([]MergeFile{{Name: "a", FirstMs: 1, LastMs: 2}})
	if single.CanMerge {
		t.Fatal("one file is not a merge")
	}
	broken := planMerge([]MergeFile{{Name: "a", FirstMs: 1, LastMs: 2}, {Name: "b", Error: "unreadable"}})
	if broken.CanMerge {
		t.Fatal("unreadable files block the merge")
	}
	huge := planMerge([]MergeFile{
		{Name: "a", FirstMs: 1, LastMs: 2, SizeBytes: maxUploadSourceBytes},
		{Name: "b", FirstMs: 3, LastMs: 4, SizeBytes: 1},
	})
	if huge.CanMerge {
		t.Fatal("over the size limit must not merge")
	}
}

func TestWriteMergedDropsOverlapDuplicatesAndExtraHeaders(t *testing.T) {
	dir := t.TempDir()
	header := "3/15 19:59:59.000  COMBAT_LOG_VERSION,9,ADVANCED_LOG_ENABLED,1"
	a := hit("3/15 20:00:00.000", "A")
	shared := hit("3/15 20:10:00.000", "Shared")
	onlySecond := hit("3/15 20:10:00.500", "OnlySecond")
	later := hit("3/15 20:20:00.000", "Later")

	first := writeLog(t, dir, "1.txt", header, a, shared)
	second := writeLog(t, dir, "2.txt", header, shared, onlySecond, later)

	plan := planMerge([]MergeFile{scanMergeFile(second), scanMergeFile(first)})
	if plan.Links[0].GapMs >= 0 && plan.Files[1].FirstMs > plan.Files[0].LastMs {
		t.Fatalf("expected the files to overlap: %+v", plan.Links)
	}

	var out bytes.Buffer
	duplicates, err := writeMerged(plan.Files, &out)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	want := []string{header, a, shared, onlySecond, later}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("merged lines:\n%s", out.String())
	}
	if duplicates != 1 {
		t.Fatalf("duplicates %d", duplicates)
	}
}

func TestWriteMergedKeepsEverythingWithoutOverlap(t *testing.T) {
	dir := t.TempDir()
	first := writeLog(t, dir, "1.txt", hit("3/15 20:00:00.000", "A"), hit("3/15 21:00:00.000", "A"))
	second := writeLog(t, dir, "2.txt", hit("3/16 20:00:00.000", "A"), hit("3/16 21:00:00.000", "A"))
	plan := planMerge([]MergeFile{scanMergeFile(first), scanMergeFile(second)})
	var out bytes.Buffer
	duplicates, err := writeMerged(plan.Files, &out)
	if err != nil {
		t.Fatal(err)
	}
	if duplicates != 0 || strings.Count(out.String(), "\n") != 4 {
		t.Fatalf("duplicates=%d output=%q", duplicates, out.String())
	}
}

func TestMergeToFileRefusesUnreadableInput(t *testing.T) {
	dir := t.TempDir()
	good := writeLog(t, dir, "1.txt", hit("3/15 20:00:00.000", "A"))
	if _, _, err := mergeToFile([]string{good, filepath.Join(dir, "missing.txt")}, filepath.Join(dir, "out.txt")); err == nil {
		t.Fatal("expected an error")
	}
}

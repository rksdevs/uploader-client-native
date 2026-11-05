package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testHistory(t *testing.T) *historyStore {
	t.Helper()
	return &historyStore{path: filepath.Join(t.TempDir(), "sub", historyFileName)}
}

func TestHistoryNewestFirstAndReplace(t *testing.T) {
	store := testHistory(t)
	for _, id := range []int{1, 2} {
		if err := store.add(HistoryEntry{PreprocessID: id, Status: historyProcessing}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.add(HistoryEntry{PreprocessID: 1, Status: historyUploaded}); err != nil {
		t.Fatal(err)
	}
	entries := store.load()
	if len(entries) != 2 || entries[0].PreprocessID != 1 || entries[0].Status != historyUploaded {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestHistorySkippedEntriesWithoutIDAreAllKept(t *testing.T) {
	store := testHistory(t)
	for i := 0; i < 2; i++ {
		if err := store.add(HistoryEntry{Status: historySkipped}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(store.load()); got != 2 {
		t.Fatalf("got %d entries, want 2", got)
	}
}

func TestHistoryIsCapped(t *testing.T) {
	store := testHistory(t)
	entries := make([]HistoryEntry, 0, historyMaxEntries+5)
	for i := 0; i < historyMaxEntries+5; i++ {
		entries = append(entries, HistoryEntry{PreprocessID: i + 1})
	}
	if err := store.save(entries); err != nil {
		t.Fatal(err)
	}
	if got := len(store.load()); got != historyMaxEntries {
		t.Fatalf("got %d entries", got)
	}
}

func TestHistoryUpdateAndUnreadableFile(t *testing.T) {
	store := testHistory(t)
	if err := store.add(HistoryEntry{PreprocessID: 3}); err != nil {
		t.Fatal(err)
	}
	if err := store.update(3, func(e *HistoryEntry) { e.LogIDs = append(e.LogIDs, 77) }); err != nil {
		t.Fatal(err)
	}
	if got := store.load()[0].LogIDs; len(got) != 1 || got[0] != 77 {
		t.Fatalf("log ids: %v", got)
	}
	if err := os.WriteFile(store.path, []byte("{broken"), 0644); err != nil {
		t.Fatal(err)
	}
	if store.load() != nil {
		t.Fatal("unreadable history should load as empty")
	}
}

func TestHistoryEntryForInstances(t *testing.T) {
	entry := historyEntryForInstances(5, historySourceManual, "Warmane_Icecrown", []Instance{
		{Name: "9/25 19:42:14.066", StartMs: 1000, EndMs: 4000, Unlisted: true, Preview: &InstancePreview{Bosses: []string{"Sindragosa"}}},
		{Name: "RS", StartMs: 5000, EndMs: 6000},
	}, false)
	if entry.DurationMs != 4000 || !entry.Unlisted || len(entry.Raids) != 2 || len(entry.Bosses) != 1 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if got := historySummary(entry); got != fmt.Sprintf("Icecrown Citadel +%d more", 1) {
		t.Fatalf("summary %q", got)
	}
}

func TestHistoryStatusForOutcome(t *testing.T) {
	cases := map[jobOutcome]string{
		jobOutcomeAllUploaded: historyUploaded,
		jobOutcomePartial:     historyPartial,
		jobOutcomeAllFailed:   historyFailed,
		jobOutcomeTimedOut:    historyTimedOut,
	}
	for outcome, want := range cases {
		if got, _ := historyStatusForOutcome(outcome); got != want {
			t.Errorf("outcome %d: got %s want %s", outcome, got, want)
		}
	}
}

func TestUnfinishedUploads(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Hour).Format(time.RFC3339)
	old := now.Add(-jobPollLimit - time.Hour).Format(time.RFC3339)
	got := unfinishedUploads([]HistoryEntry{
		{PreprocessID: 1, Status: historyProcessing, StartedAt: recent},
		{PreprocessID: 2, Status: historyProcessing, StartedAt: old},
		{PreprocessID: 3, Status: historyUploaded, StartedAt: recent},
		{PreprocessID: 0, Status: historyProcessing, StartedAt: recent},
	}, now)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestInstanceRaidNameFallsBackToInstanceName(t *testing.T) {
	if got := instanceRaidName(Instance{Name: "9/25 19:42:14.066"}); got != "9/25 19:42:14.066" {
		t.Fatalf("got %q", got)
	}
	mixed := Instance{Preview: &InstancePreview{Bosses: []string{"Halion", "Sindragosa", "The Lich King"}}}
	if got := instanceRaidName(mixed); got != "Icecrown Citadel" {
		t.Fatalf("got %q", got)
	}
}

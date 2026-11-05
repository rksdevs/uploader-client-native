package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	historyFileName   = "history.json"
	historyMaxEntries = 200

	historySourceManual = "manual"
	historySourceAuto   = "auto"
	historySourceMerge  = "merge"

	historyProcessing = "processing"
	historyUploaded   = "uploaded"
	historyPartial    = "partial"
	historyFailed     = "failed"
	historyTimedOut   = "timed_out"
	historySkipped    = "skipped"
)

// HistoryEntry is one upload shown under Recent Uploads and Automatic Upload History. Kept locally
// because public logs have no owner on the server, and token users have no account to query.
type HistoryEntry struct {
	PreprocessID int      `json:"preprocessId"`
	Source       string   `json:"source"`
	ServerName   string   `json:"serverName"`
	Raids        []string `json:"raids"`
	Bosses       []string `json:"bosses"`
	StartedAt    string   `json:"startedAt"`
	DurationMs   int64    `json:"durationMs"`
	Status       string   `json:"status"`
	Message      string   `json:"message,omitempty"`
	LogIDs       []int    `json:"logIds"`
	Unlisted     bool     `json:"unlisted,omitempty"`
	RetryPath    string   `json:"retryPath,omitempty"`
}

type historyStore struct {
	mu   sync.Mutex
	path string
}

func (s *historyStore) load() []HistoryEntry {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Printf("[History] Ignoring unreadable history file: %v\n", err)
		return nil
	}
	return entries
}

func (s *historyStore) save(entries []HistoryEntry) error {
	if len(entries) > historyMaxEntries {
		entries = entries[:historyMaxEntries]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// add puts the entry first, newest on top, replacing an entry for the same upload.
func (s *historyStore) add(entry HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.load()
	kept := make([]HistoryEntry, 0, len(entries)+1)
	kept = append(kept, entry)
	for _, existing := range entries {
		if entry.PreprocessID == 0 || existing.PreprocessID != entry.PreprocessID {
			kept = append(kept, existing)
		}
	}
	return s.save(kept)
}

func (s *historyStore) update(preprocessID int, change func(*HistoryEntry)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.load()
	for i := range entries {
		if entries[i].PreprocessID == preprocessID {
			change(&entries[i])
			return s.save(entries)
		}
	}
	return nil
}

func (a *App) history() *historyStore {
	a.historyOnce.Do(func() {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = os.TempDir()
		}
		a.historyStore = &historyStore{path: filepath.Join(configDir, "WoWLogsUploader", historyFileName)}
	})
	return a.historyStore
}

// GetUploadHistory returns recent uploads, newest first.
func (a *App) GetUploadHistory() []HistoryEntry {
	store := a.history()
	store.mu.Lock()
	defer store.mu.Unlock()
	entries := store.load()
	if entries == nil {
		return []HistoryEntry{}
	}
	return entries
}

// instanceRaidName names the raid from its bosses; the instance's own name is a start timestamp.
func instanceRaidName(instance Instance) string {
	counts := map[string]int{}
	best := ""
	if instance.Preview != nil {
		for _, boss := range instance.Preview.Bosses {
			raid, ok := bossRaids[boss]
			if !ok {
				continue
			}
			counts[raid]++
			if counts[raid] > counts[best] {
				best = raid
			}
		}
	}
	if best != "" {
		return best
	}
	return instance.Name
}

func historyEntryForInstances(preprocessID int, source, serverName string, instances []Instance, unlisted bool) HistoryEntry {
	entry := HistoryEntry{
		PreprocessID: preprocessID,
		Source:       source,
		ServerName:   serverName,
		StartedAt:    time.Now().UTC().Format(time.RFC3339),
		Status:       historyProcessing,
		Unlisted:     unlisted,
		Raids:        []string{},
		Bosses:       []string{},
		LogIDs:       []int{},
	}
	for _, instance := range instances {
		entry.Raids = append(entry.Raids, instanceRaidName(instance))
		if instance.EndMs > instance.StartMs {
			entry.DurationMs += instance.EndMs - instance.StartMs
		}
		if instance.Preview != nil {
			entry.Bosses = append(entry.Bosses, instance.Preview.Bosses...)
		}
		if instance.Unlisted {
			entry.Unlisted = true
		}
	}
	return entry
}

// recordUploadStarted adds the history entry once an upload's jobs are queued.
func (a *App) recordUploadStarted(preprocessID int, instances []Instance, unlisted bool) {
	source := uploadSource{Source: historySourceManual}
	if stored, ok := a.uploadSources.LoadAndDelete(preprocessID); ok {
		source = stored.(uploadSource)
	}
	a.recordHistory(historyEntryForInstances(preprocessID, source.Source, source.ServerName, instances, unlisted))
}

func (a *App) recordHistory(entry HistoryEntry) {
	if err := a.history().add(entry); err != nil {
		log.Printf("[History] Could not save entry: %v\n", err)
	}
}

func (a *App) addHistoryLog(preprocessID, logID int, status string) {
	err := a.history().update(preprocessID, func(entry *HistoryEntry) {
		entry.LogIDs = append(entry.LogIDs, logID)
	})
	if err != nil {
		log.Printf("[History] Could not update entry: %v\n", err)
	}
}

func historyStatusForOutcome(outcome jobOutcome) (string, string) {
	switch outcome {
	case jobOutcomeAllUploaded:
		return historyUploaded, ""
	case jobOutcomePartial:
		return historyPartial, "Some raids failed to process."
	case jobOutcomeAllFailed:
		return historyFailed, "The server could not process this log."
	default:
		return historyTimedOut, "Still in the server queue. Check View all logs later."
	}
}

func (a *App) finishHistoryEntry(preprocessID int, outcome jobOutcome) {
	status, message := historyStatusForOutcome(outcome)
	err := a.history().update(preprocessID, func(entry *HistoryEntry) {
		entry.Status = status
		entry.Message = message
	})
	if err != nil {
		log.Printf("[History] Could not update entry: %v\n", err)
	}
}

func (a *App) setHistoryRetryPath(preprocessID int, path string) {
	if err := a.history().update(preprocessID, func(entry *HistoryEntry) { entry.RetryPath = path }); err != nil {
		log.Printf("[History] Could not update entry: %v\n", err)
	}
}

// unfinishedUploads lists uploads to keep watching after a restart. Anything older than the poll
// limit is left alone: the monitor would give up on it straight away.
func unfinishedUploads(entries []HistoryEntry, now time.Time) []int {
	ids := []int{}
	for _, entry := range entries {
		if entry.Status != historyProcessing || entry.PreprocessID == 0 {
			continue
		}
		started, err := time.Parse(time.RFC3339, entry.StartedAt)
		if err == nil && now.Sub(started) < jobPollLimit {
			ids = append(ids, entry.PreprocessID)
		}
	}
	return ids
}

// resumeUnfinishedUploads restarts monitors for uploads the app stopped watching when it closed.
func (a *App) resumeUnfinishedUploads() {
	for _, id := range unfinishedUploads(a.GetUploadHistory(), time.Now()) {
		a.startJobMonitor(id, nil, "")
	}
}

func historySummary(entry HistoryEntry) string {
	if len(entry.Raids) == 0 {
		return "Upload"
	}
	if len(entry.Raids) == 1 {
		return entry.Raids[0]
	}
	return fmt.Sprintf("%s +%d more", entry.Raids[0], len(entry.Raids)-1)
}

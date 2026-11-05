package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManualTailStatePrefersManualMarker(t *testing.T) {
	state := TailState{
		TailFingerprint:       []string{"auto"},
		LastByteOffset:        10,
		BaselineEstablishedAt: "2026-01-01T00:00:00Z",
		ManualFingerprint:     []string{"manual"},
		ManualOffset:          20,
	}
	marker, ok := manualTailState(state)
	if !ok || marker.LastByteOffset != 20 || marker.TailFingerprint[0] != "manual" {
		t.Fatalf("marker=%+v ok=%v", marker, ok)
	}

	state.ManualFingerprint = nil
	marker, ok = manualTailState(state)
	if !ok || marker.LastByteOffset != 10 {
		t.Fatalf("expected the auto baseline, got %+v ok=%v", marker, ok)
	}

	if _, ok := manualTailState(TailState{}); ok {
		t.Fatal("no previous upload should report no marker")
	}
}

func TestResolveUploadPath(t *testing.T) {
	if got := resolveUploadPath("C:/x/raid.txt", "C:/logs"); got != "C:/x/raid.txt" {
		t.Fatalf("got %s", got)
	}
	if got := resolveUploadPath(" ", "C:/logs"); !strings.HasSuffix(filepath.ToSlash(got), "C:/logs/"+combatLogFileName) {
		t.Fatalf("got %s", got)
	}
}

func TestKeepStagingForRetry(t *testing.T) {
	dir := t.TempDir()
	staging := filepath.Join(dir, stagingFileName)
	if err := os.WriteFile(staging, []byte("slice"), 0644); err != nil {
		t.Fatal(err)
	}
	retry, err := keepStagingForRetry(staging, 42)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(retry) != "retry-42.txt" {
		t.Fatalf("got %s", retry)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatal("staging name should be free for the next slice")
	}
}

func TestUploadNotificationTextCoversEveryOutcome(t *testing.T) {
	for _, outcome := range []jobOutcome{jobOutcomeAllUploaded, jobOutcomePartial, jobOutcomeAllFailed, jobOutcomeTimedOut} {
		if title, body := uploadNotificationText(outcome); title == "" || body == "" {
			t.Fatalf("outcome %d has no text", outcome)
		}
	}
}

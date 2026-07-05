package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeCombatLogActivity_pendingGrowth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	baseline := "4/24 20:00:01.123  SPELL_DAMAGE,old\n"
	grown := baseline + "5/07 16:30:00.000  SPELL_DAMAGE,new\n"
	if err := os.WriteFile(path, []byte(grown), 0644); err != nil {
		t.Fatal(err)
	}

	state := TailState{
		LastByteOffset: int64(len(baseline)),
		SourceFileSize: int64(len(baseline)),
	}
	activity, err := computeCombatLogActivity(path, state)
	if err != nil {
		t.Fatal(err)
	}
	if !activity.HasPendingChanges {
		t.Fatal("expected pending changes")
	}
	if activity.PendingBytes != int64(len("5/07 16:30:00.000  SPELL_DAMAGE,new\n")) {
		t.Fatalf("unexpected pending bytes: %d", activity.PendingBytes)
	}
	if !strings.Contains(activity.LastLinePreview, "new") {
		t.Fatalf("expected latest line preview, got %q", activity.LastLinePreview)
	}
}

func TestReconcileTailStateOnStartup_appendOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	baseline := "4/24 20:00:01.123  SPELL_DAMAGE,old\n"
	if err := os.WriteFile(path, []byte(baseline), 0644); err != nil {
		t.Fatal(err)
	}

	state := TailState{
		TailFingerprint:       []string{"4/24 20:00:01.123  SPELL_DAMAGE,old"},
		LastByteOffset:        10, // legacy: fingerprint start, not EOF
		SourceFileSize:        int64(len(baseline)),
		BaselineEstablishedAt: "2026-01-01T00:00:00Z",
	}

	grown := baseline + "5/07 16:30:00.000  SPELL_DAMAGE,new\n"
	if err := os.WriteFile(path, []byte(grown), 0644); err != nil {
		t.Fatal(err)
	}

	if err := reconcileTailStateOnStartup(path, &state); err != nil {
		t.Fatal(err)
	}
	if state.LastByteOffset != int64(len(baseline)) {
		t.Fatalf("expected synced EOF %d, got %d", len(baseline), state.LastByteOffset)
	}
	if state.SourceFileSize != int64(len(grown)) {
		t.Fatalf("expected size %d, got %d", len(grown), state.SourceFileSize)
	}
}

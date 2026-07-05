package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindTailSplitOffset_newLinesAfterFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	lines := []string{
		"3/15 20:00:01.123  SPELL_DAMAGE,old1",
		"3/15 20:00:02.123  SPELL_DAMAGE,old2",
		"3/15 20:00:03.123  SPELL_DAMAGE,old3",
		"3/15 20:00:04.123  SPELL_DAMAGE,new1",
		"3/15 20:00:05.123  SPELL_DAMAGE,new2",
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	prefix := strings.Join(lines[:3], "\n") + "\n"
	fp, _, err := fingerprintFromBytes([]byte(prefix), 3)
	if err != nil {
		t.Fatal(err)
	}
	state := TailState{
		TailFingerprint: fp,
		LastByteOffset:  int64(len(prefix)),
		SourceFileSize:  int64(len(prefix)),
	}

	split, err := findTailSplitOffset(path, state)
	if err != nil {
		t.Fatalf("findTailSplitOffset: %v", err)
	}
	if split.SplitOffset <= 0 {
		t.Fatalf("expected positive split offset, got %d", split.SplitOffset)
	}

	staging := filepath.Join(dir, "staging.txt")
	written, err := copyCombatLogDelta(path, staging, split.SplitOffset)
	if err != nil {
		t.Fatalf("copyCombatLogDelta: %v", err)
	}
	if written == 0 {
		t.Fatal("expected bytes written")
	}
	data, err := os.ReadFile(staging)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "new1") || !strings.Contains(body, "new2") {
		t.Fatalf("staging missing new lines: %q", body)
	}
	if strings.Contains(body, "old1") {
		t.Fatalf("staging should not include old lines: %q", body)
	}
}

func TestFindTailSplitOffset_noDelta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	content := "3/15 20:00:01.123  SPELL_DAMAGE,only\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	fp, _, err := fingerprintFromBytes([]byte(content), 1)
	if err != nil {
		t.Fatal(err)
	}
	state := TailState{
		TailFingerprint: fp,
		LastByteOffset:  int64(len(content)),
		SourceFileSize:  int64(len(content)),
	}
	_, err = findTailSplitOffset(path, state)
	if err != errNoCombatLogDelta {
		t.Fatalf("expected errNoCombatLogDelta, got %v", err)
	}
}

func TestReconcileTailStateOnStartup_trimmedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	original := strings.Join([]string{
		"3/15 20:00:01.123  SPELL_DAMAGE,trim1",
		"3/15 20:00:02.123  SPELL_DAMAGE,trim2",
		"3/15 20:00:03.123  SPELL_DAMAGE,trim3",
	}, "\n") + "\n"
	trimmed := "3/15 20:00:02.123  SPELL_DAMAGE,trim2\n3/15 20:00:03.123  SPELL_DAMAGE,trim3\n"
	if err := os.WriteFile(path, []byte(trimmed), 0644); err != nil {
		t.Fatal(err)
	}

	fp, offset, err := fingerprintFromBytes([]byte(original), 2)
	if err != nil {
		t.Fatal(err)
	}
	state := TailState{
		TailFingerprint:     fp,
		LastByteOffset:        offset,
		SourceFileSize:        int64(len(original)),
		BaselineEstablishedAt: "2026-01-01T00:00:00Z",
	}
	if err := reconcileTailStateOnStartup(path, &state); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if state.SourceFileSize != int64(len(trimmed)) {
		t.Fatalf("expected reconciled size %d, got %d", len(trimmed), state.SourceFileSize)
	}
}

func TestParseWotLKLineTimestamp(t *testing.T) {
	ts, ok := parseWotLKLineTimestamp("3/15 20:00:01.123  SPELL_DAMAGE,x")
	if !ok {
		t.Fatal("expected ok")
	}
	if ts.Month() != 3 || ts.Day() != 15 || ts.Hour() != 20 {
		t.Fatalf("unexpected timestamp: %v", ts)
	}
}

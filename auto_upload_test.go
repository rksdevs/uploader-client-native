package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTailFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, combatLogFileName)
	content := strings.Join([]string{
		"3/15 20:00:01.123  SPELL_DAMAGE,line1",
		"",
		"3/15 20:00:02.123  SPELL_DAMAGE,line2",
		"3/15 20:00:03.123  SPELL_DAMAGE,line3",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	lines, offset, size, modTime, err := readTailFingerprint(path, 2)
	if err != nil {
		t.Fatalf("readTailFingerprint: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "line2") || !strings.Contains(lines[1], "line3") {
		t.Fatalf("unexpected tail lines: %v", lines)
	}
	if offset <= 0 {
		t.Fatalf("expected positive byte offset, got %d", offset)
	}
	if size <= 0 || modTime <= 0 {
		t.Fatalf("expected file metadata, size=%d modTime=%d", size, modTime)
	}
}

func TestIsServerAllowedForAutoUpload(t *testing.T) {
	app := &App{}
	if !app.IsServerAllowedForAutoUpload("Warmane_Icecrown") {
		t.Fatal("expected Warmane_Icecrown allowed")
	}
	if app.IsServerAllowedForAutoUpload(blockedAutoUploadServer) {
		t.Fatal("expected Whitemane_Gilneas blocked")
	}
}

func TestAutoUploadBlockReason(t *testing.T) {
	app := &App{config: Config{LogDirectory: "C:\\Logs", ApiToken: "tok"}}
	if reason := app.autoUploadBlockReason(""); reason == "" {
		t.Fatal("expected block reason for empty server")
	}
	if reason := app.autoUploadBlockReason(blockedAutoUploadServer); !strings.Contains(reason, "Gilneas") {
		t.Fatalf("expected Gilneas block reason, got %q", reason)
	}
	if reason := app.autoUploadBlockReason("Warmane_Icecrown"); reason != "" {
		t.Fatalf("expected no block reason, got %q", reason)
	}
}

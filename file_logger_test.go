package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrependDiagnosticLineCapNewestFirst(t *testing.T) {
	dir := t.TempDir()
	diagLogPath = filepath.Join(dir, "uploader.log")
	diagLogReady = true

	lineA := "LINE-A-newest"
	lineB := "LINE-B-older"
	lineC := "LINE-C-oldest"

	prependDiagnosticLine(lineC)
	prependDiagnosticLine(lineB)
	prependDiagnosticLine(lineA)

	data, err := os.ReadFile(diagLogPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, lineA) {
		t.Fatalf("expected newest line first, got:\n%s", text)
	}
	idxA := strings.Index(text, lineA)
	idxB := strings.Index(text, lineB)
	idxC := strings.Index(text, lineC)
	if idxA < 0 || idxB < 0 || idxC < 0 || idxA >= idxB || idxB >= idxC {
		t.Fatalf("expected order A,B,C in file, got:\n%s", text)
	}
}

func TestPrependDiagnosticLineTruncatesOldest(t *testing.T) {
	dir := t.TempDir()
	diagLogPath = filepath.Join(dir, "uploader.log")
	diagLogReady = true

	chunk := strings.Repeat("X", 400) + "\n"
	for i := 0; i < 4000; i++ {
		prependDiagnosticLine(chunk)
	}

	info, err := os.Stat(diagLogPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > maxDiagnosticLogBytes {
		t.Fatalf("log size %d exceeds cap %d", info.Size(), maxDiagnosticLogBytes)
	}

	data, err := os.ReadFile(diagLogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "X") {
		t.Fatalf("expected recent chunk at top after truncation")
	}
}

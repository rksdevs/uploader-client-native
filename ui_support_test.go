package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLastLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), combatLogFileName)
	if err := os.WriteFile(path, []byte("one\r\ntwo\r\n\r\nthree\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readLastLines(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "two,three" {
		t.Fatalf("got %v", got)
	}
}

func TestReadLastLinesDropsCutFirstLineOfLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), combatLogFileName)
	var content strings.Builder
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(&content, "line-%06d\n", i)
	}
	if err := os.WriteFile(path, []byte(content.String()), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readLastLines(path, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0]) != len("line-000000") || got[len(got)-1] != "line-019999" {
		t.Fatalf("first=%q last=%q", got[0], got[len(got)-1])
	}
}

func TestGetRecentLogLinesWithoutFolder(t *testing.T) {
	app := &App{}
	if got := app.GetRecentLogLines("", 8); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := app.GetRecentLogLines(t.TempDir(), 8); len(got) != 0 {
		t.Fatalf("missing log: %v", got)
	}
}

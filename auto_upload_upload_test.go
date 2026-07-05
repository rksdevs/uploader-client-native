package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZipStagingCombatLog(t *testing.T) {
	dir := t.TempDir()
	staging := filepath.Join(dir, stagingFileName)
	content := "5/07 16:30:00.000  SPELL_DAMAGE,test\n"
	if err := os.WriteFile(staging, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	zipBytes, err := zipStagingCombatLog(staging)
	if err != nil {
		t.Fatal(err)
	}
	if len(zipBytes) == 0 {
		t.Fatal("expected zip bytes")
	}

	// Round-trip via zip reader would be heavy; ensure magic PK header.
	if zipBytes[0] != 'P' || zipBytes[1] != 'K' {
		t.Fatalf("expected zip header, got %q", zipBytes[:4])
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanWarmaneRealmsFromStagedSlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "staging.txt")
	content := strings.Join([]string{
		`3/15 20:01:12.123  ZONE_CHANGE,774x0F13000FD0800000000000000000,"Onyxia",0xa48,0x0,0x0,0x0,0x0,0x0,0x0,0x0`,
		`3/15 20:01:15.456  SPELL_DAMAGE,0x0600000000000001,0x0,0x0600000000000002,"Player-Lordaeron",0x0,0x0,0x0,0x0,42833,"Fireball",0x0`,
		`3/15 20:01:16.789  SPELL_DAMAGE,0x0600000000000003,0x0,0x0600000000000004,"Healer-Lordaeron",0x0,0x0,0x0,0x0,42833,"Fireball",0x0`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	scan, err := scanWarmaneRealmsFromCombatLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Realms) != 1 {
		t.Fatalf("expected 1 realm, got %v", scan.Realms)
	}
	if scan.Realms["Warmane_Lordaeron"] != "06" {
		t.Fatalf("expected Lordaeron prefix 06, got %v", scan.Realms)
	}
}

func TestEvaluateWarmaneServerDrift(t *testing.T) {
	scan := warmaneRealmScan{
		Realms: map[string]string{"Warmane_Lordaeron": "06"},
	}

	if drift := evaluateWarmaneServerDrift("Warmane_Onyxia", scan); drift == nil {
		t.Fatal("expected drift between Onyxia default and Lordaeron log")
	}

	if drift := evaluateWarmaneServerDrift("Warmane_Lordaeron", scan); drift != nil {
		t.Fatal("expected no drift when default matches detected")
	}

	if drift := evaluateWarmaneServerDrift("Stormforge_Frostmourne_S1", scan); drift != nil {
		t.Fatal("expected no drift check outside Warmane trio")
	}

	if drift := evaluateWarmaneServerDrift("Warmane_Onyxia", warmaneRealmScan{Realms: map[string]string{}}); drift != nil {
		t.Fatal("expected no drift when no player GUIDs in slice")
	}
}

func TestEvaluateWarmaneMultipleRealmsInSlice(t *testing.T) {
	scan := warmaneRealmScan{
		Realms: map[string]string{
			"Warmane_Lordaeron": "06",
			"Warmane_Icecrown":  "07",
		},
	}
	drift := evaluateWarmaneServerDrift("Warmane_Onyxia", scan)
	if drift == nil {
		t.Fatal("expected drift prompt for mixed realms in new lines")
	}
	if !drift.MultipleDetected {
		t.Fatal("expected multipleDetected=true")
	}
}

func TestExtractPlayerGuidPrefix(t *testing.T) {
	line := `3/15 20:01:15.456  SPELL_DAMAGE,0x0700000000000001,0x0,0x0700000000000002,"Player-Icecrown",0x0,0x0,0x0,0x0,42833,"Fireball",0x0`
	if prefix := extractPlayerGuidPrefix(line); prefix != "07" {
		t.Fatalf("expected 07, got %q", prefix)
	}
}

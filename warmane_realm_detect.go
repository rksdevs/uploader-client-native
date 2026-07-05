package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Warmane WotLK realms share one game client; player GUID prefixes identify the realm.
var playerGuidPrefixToWarmaneRealm = map[string]string{
	"06": "Warmane_Lordaeron",
	"07": "Warmane_Icecrown",
	"0E": "Warmane_Onyxia",
}

var warmaneTrioServers = map[string]struct{}{
	"Warmane_Lordaeron": {},
	"Warmane_Icecrown":  {},
	"Warmane_Onyxia":    {},
}

type warmaneRealmScan struct {
	// Realms maps detected Warmane realm -> GUID prefix seen in the slice.
	Realms map[string]string
}

func isWarmaneTrioServer(serverName string) bool {
	_, ok := warmaneTrioServers[strings.TrimSpace(serverName)]
	return ok
}

func warmaneRealmLabel(serverName string) string {
	switch serverName {
	case "Warmane_Lordaeron":
		return "Warmane - Lordaeron"
	case "Warmane_Icecrown":
		return "Warmane - Icecrown"
	case "Warmane_Onyxia":
		return "Warmane - Onyxia"
	default:
		return serverName
	}
}

// scanWarmaneRealmsFromCombatLog reads player GUID prefixes from a combat log file.
// Used on the staged delta slice only (new lines since last tail), not the full WoWCombatLog.txt history.
func scanWarmaneRealmsFromCombatLog(path string) (warmaneRealmScan, error) {
	result := warmaneRealmScan{Realms: make(map[string]string)}
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		prefix := extractPlayerGuidPrefix(scanner.Text())
		if prefix == "" {
			continue
		}
		if realm, ok := playerGuidPrefixToWarmaneRealm[prefix]; ok {
			if _, seen := result.Realms[realm]; !seen {
				result.Realms[realm] = prefix
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func extractPlayerGuidPrefix(line string) string {
	prefix, _ := extractPlayerInfoFromLine(line)
	return prefix
}

func extractPlayerInfoFromLine(line string) (string, []string) {
	if !strings.Contains(line, "0x0") && !strings.Contains(line, "Player-") {
		return "", nil
	}
	parts := strings.SplitN(line, "  ", 2)
	if len(parts) < 2 {
		return "", nil
	}
	eventParts := strings.Split(parts[1], ",")

	var prefix string
	keys := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)

	tryAdd := func(raw string) {
		guid := strings.Trim(strings.TrimSpace(raw), "\"")
		if !isPlayerGUID(guid) {
			return
		}
		canon := canonicalPlayerGUIDKey(guid)
		if canon == "" {
			return
		}
		if _, ok := seen[canon]; ok {
			return
		}
		seen[canon] = struct{}{}
		keys = append(keys, canon)
		if prefix == "" {
			if p := guidPrefixIfPlayer(guid); p != "" {
				prefix = p
			}
		}
	}

	if len(eventParts) > 1 {
		tryAdd(eventParts[1])
	}
	if len(eventParts) > 4 {
		tryAdd(eventParts[4])
	}
	return prefix, keys
}

func isPlayerGUID(guid string) bool {
	if guid == "" {
		return false
	}
	if guid == "0x0000000000000000" || guid == "0000000000000000" {
		return false
	}
	if strings.HasPrefix(guid, "Player-") {
		return true
	}
	stripped := strings.ToUpper(strings.Replace(guid, "0x", "", 1))
	if len(stripped) < 2 {
		return false
	}
	value, err := strconv.ParseInt(stripped[:2], 16, 64)
	if err != nil {
		return false
	}
	return value >= 0x00 && value <= 0x0F
}

func canonicalPlayerGUIDKey(guid string) string {
	s := strings.TrimSpace(guid)
	s = strings.ToUpper(s)
	s = strings.TrimPrefix(s, "0X")
	return s
}

func guidPrefixIfPlayer(rawGUID string) string {
	guid := strings.Trim(strings.TrimSpace(rawGUID), "\"")
	if !isPlayerGUID(guid) {
		return ""
	}
	canon := canonicalPlayerGUIDKey(guid)
	if len(canon) < 2 {
		return ""
	}
	return strings.ToUpper(canon[:2])
}

type warmaneDriftCheck struct {
	DefaultServer    string
	DetectedServers  []string
	GuidPrefixByRealm map[string]string
	MultipleDetected bool
}

// evaluateWarmaneServerDrift compares configured default against GUIDs in the staged slice.
// Returns nil when no user confirmation is required.
func evaluateWarmaneServerDrift(configuredServer string, scan warmaneRealmScan) *warmaneDriftCheck {
	configuredServer = strings.TrimSpace(configuredServer)
	if !isWarmaneTrioServer(configuredServer) {
		return nil
	}
	if len(scan.Realms) == 0 {
		return nil
	}

	detected := make([]string, 0, len(scan.Realms))
	for realm := range scan.Realms {
		detected = append(detected, realm)
	}
	sortStrings(detected)

	if len(detected) == 1 && detected[0] == configuredServer {
		return nil
	}

	return &warmaneDriftCheck{
		DefaultServer:     configuredServer,
		DetectedServers:   detected,
		GuidPrefixByRealm: scan.Realms,
		MultipleDetected:  len(detected) > 1,
	}
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func formatWarmaneDriftLog(check *warmaneDriftCheck) string {
	if check == nil {
		return ""
	}
	parts := make([]string, 0, len(check.DetectedServers))
	for _, realm := range check.DetectedServers {
		prefix := check.GuidPrefixByRealm[realm]
		parts = append(parts, fmt.Sprintf("%s (GUID %s)", warmaneRealmLabel(realm), prefix))
	}
	return fmt.Sprintf(
		"default=%s detected_in_new_lines=[%s]",
		warmaneRealmLabel(check.DefaultServer),
		strings.Join(parts, ", "),
	)
}

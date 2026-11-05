package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// Same limit as the server's detect.RaidAwareBigGapMs and the speed-run rule.
	mergeMultiDaySpanMs = 14 * 60 * 60 * 1000
	mergeLockoutSpanMs  = 7 * 24 * 60 * 60 * 1000
	mergeRealmScanLines = 20000
	combatLogHeaderTag  = "COMBAT_LOG_VERSION"
)

// MergeFile is one scanned log file.
type MergeFile struct {
	Path      string   `json:"path"`
	Name      string   `json:"name"`
	SizeBytes int64    `json:"sizeBytes"`
	FirstMs   int64    `json:"firstMs"`
	LastMs    int64    `json:"lastMs"`
	Bosses    []string `json:"bosses"`
	Realms    []string `json:"realms"`
	Header    string   `json:"header,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// MergeLink describes the time between two neighbouring files. Negative GapMs is an overlap.
type MergeLink struct {
	GapMs   int64 `json:"gapMs"`
	Overlap bool  `json:"overlap"`
	NextDay bool  `json:"nextDay"`
}

// MergePlan is what the Merge screen shows before anything is written.
type MergePlan struct {
	Files      []MergeFile `json:"files"`
	Links      []MergeLink `json:"links"`
	Bosses     []string    `json:"bosses"`
	TotalBytes int64       `json:"totalBytes"`
	ActiveMs   int64       `json:"activeMs"`
	SpanMs     int64       `json:"spanMs"`
	MultiDay   bool        `json:"multiDay"`
	Warnings   []string    `json:"warnings"`
	CanMerge   bool        `json:"canMerge"`
}

// lineStamp returns the "M/D HH:MM:SS.mmm" prefix of a combat log line.
func lineStamp(line string) (string, bool) {
	stamp, _, found := strings.Cut(line, "  ")
	if !found || len(stamp) < 14 || len(stamp) > 18 || !strings.Contains(stamp, "/") {
		return "", false
	}
	return stamp, true
}

// stampMs converts a line stamp to Unix ms. Combat logs carry no year, so the caller supplies it.
func stampMs(stamp string, year int) (int64, bool) {
	datePart, timePart, found := strings.Cut(stamp, " ")
	if !found {
		return 0, false
	}
	monthText, dayText, found := strings.Cut(datePart, "/")
	if !found {
		return 0, false
	}
	month, err1 := strconv.Atoi(monthText)
	day, err2 := strconv.Atoi(dayText)
	clock, err3 := time.Parse("15:04:05.000", timePart)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	moment := time.Date(year, time.Month(month), day, clock.Hour(), clock.Minute(), clock.Second(), clock.Nanosecond(), time.UTC)
	return moment.UnixMilli(), true
}

// inferLogYear picks the year for a stamp from the file's modified time: a log cannot end after it
// was last written, so a December stamp in a file saved in January belongs to the year before.
func inferLogYear(stamp string, modified time.Time) int {
	year := modified.Year()
	if ms, ok := stampMs(stamp, year); ok && ms > modified.Add(48*time.Hour).UnixMilli() {
		return year - 1
	}
	return year
}

// lineNames returns the source and target names of an event line: its first two quoted fields.
// Found by quotes because the columns between them differ by expansion (Cata adds raid flags).
func lineNames(line string) (string, string) {
	_, body, found := strings.Cut(line, "  ")
	if !found {
		return "", ""
	}
	var names [2]string
	count := 0
	for _, field := range strings.SplitN(body, ",", 9) {
		if strings.HasPrefix(field, `"`) {
			names[count] = strings.Trim(field, `"`)
			count++
			if count == 2 {
				break
			}
		}
	}
	return names[0], names[1]
}

func isCreatureLine(line string) bool {
	return strings.Contains(line, "0xF13") || strings.Contains(line, "0xF15") || strings.Contains(line, "Creature-")
}

// scanMergeFile reads one log for its time range, bosses, realm and header.
func scanMergeFile(path string) MergeFile {
	result := MergeFile{Path: path, Name: filepath.Base(path), Bosses: []string{}, Realms: []string{}}
	file, err := os.Open(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.SizeBytes = info.Size()

	var firstStamp, lastStamp string
	bosses := map[string]bool{}
	realms := map[string]bool{}
	lineCount := 0

	reader := bufio.NewReaderSize(file, 1024*1024)
	for {
		raw, readErr := reader.ReadString('\n')
		line := strings.TrimRight(raw, "\r\n")
		if line != "" {
			lineCount++
			if stamp, ok := lineStamp(line); ok && !strings.Contains(line, combatLogHeaderTag) {
				if firstStamp == "" {
					firstStamp = stamp
				}
				lastStamp = stamp
			}
			if result.Header == "" && strings.Contains(line, combatLogHeaderTag) {
				result.Header = line[strings.Index(line, combatLogHeaderTag):]
			}
			if isCreatureLine(line) {
				source, target := lineNames(line)
				if boss, ok := mergeBossNames[target]; ok {
					bosses[boss] = true
				} else if boss, ok := mergeBossNames[source]; ok {
					bosses[boss] = true
				}
			}
			if lineCount <= mergeRealmScanLines {
				if realm, ok := playerGuidPrefixToWarmaneRealm[extractPlayerGuidPrefix(line)]; ok {
					realms[realm] = true
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			result.Error = readErr.Error()
			return result
		}
	}

	if firstStamp == "" {
		result.Error = "No combat log lines found in this file."
		return result
	}
	year := inferLogYear(firstStamp, info.ModTime())
	result.FirstMs, _ = stampMs(firstStamp, year)
	result.LastMs, _ = stampMs(lastStamp, year)
	if result.LastMs < result.FirstMs {
		result.LastMs, _ = stampMs(lastStamp, year+1)
	}
	for boss := range bosses {
		result.Bosses = append(result.Bosses, boss)
	}
	for realm := range realms {
		result.Realms = append(result.Realms, realm)
	}
	sort.Strings(result.Bosses)
	sort.Strings(result.Realms)
	return result
}

// planMerge orders files by time and reports gaps, overlaps and anything that needs a warning.
func planMerge(files []MergeFile) MergePlan {
	plan := MergePlan{Files: append([]MergeFile{}, files...), Links: []MergeLink{}, Bosses: []string{}, Warnings: []string{}}
	sort.SliceStable(plan.Files, func(i, j int) bool { return plan.Files[i].FirstMs < plan.Files[j].FirstMs })

	seenBoss := map[string]bool{}
	realms := map[string]bool{}
	headers := map[string]bool{}
	readable := 0
	for i, file := range plan.Files {
		plan.TotalBytes += file.SizeBytes
		if file.Error != "" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: %s", file.Name, file.Error))
			continue
		}
		readable++
		plan.ActiveMs += file.LastMs - file.FirstMs
		for _, boss := range file.Bosses {
			if !seenBoss[boss] {
				seenBoss[boss] = true
				plan.Bosses = append(plan.Bosses, boss)
			}
		}
		for _, realm := range file.Realms {
			realms[realm] = true
		}
		if file.Header != "" {
			headers[file.Header] = true
		}
		if i > 0 && plan.Files[i-1].Error == "" {
			gap := file.FirstMs - plan.Files[i-1].LastMs
			link := MergeLink{GapMs: gap, Overlap: gap < 0}
			link.NextDay = gap > mergeMultiDaySpanMs
			plan.Links = append(plan.Links, link)
		}
	}

	if readable > 0 {
		var first, last int64
		for _, file := range plan.Files {
			if file.Error != "" {
				continue
			}
			if first == 0 || file.FirstMs < first {
				first = file.FirstMs
			}
			if file.LastMs > last {
				last = file.LastMs
			}
		}
		plan.SpanMs = last - first
		plan.MultiDay = plan.SpanMs > mergeMultiDaySpanMs
	}
	if len(realms) > 1 {
		names := make([]string, 0, len(realms))
		for realm := range realms {
			names = append(names, realm)
		}
		sort.Strings(names)
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("These files look like different realms: %s.", strings.Join(names, ", ")))
	}
	if len(headers) > 1 {
		plan.Warnings = append(plan.Warnings, "These files have different combat log versions.")
	}
	if plan.SpanMs > mergeLockoutSpanMs {
		plan.Warnings = append(plan.Warnings, "These files are more than a week apart, so they cannot be the same raid lockout.")
	}
	plan.CanMerge = readable == len(plan.Files) && readable >= 2 && plan.TotalBytes <= maxUploadSourceBytes
	if plan.TotalBytes > maxUploadSourceBytes {
		plan.Warnings = append(plan.Warnings, "Together these files are over the 1024 MB upload limit.")
	}
	return plan
}

// writeMerged copies files in order. A later file's lines inside the earlier file's time range are
// dropped when identical, and only the first combat log header is kept. Lines that exist only in
// the later file stay, so nothing recorded is lost.
func writeMerged(files []MergeFile, out io.Writer) (int, error) {
	writer := bufio.NewWriterSize(out, 1024*1024)
	duplicates := 0
	headerWritten := false
	var overlapLines map[string]bool
	var overlapEndMs int64

	for index, entry := range files {
		file, err := os.Open(entry.Path)
		if err != nil {
			return duplicates, err
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return duplicates, err
		}

		var nextFirstMs int64
		if index+1 < len(files) && files[index+1].FirstMs <= entry.LastMs {
			nextFirstMs = files[index+1].FirstMs
		}
		nextOverlap := map[string]bool{}
		year := time.UnixMilli(entry.FirstMs).UTC().Year()
		if entry.FirstMs == 0 {
			year = info.ModTime().Year()
		}

		reader := bufio.NewReaderSize(file, 1024*1024)
		for {
			raw, readErr := reader.ReadString('\n')
			line := strings.TrimRight(raw, "\r\n")
			if line != "" {
				keep := true
				if strings.Contains(line, combatLogHeaderTag) {
					keep = !headerWritten
					headerWritten = true
				} else if overlapLines != nil || nextFirstMs > 0 {
					if stamp, ok := lineStamp(line); ok {
						if ms, ok := stampMs(stamp, year); ok {
							if overlapLines != nil && ms <= overlapEndMs && overlapLines[line] {
								keep = false
								duplicates++
							}
							if nextFirstMs > 0 && ms >= nextFirstMs {
								nextOverlap[line] = true
							}
						}
					}
				}
				if keep {
					if _, err := writer.WriteString(line + "\n"); err != nil {
						file.Close()
						return duplicates, err
					}
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				file.Close()
				return duplicates, readErr
			}
		}
		file.Close()

		overlapLines = nil
		if nextFirstMs > 0 {
			overlapLines = nextOverlap
			overlapEndMs = entry.LastMs
		}
	}
	return duplicates, writer.Flush()
}

// SelectMergeFiles lets the user pick log files.
func (a *App) SelectMergeFiles() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select combat log files",
		Filters: []runtime.FileFilter{{DisplayName: "Combat logs (*.txt)", Pattern: "*.txt"}},
	})
}

// ScanMergeFiles reads the files and returns the merge plan. Nothing is written.
func (a *App) ScanMergeFiles(paths []string) *MergePlan {
	files := make([]MergeFile, len(paths))
	done := make(chan int, len(paths))
	for i, path := range paths {
		go func(i int, path string) {
			files[i] = scanMergeFile(path)
			done <- i
		}(i, path)
	}
	for range paths {
		<-done
	}
	plan := planMerge(files)
	return &plan
}

func mergeToFile(paths []string, destination string) (*MergePlan, int, error) {
	files := make([]MergeFile, 0, len(paths))
	for _, path := range paths {
		files = append(files, scanMergeFile(path))
	}
	plan := planMerge(files)
	if !plan.CanMerge {
		return &plan, 0, fmt.Errorf("these files cannot be merged: %s", strings.Join(plan.Warnings, " "))
	}
	out, err := os.Create(destination)
	if err != nil {
		return &plan, 0, err
	}
	duplicates, err := writeMerged(plan.Files, out)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return &plan, duplicates, err
}

// SaveMergedFile writes the merged log where the user chooses and returns the path.
func (a *App) SaveMergedFile(paths []string) (string, error) {
	destination, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save merged combat log",
		DefaultFilename: "WoWCombatLog-merged.txt",
		Filters:         []runtime.FileFilter{{DisplayName: "Combat logs (*.txt)", Pattern: "*.txt"}},
	})
	if err != nil || destination == "" {
		return "", err
	}
	if _, _, err := mergeToFile(paths, destination); err != nil {
		os.Remove(destination)
		return "", err
	}
	return destination, nil
}

// MergeAndUpload merges the files into a temporary log and uploads it as one raid.
func (a *App) MergeAndUpload(paths []string, serverName string, unlisted bool) (*PreprocessResponse, error) {
	tmp, err := os.CreateTemp("", "wow-logs-merged-*.txt")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if _, _, err := mergeToFile(paths, tmp.Name()); err != nil {
		return nil, err
	}
	merged, err := os.Open(tmp.Name())
	if err != nil {
		return nil, err
	}
	defer merged.Close()
	info, err := merged.Stat()
	if err != nil {
		return nil, err
	}
	return a.uploadCombatLog(merged, info.Size(), preprocessUploadOptions{
		Endpoint:   "/api/v5/uploader/preprocess",
		ServerName: serverName,
		Unlisted:   unlisted,
		Merged:     true,
		Source:     historySourceMerge,
	})
}

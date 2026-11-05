package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectCombatLogFile lets the user upload a file other than WoWCombatLog.txt.
func (a *App) SelectCombatLogFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select a combat log",
		Filters: []runtime.FileFilter{{DisplayName: "Combat logs (*.txt)", Pattern: "*.txt"}},
	})
}

// GetSiteURL is the website base for links such as Manage devices.
func (a *App) GetSiteURL() string {
	return a.viewLogURL
}

// GetRecentLogLines returns the newest lines of the combat log for the live view.
func (a *App) GetRecentLogLines(logDirectory string, count int) []string {
	if strings.TrimSpace(logDirectory) == "" || count <= 0 {
		return []string{}
	}
	line, err := readLastLines(combatLogPath(logDirectory), count)
	if err != nil {
		return []string{}
	}
	return line
}

// readLastLines reads backwards from the end, so a large log costs one small read.
func readLastLines(path string, count int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	const chunk = 64 * 1024
	size := info.Size()
	start := size - chunk
	if start < 0 {
		start = 0
	}
	buf := make([]byte, size-start)
	if _, err := file.ReadAt(buf, start); err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(buf), "\n") {
		if trimmed := strings.TrimRight(line, "\r"); strings.TrimSpace(trimmed) != "" {
			lines = append(lines, trimmed)
		}
	}
	if start > 0 && len(lines) > 0 {
		lines = lines[1:] // the first line of the chunk may be cut off
	}
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return lines, nil
}

// OpenDiagnosticLog shows the app log in Notepad.
func (a *App) OpenDiagnosticLog() error {
	return exec.Command("notepad.exe", a.GetDiagnosticLogPath()).Start()
}

//go:build windows

package main

import "syscall"

const (
	smXVIRTUALSCREEN = 76
	smYVIRTUALSCREEN = 77
	smCXVIRTUALSCREEN = 78
	smCYVIRTUALSCREEN = 79
)

var (
	modUser32         = syscall.NewLazyDLL("user32.dll")
	procGetSystemMetrics = modUser32.NewProc("GetSystemMetrics")
)

func getSystemMetrics(index int) int {
	r, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(r)
}

func virtualScreenRect() (x, y, w, h int) {
	return getSystemMetrics(smXVIRTUALSCREEN),
		getSystemMetrics(smYVIRTUALSCREEN),
		getSystemMetrics(smCXVIRTUALSCREEN),
		getSystemMetrics(smCYVIRTUALSCREEN)
}

// isWindowPlacementVisible reports whether at least a small part of the window rect
// intersects the Windows virtual screen (all monitors). Used when a monitor was unplugged
// or the window was saved on a display that is no longer connected.
func isWindowPlacementVisible(winX, winY, winW, winH int) bool {
	if winW <= 0 || winH <= 0 {
		return false
	}
	vx, vy, vw, vh := virtualScreenRect()
	if vw <= 0 || vh <= 0 {
		return true
	}

	const minVisible = 120

	left := max(winX, vx)
	top := max(winY, vy)
	right := min(winX+winW, vx+vw)
	bottom := min(winY+winH, vy+vh)

	return right-left >= minVisible && bottom-top >= minVisible
}

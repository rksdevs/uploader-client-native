//go:build !windows

package main

func isWindowPlacementVisible(_ int, _ int, _ int, _ int) bool {
	return true
}

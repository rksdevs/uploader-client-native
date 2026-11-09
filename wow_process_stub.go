//go:build !windows

package main

func isWowProcessRunning() bool {
	return false
}

func isAsGameRunning() bool {
	return false
}

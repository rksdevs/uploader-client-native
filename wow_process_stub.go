//go:build !windows

package main

func isWowProcessRunning() bool {
	return false
}

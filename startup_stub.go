//go:build !windows

package main

func isStartWithWindowsEnabled() bool { return false }

func setStartWithWindows(enabled bool) error { return nil }

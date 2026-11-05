//go:build !windows

package main

func showDesktopNotification(title, message string) error { return nil }

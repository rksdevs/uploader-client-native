//go:build windows

package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	startupRunKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupRunValue = "WoWLogsUploader"
)

func isStartWithWindowsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(startupRunValue)
	return err == nil
}

func setStartWithWindows(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(startupRunValue); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	log.Printf("[Startup] Registering %s\n", exe)
	return key.SetStringValue(startupRunValue, fmt.Sprintf(`"%s"`, exe))
}

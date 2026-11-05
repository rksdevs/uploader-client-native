//go:build windows

package main

import (
	"os/exec"
	"strings"
	"syscall"
)

// showDesktopNotification raises a Windows toast through PowerShell, which needs no extra
// dependency or registered app id. Values travel in environment variables, never in the script.
func showDesktopNotification(title, message string) error {
	script := strings.Join([]string{
		`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null`,
		`$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)`,
		`$text = $xml.GetElementsByTagName('text')`,
		`$text.Item(0).AppendChild($xml.CreateTextNode($env:WOWLOGS_TOAST_TITLE)) | Out-Null`,
		`$text.Item(1).AppendChild($xml.CreateTextNode($env:WOWLOGS_TOAST_BODY)) | Out-Null`,
		`$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)`,
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('WoW Logs Uploader').Show($toast)`,
	}, "; ")
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(cmd.Environ(), "WOWLOGS_TOAST_TITLE="+title, "WOWLOGS_TOAST_BODY="+message)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

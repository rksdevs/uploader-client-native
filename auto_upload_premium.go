package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const autoUploadPremiumRecheck = 15 * time.Minute

type autoUploadVerifyResponse struct {
	Premium           bool   `json:"premium"`
	TokenType         string `json:"tokenType"`
	DeviceRegistered  bool   `json:"deviceRegistered"`
	DeviceCount       int    `json:"deviceCount"`
	DeviceLimit       int    `json:"deviceLimit"`
	CanRegisterDevice bool   `json:"canRegisterDevice"`
	HourlyLimit       int    `json:"hourlyLimit"`
	Code              string `json:"code"`
	Message           string `json:"message"`
	Reason            string `json:"reason"`
}

func (a *App) verifyAutoUploadAccessRemote() (ok bool, message string, err error) {
	token := strings.TrimSpace(a.config.ApiToken)
	if token == "" {
		return false, "Premium API token is missing.", nil
	}

	apiURL := fmt.Sprintf("%s/api/v5/uploader/auto-upload-verify", a.apiBaseURL)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("X-API-Token", token)
	req.Header.Set("X-Device-Id", a.ensureDeviceID())

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	var parsed autoUploadVerifyResponse
	_ = json.Unmarshal(body, &parsed)

	if resp.StatusCode == http.StatusOK && parsed.Premium {
		return true, "", nil
	}

	message = strings.TrimSpace(parsed.Message)
	if message == "" {
		message = strings.TrimSpace(resp.Status)
	}
	if parsed.Reason != "" && message != "" {
		message = fmt.Sprintf("%s (%s)", message, parsed.Reason)
	}
	return false, message, nil
}

func (a *App) disableAutoUploadForPremiumLoss(detail string) {
	if detail == "" {
		detail = "Premium subscription is no longer active."
	}
	log.Printf("[AutoUpload] Disabling auto-upload: %s\n", detail)
	a.config.AutoUploadEnabled = false
	_ = a.saveConfig()
	a.stopAutoUploadWatcher()
	a.setWatcherStatus("error", detail)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auto_upload_premium_lost", map[string]interface{}{
			"message": detail,
		})
	}
}

func (a *App) markPremiumChecked() {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	a.autoUploadWatcher.lastPremiumCheckAt = time.Now()
	a.autoUploadWatcher.mu.Unlock()
}

func (a *App) shouldRecheckAutoUploadPremium() bool {
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	defer a.autoUploadWatcher.mu.Unlock()
	if a.autoUploadWatcher.lastPremiumCheckAt.IsZero() {
		return true
	}
	return time.Since(a.autoUploadWatcher.lastPremiumCheckAt) >= autoUploadPremiumRecheck
}

// ensureAutoUploadPremiumActive verifies remote premium access.
// Network failures fail open (returns true) so brief outages do not disable auto-upload.
func (a *App) ensureAutoUploadPremiumActive() bool {
	if strings.TrimSpace(a.config.ApiToken) == "" {
		a.disableAutoUploadForPremiumLoss("Premium API token is missing.")
		return false
	}

	ok, message, err := a.verifyAutoUploadAccessRemote()
	if err != nil {
		log.Printf("[AutoUpload] Premium verify network error (ignored): %v\n", err)
		return true
	}
	if !ok {
		a.disableAutoUploadForPremiumLoss(message)
		return false
	}

	a.markPremiumChecked()
	return true
}

// CheckAutoUploadPremiumNow is exposed for manual/debug refresh from the UI layer.
func (a *App) CheckAutoUploadPremiumNow() (bool, string) {
	if a.ensureAutoUploadPremiumActive() {
		return true, "Premium auto-upload access is active."
	}
	a.initAutoUploadWatcherFields()
	a.autoUploadWatcher.mu.Lock()
	detail := a.autoUploadWatcher.detail
	a.autoUploadWatcher.mu.Unlock()
	if detail == "" {
		detail = "Premium auto-upload access is not available."
	}
	return false, detail
}

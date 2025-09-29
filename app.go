// package main

// import (
// 	"archive/zip"
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"mime/multipart"
// 	"net/http"
// 	"os"
// 	"path/filepath"
// 	"sync"
// 	"time"

// 	"github.com/wailsapp/wails/v2/pkg/runtime"
// )

// type Config struct {
// 	LogDirectory string `json:"logDirectory"`
// }
// type App struct {
// 	ctx          context.Context
// 	apiBaseURL   string
// 	viewLogURL   string
// 	pendingPolls map[int]bool
// 	pollLock     sync.Mutex
// 	config       Config
// 	configPath   string
// }

// func NewApp() *App {
// 	return &App{
// 		pendingPolls: make(map[int]bool),
// 	}
// }

// func (a *App) startup(ctx context.Context) {
// 	a.ctx = ctx
// 	log.Println("[Go Backend] Startup complete, context saved.")

// 	env := runtime.Environment(a.ctx)
// 	if env.BuildType == "dev" {
// 		log.Println("[Go Backend] Development environment detected.")
// 		a.apiBaseURL = "http://localhost:8000"
// 		a.viewLogURL = "http://localhost:3000"
// 	} else {
// 		log.Println("[Go Backend] Production environment detected.")
// 		a.apiBaseURL = "https://wow-logs.co.in"
// 		a.viewLogURL = "https://wow-logs.co.in"
// 	}

// 	if err := a.loadConfig(); err != nil {
// 		log.Printf("[Go Backend] Could not load config file (this is normal on first run): %v\n", err)
// 	}
// }

// func (a *App) loadConfig() error {
// 	configDir, err := os.UserConfigDir()
// 	if err != nil {
// 		return fmt.Errorf("could not get user config directory: %w", err)
// 	}
// 	appConfigDir := filepath.Join(configDir, "WoWLogsUploader")
// 	if err := os.MkdirAll(appConfigDir, os.ModePerm); err != nil {
// 		return fmt.Errorf("could not create app config directory: %w", err)
// 	}
// 	a.configPath = filepath.Join(appConfigDir, "config.json")
// 	log.Printf("[Go Backend] Config file path set to: %s\n", a.configPath)

// 	data, err := os.ReadFile(a.configPath)
// 	if err != nil {
// 		return fmt.Errorf("could not read config file: %w", err)
// 	}

// 	if err := json.Unmarshal(data, &a.config); err != nil {
// 		return fmt.Errorf("could not parse config file: %w", err)
// 	}

// 	log.Printf("[Go Backend] Successfully loaded config. Log Directory: %s\n", a.config.LogDirectory)
// 	return nil
// }

// func (a *App) saveConfig() error {
// 	data, err := json.MarshalIndent(a.config, "", "  ")
// 	if err != nil {
// 		return fmt.Errorf("could not marshal config to JSON: %w", err)
// 	}
// 	if err := os.WriteFile(a.configPath, data, 0644); err != nil {
// 		return fmt.Errorf("could not write config file: %w", err)
// 	}
// 	log.Printf("[Go Backend] Successfully saved config to: %s\n", a.configPath)
// 	return nil
// }

// func (a *App) GetSavedDirectory() string {
// 	log.Printf("[Go Backend] Frontend requested saved directory. Returning: %s\n", a.config.LogDirectory)
// 	return a.config.LogDirectory
// }

// func (a *App) SelectDirectory() (string, error) {
// 	log.Println("[Go Backend] Received request to select directory.")
// 	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
// 		Title: "Select your WoW Logs folder",
// 	})
// 	if err != nil {
// 		return "", err
// 	}
// 	if directory != "" {
// 		log.Printf("[Go Backend] User selected new directory: %s\n", directory)
// 		a.config.LogDirectory = directory
// 		if err := a.saveConfig(); err != nil {
// 			log.Printf("[Go Backend] ERROR: Failed to save config: %v\n", err)
// 		}
// 	}
// 	return directory, nil
// }

// func (a *App) PreprocessLog(logDirectory string, serverName string) (*PreprocessResponse, error) {
// 	log.Printf("[Go Backend] PREPROCESS: Starting for directory '%s', Server: '%s'\n", logDirectory, serverName)
// 	logPath := filepath.Join(logDirectory, "WoWCombatLog.txt")

// 	logData, err := os.ReadFile(logPath)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not read WoWCombatLog.txt: %w", err)
// 	}

// 	buf := new(bytes.Buffer)
// 	zipWriter := zip.NewWriter(buf)
// 	fileWriter, err := zipWriter.Create("WoWCombatLog.txt")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create zip entry: %w", err)
// 	}
// 	_, err = fileWriter.Write(logData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to write data to zip entry: %w", err)
// 	}
// 	zipWriter.Close()

// 	requestBody := &bytes.Buffer{}
// 	writer := multipart.NewWriter(requestBody)
// 	part, err := writer.CreateFormFile("logFile", "WoWCombatLog.zip")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create form file: %w", err)
// 	}
// 	_, err = io.Copy(part, buf)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to copy zip data to form: %w", err)
// 	}
// 	_ = writer.WriteField("serverName", serverName)
// 	writer.Close()

// 	apiURL := fmt.Sprintf("%s/api/v5/uploader/preprocess", a.apiBaseURL)
// 	req, err := http.NewRequest("POST", apiURL, requestBody)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create preprocess request: %w", err)
// 	}
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("X-Socket-ID", "wails-native-client-polling")

// 	client := &http.Client{Timeout: 60 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to send preprocess request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	respBody, _ := io.ReadAll(resp.Body)
// 	log.Printf("[Go Backend] PREPROCESS Response: Status: %s, Body: %s\n", resp.Status, string(respBody))

// 	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
// 		return nil, fmt.Errorf("server returned error: %s", resp.Status)
// 	}

// 	var preprocessResponse PreprocessResponse
// 	if err := json.Unmarshal(respBody, &preprocessResponse); err != nil {
// 		return nil, fmt.Errorf("failed to decode preprocess response: %w", err)
// 	}

// 	preprocessResponse.ViewLogURL = a.viewLogURL
// 	return &preprocessResponse, nil
// }

// func (a *App) EnqueueJobs(preprocessId int, selectedInstances []Instance) (string, error) {
// 	log.Printf("[Go Backend] ENQUEUE: Queuing %d jobs for PreprocessID: %d\n", len(selectedInstances), preprocessId)

// 	requestData, err := json.Marshal(map[string]interface{}{
// 		"preprocessId":      preprocessId,
// 		"selectedInstances": selectedInstances,
// 		"socketId":          "wails-native-client-polling",
// 	})
// 	if err != nil {
// 		return "", fmt.Errorf("failed to marshal enqueue request: %w", err)
// 	}

// 	apiURL := fmt.Sprintf("%s/api/v5/uploader/enqueue-jobs", a.apiBaseURL)
// 	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create enqueue request: %w", err)
// 	}
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{Timeout: 30 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to send enqueue request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	respBody, _ := io.ReadAll(resp.Body)
// 	log.Printf("[Go Backend] ENQUEUE Response: Status: %s, Body: %s\n", resp.Status, string(respBody))

// 	if resp.StatusCode != http.StatusAccepted {
// 		return "", fmt.Errorf("server returned non-202 status: %s", resp.Status)
// 	}

// 	return "Jobs successfully queued! You will be notified upon completion.", nil
// }

// func (a *App) StartMonitoringJob(preprocessId int) {
// 	a.pollLock.Lock()
// 	if a.pendingPolls[preprocessId] {
// 		log.Printf("[Go Backend] POLLING: Monitoring for PreprocessID %d is already active.", preprocessId)
// 		a.pollLock.Unlock()
// 		return
// 	}
// 	a.pendingPolls[preprocessId] = true
// 	a.pollLock.Unlock()

// 	log.Printf("[Go Backend] POLLING: Started monitoring for PreprocessID: %d\n", preprocessId)

// 	go func() {
// 		defer func() {
// 			a.pollLock.Lock()
// 			delete(a.pendingPolls, preprocessId)
// 			a.pollLock.Unlock()
// 			log.Printf("[Go Backend] POLLING: Stopped monitoring for PreprocessID: %d\n", preprocessId)
// 		}()

// 		timeout := time.After(15 * time.Minute)
// 		ticker := time.NewTicker(10 * time.Second)
// 		defer ticker.Stop()

// 		notifiedLogs := make(map[int]bool)

// 		for {
// 			select {
// 			case <-timeout:
// 				log.Printf("[Go Backend] POLLING: Timed out for PreprocessID: %d\n", preprocessId)
// 				return
// 			case <-ticker.C:
// 				status, err := a.checkJobStatus(preprocessId)
// 				if err != nil {
// 					log.Printf("[Go Backend] POLLING: Error checking status for %d: %v\n", preprocessId, err)
// 					continue
// 				}

// 				allJobsConsideredDone := true
// 				if status.TotalJobs == 0 || status.JobsCompleted < status.TotalJobs {
// 					allJobsConsideredDone = false
// 				}

// 				for _, logStatus := range status.Logs {
// 					if !notifiedLogs[logStatus.ID] && (logStatus.Status == "uploaded" || logStatus.Status == "failed") {
// 						log.Printf("[Go Backend] POLLING: Detected completed log %d with status '%s'. Notifying frontend.\n", logStatus.ID, logStatus.Status)
// 						runtime.EventsEmit(a.ctx, "job_notification", map[string]interface{}{
// 							"logId":      logStatus.ID,
// 							"status":     logStatus.Status,
// 							"viewLogURL": status.ViewLogURL,
// 						})
// 						notifiedLogs[logStatus.ID] = true
// 					}
// 				}

// 				if allJobsConsideredDone && status.TotalJobs > 0 && len(notifiedLogs) == status.TotalJobs {
// 					log.Printf("[Go Backend] POLLING: All %d jobs for PreprocessID %d are complete. Stopping poll.\n", status.TotalJobs, preprocessId)
// 					return
// 				}
// 			}
// 		}
// 	}()
// }

// func (a *App) checkJobStatus(preprocessId int) (*JobStatusResponse, error) {
// 	log.Printf("[Go Backend] POLLING: Checking status for PreprocessID: %d\n", preprocessId)
// 	apiURL := fmt.Sprintf("%s/api/v5/uploader/status/%d", a.apiBaseURL, preprocessId)

// 	resp, err := http.Get(apiURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to poll for job status: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("polling returned non-200 status: %s", resp.Status)
// 	}

// 	var statusResponse JobStatusResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
// 		return nil, fmt.Errorf("failed to decode job status response: %w", err)
// 	}

// 	statusResponse.ViewLogURL = a.viewLogURL
// 	return &statusResponse, nil
// }

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	LogDirectory string `json:"logDirectory"`
}

type App struct {
	ctx          context.Context
	apiBaseURL   string
	viewLogURL   string
	pendingPolls map[int]bool
	pollLock     sync.Mutex
	config       Config
	configPath   string
}

func NewApp() *App {
	return &App{
		pendingPolls: make(map[int]bool),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("[Go Backend] Startup complete.")

	env := runtime.Environment(a.ctx)
	if env.BuildType == "dev" {
		log.Println("[Go Backend] Development environment detected.")
		a.apiBaseURL = "http://localhost:8000"
		a.viewLogURL = "http://localhost:3000"
	} else {
		log.Println("[Go Backend] Production environment detected.")
		a.apiBaseURL = "https://wow-logs.co.in"
		a.viewLogURL = "https://wow-logs.co.in"
	}

	if err := a.loadConfig(); err != nil {
		log.Printf("[Go Backend] Could not load config file (this is normal on first run): %v\n", err)
	}
}

func (a *App) loadConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("could not get user config directory: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "WoWLogsUploader")
	if err := os.MkdirAll(appConfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create app config directory: %w", err)
	}
	a.configPath = filepath.Join(appConfigDir, "config.json")

	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}

	if err := json.Unmarshal(data, &a.config); err != nil {
		return fmt.Errorf("could not parse config file: %w", err)
	}

	log.Printf("[Go Backend] Successfully loaded config. Log Directory: %s\n", a.config.LogDirectory)
	return nil
}

func (a *App) saveConfig() error {
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config to JSON: %w", err)
	}
	if err := os.WriteFile(a.configPath, data, 0644); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}
	log.Printf("[Go Backend] Successfully saved config to: %s\n", a.configPath)
	return nil
}

func (a *App) GetSavedDirectory() string {
	return a.config.LogDirectory
}

func (a *App) SelectDirectory() (string, error) {
	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select your WoW Logs folder",
	})
	if err != nil {
		return "", err
	}
	if directory != "" {
		log.Printf("[Go Backend] User selected new directory: %s\n", directory)
		a.config.LogDirectory = directory
		if err := a.saveConfig(); err != nil {
			log.Printf("[Go Backend] ERROR: Failed to save config: %v\n", err)
		}
	}
	return directory, nil
}

func (a *App) PreprocessLog(logDirectory string, serverName string) (*PreprocessResponse, error) {
	log.Printf("[Go Backend] PREPROCESS: Starting for directory '%s', Server: '%s'\n", logDirectory, serverName)
	logPath := filepath.Join(logDirectory, "WoWCombatLog.txt")

	logData, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("could not read WoWCombatLog.txt: %w", err)
	}

	// Create a zip archive in memory.
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	fileWriter, err := zipWriter.Create("WoWCombatLog.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create zip entry: %w", err)
	}
	if _, err = fileWriter.Write(logData); err != nil {
		return nil, fmt.Errorf("failed to write data to zip entry: %w", err)
	}
	zipWriter.Close()

	// Create a multipart form request.
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	part, err := writer.CreateFormFile("logFile", "WoWCombatLog.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(part, buf); err != nil {
		return nil, fmt.Errorf("failed to copy zip data to form: %w", err)
	}
	_ = writer.WriteField("serverName", serverName)
	writer.Close()

	// Make the HTTP POST request.
	apiURL := fmt.Sprintf("%s/api/v5/uploader/preprocess", a.apiBaseURL)
	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create preprocess request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Socket-ID", "wails-native-client-polling") // Use a static ID for polling

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send preprocess request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Go Backend] PREPROCESS Response: Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("server returned error: %s", resp.Status)
	}

	var preprocessResponse PreprocessResponse
	if err := json.Unmarshal(respBody, &preprocessResponse); err != nil {
		return nil, fmt.Errorf("failed to decode preprocess response: %w", err)
	}

	preprocessResponse.ViewLogURL = a.viewLogURL
	return &preprocessResponse, nil
}

func (a *App) EnqueueJobs(preprocessId int, selectedInstances []Instance) (string, error) {
	log.Printf("[Go Backend] ENQUEUE: Queuing %d jobs for PreprocessID: %d\n", len(selectedInstances), preprocessId)

	requestData, err := json.Marshal(map[string]interface{}{
		"preprocessId":      preprocessId,
		"selectedInstances": selectedInstances,
		"socketId":          "wails-native-client-polling",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal enqueue request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v5/uploader/enqueue-jobs", a.apiBaseURL)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("failed to create enqueue request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send enqueue request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Go Backend] ENQUEUE Response: Status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("server returned non-202 status: %s", resp.Status)
	}

	return "Jobs successfully queued! You will be notified upon completion.", nil
}

func (a *App) StartMonitoringJob(preprocessId int) {
	a.pollLock.Lock()
	if a.pendingPolls[preprocessId] {
		log.Printf("[Go Backend] POLLING: Monitoring for PreprocessID %d is already active.", preprocessId)
		a.pollLock.Unlock()
		return
	}
	a.pendingPolls[preprocessId] = true
	a.pollLock.Unlock()

	log.Printf("[Go Backend] POLLING: Started monitoring for PreprocessID: %d\n", preprocessId)

	go func() {
		defer func() {
			a.pollLock.Lock()
			delete(a.pendingPolls, preprocessId)
			a.pollLock.Unlock()
			log.Printf("[Go Backend] POLLING: Stopped monitoring for PreprocessID: %d\n", preprocessId)
		}()

		timeout := time.After(15 * time.Minute)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		notifiedLogs := make(map[int]bool)

		for {
			select {
			case <-timeout:
				log.Printf("[Go Backend] POLLING: Timed out for PreprocessID: %d\n", preprocessId)
				return
			case <-ticker.C:
				status, err := a.checkJobStatus(preprocessId)
				if err != nil {
					log.Printf("[Go Backend] POLLING: Error checking status for %d: %v\n", preprocessId, err)
					continue
				}

				allJobsConsideredDone := status.TotalJobs > 0 && status.JobsCompleted >= status.TotalJobs

				for _, logStatus := range status.Logs {
					if !notifiedLogs[logStatus.ID] && (logStatus.Status == "uploaded" || logStatus.Status == "failed") {
						log.Printf("[Go Backend] POLLING: Detected completed log %d with status '%s'. Notifying frontend.\n", logStatus.ID, logStatus.Status)
						runtime.EventsEmit(a.ctx, "job_notification", map[string]interface{}{
							"logId":      logStatus.ID,
							"status":     logStatus.Status,
							"viewLogURL": status.ViewLogURL,
						})
						notifiedLogs[logStatus.ID] = true
					}
				}

				if allJobsConsideredDone && len(notifiedLogs) == status.TotalJobs {
					log.Printf("[Go Backend] POLLING: All %d jobs for PreprocessID %d are complete. Stopping poll.\n", status.TotalJobs, preprocessId)
					return
				}
			}
		}
	}()
}

func (a *App) checkJobStatus(preprocessId int) (*JobStatusResponse, error) {
	apiURL := fmt.Sprintf("%s/api/v5/uploader/status/%d", a.apiBaseURL, preprocessId)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to poll for job status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("polling returned non-200 status: %s", resp.Status)
	}

	var statusResponse JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return nil, fmt.Errorf("failed to decode job status response: %w", err)
	}

	statusResponse.ViewLogURL = a.viewLogURL
	return &statusResponse, nil
}

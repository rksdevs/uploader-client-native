package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	jobPollFast       = 10 * time.Second
	jobPollSlow       = 60 * time.Second
	jobPollFastWindow = 15 * time.Minute
	// The job worker is a single FIFO queue, so a busy evening can hold a log for hours.
	jobPollLimit = 6 * time.Hour
)

type jobOutcome int

const (
	jobOutcomeAllUploaded jobOutcome = iota
	jobOutcomePartial
	jobOutcomeAllFailed
	jobOutcomeTimedOut
)

// classifyJobOutcome: a poll that ran out of time says nothing about the job, so it must never be
// reported as a failure or retried (the job may still finish, and a retry would duplicate it).
func classifyJobOutcome(uploaded, failed int, timedOut bool) jobOutcome {
	switch {
	case timedOut:
		return jobOutcomeTimedOut
	case failed == 0:
		return jobOutcomeAllUploaded
	case uploaded == 0:
		return jobOutcomeAllFailed
	default:
		return jobOutcomePartial
	}
}

func jobPollInterval(elapsed time.Duration) time.Duration {
	if elapsed < jobPollFastWindow {
		return jobPollFast
	}
	return jobPollSlow
}

func (a *App) StartMonitoringJob(preprocessId int) {
	a.startJobMonitor(preprocessId, nil, "")
}

// StartMonitoringJobWithTailRefresh polls job status and advances the upload markers on success.
func (a *App) StartMonitoringJobWithTailRefresh(preprocessId int, logDirectory string) {
	logPath := ""
	if logDirectory != "" {
		logPath = combatLogPath(logDirectory)
	}
	a.startJobMonitor(preprocessId, nil, logPath)
}

func (a *App) startJobMonitor(preprocessId int, autoCtx *autoUploadJobContext, tailRefreshLogPath string) {
	a.pollLock.Lock()
	if a.pendingPolls[preprocessId] {
		log.Printf("[Go Backend] POLLING: Monitoring for PreprocessID %d is already active.", preprocessId)
		a.pollLock.Unlock()
		return
	}
	a.pendingPolls[preprocessId] = true
	a.pollLock.Unlock()

	if autoCtx != nil {
		a.rememberPendingAutoUpload(preprocessId, autoCtx)
	}
	log.Printf("[Go Backend] POLLING: Started monitoring for PreprocessID: %d\n", preprocessId)

	go func() {
		defer func() {
			a.pollLock.Lock()
			delete(a.pendingPolls, preprocessId)
			a.pollLock.Unlock()
			log.Printf("[Go Backend] POLLING: Stopped monitoring for PreprocessID: %d\n", preprocessId)
		}()

		started := time.Now()
		notifiedLogs := make(map[int]bool)
		uploaded, failed := 0, 0

		finish := func(timedOut bool) {
			outcome := classifyJobOutcome(uploaded, failed, timedOut)
			a.finishHistoryEntry(preprocessId, outcome)
			a.notifyUploadResult(preprocessId, outcome)
			if autoCtx != nil {
				a.handleAutoUploadJobsFinished(autoCtx, outcome)
			} else if uploaded > 0 && tailRefreshLogPath != "" {
				if err := a.markUploadedThrough(tailRefreshLogPath); err != nil {
					log.Printf("[Go Backend] Could not refresh upload markers: %v\n", err)
				}
			}
			if timedOut && a.ctx != nil {
				runtime.EventsEmit(a.ctx, "job_timed_out", map[string]interface{}{"preprocessId": preprocessId})
			}
		}

		for {
			elapsed := time.Since(started)
			if elapsed >= jobPollLimit {
				log.Printf("[Go Backend] POLLING: Gave up waiting for PreprocessID: %d\n", preprocessId)
				finish(true)
				return
			}
			time.Sleep(jobPollInterval(elapsed))

			status, err := a.checkJobStatus(preprocessId)
			if err != nil {
				log.Printf("[Go Backend] POLLING: Error checking status for %d: %v\n", preprocessId, err)
				continue
			}

			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "job_progress", map[string]interface{}{
					"preprocessId":  preprocessId,
					"queuePosition": status.QueuePosition,
					"jobsCompleted": status.JobsCompleted,
					"totalJobs":     status.TotalJobs,
				})
			}

			for _, logStatus := range status.Logs {
				if notifiedLogs[logStatus.ID] || (logStatus.Status != "uploaded" && logStatus.Status != "failed") {
					continue
				}
				notifiedLogs[logStatus.ID] = true
				if logStatus.Status == "uploaded" {
					uploaded++
				} else {
					failed++
				}
				a.addHistoryLog(preprocessId, logStatus.ID, logStatus.Status)
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, "job_notification", map[string]interface{}{
						"logId":      logStatus.ID,
						"status":     logStatus.Status,
						"viewLogURL": status.ViewLogURL,
					})
				}
			}

			if status.TotalJobs > 0 && status.JobsCompleted >= status.TotalJobs && len(notifiedLogs) == status.TotalJobs {
				finish(false)
				return
			}
		}
	}()
}

func (a *App) checkJobStatus(preprocessId int) (*JobStatusResponse, error) {
	apiURL := fmt.Sprintf("%s/api/v5/uploader/status/%d", a.apiBaseURL, preprocessId)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	a.setAPIHeaders(req)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
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

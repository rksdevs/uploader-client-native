package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	nativeSocketID       = "wails-native-client-polling"
	uploadRequestTimeout = 30 * time.Minute
	preprocessPollEvery  = 1500 * time.Millisecond
	preprocessPollLimit  = 15 * time.Minute
	progressMinStepBytes = 1024 * 1024
	maxUploadSourceBytes = 1024 * 1024 * 1024
	uploadStageCompress  = "compressing"
	uploadStageUpload    = "uploading"
	uploadStageDetect    = "detecting"
)

// UploadProgress is emitted to the UI as "upload_progress".
type UploadProgress struct {
	Stage      string `json:"stage"`
	Percent    int    `json:"percent"`
	BytesDone  int64  `json:"bytesDone"`
	BytesTotal int64  `json:"bytesTotal"`
}

func (a *App) emitUploadProgress(p UploadProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "upload_progress", p)
	}
}

// progressCounter reports bytes passing through a Reader, at most once per progressMinStepBytes.
type progressCounter struct {
	src      io.Reader
	done     int64
	reported int64
	total    int64
	report   func(done, total int64)
}

func (p *progressCounter) Read(b []byte) (int, error) {
	n, err := p.src.Read(b)
	p.done += int64(n)
	if p.report != nil && (p.done-p.reported >= progressMinStepBytes || err == io.EOF) {
		p.reported = p.done
		p.report(p.done, p.total)
	}
	return n, err
}

// compressToTempZip streams src into a temporary zip holding one combat log entry, so memory use
// does not grow with the log size. The caller removes the returned file.
func compressToTempZip(src io.Reader, size int64, report func(done, total int64)) (string, error) {
	tmp, err := os.CreateTemp("", "wow-logs-upload-*.zip")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file: %w", err)
	}
	fail := func(err error) (string, error) {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	zipWriter := zip.NewWriter(tmp)
	entry, err := zipWriter.Create(combatLogFileName)
	if err != nil {
		return fail(fmt.Errorf("could not create zip entry: %w", err))
	}
	counted := &progressCounter{src: src, total: size, report: report}
	written, err := io.Copy(entry, counted)
	if err != nil {
		return fail(fmt.Errorf("could not compress combat log: %w", err))
	}
	if written == 0 {
		return fail(fmt.Errorf("combat log is empty"))
	}
	if err := zipWriter.Close(); err != nil {
		return fail(fmt.Errorf("could not finish zip: %w", err))
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("could not finish zip: %w", err)
	}
	return tmp.Name(), nil
}

// postZipMultipart sends the zip as the logFile form field with a known Content-Length, reading it
// from disk as it goes. Form fields go first so the gateway has them when the file arrives.
func postZipMultipart(
	ctx context.Context,
	apiURL, zipPath string,
	fields map[string]string,
	prepare func(*http.Request),
	report func(done, total int64),
) (int, string, []byte, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return 0, "", nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, "", nil, err
	}

	var head bytes.Buffer
	writer := multipart.NewWriter(&head)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return 0, "", nil, err
		}
	}
	if _, err := writer.CreateFormFile("logFile", "WoWCombatLog.zip"); err != nil {
		return 0, "", nil, err
	}
	headLen := head.Len()
	if err := writer.Close(); err != nil {
		return 0, "", nil, err
	}
	prefix := head.Bytes()[:headLen]
	suffix := head.Bytes()[headLen:]

	total := int64(len(prefix)) + info.Size() + int64(len(suffix))
	body := &progressCounter{
		src:    io.MultiReader(bytes.NewReader(prefix), file, bytes.NewReader(suffix)),
		total:  total,
		report: report,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return 0, "", nil, err
	}
	req.ContentLength = total
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if prepare != nil {
		prepare(req)
	}

	client := &http.Client{Timeout: uploadRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Status, nil, err
	}
	return resp.StatusCode, resp.Status, respBody, nil
}

func percentOf(done, total int64) int {
	if total <= 0 {
		return 0
	}
	percent := int(done * 100 / total)
	if percent > 100 {
		return 100
	}
	return percent
}

type preprocessUploadOptions struct {
	Endpoint   string
	ServerName string
	Unlisted   bool
	Merged     bool
	Source     string
	Prepare    func(*http.Request)
}

// preprocessResponseBody also covers the 4.0 async reply (202 with streaming: true).
type preprocessResponseBody struct {
	PreprocessResponse
	Streaming bool `json:"streaming"`
}

// uploadCombatLog compresses src and sends it through preprocess. Older gateways reply with the
// instance list directly; current ones reply 202 and the instances arrive through polling.
func (a *App) uploadCombatLog(src io.Reader, size int64, opts preprocessUploadOptions) (*PreprocessResponse, error) {
	if size > maxUploadSourceBytes {
		return nil, fmt.Errorf(
			"this log is too large to upload (%d MB, limit 1024 MB). Use Archive & clear, or type /combatlog in-game to start a new file",
			size/1024/1024,
		)
	}

	zipPath, err := compressToTempZip(src, size, func(done, total int64) {
		a.emitUploadProgress(UploadProgress{uploadStageCompress, percentOf(done, total), done, total})
	})
	if err != nil {
		return nil, err
	}
	defer os.Remove(zipPath)

	fields := map[string]string{"serverName": opts.ServerName}
	if opts.Unlisted {
		fields["unlisted"] = "true"
	}
	if opts.Merged {
		fields["merged"] = "true"
	}

	apiURL := fmt.Sprintf("%s%s", a.apiBaseURL, opts.Endpoint)
	status, statusText, respBody, err := postZipMultipart(
		context.Background(), apiURL, zipPath, fields,
		func(req *http.Request) {
			req.Header.Set("X-Socket-ID", nativeSocketID)
			req.Header.Set("X-Uploader-Async", "1")
			a.setAPIHeaders(req)
			if opts.Prepare != nil {
				opts.Prepare(req)
			}
		},
		func(done, total int64) {
			a.emitUploadProgress(UploadProgress{uploadStageUpload, percentOf(done, total), done, total})
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not reach the server: %w", err)
	}
	if status != http.StatusOK && status != http.StatusAccepted {
		return nil, newAPIError(status, statusText, respBody)
	}

	var parsed preprocessResponseBody
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("could not read the server response: %w", err)
	}
	response := parsed.PreprocessResponse
	response.ViewLogURL = a.viewLogURL
	a.uploadSources.Store(response.PreprocessID, uploadSource{Source: opts.Source, ServerName: opts.ServerName})
	if !parsed.Streaming {
		if response.AutoQueued {
			a.recordUploadStarted(response.PreprocessID, response.Instances, opts.Unlisted)
		}
		return &response, nil
	}

	if err := a.awaitPreprocess(&response); err != nil {
		return nil, err
	}
	if len(response.Instances) == 1 {
		if _, err := a.EnqueueJobs(response.PreprocessID, response.Instances, opts.Unlisted); err != nil {
			return nil, err
		}
		response.AutoQueued = true
	}
	return &response, nil
}

type preprocessStatusBody struct {
	Status                     string     `json:"status"`
	Complete                   bool       `json:"complete"`
	Error                      string     `json:"error"`
	Instances                  []Instance `json:"instances"`
	HasMultipleDetectedServers bool       `json:"hasMultipleDetectedServers"`
	StreamProgress             *struct {
		Percent int `json:"percent"`
	} `json:"streamProgress"`
}

// applyPreprocessStatus folds one poll into the response; done means stop polling.
func applyPreprocessStatus(status preprocessStatusBody, response *PreprocessResponse) (bool, error) {
	if !status.Complete {
		return false, nil
	}
	if strings.TrimSpace(status.Error) != "" {
		return true, fmt.Errorf("%s", strings.TrimSpace(status.Error))
	}
	if len(status.Instances) == 0 {
		return true, fmt.Errorf("no raid instances found in this log")
	}
	response.Instances = status.Instances
	response.HasMultipleDetectedServers = status.HasMultipleDetectedServers
	return true, nil
}

func (a *App) awaitPreprocess(response *PreprocessResponse) error {
	deadline := time.Now().Add(preprocessPollLimit)
	apiURL := fmt.Sprintf("%s/api/v5/uploader/preprocess-status/%d", a.apiBaseURL, response.PreprocessID)
	client := &http.Client{Timeout: 15 * time.Second}

	for time.Now().Before(deadline) {
		time.Sleep(preprocessPollEvery)

		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			return err
		}
		a.setAPIHeaders(req)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return newAPIError(resp.StatusCode, resp.Status, body)
		}

		var status preprocessStatusBody
		if err := json.Unmarshal(body, &status); err != nil {
			continue
		}
		if status.StreamProgress != nil {
			a.emitUploadProgress(UploadProgress{Stage: uploadStageDetect, Percent: status.StreamProgress.Percent})
		}
		done, err := applyPreprocessStatus(status, response)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return fmt.Errorf("the server is taking too long to read this log; check View all logs shortly")
}

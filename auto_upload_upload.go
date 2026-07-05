package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

type apiErrorBody struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func zipCombatLogBytes(logData []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	fileWriter, err := zipWriter.Create(combatLogFileName)
	if err != nil {
		return nil, fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := fileWriter.Write(logData); err != nil {
		return nil, fmt.Errorf("write zip entry: %w", err)
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}

func zipStagingCombatLog(stagingPath string) ([]byte, error) {
	data, err := os.ReadFile(stagingPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("staging file is empty")
	}
	return zipCombatLogBytes(data)
}

func (a *App) postAutoPreprocess(zipBytes []byte, serverName string) (*PreprocessResponse, int, []byte, error) {
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	part, err := writer.CreateFormFile("logFile", "WoWCombatLog.zip")
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err = io.Copy(part, bytes.NewReader(zipBytes)); err != nil {
		return nil, 0, nil, fmt.Errorf("copy zip to form: %w", err)
	}
	_ = writer.WriteField("serverName", serverName)
	writer.Close()

	apiURL := fmt.Sprintf("%s/api/v5/uploader/auto-preprocess", a.apiBaseURL)
	req, err := http.NewRequest(http.MethodPost, apiURL, requestBody)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Socket-ID", "wails-native-client-polling")
	req.Header.Set("X-API-Token", strings.TrimSpace(a.config.ApiToken))
	req.Header.Set("X-Device-Id", a.ensureDeviceID())

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, resp.StatusCode, respBody, fmt.Errorf("auto-preprocess failed: %s", resp.Status)
	}

	var preprocessResponse PreprocessResponse
	if err := json.Unmarshal(respBody, &preprocessResponse); err != nil {
		return nil, resp.StatusCode, respBody, fmt.Errorf("decode auto-preprocess response: %w", err)
	}
	preprocessResponse.ViewLogURL = a.viewLogURL
	return &preprocessResponse, resp.StatusCode, respBody, nil
}

func parseAPIErrorBody(body []byte) apiErrorBody {
	var parsed apiErrorBody
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}
	return parsed
}

func (a *App) advanceTailFingerprintFromLog(logPath string) error {
	lines, _, fileSize, modTime, err := readTailFingerprint(logPath, tailFingerprintLineCount)
	if err != nil {
		return err
	}

	state, err := a.loadTailState()
	if err != nil {
		return err
	}
	if state.BaselineEstablishedAt == "" {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state.TailFingerprint = append([]string(nil), lines...)
	state.LastByteOffset = fileSize
	state.SourceFileSize = fileSize
	state.SourceFileMtimeUnix = modTime
	state.LastUploadAt = now
	return a.saveTailState(state)
}

func removeStagingFile(stagingPath string) {
	if strings.TrimSpace(stagingPath) == "" {
		return
	}
	if err := os.Remove(stagingPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[AutoUpload] Could not delete staging file %s: %v\n", stagingPath, err)
	}
}

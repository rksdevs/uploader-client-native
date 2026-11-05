package main

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompressToTempZipRoundTrip(t *testing.T) {
	content := strings.Repeat("3/15 20:00:01.123  SPELL_DAMAGE,x\n", 5000)
	var lastDone int64
	path, err := compressToTempZip(strings.NewReader(content), int64(len(content)), func(done, total int64) {
		lastDone = done
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	if lastDone != int64(len(content)) {
		t.Fatalf("final progress %d, want %d", lastDone, len(content))
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || reader.File[0].Name != combatLogFileName {
		t.Fatalf("unexpected entries: %+v", reader.File)
	}
	entry, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer entry.Close()
	got, err := io.ReadAll(entry)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatal("zip content differs from source")
	}
}

func TestCompressToTempZipRejectsEmptyAndCleansUp(t *testing.T) {
	before, _ := filepath.Glob(filepath.Join(os.TempDir(), "wow-logs-upload-*.zip"))
	if _, err := compressToTempZip(strings.NewReader(""), 0, nil); err == nil {
		t.Fatal("expected an error for an empty log")
	}
	after, _ := filepath.Glob(filepath.Join(os.TempDir(), "wow-logs-upload-*.zip"))
	if len(after) != len(before) {
		t.Fatalf("temp zip left behind: %d -> %d", len(before), len(after))
	}
}

func TestPostZipMultipartSendsFieldsFileAndLength(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "log.zip")
	payload := strings.Repeat("zip-bytes", 1000)
	if err := os.WriteFile(zipPath, []byte(payload), 0644); err != nil {
		t.Fatal(err)
	}

	var gotServer, gotMerged, gotFile, gotHeader string
	var gotLength int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLength = r.ContentLength
		gotHeader = r.Header.Get("X-Test")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotServer = r.FormValue("serverName")
		gotMerged = r.FormValue("merged")
		file, header, err := r.FormFile("logFile")
		if err != nil {
			t.Errorf("form file: %v", err)
			return
		}
		defer file.Close()
		if header.Filename != "WoWCombatLog.zip" {
			t.Errorf("filename %q", header.Filename)
		}
		data, _ := io.ReadAll(file)
		gotFile = string(data)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"preprocessId":5}`))
	}))
	defer server.Close()

	var reported int64
	status, _, body, err := postZipMultipart(
		context.Background(), server.URL, zipPath,
		map[string]string{"serverName": "Warmane_Icecrown", "merged": "true"},
		func(r *http.Request) { r.Header.Set("X-Test", "yes") },
		func(done, total int64) { reported = done },
	)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusAccepted || !strings.Contains(string(body), "preprocessId") {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if gotServer != "Warmane_Icecrown" || gotMerged != "true" || gotHeader != "yes" {
		t.Fatalf("server=%q merged=%q header=%q", gotServer, gotMerged, gotHeader)
	}
	if gotFile != payload {
		t.Fatal("uploaded file differs from the zip")
	}
	if gotLength <= int64(len(payload)) {
		t.Fatalf("expected a known Content-Length, got %d", gotLength)
	}
	if reported != gotLength {
		t.Fatalf("final progress %d, want %d", reported, gotLength)
	}
}

func uploadTestApp(t *testing.T, handler http.HandlerFunc) *App {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	app := &App{apiBaseURL: server.URL, viewLogURL: "https://site.test", pendingPolls: map[int]bool{}}
	// Keep test uploads out of the real history file.
	app.historyOnce.Do(func() {
		app.historyStore = &historyStore{path: filepath.Join(t.TempDir(), historyFileName)}
	})
	return app
}

func TestUploadCombatLogSyncGatewayResponse(t *testing.T) {
	app := uploadTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Uploader-Async") != "1" {
			t.Errorf("missing async opt-in header")
		}
		if r.Header.Get(nativeUploaderClientHeader) != "1" {
			t.Errorf("missing native header")
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"preprocessId":9,"autoQueued":true}`))
	})
	content := "3/15 20:00:01.123  SPELL_DAMAGE,x\n"
	resp, err := app.uploadCombatLog(strings.NewReader(content), int64(len(content)), preprocessUploadOptions{
		Endpoint: "/api/v5/uploader/preprocess", ServerName: "Warmane_Icecrown",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.PreprocessID != 9 || !resp.AutoQueued || resp.ViewLogURL != "https://site.test" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUploadCombatLogSurfacesServerMessage(t *testing.T) {
	app := uploadTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"Uploads are disabled for maintenance.","code":"MAINTENANCE_MODE"}`))
	})
	content := "line\n"
	_, err := app.uploadCombatLog(strings.NewReader(content), int64(len(content)), preprocessUploadOptions{
		Endpoint: "/api/v5/uploader/preprocess", ServerName: "x",
	})
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apiError, got %v", err)
	}
	if apiErr.Code != "MAINTENANCE_MODE" || !strings.Contains(apiErr.Message, "maintenance") {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
}

func TestUploadCombatLogRejectsOversizedSource(t *testing.T) {
	app := &App{}
	_, err := app.uploadCombatLog(strings.NewReader(""), maxUploadSourceBytes+1, preprocessUploadOptions{})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected a size error, got %v", err)
	}
}

func TestApplyPreprocessStatus(t *testing.T) {
	var response PreprocessResponse
	if done, err := applyPreprocessStatus(preprocessStatusBody{Complete: false}, &response); done || err != nil {
		t.Fatalf("incomplete: done=%v err=%v", done, err)
	}
	if _, err := applyPreprocessStatus(preprocessStatusBody{Complete: true, Error: "bad archive"}, &response); err == nil {
		t.Fatal("expected the worker error")
	}
	if _, err := applyPreprocessStatus(preprocessStatusBody{Complete: true}, &response); err == nil {
		t.Fatal("expected an error when no instances were found")
	}
	done, err := applyPreprocessStatus(preprocessStatusBody{Complete: true, Instances: []Instance{{Name: "ICC"}}}, &response)
	if !done || err != nil || len(response.Instances) != 1 {
		t.Fatalf("done=%v err=%v instances=%d", done, err, len(response.Instances))
	}
}

func TestNewAPIErrorFallbacks(t *testing.T) {
	if got := newAPIError(500, "500 Internal", []byte("<html>")).Message; !strings.Contains(got, "500 Internal") {
		t.Fatalf("got %q", got)
	}
	both := newAPIError(500, "500", []byte(`{"message":"Server error.","error":"Unsupported format"}`))
	if both.Message != "Server error. Unsupported format" {
		t.Fatalf("got %q", both.Message)
	}
}

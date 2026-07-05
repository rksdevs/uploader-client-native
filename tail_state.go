package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const tailStateFileName = "tail-state.json"

// TailState tracks where the auto-upload tail last synced with the source combat log.
type TailState struct {
	TailFingerprint       []string `json:"tailFingerprint"`
	LastByteOffset        int64    `json:"lastByteOffset"`
	SourceFileSize        int64    `json:"sourceFileSize"`
	SourceFileMtimeUnix   int64    `json:"sourceFileMtimeUnix"`
	LastUploadAt          string   `json:"lastUploadAt,omitempty"`
	LastPreprocessID      int      `json:"lastPreprocessId,omitempty"`
	BaselineEstablishedAt string   `json:"baselineEstablishedAt,omitempty"`
}

func (a *App) tailStatePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "WoWLogsUploader", tailStateFileName)
}

func (a *App) loadTailState() (TailState, error) {
	path := a.tailStatePath()
	if path == "" {
		return TailState{}, fmt.Errorf("could not resolve tail state path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TailState{}, nil
		}
		return TailState{}, err
	}
	var state TailState
	if err := json.Unmarshal(data, &state); err != nil {
		return TailState{}, err
	}
	return state, nil
}

func (a *App) saveTailState(state TailState) error {
	path := a.tailStatePath()
	if path == "" {
		return fmt.Errorf("could not resolve tail state path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *App) hasTailBaseline() bool {
	state, err := a.loadTailState()
	if err != nil {
		return false
	}
	return state.BaselineEstablishedAt != "" && len(state.TailFingerprint) > 0
}

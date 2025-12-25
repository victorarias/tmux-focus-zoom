package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	defaultConfigDir = ".config/tmux-focus-zoom"
	stateFile        = "state.json"
)

// configDir returns the config directory, respecting FOCUS_ZOOM_CONFIG_DIR env var
func configDir() string {
	if dir := os.Getenv("FOCUS_ZOOM_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfigDir
	}
	return filepath.Join(home, defaultConfigDir)
}

// State represents the focus-zoom state for a session/window
type State struct {
	Enabled  bool   `json:"enabled"`
	Session  string `json:"session"`
	Window   string `json:"window"`
	Snapshot string `json:"snapshot"`
}

// stateFilePath returns the full path to the state file
func stateFilePath() (string, error) {
	return filepath.Join(configDir(), stateFile), nil
}

// ensureConfigDir creates the config directory if it doesn't exist
func ensureConfigDir() error {
	return os.MkdirAll(configDir(), 0755)
}

// LoadState reads the state from disk
func LoadState() (*State, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveState writes the state to disk
func SaveState(state *State) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	path, err := stateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ClearState removes the state file
func ClearState() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	// Use temp directory for test
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Test saving state
	state := &State{
		Enabled:  true,
		Session:  "main",
		Window:   "1",
		Snapshot: "@bbe6,200x50,0,0{140x50,0,0,1,59x50,141,0}",
	}

	err := SaveState(state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, configDir, stateFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Test loading state
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loaded.Enabled != state.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", loaded.Enabled, state.Enabled)
	}
	if loaded.Session != state.Session {
		t.Errorf("Session mismatch: got %v, want %v", loaded.Session, state.Session)
	}
	if loaded.Window != state.Window {
		t.Errorf("Window mismatch: got %v, want %v", loaded.Window, state.Window)
	}
	if loaded.Snapshot != state.Snapshot {
		t.Errorf("Snapshot mismatch: got %v, want %v", loaded.Snapshot, state.Snapshot)
	}
}

func TestLoadStateNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Load should return empty state, not error
	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state.Enabled {
		t.Error("Expected disabled state for missing file")
	}
}

func TestClearState(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create state
	state := &State{Enabled: true, Session: "test", Window: "1", Snapshot: "layout"}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Clear it
	if err := ClearState(); err != nil {
		t.Fatalf("ClearState failed: %v", err)
	}

	// Verify file is gone
	path := filepath.Join(tmpDir, configDir, stateFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("State file should have been removed")
	}

	// Load should return empty state
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState after clear failed: %v", err)
	}
	if loaded.Enabled {
		t.Error("Expected disabled state after clear")
	}
}

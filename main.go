package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Status bar colors (Catppuccin Mocha yellow)
	statusColor = "#f9e2af"
	zoomIcon    = "󰍉"
)

var debugLog *os.File

func initDebugLog() {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".config", "tmux-focus-zoom", "debug.log")
	debugLog, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func debugf(format string, args ...interface{}) {
	if debugLog != nil {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Fprintf(debugLog, "[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
		debugLog.Sync()
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: tmux-focus-zoom <toggle|apply|status>")
		os.Exit(1)
	}

	initDebugLog()

	cmd := os.Args[1]

	var err error
	switch cmd {
	case "toggle":
		err = cmdToggle()
	case "apply":
		err = cmdApply()
	case "status":
		err = cmdStatus()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdToggle enables or disables focus-zoom
func cmdToggle() error {
	debugf("cmdToggle: loading state")
	state, err := LoadState()
	if err != nil {
		debugf("cmdToggle: LoadState error: %v", err)
		return fmt.Errorf("LoadState: %w", err)
	}
	debugf("cmdToggle: state.Enabled=%v", state.Enabled)

	if state.Enabled {
		// Disable: try to restore snapshot, then clear state
		debugf("cmdToggle: disabling")

		// Check if snapshot is still valid (same pane count)
		canRestore := false
		if state.Snapshot != "" {
			if snapshotTree, err := ParseLayout(state.Snapshot); err == nil {
				snapshotPanes := countPanes(snapshotTree)
				currentPanes, _ := GetPaneCount()
				canRestore = snapshotPanes == currentPanes
				debugf("cmdToggle: snapshot panes=%d, current panes=%d, canRestore=%v",
					snapshotPanes, currentPanes, canRestore)
			}
		}

		if canRestore {
			debugf("cmdToggle: restoring snapshot")
			if err := RestoreSnapshot(state); err != nil {
				debugf("cmdToggle: RestoreSnapshot error (ignored): %v", err)
			}
		} else {
			debugf("cmdToggle: skipping restore (pane count changed)")
		}

		if err := ClearState(); err != nil {
			debugf("cmdToggle: ClearState error: %v", err)
			return fmt.Errorf("ClearState: %w", err)
		}
		return DisplayMessage("Focus zoom: OFF")
	}

	// Enable: capture snapshot and apply zoom
	debugf("cmdToggle: enabling, capturing snapshot")
	newState, err := CaptureSnapshot()
	if err != nil {
		debugf("cmdToggle: CaptureSnapshot error: %v", err)
		return fmt.Errorf("CaptureSnapshot: %w", err)
	}
	debugf("cmdToggle: snapshot=%s", newState.Snapshot)

	if err := SaveState(newState); err != nil {
		debugf("cmdToggle: SaveState error: %v", err)
		return fmt.Errorf("SaveState: %w", err)
	}
	debugf("cmdToggle: state saved")

	// Apply zoom immediately
	if err := ApplyZoom(newState); err != nil {
		debugf("cmdToggle: ApplyZoom error: %v", err)
		return fmt.Errorf("ApplyZoom: %w", err)
	}
	debugf("cmdToggle: zoom applied")

	return DisplayMessage("Focus zoom: ON")
}

// cmdApply is called on pane-focus-in to apply zoom effect
func cmdApply() error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if !state.Enabled {
		return nil
	}

	return ApplyZoom(state)
}

// cmdStatus outputs the status for the tmux status bar
func cmdStatus() error {
	state, err := LoadState()
	if err != nil || !state.Enabled {
		fmt.Printf("#[fg=%s]%s OFF#[default] ", statusColor, zoomIcon)
		return nil
	}

	// Check if current window matches the state
	match, err := IsMatchingWindow(state)
	if err != nil || !match {
		fmt.Printf("#[fg=%s]%s OFF#[default] ", statusColor, zoomIcon)
		return nil
	}

	fmt.Printf("#[fg=%s]%s ON#[default] ", statusColor, zoomIcon)
	return nil
}

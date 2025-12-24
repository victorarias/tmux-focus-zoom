package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// TmuxCmd executes a tmux command and returns the output
func TmuxCmd(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// TmuxCmdNoOutput executes a tmux command without capturing output
func TmuxCmdNoOutput(args ...string) error {
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// GetCurrentSession returns the current session name
func GetCurrentSession() (string, error) {
	return TmuxCmd("display-message", "-p", "#{session_name}")
}

// GetCurrentWindow returns the current window index
func GetCurrentWindow() (string, error) {
	return TmuxCmd("display-message", "-p", "#{window_index}")
}

// GetCurrentPane returns the current pane ID (e.g., "%42")
func GetCurrentPane() (string, error) {
	return TmuxCmd("display-message", "-p", "#{pane_id}")
}

// GetActivePaneID returns the current pane's numeric ID
func GetActivePaneID() (int, error) {
	paneID, err := GetCurrentPane()
	if err != nil {
		return 0, err
	}
	// pane_id is in format "%42", strip the % prefix
	if len(paneID) > 0 && paneID[0] == '%' {
		paneID = paneID[1:]
	}
	return strconv.Atoi(paneID)
}

// GetWindowLayout returns the current window layout string
func GetWindowLayout() (string, error) {
	return TmuxCmd("display-message", "-p", "#{window_layout}")
}

// GetPaneCount returns the number of panes in the current window
func GetPaneCount() (int, error) {
	out, err := TmuxCmd("display-message", "-p", "#{window_panes}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// GetWindowSize returns window width and height
func GetWindowSize() (width, height int, err error) {
	w, err := TmuxCmd("display-message", "-p", "#{window_width}")
	if err != nil {
		return 0, 0, err
	}
	h, err := TmuxCmd("display-message", "-p", "#{window_height}")
	if err != nil {
		return 0, 0, err
	}

	width, err = strconv.Atoi(w)
	if err != nil {
		return 0, 0, err
	}
	height, err = strconv.Atoi(h)
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}

// SelectLayout applies a layout string to the current window
func SelectLayout(layout string) error {
	return TmuxCmdNoOutput("select-layout", layout)
}

// ResizePane resizes the current pane to the specified dimensions
func ResizePane(width, height int) error {
	if err := TmuxCmdNoOutput("resize-pane", "-x", strconv.Itoa(width)); err != nil {
		return err
	}
	return TmuxCmdNoOutput("resize-pane", "-y", strconv.Itoa(height))
}

// DisplayMessage shows a message in tmux
func DisplayMessage(msg string) error {
	return TmuxCmdNoOutput("display-message", msg)
}

// PaneInfo holds information about a pane
type PaneInfo struct {
	ID     string
	Index  int
	Width  int
	Height int
	Left   int
	Top    int
	Active bool
}

// GetPanes returns info about all panes in current window
func GetPanes() ([]PaneInfo, error) {
	// Format: id:index:width:height:left:top:active
	out, err := TmuxCmd("list-panes", "-F", "#{pane_id}:#{pane_index}:#{pane_width}:#{pane_height}:#{pane_left}:#{pane_top}:#{pane_active}")
	if err != nil {
		return nil, err
	}

	var panes []PaneInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 7 {
			continue
		}

		index, _ := strconv.Atoi(parts[1])
		width, _ := strconv.Atoi(parts[2])
		height, _ := strconv.Atoi(parts[3])
		left, _ := strconv.Atoi(parts[4])
		top, _ := strconv.Atoi(parts[5])
		active := parts[6] == "1"

		panes = append(panes, PaneInfo{
			ID:     parts[0],
			Index:  index,
			Width:  width,
			Height: height,
			Left:   left,
			Top:    top,
			Active: active,
		})
	}

	return panes, nil
}

// ResizePaneWidth resizes a pane's width only
func ResizePaneWidth(paneID string, width int) error {
	debugf("resize-pane -t %s -x %d", paneID, width)
	return TmuxCmdNoOutput("resize-pane", "-t", paneID, "-x", strconv.Itoa(width))
}

// ResizePaneHeight resizes a pane's height only
func ResizePaneHeight(paneID string, height int) error {
	debugf("resize-pane -t %s -y %d", paneID, height)
	return TmuxCmdNoOutput("resize-pane", "-t", paneID, "-y", strconv.Itoa(height))
}

// GetZoomPercent returns the configured zoom percentage from tmux option @focus-zoom-percent
// Falls back to DefaultZoomPercent if not set or invalid
func GetZoomPercent() int {
	out, err := TmuxCmd("show-option", "-gqv", "@focus-zoom-percent")
	if err != nil || out == "" {
		return DefaultZoomPercent
	}
	percent, err := strconv.Atoi(out)
	if err != nil || percent < 10 || percent > 95 {
		return DefaultZoomPercent
	}
	return percent
}

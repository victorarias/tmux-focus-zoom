package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Integration tests that run against a real tmux server.
// These tests spawn a tmux server, create panes, run the plugin, and verify behavior.
//
// Run with: go test -v -tags=integration -run Integration
// Skip with: go test -v -short (skips integration tests)

const (
	testSocketName = "focus-zoom-test"
	testSession    = "test"
)

// tmuxTest provides a test harness for running tmux integration tests
type tmuxTest struct {
	t          *testing.T
	socketPath string
	configDir  string
	binary     string
	cleanup    func()
}

// newTmuxTest creates a new tmux test environment
func newTmuxTest(t *testing.T) *tmuxTest {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	binary := filepath.Join(t.TempDir(), "tmux-focus-zoom")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}

	// Create unique socket path (short path to avoid Unix socket length limit)
	socketPath := fmt.Sprintf("/tmp/fz-test-%d", os.Getpid())

	// Create temp config directory for state
	configDir := t.TempDir()

	tt := &tmuxTest{
		t:          t,
		socketPath: socketPath,
		configDir:  configDir,
		binary:     binary,
	}

	// Start tmux server
	if err := tt.tmux("new-session", "-d", "-s", testSession, "-x", "200", "-y", "50"); err != nil {
		t.Fatalf("failed to start tmux server: %v", err)
	}

	// Give tmux time to start
	time.Sleep(100 * time.Millisecond)

	tt.cleanup = func() {
		// Kill the tmux server
		_ = tt.tmux("kill-server")
		// Remove socket file
		_ = os.Remove(socketPath)
	}

	return tt
}

// tmux runs a tmux command with the test socket
func (tt *tmuxTest) tmux(args ...string) error {
	fullArgs := append([]string{"-S", tt.socketPath}, args...)
	cmd := exec.Command("tmux", fullArgs...)
	cmd.Env = append(os.Environ(), "TMUX=") // Ensure we're not inside another tmux
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %v failed: %v\n%s", args, err, out)
	}
	return nil
}

// tmuxOutput runs a tmux command and returns output
func (tt *tmuxTest) tmuxOutput(args ...string) (string, error) {
	fullArgs := append([]string{"-S", tt.socketPath}, args...)
	cmd := exec.Command("tmux", fullArgs...)
	cmd.Env = append(os.Environ(), "TMUX=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// runPlugin runs the focus-zoom binary with the test socket
func (tt *tmuxTest) runPlugin(args ...string) error {
	cmd := exec.Command(tt.binary, args...)
	cmd.Env = append(os.Environ(),
		"TMUX_SOCKET="+tt.socketPath,       // Tell binary which socket to use
		"FOCUS_ZOOM_CONFIG_DIR="+tt.configDir, // Use isolated config
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("plugin %v failed: %v\n%s", args, err, out)
	}
	return nil
}

// getLayout returns the current window layout
func (tt *tmuxTest) getLayout() (string, error) {
	return tt.tmuxOutput("display-message", "-p", "#{window_layout}")
}

// getPaneCount returns number of panes in current window
func (tt *tmuxTest) getPaneCount() (int, error) {
	out, err := tt.tmuxOutput("display-message", "-p", "#{window_panes}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// getPaneWidths returns widths of all panes
func (tt *tmuxTest) getPaneWidths() ([]int, error) {
	out, err := tt.tmuxOutput("list-panes", "-F", "#{pane_width}")
	if err != nil {
		return nil, err
	}
	var widths []int
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		w, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		widths = append(widths, w)
	}
	return widths, nil
}

// splitHorizontal splits the current pane horizontally
func (tt *tmuxTest) splitHorizontal() error {
	return tt.tmux("split-window", "-h")
}

// splitVertical splits the current pane vertically
func (tt *tmuxTest) splitVertical() error {
	return tt.tmux("split-window", "-v")
}

// selectPane selects a pane by index (0-based, converted to tmux %id format)
func (tt *tmuxTest) selectPane(index int) error {
	return tt.tmux("select-pane", "-t", fmt.Sprintf("%%%d", index))
}

// evenHorizontal applies even-horizontal layout
func (tt *tmuxTest) evenHorizontal() error {
	return tt.tmux("select-layout", "even-horizontal")
}

// close cleans up the test environment
func (tt *tmuxTest) close() {
	if tt.cleanup != nil {
		tt.cleanup()
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestIntegration_TwoPanesHorizontal(t *testing.T) {
	tt := newTmuxTest(t)
	defer tt.close()

	// Create two panes side by side
	if err := tt.splitHorizontal(); err != nil {
		t.Fatalf("split failed: %v", err)
	}
	if err := tt.evenHorizontal(); err != nil {
		t.Fatalf("even layout failed: %v", err)
	}

	// Verify we have 2 panes
	count, err := tt.getPaneCount()
	if err != nil {
		t.Fatalf("getPaneCount failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 panes, got %d", count)
	}

	// Get initial widths
	initialWidths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	t.Logf("Initial widths: %v", initialWidths)

	// Widths should be roughly equal (within 1 due to border)
	if abs(initialWidths[0]-initialWidths[1]) > 2 {
		t.Errorf("initial widths not equal: %v", initialWidths)
	}

	// Enable zoom
	if err := tt.runPlugin("toggle"); err != nil {
		t.Fatalf("toggle failed: %v", err)
	}

	// Give it time to apply
	time.Sleep(100 * time.Millisecond)

	// Get zoomed widths
	zoomedWidths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	t.Logf("Zoomed widths: %v", zoomedWidths)

	// Active pane should be larger (65% default)
	// Pane 1 is active after split, so it should be larger
	if zoomedWidths[1] <= zoomedWidths[0] {
		t.Errorf("expected pane 1 to be larger after zoom, got %v", zoomedWidths)
	}

	// Switch to pane 0
	if err := tt.selectPane(0); err != nil {
		t.Fatalf("select pane failed: %v", err)
	}

	// Trigger apply (simulating focus change)
	if err := tt.runPlugin("apply"); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Now pane 0 should be larger
	switchedWidths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	t.Logf("After switch widths: %v", switchedWidths)

	if switchedWidths[0] <= switchedWidths[1] {
		t.Errorf("expected pane 0 to be larger after switch, got %v", switchedWidths)
	}

	// Toggle off - should restore original
	if err := tt.runPlugin("toggle"); err != nil {
		t.Fatalf("toggle off failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	restoredWidths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	t.Logf("Restored widths: %v", restoredWidths)

	// Widths should be roughly equal again
	if abs(restoredWidths[0]-restoredWidths[1]) > 2 {
		t.Errorf("restored widths not equal: %v", restoredWidths)
	}
}

func TestIntegration_PaneCloseWhileZoomed(t *testing.T) {
	tt := newTmuxTest(t)
	defer tt.close()

	// Create three panes
	if err := tt.splitHorizontal(); err != nil {
		t.Fatalf("split 1 failed: %v", err)
	}
	if err := tt.splitHorizontal(); err != nil {
		t.Fatalf("split 2 failed: %v", err)
	}
	if err := tt.evenHorizontal(); err != nil {
		t.Fatalf("even layout failed: %v", err)
	}

	count, _ := tt.getPaneCount()
	if count != 3 {
		t.Fatalf("expected 3 panes, got %d", count)
	}

	// Enable zoom
	if err := tt.runPlugin("toggle"); err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Close the rightmost pane (pane 2)
	if err := tt.tmux("kill-pane", "-t", "2"); err != nil {
		t.Fatalf("kill-pane failed: %v", err)
	}

	count, _ = tt.getPaneCount()
	if count != 2 {
		t.Fatalf("expected 2 panes after close, got %d", count)
	}

	// Apply should work without error
	if err := tt.runPlugin("apply"); err != nil {
		t.Fatalf("apply after pane close failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Layout should be valid (no corruption)
	layout, err := tt.getLayout()
	if err != nil {
		t.Fatalf("getLayout failed: %v", err)
	}
	t.Logf("Layout after pane close: %s", layout)

	// Should contain 2 pane IDs
	widths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	if len(widths) != 2 {
		t.Errorf("expected 2 pane widths, got %d", len(widths))
	}
}

func TestIntegration_ThreePanesZoom(t *testing.T) {
	tt := newTmuxTest(t)
	defer tt.close()

	// Create three panes horizontally
	if err := tt.splitHorizontal(); err != nil {
		t.Fatalf("split 1 failed: %v", err)
	}
	if err := tt.splitHorizontal(); err != nil {
		t.Fatalf("split 2 failed: %v", err)
	}
	if err := tt.evenHorizontal(); err != nil {
		t.Fatalf("even layout failed: %v", err)
	}

	// Select pane 0 (leftmost)
	if err := tt.selectPane(0); err != nil {
		t.Fatalf("select pane failed: %v", err)
	}

	// Enable zoom
	if err := tt.runPlugin("toggle"); err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	widths, err := tt.getPaneWidths()
	if err != nil {
		t.Fatalf("getPaneWidths failed: %v", err)
	}
	t.Logf("Three pane widths after zoom on pane 0: %v", widths)

	// Pane 0 should be largest
	if widths[0] <= widths[1] || widths[0] <= widths[2] {
		t.Errorf("expected pane 0 to be largest, got %v", widths)
	}

	// Panes 1 and 2 should share remaining space roughly equally
	if abs(widths[1]-widths[2]) > 5 {
		t.Errorf("expected panes 1 and 2 to be similar size, got %v", widths)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

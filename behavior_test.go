package main

import (
	"testing"
)

/*
=============================================================================
BEHAVIOR TESTS FOR TMUX-FOCUS-ZOOM
=============================================================================

These tests document the DESIRED behavior vs ACTUAL tmux behavior.

Layout under test (claude-cortex window):
┌─────────────┬─────────────┬─────────────┐
│ P1 (%26)    │             │             │
│ 84x30       │  P3 (%36)   │  P4 (%42)   │
│ left=0      │  85x61      │  84x61      │
├─────────────┤  left=85    │  left=171   │
│ P2 (%41)    │             │             │
│ 84x30       │             │             │
│ left=0      │             │             │
└─────────────┴─────────────┴─────────────┘

Window: 255x61 (with 2 column borders)
*/

// TestDesiredBehavior_P4Active documents what SHOULD happen when P4 is focused
func TestDesiredBehavior_P4Active(t *testing.T) {
	t.Log("=== DESIRED: P4 (%42) focused ===")
	t.Log("")
	t.Log("Before:")
	t.Log("  col0 (P1,P2): 84 wide")
	t.Log("  col1 (P3):    85 wide")
	t.Log("  col2 (P4):    84 wide")
	t.Log("")
	t.Log("After (65% zoom):")
	t.Log("  col0 (P1,P2): 44 wide  (shrinks proportionally)")
	t.Log("  col1 (P3):    44 wide  (shrinks proportionally)")
	t.Log("  col2 (P4):    165 wide (grows to 65%)")
	t.Log("")
	t.Log("All non-active columns shrink PROPORTIONALLY.")
}

// TestDesiredBehavior_P1Active documents what SHOULD happen when P1 is focused
func TestDesiredBehavior_P1Active(t *testing.T) {
	t.Log("=== DESIRED: P1 (%26) focused ===")
	t.Log("")
	t.Log("Before:")
	t.Log("  col0 (P1,P2): 84 wide")
	t.Log("  col1 (P3):    85 wide")
	t.Log("  col2 (P4):    84 wide")
	t.Log("")
	t.Log("After (65% zoom):")
	t.Log("  col0 (P1,P2): 165 wide (grows to 65%)")
	t.Log("  col1 (P3):    44 wide  (shrinks proportionally)")
	t.Log("  col2 (P4):    44 wide  (shrinks proportionally)")
	t.Log("")
	t.Log("All non-active columns shrink PROPORTIONALLY.")
}

// TestActualTmuxBehavior_ResizePane documents what tmux ACTUALLY does
func TestActualTmuxBehavior_ResizePane(t *testing.T) {
	t.Log("=== ACTUAL: tmux resize-pane behavior ===")
	t.Log("")
	t.Log("Command: tmux resize-pane -t %26 -x 165")
	t.Log("")
	t.Log("Result:")
	t.Log("  col0 (P1,P2): 165 wide (grows as requested)")
	t.Log("  col1 (P3):    4 wide   (ONLY adjacent column shrinks)")
	t.Log("  col2 (P4):    84 wide  (unchanged - not adjacent)")
	t.Log("")
	t.Log("PROBLEM: resize-pane only affects ADJACENT panes.")
	t.Log("         Non-adjacent panes are unchanged.")
	t.Log("")
	t.Log("This is why P1/P2 zoom doesn't work correctly - P4 stays")
	t.Log("the same size while P3 gets crushed to almost nothing.")
}

// TestSolution_LayoutString documents the solution approach
func TestSolution_LayoutString(t *testing.T) {
	t.Log("=== SOLUTION: Layout String Approach ===")
	t.Log("")
	t.Log("Instead of multiple resize-pane calls, we need to:")
	t.Log("")
	t.Log("1. Parse the snapshot layout string")
	t.Log("2. Calculate new sizes for ALL panes")
	t.Log("3. Build a new layout string with desired sizes")
	t.Log("4. Apply with: tmux select-layout <new-layout-string>")
	t.Log("")
	t.Log("Layout string format:")
	t.Log("  checksum,WxH,x,y{child1,child2,...} or [child1,child2,...]")
	t.Log("  {} = horizontal split")
	t.Log("  [] = vertical split")
	t.Log("")
	t.Log("Example original layout:")
	t.Log("  b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}")
	t.Log("")
	t.Log("Parsed structure:")
	t.Log("  Window 255x61 at (0,0)")
	t.Log("  └── Horizontal split {}")
	t.Log("      ├── Vertical split [] at x=0, width=84")
	t.Log("      │   ├── Pane 26: 84x30 at (0,0)")
	t.Log("      │   └── Pane 41: 84x30 at (0,31)")
	t.Log("      ├── Pane 36: 85x61 at (85,0)")
	t.Log("      └── Pane 42: 84x61 at (171,0)")
	t.Log("")
	t.Log("Modified for P1 zoom (col0=165, col1=44, col2=44):")
	t.Log("  xxxx,255x61,0,0{165x61,0,0[165x30,0,0,26,165x30,0,31,41],44x61,166,0,36,44x61,211,0,42}")
}

// =============================================================================
// ACTUAL FAILING TESTS - test production code in layout.go
// =============================================================================

// TestParseLayoutString tests that we can parse a tmux layout string
func TestParseLayoutString(t *testing.T) {
	layout := "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}"

	node, err := ParseLayout(layout)
	if err != nil {
		t.Fatalf("ParseLayout failed: %v", err)
	}

	// Verify root node
	if node.Width != 255 || node.Height != 61 {
		t.Errorf("Root size: got %dx%d, want 255x61", node.Width, node.Height)
	}

	// Verify it's a horizontal split with 3 children
	if node.SplitType != SplitHorizontal {
		t.Errorf("Root split type: got %v, want horizontal", node.SplitType)
	}
	if len(node.Children) != 3 {
		t.Fatalf("Root children: got %d, want 3", len(node.Children))
	}

	// Verify first child is vertical split (col0 with P1, P2)
	col0 := node.Children[0]
	if col0.Width != 84 {
		t.Errorf("Col0 width: got %d, want 84", col0.Width)
	}
	if col0.SplitType != SplitVertical {
		t.Errorf("Col0 split type: got %v, want vertical", col0.SplitType)
	}
	if len(col0.Children) != 2 {
		t.Errorf("Col0 children: got %d, want 2", len(col0.Children))
	}

	// Verify col0 children are panes 26 and 41
	if col0.Children[0].PaneID != 26 {
		t.Errorf("P1 pane ID: got %d, want 26", col0.Children[0].PaneID)
	}
	if col0.Children[1].PaneID != 41 {
		t.Errorf("P2 pane ID: got %d, want 41", col0.Children[1].PaneID)
	}

	// Verify second child is pane 36 (col1)
	col1 := node.Children[1]
	if col1.Width != 85 || col1.PaneID != 36 {
		t.Errorf("Col1: got width=%d pane=%d, want width=85 pane=36", col1.Width, col1.PaneID)
	}

	// Verify third child is pane 42 (col2)
	col2 := node.Children[2]
	if col2.Width != 84 || col2.PaneID != 42 {
		t.Errorf("Col2: got width=%d pane=%d, want width=84 pane=42", col2.Width, col2.PaneID)
	}
}

// TestBuildLayoutString tests that we can rebuild a layout string from parsed nodes
func TestBuildLayoutString(t *testing.T) {
	original := "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}"

	node, err := ParseLayout(original)
	if err != nil {
		t.Fatalf("ParseLayout failed: %v", err)
	}

	rebuilt := BuildLayout(node)

	// The checksum will be different, but structure should match
	// For now just verify we get a non-empty string
	if rebuilt == "" {
		t.Fatal("BuildLayout returned empty string")
	}

	t.Logf("Original: %s", original)
	t.Logf("Rebuilt:  %s", rebuilt)
}

// =============================================================================
// ZOOM LAYOUT MODIFICATION TESTS
// =============================================================================

// TestApplyZoomToLayout_P1Active tests zooming when P1 (pane 26) is focused
// P1 is in col0 (the leftmost column with vertical split P1/P2)
func TestApplyZoomToLayout_P1Active(t *testing.T) {
	// Original layout: 3 columns, col0 has vertical split
	// col0: 84, col1: 85, col2: 84 (total: 253 + 2 borders = 255)
	layout := "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}"

	node, err := ParseLayout(layout)
	if err != nil {
		t.Fatalf("ParseLayout failed: %v", err)
	}

	// Zoom pane 26 (P1) - should grow col0 to 65%
	activePaneID := 26
	zoomed := ApplyZoomToLayout(node, activePaneID, DefaultZoomPercent)

	rebuilt := BuildLayout(zoomed)
	t.Logf("Zoomed layout: %s", rebuilt)

	// Verify col0 (containing P1) grew to ~165 (65% of 255)
	col0 := zoomed.Children[0]
	expectedWidth := (255 * DefaultZoomPercent) / 100 // 165
	if col0.Width < expectedWidth-5 || col0.Width > expectedWidth+5 {
		t.Errorf("Col0 width after zoom: got %d, want ~%d", col0.Width, expectedWidth)
	}

	// Verify col1 and col2 shrunk proportionally
	// Original: col1=85, col2=84 (total=169)
	// Remaining: 255 - 165 - 2 borders = 88
	// col1 should get: 85/169 * 88 ≈ 44
	// col2 should get: 84/169 * 88 ≈ 43
	col1 := zoomed.Children[1]
	col2 := zoomed.Children[2]

	if col1.Width < 40 || col1.Width > 50 {
		t.Errorf("Col1 width after zoom: got %d, want ~44", col1.Width)
	}
	if col2.Width < 40 || col2.Width > 50 {
		t.Errorf("Col2 width after zoom: got %d, want ~43", col2.Width)
	}

	// Verify P1 and P2 (in col0) got their widths updated too
	p1 := col0.Children[0]
	p2 := col0.Children[1]
	if p1.Width != col0.Width {
		t.Errorf("P1 width should match col0: got %d, want %d", p1.Width, col0.Width)
	}
	if p2.Width != col0.Width {
		t.Errorf("P2 width should match col0: got %d, want %d", p2.Width, col0.Width)
	}

	t.Logf("Result: col0=%d, col1=%d, col2=%d (sum=%d)",
		col0.Width, col1.Width, col2.Width,
		col0.Width+col1.Width+col2.Width+2)
}

// TestApplyZoomToLayout_P4Active tests zooming when P4 (pane 42) is focused
// P4 is in col2 (the rightmost column, single pane)
func TestApplyZoomToLayout_P4Active(t *testing.T) {
	layout := "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}"

	node, err := ParseLayout(layout)
	if err != nil {
		t.Fatalf("ParseLayout failed: %v", err)
	}

	// Zoom pane 42 (P4) - should grow col2 to 65%
	activePaneID := 42
	zoomed := ApplyZoomToLayout(node, activePaneID, DefaultZoomPercent)

	rebuilt := BuildLayout(zoomed)
	t.Logf("Zoomed layout: %s", rebuilt)

	// Verify col2 (containing P4) grew to ~165
	col2 := zoomed.Children[2]
	expectedWidth := (255 * DefaultZoomPercent) / 100 // 165
	if col2.Width < expectedWidth-5 || col2.Width > expectedWidth+5 {
		t.Errorf("Col2 width after zoom: got %d, want ~%d", col2.Width, expectedWidth)
	}

	// Verify col0 and col1 shrunk proportionally
	col0 := zoomed.Children[0]
	col1 := zoomed.Children[1]

	if col0.Width < 40 || col0.Width > 50 {
		t.Errorf("Col0 width after zoom: got %d, want ~43", col0.Width)
	}
	if col1.Width < 40 || col1.Width > 50 {
		t.Errorf("Col1 width after zoom: got %d, want ~44", col1.Width)
	}

	t.Logf("Result: col0=%d, col1=%d, col2=%d (sum=%d)",
		col0.Width, col1.Width, col2.Width,
		col0.Width+col1.Width+col2.Width+2)
}

// TestApplyZoomToLayout_P3Active tests zooming when P3 (pane 36) is focused
// P3 is in col1 (the middle column, single pane)
func TestApplyZoomToLayout_P3Active(t *testing.T) {
	layout := "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}"

	node, err := ParseLayout(layout)
	if err != nil {
		t.Fatalf("ParseLayout failed: %v", err)
	}

	// Zoom pane 36 (P3) - should grow col1 to 65%
	activePaneID := 36
	zoomed := ApplyZoomToLayout(node, activePaneID, DefaultZoomPercent)

	rebuilt := BuildLayout(zoomed)
	t.Logf("Zoomed layout: %s", rebuilt)

	// Verify col1 (containing P3) grew to ~165
	col1 := zoomed.Children[1]
	expectedWidth := (255 * DefaultZoomPercent) / 100 // 165
	if col1.Width < expectedWidth-5 || col1.Width > expectedWidth+5 {
		t.Errorf("Col1 width after zoom: got %d, want ~%d", col1.Width, expectedWidth)
	}

	// Verify col0 and col2 shrunk proportionally
	col0 := zoomed.Children[0]
	col2 := zoomed.Children[2]

	if col0.Width < 40 || col0.Width > 50 {
		t.Errorf("Col0 width after zoom: got %d, want ~43", col0.Width)
	}
	if col2.Width < 40 || col2.Width > 50 {
		t.Errorf("Col2 width after zoom: got %d, want ~43", col2.Width)
	}

	t.Logf("Result: col0=%d, col1=%d, col2=%d (sum=%d)",
		col0.Width, col1.Width, col2.Width,
		col0.Width+col1.Width+col2.Width+2)
}

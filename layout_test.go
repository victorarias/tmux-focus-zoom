package main

import (
	"strings"
	"testing"
)

// Test layout representing the claude-cortex window:
// ┌────────────────┬─────────────────┬────────────────┐
// │ P1 (84x30)     │                 │                │
// │ left=0, top=0  │  P3 (85x61)     │  P4 (84x61)    │
// ├────────────────┤  left=85, top=0 │  left=171,top=0│
// │ P2 (84x30)     │                 │                │
// │ left=0, top=31 │                 │                │
// └────────────────┴─────────────────┴────────────────┘
func makeTestPanes() []PaneInfo {
	return []PaneInfo{
		{ID: "%1", Index: 1, Width: 84, Height: 30, Left: 0, Top: 0, Active: false},   // P1
		{ID: "%2", Index: 2, Width: 84, Height: 30, Left: 0, Top: 31, Active: false},  // P2
		{ID: "%3", Index: 3, Width: 85, Height: 61, Left: 85, Top: 0, Active: false},  // P3
		{ID: "%4", Index: 4, Width: 84, Height: 61, Left: 171, Top: 0, Active: false}, // P4
	}
}

func TestFindColumns(t *testing.T) {
	panes := makeTestPanes()
	columns := findColumns(panes)

	if len(columns) != 3 {
		t.Fatalf("Expected 3 columns, got %d", len(columns))
	}

	// Column 0: left=0, contains P1 and P2
	if columns[0].left != 0 {
		t.Errorf("Column 0 left: expected 0, got %d", columns[0].left)
	}
	if len(columns[0].panes) != 2 {
		t.Errorf("Column 0 panes: expected 2, got %d", len(columns[0].panes))
	}
	if columns[0].width != 84 {
		t.Errorf("Column 0 width: expected 84, got %d", columns[0].width)
	}

	// Column 1: left=85, contains P3
	if columns[1].left != 85 {
		t.Errorf("Column 1 left: expected 85, got %d", columns[1].left)
	}
	if len(columns[1].panes) != 1 {
		t.Errorf("Column 1 panes: expected 1, got %d", len(columns[1].panes))
	}

	// Column 2: left=171, contains P4
	if columns[2].left != 171 {
		t.Errorf("Column 2 left: expected 171, got %d", columns[2].left)
	}
	if len(columns[2].panes) != 1 {
		t.Errorf("Column 2 panes: expected 1, got %d", len(columns[2].panes))
	}
}

func TestFindRowsInColumn_SplitColumn(t *testing.T) {
	panes := makeTestPanes()
	columns := findColumns(panes)

	// Column 0 (left column) has P1 and P2 vertically stacked
	rows := findRowsInColumn(columns[0].panes)

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows in column 0, got %d", len(rows))
	}

	// Row 0: top=0 (P1)
	if rows[0].top != 0 {
		t.Errorf("Row 0 top: expected 0, got %d", rows[0].top)
	}
	if rows[0].height != 30 {
		t.Errorf("Row 0 height: expected 30, got %d", rows[0].height)
	}

	// Row 1: top=31 (P2)
	if rows[1].top != 31 {
		t.Errorf("Row 1 top: expected 31, got %d", rows[1].top)
	}
	if rows[1].height != 30 {
		t.Errorf("Row 1 height: expected 30, got %d", rows[1].height)
	}
}

func TestFindRowsInColumn_SinglePaneColumn(t *testing.T) {
	panes := makeTestPanes()
	columns := findColumns(panes)

	// Column 1 (middle) and Column 2 (right) each have one full-height pane
	rows1 := findRowsInColumn(columns[1].panes)
	if len(rows1) != 1 {
		t.Errorf("Expected 1 row in column 1, got %d", len(rows1))
	}

	rows2 := findRowsInColumn(columns[2].panes)
	if len(rows2) != 1 {
		t.Errorf("Expected 1 row in column 2, got %d", len(rows2))
	}
}

// ============================================================================
// Documentation tests - document problem and solution approach
// ============================================================================

// TestSimulateResizeBehavior demonstrates why resize doesn't work as expected
func TestSimulateResizeBehavior(t *testing.T) {
	// Initial layout:
	// | col0 (84) | col1 (85) | col2 (84) | = 255 (with 2 borders)
	// | P1,P2     | P3        | P4        |

	// When P1 is focused and we want 65% zoom:
	// Target: col0=165, col1=44, col2=43

	// What ACTUALLY happens with tmux resize-pane:

	t.Log("=== Simulating tmux resize behavior ===")
	t.Log("")
	t.Log("Initial: col0=84, col1=85, col2=84 (total=255 with borders)")
	t.Log("")

	// Step 1: Resize col0 (P1) to 165
	t.Log("Step 1: resize-pane -t P1 -x 165")
	t.Log("  tmux takes space from col0's RIGHT NEIGHBOR (col1)")
	t.Log("  col0: 84 -> 165 (+81)")
	t.Log("  col1: 85 -> 4 (-81)  <-- shrinks drastically!")
	t.Log("  col2: 84 -> 84 (unchanged - not adjacent to col0)")
	t.Log("")

	// Step 2: Resize col1 to 44
	t.Log("Step 2: resize-pane -t P3 -x 44")
	t.Log("  tmux adjusts boundary between col1 and its neighbors")
	t.Log("  Result depends on current state - unpredictable")
	t.Log("")

	// Step 3: Resize col2 to 43
	t.Log("Step 3: resize-pane -t P4 -x 43")
	t.Log("  col2 shrinks, space goes to col1")
	t.Log("")

	t.Log("=== THE PROBLEM ===")
	t.Log("tmux resize-pane -x N only affects the boundary with ADJACENT panes.")
	t.Log("It does NOT redistribute space across ALL panes proportionally.")
	t.Log("")
	t.Log("=== POTENTIAL SOLUTIONS ===")
	t.Log("1. Construct a custom layout string and use select-layout")
	t.Log("2. Resize in a very specific order to achieve desired result")
	t.Log("3. Accept that perfect proportional resize isn't possible")
}

// TestLayoutStringApproach shows how to construct a layout string
func TestLayoutStringApproach(t *testing.T) {
	// tmux layout strings encode the exact position and size of every pane
	// Format: checksum,WxH,x,y{children} or [children]
	// {} = horizontal split, [] = vertical split

	// Original layout for claude-cortex:
	// b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}
	//
	// Breakdown:
	// - Window: 255x61 at (0,0)
	// - Horizontal split {} containing:
	//   - Vertical split [] at x=0, width=84:
	//     - Pane 26: 84x30 at (0,0)
	//     - Pane 41: 84x30 at (0,31)
	//   - Pane 36: 85x61 at (85,0)
	//   - Pane 42: 84x61 at (171,0)

	// To zoom P1 (pane 26), we want:
	// - col0: 165 wide (was 84)
	// - col1: 44 wide (was 85)
	// - col2: 43 wide (was 84)
	//
	// New positions:
	// - col0: x=0, width=165
	// - col1: x=166, width=44
	// - col2: x=211, width=43

	t.Log("=== LAYOUT STRING APPROACH ===")
	t.Log("")
	t.Log("Instead of multiple resize-pane calls, construct a layout string")
	t.Log("that specifies exact sizes for all panes, then apply with select-layout.")
	t.Log("")
	t.Log("This would give us perfect proportional distribution in one operation.")
	t.Log("")
	t.Log("Challenge: need to parse and reconstruct the layout string format.")
}

// TestCountPanes tests counting panes in a layout tree
func TestCountPanes(t *testing.T) {
	tests := []struct {
		name     string
		layout   string
		expected int
	}{
		{
			name:     "single pane",
			layout:   "1234,100x50,0,0,1",
			expected: 1,
		},
		{
			name:     "two panes horizontal",
			layout:   "1234,199x53,0,0{99x53,0,0,1,99x53,100,0,2}",
			expected: 2,
		},
		{
			name:     "two panes vertical",
			layout:   "1234,100x100,0,0[100x49,0,0,1,100x49,0,50,2]",
			expected: 2,
		},
		{
			name:     "four panes complex",
			layout:   "b2d9,255x61,0,0{84x61,0,0[84x30,0,0,26,84x30,0,31,41],85x61,85,0,36,84x61,171,0,42}",
			expected: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node, err := ParseLayout(tc.layout)
			if err != nil {
				t.Fatalf("ParseLayout error: %v", err)
			}
			count := countPanes(node)
			if count != tc.expected {
				t.Errorf("countPanes() = %d, expected %d", count, tc.expected)
			}
		})
	}
}

// TestTwoColumnLayout tests the specific case that's broken:
// Two panes side by side (one vertical split creating two columns)
func TestTwoColumnLayout(t *testing.T) {
	// Layout: 2 panes side by side (horizontal split)
	// {pane1, pane2} - using {} means horizontal split
	layout := "1234,199x53,0,0{99x53,0,0,1,99x53,100,0,2}"
	
	t.Logf("Input layout: %s", layout)
	
	node, err := ParseLayout(layout)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	t.Logf("Root: %dx%d, SplitType=%d (1=Horiz, 2=Vert)", 
		node.Width, node.Height, node.SplitType)
	
	if node.SplitType != SplitHorizontal {
		t.Errorf("Root should be SplitHorizontal (1), got %d", node.SplitType)
	}
	
	for i, c := range node.Children {
		t.Logf("  Child %d: %dx%d at (%d,%d), PaneID=%d, SplitType=%d",
			i, c.Width, c.Height, c.X, c.Y, c.PaneID, c.SplitType)
	}
	
	// Apply zoom to pane 1
	zoomed := ApplyZoomToLayout(node, 1, 65)
	
	t.Logf("After zoom:")
	t.Logf("Root: %dx%d, SplitType=%d", zoomed.Width, zoomed.Height, zoomed.SplitType)
	
	if zoomed.SplitType != SplitHorizontal {
		t.Errorf("Root should STILL be SplitHorizontal (1) after zoom, got %d", zoomed.SplitType)
	}
	
	for i, c := range zoomed.Children {
		t.Logf("  Child %d: %dx%d at (%d,%d), PaneID=%d, SplitType=%d",
			i, c.Width, c.Height, c.X, c.Y, c.PaneID, c.SplitType)
	}
	
	rebuilt := BuildLayout(zoomed)
	t.Logf("Rebuilt: %s", rebuilt)
	
	// The rebuilt layout should use {} not []
	if strings.Contains(rebuilt, "[") {
		t.Errorf("BUG: Layout changed from {} to [] (horizontal to vertical)!")
		t.Errorf("Expected horizontal split {}, got vertical split []")
	}
}

package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// DefaultZoomPercent is the default percentage of window size the focused pane should occupy
	DefaultZoomPercent = 65
)

// SplitType represents how a layout node is split
type SplitType int

const (
	SplitNone       SplitType = iota // Leaf pane (no split)
	SplitHorizontal                  // Children arranged side-by-side (uses {})
	SplitVertical                    // Children arranged top-to-bottom (uses [])
)

// LayoutNode represents a node in the tmux layout tree
type LayoutNode struct {
	Width     int
	Height    int
	X         int
	Y         int
	PaneID    int // -1 if not a leaf pane
	SplitType SplitType
	Children  []*LayoutNode
}

// ParseLayout parses a tmux layout string into a tree structure
// Format: checksum,WxH,x,y{children} or checksum,WxH,x,y[children] or checksum,WxH,x,y,paneID
func ParseLayout(layout string) (*LayoutNode, error) {
	// Skip the checksum (first part before comma)
	idx := strings.Index(layout, ",")
	if idx == -1 {
		return nil, fmt.Errorf("invalid layout: no checksum separator")
	}
	rest := layout[idx+1:]

	node, _, err := parseNode(rest)
	return node, err
}

// parseNode parses a single node from the layout string
// Returns the node and the remaining unparsed string
func parseNode(s string) (*LayoutNode, string, error) {
	node := &LayoutNode{PaneID: -1}

	// Parse WxH
	xIdx := strings.Index(s, "x")
	if xIdx == -1 {
		return nil, "", fmt.Errorf("invalid node: no 'x' in dimensions")
	}
	width, err := strconv.Atoi(s[:xIdx])
	if err != nil {
		return nil, "", fmt.Errorf("invalid width: %v", err)
	}
	node.Width = width
	s = s[xIdx+1:]

	// Find end of height (next comma)
	commaIdx := strings.Index(s, ",")
	if commaIdx == -1 {
		return nil, "", fmt.Errorf("invalid node: no comma after height")
	}
	height, err := strconv.Atoi(s[:commaIdx])
	if err != nil {
		return nil, "", fmt.Errorf("invalid height: %v", err)
	}
	node.Height = height
	s = s[commaIdx+1:]

	// Parse x position
	commaIdx = strings.Index(s, ",")
	if commaIdx == -1 {
		return nil, "", fmt.Errorf("invalid node: no comma after x")
	}
	x, err := strconv.Atoi(s[:commaIdx])
	if err != nil {
		return nil, "", fmt.Errorf("invalid x: %v", err)
	}
	node.X = x
	s = s[commaIdx+1:]

	// Parse y position - ends at comma, {, [, or end
	endIdx := len(s)
	for i, c := range s {
		if c == ',' || c == '{' || c == '[' || c == '}' || c == ']' {
			endIdx = i
			break
		}
	}
	y, err := strconv.Atoi(s[:endIdx])
	if err != nil {
		return nil, "", fmt.Errorf("invalid y: %v", err)
	}
	node.Y = y
	s = s[endIdx:]

	// Check what follows
	if len(s) == 0 {
		// End of string - this shouldn't happen for valid layout
		return node, "", nil
	}

	switch s[0] {
	case '{':
		// Horizontal split
		node.SplitType = SplitHorizontal
		children, rest, err := parseChildren(s[1:], '}')
		if err != nil {
			return nil, "", err
		}
		node.Children = children
		return node, rest, nil

	case '[':
		// Vertical split
		node.SplitType = SplitVertical
		children, rest, err := parseChildren(s[1:], ']')
		if err != nil {
			return nil, "", err
		}
		node.Children = children
		return node, rest, nil

	case ',':
		// Leaf pane - pane ID follows
		s = s[1:]
		endIdx := len(s)
		for i, c := range s {
			if c == ',' || c == '}' || c == ']' {
				endIdx = i
				break
			}
		}
		paneID, err := strconv.Atoi(s[:endIdx])
		if err != nil {
			return nil, "", fmt.Errorf("invalid pane ID: %v", err)
		}
		node.PaneID = paneID
		node.SplitType = SplitNone
		return node, s[endIdx:], nil

	case '}', ']':
		// End of parent's children list
		return node, s, nil

	default:
		return nil, "", fmt.Errorf("unexpected character: %c", s[0])
	}
}

// parseChildren parses a comma-separated list of child nodes
func parseChildren(s string, endChar byte) ([]*LayoutNode, string, error) {
	var children []*LayoutNode

	for len(s) > 0 && s[0] != endChar {
		child, rest, err := parseNode(s)
		if err != nil {
			return nil, "", err
		}
		children = append(children, child)
		s = rest

		// Skip comma between children
		if len(s) > 0 && s[0] == ',' {
			s = s[1:]
		}
	}

	// Skip the closing bracket
	if len(s) > 0 && s[0] == endChar {
		s = s[1:]
	}

	return children, s, nil
}

// BuildLayout builds a tmux layout string from a tree structure
func BuildLayout(node *LayoutNode) string {
	body := buildNodeString(node)
	checksum := calculateChecksum(body)
	return fmt.Sprintf("%s,%s", checksum, body)
}

// buildNodeString builds the layout string for a node (without checksum)
func buildNodeString(node *LayoutNode) string {
	base := fmt.Sprintf("%dx%d,%d,%d", node.Width, node.Height, node.X, node.Y)

	switch node.SplitType {
	case SplitNone:
		return fmt.Sprintf("%s,%d", base, node.PaneID)
	case SplitHorizontal:
		var childStrs []string
		for _, child := range node.Children {
			childStrs = append(childStrs, buildNodeString(child))
		}
		return fmt.Sprintf("%s{%s}", base, strings.Join(childStrs, ","))
	case SplitVertical:
		var childStrs []string
		for _, child := range node.Children {
			childStrs = append(childStrs, buildNodeString(child))
		}
		return fmt.Sprintf("%s[%s]", base, strings.Join(childStrs, ","))
	default:
		return base
	}
}

// calculateChecksum calculates the tmux layout checksum
// tmux uses a rotate-right-and-add algorithm
func calculateChecksum(s string) string {
	var csum uint16 = 0
	for i := 0; i < len(s); i++ {
		csum = (csum >> 1) + ((csum & 1) << 15) // Rotate right
		csum += uint16(s[i])
	}
	return fmt.Sprintf("%04x", csum)
}

// ApplyZoomToLayout modifies a layout tree so the pane with activePaneID
// gets zoomPercent of the available space, while others shrink proportionally.
// Returns a new layout tree (does not modify the input).
func ApplyZoomToLayout(node *LayoutNode, activePaneID int, zoomPercent int) *LayoutNode {
	// Deep copy the tree
	result := copyLayoutNode(node)

	// Find which child contains the active pane
	activeChildIdx := -1
	for i, child := range result.Children {
		if containsPane(child, activePaneID) {
			activeChildIdx = i
			break
		}
	}

	if activeChildIdx == -1 {
		// Active pane not found in children - return unchanged
		return result
	}

	// Apply zoom based on split type
	if result.SplitType == SplitHorizontal {
		applyHorizontalZoom(result, activeChildIdx, zoomPercent)
	} else if result.SplitType == SplitVertical {
		applyVerticalZoom(result, activeChildIdx, zoomPercent)
	}

	return result
}

// containsPane checks if a node or its descendants contain the given pane ID
func containsPane(node *LayoutNode, paneID int) bool {
	if node.PaneID == paneID {
		return true
	}
	for _, child := range node.Children {
		if containsPane(child, paneID) {
			return true
		}
	}
	return false
}

// countPanes returns the number of panes in a layout tree
func countPanes(node *LayoutNode) int {
	if node == nil {
		return 0
	}
	if node.SplitType == SplitNone {
		// Leaf node = 1 pane
		return 1
	}
	count := 0
	for _, child := range node.Children {
		count += countPanes(child)
	}
	return count
}

// copyLayoutNode creates a deep copy of a layout node
func copyLayoutNode(node *LayoutNode) *LayoutNode {
	if node == nil {
		return nil
	}
	result := &LayoutNode{
		Width:     node.Width,
		Height:    node.Height,
		X:         node.X,
		Y:         node.Y,
		PaneID:    node.PaneID,
		SplitType: node.SplitType,
	}
	for _, child := range node.Children {
		result.Children = append(result.Children, copyLayoutNode(child))
	}
	return result
}

// applyHorizontalZoom resizes children of a horizontal split
// Active child gets zoomPercent of width, others shrink proportionally
func applyHorizontalZoom(node *LayoutNode, activeIdx int, zoomPercent int) {
	if len(node.Children) <= 1 {
		return
	}

	// Calculate total available width (without borders)
	borders := len(node.Children) - 1
	availableWidth := node.Width - borders

	// Target width for active child
	targetWidth := (availableWidth * zoomPercent) / 100

	// Total width of other children (for proportional distribution)
	var otherWidth int
	for i, child := range node.Children {
		if i != activeIdx {
			otherWidth += child.Width
		}
	}

	// Remaining width for other children
	remainingWidth := availableWidth - targetWidth

	// Distribute widths
	newWidths := make([]int, len(node.Children))
	widthUsed := 0
	for i, child := range node.Children {
		if i == activeIdx {
			newWidths[i] = targetWidth
		} else if otherWidth > 0 {
			// Proportional share
			newWidths[i] = (child.Width * remainingWidth) / otherWidth
		} else {
			// Fallback: equal distribution
			newWidths[i] = remainingWidth / (len(node.Children) - 1)
		}
		widthUsed += newWidths[i]
	}

	// Adjust for rounding errors - add remainder to active child
	if widthUsed != availableWidth {
		newWidths[activeIdx] += availableWidth - widthUsed
	}

	// Apply new widths and update X positions
	currentX := node.X
	for i, child := range node.Children {
		child.Width = newWidths[i]
		child.X = currentX
		// Recursively update widths of all descendants
		updateChildWidths(child, newWidths[i])
		currentX += newWidths[i] + 1 // +1 for border
	}
}

// applyVerticalZoom resizes children of a vertical split
// Active child gets zoomPercent of height, others shrink proportionally
func applyVerticalZoom(node *LayoutNode, activeIdx int, zoomPercent int) {
	if len(node.Children) <= 1 {
		return
	}

	// Calculate total available height (without borders)
	borders := len(node.Children) - 1
	availableHeight := node.Height - borders

	// Target height for active child
	targetHeight := (availableHeight * zoomPercent) / 100

	// Total height of other children (for proportional distribution)
	var otherHeight int
	for i, child := range node.Children {
		if i != activeIdx {
			otherHeight += child.Height
		}
	}

	// Remaining height for other children
	remainingHeight := availableHeight - targetHeight

	// Distribute heights
	newHeights := make([]int, len(node.Children))
	heightUsed := 0
	for i, child := range node.Children {
		if i == activeIdx {
			newHeights[i] = targetHeight
		} else if otherHeight > 0 {
			// Proportional share
			newHeights[i] = (child.Height * remainingHeight) / otherHeight
		} else {
			// Fallback: equal distribution
			newHeights[i] = remainingHeight / (len(node.Children) - 1)
		}
		heightUsed += newHeights[i]
	}

	// Adjust for rounding errors
	if heightUsed != availableHeight {
		newHeights[activeIdx] += availableHeight - heightUsed
	}

	// Apply new heights and update Y positions
	currentY := node.Y
	for i, child := range node.Children {
		child.Height = newHeights[i]
		child.Y = currentY
		// Recursively update heights of all descendants
		updateChildHeights(child, newHeights[i])
		currentY += newHeights[i] + 1 // +1 for border
	}
}

// updateChildWidths recursively updates width of all children to match parent
func updateChildWidths(node *LayoutNode, width int) {
	node.Width = width
	for _, child := range node.Children {
		if node.SplitType == SplitVertical {
			// Vertical split: all children have same width as parent
			updateChildWidths(child, width)
		}
		// Horizontal split: children keep their own widths (already set)
	}
}

// updateChildHeights recursively updates height of all children to match parent
func updateChildHeights(node *LayoutNode, height int) {
	node.Height = height
	for _, child := range node.Children {
		if node.SplitType == SplitHorizontal {
			// Horizontal split: all children have same height as parent
			updateChildHeights(child, height)
		}
		// Vertical split: children keep their own heights (already set)
	}
}

// CaptureSnapshot takes a snapshot of the current layout
func CaptureSnapshot() (*State, error) {
	session, err := GetCurrentSession()
	if err != nil {
		return nil, err
	}

	window, err := GetCurrentWindow()
	if err != nil {
		return nil, err
	}

	layout, err := GetWindowLayout()
	if err != nil {
		return nil, err
	}

	return &State{
		Enabled:  true,
		Session:  session,
		Window:   window,
		Snapshot: layout,
	}, nil
}

// RestoreSnapshot restores the saved layout
func RestoreSnapshot(state *State) error {
	if state.Snapshot == "" {
		return nil
	}
	return SelectLayout(state.Snapshot)
}

// ApplyZoom restores snapshot and enlarges the focused pane proportionally
func ApplyZoom(state *State) error {
	// Check pane count - skip if only 1 pane
	paneCount, err := GetPaneCount()
	if err != nil {
		return err
	}
	if paneCount <= 1 {
		return nil
	}

	// Check if we're in the same session/window as the snapshot
	session, err := GetCurrentSession()
	if err != nil {
		return err
	}
	window, err := GetCurrentWindow()
	if err != nil {
		return err
	}

	// Different window - don't apply zoom
	if session != state.Session || window != state.Window {
		return nil
	}

	// Get active pane ID before restoring snapshot
	activePaneID, err := GetActivePaneID()
	if err != nil {
		return err
	}
	debugf("Active pane ID: %d", activePaneID)

	// Parse the snapshot layout
	layoutTree, err := ParseLayout(state.Snapshot)
	if err != nil {
		debugf("Failed to parse layout: %v", err)
		// Fall back to old approach
		if err := RestoreSnapshot(state); err != nil {
			_ = ClearState()
			_ = DisplayMessage("Focus zoom: disabled (layout changed)")
			return err
		}
		return applyZoomFallback(state, activePaneID)
	}

	// Check if pane count has changed (pane was closed or opened)
	snapshotPaneCount := countPanes(layoutTree)
	if snapshotPaneCount != paneCount {
		debugf("Pane count changed: snapshot=%d, current=%d - updating snapshot", snapshotPaneCount, paneCount)
		// Capture fresh snapshot with current layout
		newState, err := CaptureSnapshot()
		if err != nil {
			debugf("Failed to capture new snapshot: %v", err)
			return err
		}
		// Save the updated state
		if err := SaveState(newState); err != nil {
			debugf("Failed to save updated state: %v", err)
			return err
		}
		// Update our working state and re-parse
		state.Snapshot = newState.Snapshot
		layoutTree, err = ParseLayout(state.Snapshot)
		if err != nil {
			debugf("Failed to parse new layout: %v", err)
			return err
		}
	}

	// Get configured zoom percentage
	zoomPercent := GetZoomPercent()

	// Apply zoom to the layout tree (horizontal zoom at root level)
	zoomedTree := ApplyZoomToLayout(layoutTree, activePaneID, zoomPercent)

	// Also apply vertical zoom if the active pane is in a column with multiple panes
	applyNestedZoom(zoomedTree, activePaneID, zoomPercent)

	// Build and apply the new layout
	newLayout := BuildLayout(zoomedTree)
	debugf("Applying zoomed layout: %s", newLayout)

	if err := SelectLayout(newLayout); err != nil {
		debugf("Failed to apply layout: %v", err)
		// Fall back to restoring snapshot
		_ = RestoreSnapshot(state)
		return err
	}

	return nil
}

// applyNestedZoom recursively applies zoom to nested splits containing the active pane
func applyNestedZoom(node *LayoutNode, activePaneID int, zoomPercent int) {
	// Find the child that contains the active pane
	for i, child := range node.Children {
		if containsPane(child, activePaneID) {
			// If this child has children (is a split), apply zoom to it
			if len(child.Children) > 1 {
				// Find which grandchild contains the active pane
				activeGrandchildIdx := -1
				for j, grandchild := range child.Children {
					if containsPane(grandchild, activePaneID) {
						activeGrandchildIdx = j
						break
					}
				}

				if activeGrandchildIdx >= 0 {
					// Apply zoom based on the child's split type
					if child.SplitType == SplitHorizontal {
						applyHorizontalZoom(child, activeGrandchildIdx, zoomPercent)
					} else if child.SplitType == SplitVertical {
						applyVerticalZoom(child, activeGrandchildIdx, zoomPercent)
					}

					// Recursively apply to deeper levels
					applyNestedZoom(child, activePaneID, zoomPercent)
				}
			}
			// Update this child's reference in parent
			node.Children[i] = child
			return
		}
	}
}

// applyZoomFallback uses the old resize-pane approach as a fallback
func applyZoomFallback(state *State, activePaneID int) error {
	debugf("Using fallback zoom approach")

	panes, err := GetPanes()
	if err != nil {
		return err
	}

	winWidth, winHeight, err := GetWindowSize()
	if err != nil {
		return err
	}

	// Find active pane
	var activePane *PaneInfo
	for i := range panes {
		if panes[i].Active {
			activePane = &panes[i]
			break
		}
	}
	if activePane == nil {
		return nil
	}

	// Apply old proportional zoom logic
	zoomPercent := GetZoomPercent()
	applyProportionalZoom(panes, activePane, winWidth, winHeight, zoomPercent)
	return nil
}

// applyProportionalZoom resizes all panes so focused gets zoomPercent, others shrink proportionally
func applyProportionalZoom(panes []PaneInfo, active *PaneInfo, winWidth, winHeight, zoomPercent int) {
	debugf("=== applyProportionalZoom ===")
	debugf("Active pane: %s (index=%d) at (%d,%d) size=%dx%d",
		active.ID, active.Index, active.Left, active.Top, active.Width, active.Height)
	debugf("Window size: %dx%d", winWidth, winHeight)

	// Find unique columns (by left position) and their widths
	columns := findColumns(panes)
	debugf("Found %d columns:", len(columns))
	for i, col := range columns {
		debugf("  col[%d]: left=%d, width=%d, panes=%d", i, col.left, col.width, len(col.panes))
	}

	// Calculate target width for focused pane's column
	targetWidth := (winWidth * zoomPercent) / 100
	debugf("Target width for active column: %d (%d%% of %d)", targetWidth, zoomPercent, winWidth)

	// Find which column the active pane is in
	activeColIdx := -1
	var activeColumn *column
	for i, col := range columns {
		if col.left == active.Left {
			activeColIdx = i
			activeColumn = &columns[i]
			break
		}
	}
	debugf("Active column index: %d", activeColIdx)

	// Resize columns proportionally (always do this if multiple columns)
	if len(columns) > 1 && activeColIdx >= 0 {
		resizeColumnsProportionally(panes, columns, activeColIdx, targetWidth, winWidth)
	}

	// Only do row resizing if there are multiple panes in the active column
	// (i.e., the column is vertically split)
	if activeColumn != nil && len(activeColumn.panes) > 1 {
		// Find rows ONLY within the active column
		rows := findRowsInColumn(activeColumn.panes)

		// Find which row the active pane is in
		activeRowIdx := -1
		for i, row := range rows {
			if row.top == active.Top {
				activeRowIdx = i
				break
			}
		}

		// Get the column's total height (use the tallest pane or sum of panes)
		colHeight := 0
		for _, p := range activeColumn.panes {
			colHeight += p.Height
		}
		// Add borders between rows
		colHeight += len(rows) - 1

		targetHeight := (colHeight * zoomPercent) / 100

		if len(rows) > 1 && activeRowIdx >= 0 {
			resizeRowsProportionally(panes, rows, activeRowIdx, targetHeight, colHeight)
		}
	}
}

type column struct {
	left  int
	width int
	panes []PaneInfo
}

type row struct {
	top    int
	height int
	panes  []PaneInfo
}

// findColumns groups panes by their left position (column)
func findColumns(panes []PaneInfo) []column {
	colMap := make(map[int]*column)

	for _, p := range panes {
		if col, exists := colMap[p.Left]; exists {
			col.panes = append(col.panes, p)
			// Column width is the width of any pane in it
			if p.Width > col.width {
				col.width = p.Width
			}
		} else {
			colMap[p.Left] = &column{
				left:  p.Left,
				width: p.Width,
				panes: []PaneInfo{p},
			}
		}
	}

	// Convert to slice and sort by left position
	var cols []column
	for _, col := range colMap {
		cols = append(cols, *col)
	}
	sort.Slice(cols, func(i, j int) bool {
		return cols[i].left < cols[j].left
	})

	return cols
}

// findRowsInColumn groups panes within a single column by their top position
func findRowsInColumn(columnPanes []PaneInfo) []row {
	rowMap := make(map[int]*row)

	for _, p := range columnPanes {
		if r, exists := rowMap[p.Top]; exists {
			r.panes = append(r.panes, p)
			if p.Height > r.height {
				r.height = p.Height
			}
		} else {
			rowMap[p.Top] = &row{
				top:    p.Top,
				height: p.Height,
				panes:  []PaneInfo{p},
			}
		}
	}

	var rows []row
	for _, r := range rowMap {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].top < rows[j].top
	})

	return rows
}

// resizeColumnsProportionally grows active column - tmux handles shrinking others
func resizeColumnsProportionally(panes []PaneInfo, columns []column, activeIdx int, targetWidth, winWidth int) {
	debugf("=== resizeColumnsProportionally ===")
	debugf("activeIdx=%d, targetWidth=%d, winWidth=%d", activeIdx, targetWidth, winWidth)

	if len(columns) <= 1 {
		debugf("Only 1 column, skipping")
		return
	}

	// Only resize the active column's WIDTH - tmux will redistribute space from neighbors
	if len(columns[activeIdx].panes) > 0 {
		p := columns[activeIdx].panes[0]
		debugf("Resize active col[%d] pane %s to width=%d", activeIdx, p.ID, targetWidth)
		_ = ResizePaneWidth(p.ID, targetWidth)
	}
}

// resizeRowsProportionally grows active row - tmux handles shrinking others
func resizeRowsProportionally(panes []PaneInfo, rows []row, activeIdx int, targetHeight, winHeight int) {
	debugf("=== resizeRowsProportionally ===")
	debugf("activeIdx=%d, targetHeight=%d, winHeight=%d", activeIdx, targetHeight, winHeight)

	if len(rows) <= 1 {
		debugf("Only 1 row, skipping")
		return
	}

	// Only resize the active row's HEIGHT - don't touch width!
	if len(rows[activeIdx].panes) > 0 {
		p := rows[activeIdx].panes[0]
		debugf("Resize active row[%d] pane %s to height=%d", activeIdx, p.ID, targetHeight)
		_ = ResizePaneHeight(p.ID, targetHeight)
	}
}

// IsMatchingWindow checks if current session/window matches state
func IsMatchingWindow(state *State) (bool, error) {
	session, err := GetCurrentSession()
	if err != nil {
		return false, err
	}
	window, err := GetCurrentWindow()
	if err != nil {
		return false, err
	}
	return session == state.Session && window == state.Window, nil
}

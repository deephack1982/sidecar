package filebrowser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// ansiResetRe matches ANSI reset sequences (both \x1b[0m and \x1b[m)
var ansiResetRe = regexp.MustCompile(`\x1b\[0?m`)

// selectionBgANSI is the ANSI 24-bit background code for selection highlight (#374151)
const selectionBgANSI = "\x1b[48;2;55;65;81m"

// injectSelectionBackground injects selection background color into an ANSI-styled string.
// It prepends the background at the start and re-injects after any reset sequences.
func injectSelectionBackground(s string) string {
	// Prepend background color
	result := selectionBgANSI + s
	// Re-inject background after any reset sequences
	result = ansiResetRe.ReplaceAllString(result, "${0}"+selectionBgANSI)
	// Append reset at end to not bleed into next line
	return result + "\x1b[0m"
}

// FocusPane represents which pane is active.
type FocusPane int

const (
	PaneTree FocusPane = iota
	PanePreview
)

// dividerWidth is the width of the draggable divider between panes.
const dividerWidth = 1

// calculatePaneWidths sets the tree and preview pane widths.
// If treeWidth is already set (from drag), only updates previewWidth.
func (p *Plugin) calculatePaneWidths() {
	// Original: available = width - 6 (4 for borders + 2 margin)
	// With divider: subtract 1 more for the divider character
	available := p.width - 6 - dividerWidth

	// Only set default treeWidth if not yet initialized
	if p.treeWidth == 0 {
		p.treeWidth = available * 30 / 100
	}

	// Clamp treeWidth to valid bounds
	minWidth := 20
	maxWidth := available - 40 // Leave at least 40 for preview
	if maxWidth < minWidth {
		maxWidth = minWidth
	}
	if p.treeWidth < minWidth {
		p.treeWidth = minWidth
	} else if p.treeWidth > maxWidth {
		p.treeWidth = maxWidth
	}

	// Calculate previewWidth from remaining space
	p.previewWidth = available - p.treeWidth
	if p.previewWidth < 40 {
		p.previewWidth = 40
	}
}

// renderView creates the 2-pane layout.
func (p *Plugin) renderView() string {
	// Clear mouse hit regions at start of each render
	p.mouseHandler.Clear()

	// Project search is a full overlay - render modal over dimmed background
	if p.projectSearchMode {
		background := p.renderNormalPanes()
		modal := p.renderProjectSearchModalContent()
		return ui.OverlayModal(background, modal, p.width, p.height)
	}

	// Quick open is a full overlay - render modal over dimmed background
	if p.quickOpenMode {
		background := p.renderNormalPanes()
		modal := p.renderQuickOpenModalContent()
		return ui.OverlayModal(background, modal, p.width, p.height)
	}

	// Info modal is a full overlay - render modal over dimmed background
	if p.infoMode {
		background := p.renderNormalPanes()
		modal := p.renderInfoModalContent()
		return ui.OverlayModal(background, modal, p.width, p.height)
	}

	// File operation modals - render modal over dimmed background
	if p.fileOpMode != FileOpNone {
		background := p.renderNormalPanes()
		var modal string
		switch p.fileOpMode {
		case FileOpRename:
			modal = p.renderRenameModal()
		case FileOpDelete:
			modal = p.renderDeleteModal()
		case FileOpMove:
			if p.fileOpConfirmCreate {
				modal = p.renderCreateDirConfirmModal()
			} else {
				modal = p.renderMoveModal()
			}
		case FileOpCreateDir:
			modal = p.renderMkdirModal()
		case FileOpCreateFile:
			modal = p.renderCreateFileModal()
		}
		return ui.OverlayModal(background, modal, p.width, p.height)
	}

	return p.renderNormalPanes()
}

// renderNormalPanes renders the standard 2-pane layout without modals.
func (p *Plugin) renderNormalPanes() string {

	p.calculatePaneWidths()

	// Determine border styles based on focus
	treeBorder := styles.PanelInactive
	previewBorder := styles.PanelInactive
	if p.activePane == PaneTree && !p.searchMode && !p.contentSearchMode {
		treeBorder = styles.PanelActive
	} else if p.activePane == PanePreview && !p.searchMode && !p.contentSearchMode {
		previewBorder = styles.PanelActive
	}

	// Account for input bar if active (content search only)
	// Note: tree search bar is rendered inside the tree pane, not here
	// Note: file op uses modals now, not inline bar
	inputBarHeight := 0
	if p.contentSearchMode {
		inputBarHeight = 1
	}

	// Calculate pane height for content (excluding borders)
	// Note: footer is rendered by the app, not by the plugin
	paneHeight := p.height - inputBarHeight - 2
	if paneHeight < 4 {
		paneHeight = 4
	}

	// Inner content height = pane height - header lines (2)
	innerHeight := paneHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	treeContent := p.renderTreePane(innerHeight)
	previewContent := p.renderPreviewPane(innerHeight)

	// Apply styles - border adds 2 lines to visual height
	leftPane := treeBorder.
		Width(p.treeWidth).
		Height(paneHeight).
		Render(treeContent)

	rightPane := previewBorder.
		Width(p.previewWidth).
		Height(paneHeight).
		Render(previewContent)

	// Render visible divider between panes
	// MarginTop(1) in renderDivider shifts it down, so use paneHeight directly
	divider := p.renderDivider(paneHeight)

	// Join panes horizontally with divider in between
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, divider, rightPane)

	// Build final layout
	var parts []string

	// Note: tree search bar is rendered inside renderTreePane(), not here

	// Add content search bar if in content search mode
	if p.contentSearchMode {
		parts = append(parts, p.renderContentSearchBar())
	}

	// Note: file operations now use modals (see renderView), not inline bar

	parts = append(parts, panes)

	// Register mouse hit regions for panes
	// Panes start after any input bars
	paneY := inputBarHeight
	treeItemY := paneY + 3 // border(1) + header(2)

	// Register pane regions - tested in reverse order (last added = highest priority)
	// Tree pane region (x=0, full width) - lowest priority fallback
	p.mouseHandler.HitMap.AddRect(regionTreePane, 0, paneY, p.treeWidth, paneHeight, nil)

	// Preview pane region (after divider) - medium priority
	previewX := p.treeWidth + dividerWidth
	p.mouseHandler.HitMap.AddRect(regionPreviewPane, previewX, paneY, p.previewWidth, paneHeight, nil)

	// Pane divider region - HIGH PRIORITY (registered after panes so it wins in overlap)
	// Left pane is Width(treeWidth), so occupies columns 0 to treeWidth-1
	// Divider is at column treeWidth
	// Hit region is wider for easier clicking
	dividerX := p.treeWidth
	dividerHitWidth := 3
	p.mouseHandler.HitMap.AddRect(regionPaneDivider, dividerX, paneY, dividerHitWidth, paneHeight, nil)

	// Register individual tree items LAST (tested first = higher priority)
	// Note: regions are tested in reverse order, so items added last take precedence
	if p.tree != nil && p.tree.Len() > 0 {
		end := p.treeScrollOff + innerHeight
		if end > p.tree.Len() {
			end = p.tree.Len()
		}
		for i := p.treeScrollOff; i < end; i++ {
			itemY := treeItemY + (i - p.treeScrollOff)
			// Register region: x=1 (inside border), width=treeWidth-2, height=1, data=tree index
			p.mouseHandler.HitMap.AddRect(regionTreeItem, 1, itemY, p.treeWidth-2, 1, i)
		}
	}

	// Register individual preview lines for text selection (LAST for highest priority)
	if p.previewFile != "" && !p.isBinary && len(p.previewLines) > 0 {
		previewContentStartY := paneY + 3 // border(1) + header(2 lines: title + metadata)
		contentStart := p.previewScroll
		contentEnd := contentStart + innerHeight
		if contentEnd > len(p.previewLines) {
			contentEnd = len(p.previewLines)
		}
		for i := contentStart; i < contentEnd; i++ {
			lineY := previewContentStartY + (i - contentStart)
			// Region covers content area within preview pane
			p.mouseHandler.HitMap.AddRect(regionPreviewLine, previewX+1, lineY, p.previewWidth-2, 1, i)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, parts...)
}

// renderDivider renders the vertical divider between panes.
func (p *Plugin) renderDivider(height int) string {
	// Use a subtle vertical bar as the divider
	// MarginTop(1) shifts it down to align with pane content (below top border)
	dividerStyle := lipgloss.NewStyle().
		Foreground(styles.BorderNormal).
		MarginTop(1)

	// Build vertical bar with exact height
	var sb strings.Builder
	for i := 0; i < height; i++ {
		sb.WriteString("│")
		if i < height-1 {
			sb.WriteString("\n")
		}
	}

	return dividerStyle.Render(sb.String())
}

// renderContentSearchBar renders the content search input bar for preview pane.
func (p *Plugin) renderContentSearchBar() string {
	// Show cursor while typing, hide when committed
	cursor := ""
	if !p.contentSearchCommitted {
		cursor = "█"
	}

	matchInfo := ""
	if len(p.contentSearchMatches) > 0 {
		matchInfo = fmt.Sprintf(" (%d/%d)", p.contentSearchCursor+1, len(p.contentSearchMatches))
		if p.contentSearchCommitted {
			matchInfo += " [n/N j/k]" // Hint for navigation
		}
	} else if p.contentSearchQuery != "" {
		matchInfo = " (0 matches)"
	}

	searchLine := fmt.Sprintf(" / %s%s%s", p.contentSearchQuery, cursor, matchInfo)
	return styles.ModalTitle.Render(searchLine)
}

// renderSearchBar renders the search input bar (legacy - for external use).
func (p *Plugin) renderSearchBar() string {
	return p.renderTreeSearchBar()
}

// renderTreeSearchBar renders the tree search bar inline within the tree pane.
func (p *Plugin) renderTreeSearchBar() string {
	cursor := "█"
	matchInfo := ""
	if len(p.searchMatches) > 0 {
		matchInfo = fmt.Sprintf(" (%d/%d)", p.searchCursor+1, len(p.searchMatches))
	} else if p.searchQuery != "" {
		matchInfo = " (no matches)"
	}

	searchLine := fmt.Sprintf("/%s%s%s", p.searchQuery, cursor, matchInfo)
	// Use a subtle style that fits inside the pane
	return styles.StatusInProgress.Render(searchLine)
}

// renderTreePane renders the file tree in the left pane.
func (p *Plugin) renderTreePane(visibleHeight int) string {
	var sb strings.Builder

	// Header with sort mode indicator
	header := styles.Title.Render("Files")
	sb.WriteString(header)
	if p.tree != nil {
		sb.WriteString("  ")
		sb.WriteString(styles.Muted.Render("[" + p.tree.SortMode.Label() + "]"))
	}
	sb.WriteString("\n")

	// Search bar (if in search mode) - rendered inside the pane like conversations plugin
	if p.searchMode {
		searchLine := p.renderTreeSearchBar()
		sb.WriteString(searchLine)
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n") // Empty line when not searching
	}

	// In search mode, show filtered results instead of full tree
	if p.searchMode {
		if len(p.searchMatches) > 0 {
			return p.renderSearchResults(&sb, visibleHeight)
		} else if p.searchQuery != "" {
			// Show "no matches" when query exists but no results
			sb.WriteString(styles.Muted.Render("No matching files"))
			return sb.String()
		}
		// Empty query - fall through to show full tree
	}

	if p.tree == nil || p.tree.Len() == 0 {
		sb.WriteString(styles.Muted.Render("No files"))
		return sb.String()
	}

	// visibleHeight already accounts for header (2 lines) in renderNormalPanes()
	// So we use it directly - no additional subtraction needed
	end := p.treeScrollOff + visibleHeight
	if end > p.tree.Len() {
		end = p.tree.Len()
	}

	for i := p.treeScrollOff; i < end; i++ {
		node := p.tree.GetNode(i)
		if node == nil {
			continue
		}

		selected := i == p.treeCursor
		maxWidth := p.treeWidth - 4 // Account for border padding
		line := p.renderTreeNode(node, selected, maxWidth)

		sb.WriteString(line)
		// Don't add newline after last line
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderSearchResults renders the filtered search results list.
func (p *Plugin) renderSearchResults(sb *strings.Builder, visibleHeight int) string {
	maxWidth := p.treeWidth - 4

	// Calculate scroll offset for search results
	searchScrollOff := 0
	if p.searchCursor >= visibleHeight {
		searchScrollOff = p.searchCursor - visibleHeight + 1
	}

	end := searchScrollOff + visibleHeight
	if end > len(p.searchMatches) {
		end = len(p.searchMatches)
	}

	for i := searchScrollOff; i < end; i++ {
		node := p.searchMatches[i]
		selected := i == p.searchCursor

		// Show full path for search results (more useful than just name)
		displayPath := node.Path
		if len(displayPath) > maxWidth-2 {
			displayPath = "…" + displayPath[len(displayPath)-maxWidth+3:]
		}

		if selected {
			// Full-width highlight for selected item
			if len(displayPath) < maxWidth {
				displayPath += strings.Repeat(" ", maxWidth-len(displayPath))
			}
			sb.WriteString(styles.ListItemSelected.Render(displayPath))
		} else {
			// Style based on file type
			if node.IsDir {
				sb.WriteString(styles.FileBrowserDir.Render(displayPath))
			} else if node.IsIgnored {
				sb.WriteString(styles.FileBrowserIgnored.Render(displayPath))
			} else {
				sb.WriteString(styles.FileBrowserFile.Render(displayPath))
			}
		}

		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderTreeNode renders a single tree node.
func (p *Plugin) renderTreeNode(node *FileNode, selected bool, maxWidth int) string {
	// Indentation
	indent := strings.Repeat("  ", node.Depth)

	// Icon for directories
	icon := "  "
	if node.IsDir {
		if node.IsExpanded {
			icon = "> "
		} else {
			icon = "+ "
		}
	}

	// Calculate available width for name (after indent and icon)
	prefixLen := len(indent) + len(icon)
	availableWidth := maxWidth - prefixLen
	if availableWidth < 3 {
		availableWidth = 3
	}

	// Truncate name before styling to avoid cutting ANSI escape codes
	displayName := node.Name
	if len(displayName) > availableWidth {
		displayName = displayName[:availableWidth-1] + "…"
	}

	// Name styling
	var name string
	if node.IsDir {
		name = styles.FileBrowserDir.Render(displayName)
	} else if node.IsIgnored {
		name = styles.FileBrowserIgnored.Render(displayName)
	} else {
		name = styles.FileBrowserFile.Render(displayName)
	}

	line := fmt.Sprintf("%s%s%s", indent, styles.FileBrowserIcon.Render(icon), name)

	if selected {
		// Build plain text version for full-width highlight
		plainLine := indent + icon + displayName
		// Pad to full width
		if len(plainLine) < maxWidth {
			plainLine += strings.Repeat(" ", maxWidth-len(plainLine))
		}
		return styles.ListItemSelected.Render(plainLine)
	}
	return line
}

// renderPreviewPane renders the file preview in the right pane.
func (p *Plugin) renderPreviewPane(visibleHeight int) string {
	var sb strings.Builder

	// Header with file path
	header := "Preview"
	if p.previewFile != "" {
		header = truncatePath(p.previewFile, p.previewWidth-4)
		// Add markdown render indicator
		if p.isMarkdownFile() && p.markdownRenderMode {
			header += " [rendered]"
		}
	}
	sb.WriteString(styles.Title.Render(header))

	// Metadata line (size, mod time, permissions)
	if p.previewFile != "" && p.previewSize > 0 {
		meta := fmt.Sprintf("%s  %s  %s",
			formatSize(p.previewSize),
			p.previewModTime.Format("Jan 2 15:04"),
			p.previewMode.String(),
		)
		sb.WriteString("  ")
		sb.WriteString(styles.Muted.Render(meta))
	}
	sb.WriteString("\n\n")

	if p.previewFile == "" {
		sb.WriteString(styles.Muted.Render("Select a file to preview"))
		return sb.String()
	}

	if p.previewError != nil {
		sb.WriteString(styles.StatusDeleted.Render(p.previewError.Error()))
		return sb.String()
	}

	if p.isBinary {
		sb.WriteString(styles.Muted.Render("Binary file"))
		return sb.String()
	}

	// Determine which lines to display
	var lines []string
	showLineNumbers := true

	// Use markdown-rendered lines if in render mode for markdown files
	if p.markdownRenderMode && p.isMarkdownFile() && len(p.markdownRendered) > 0 {
		lines = p.markdownRendered
		showLineNumbers = false // Glamour output doesn't map 1:1 to source lines
	} else if len(p.previewHighlighted) > 0 {
		lines = p.previewHighlighted
	} else {
		lines = p.previewLines
	}

	start := p.previewScroll
	end := start + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	// Calculate max line width (pane width - line number - padding)
	lineNumWidth := 5 // "1234 " = 5 chars
	if !showLineNumbers {
		lineNumWidth = 0
	}
	maxLineWidth := p.previewWidth - lineNumWidth - 4
	if maxLineWidth < 10 {
		maxLineWidth = 10
	}

	// Style for truncating lines with ANSI codes
	lineStyle := lipgloss.NewStyle().MaxWidth(maxLineWidth)

	// Reserve 1 line for truncation message if needed
	contentEnd := end
	if p.isTruncated && end-start > 1 {
		contentEnd = end - 1
	}

	for i := start; i < contentEnd; i++ {
		// Check if this line is selected for text selection highlighting
		if p.isLineSelected(i) && showLineNumbers {
			// Line number with selection background
			lineNumStr := fmt.Sprintf("%4d ", i+1)
			sb.WriteString(injectSelectionBackground(lineNumStr))

			// Get syntax-highlighted content and inject selection background
			var lineContent string
			if i < len(lines) {
				lineContent = lines[i]
			}
			// Truncate using lipgloss (handles ANSI codes properly)
			lineContent = lipgloss.NewStyle().MaxWidth(maxLineWidth).Render(lineContent)
			sb.WriteString(injectSelectionBackground(lineContent))

			// Pad remaining width with selection background
			contentWidth := lipgloss.Width(lineNumStr) + lipgloss.Width(lineContent)
			if contentWidth < p.previewWidth-4 {
				padding := strings.Repeat(" ", p.previewWidth-4-contentWidth)
				sb.WriteString(injectSelectionBackground(padding))
			}
		} else {
			// Get line content
			var lineContent string
			if p.contentSearchMode && len(p.contentSearchMatches) > 0 && showLineNumbers {
				// Use raw lines for highlighting (loses syntax highlighting on matched lines)
				lineContent = p.highlightLineMatches(i)
			} else if i < len(lines) {
				lineContent = lines[i]
			}

			line := lineStyle.Render(lineContent)

			// Render with or without line numbers
			if showLineNumbers {
				lineNum := styles.FileBrowserLineNumber.Render(fmt.Sprintf("%4d ", i+1))
				sb.WriteString(lineNum)
			}
			sb.WriteString(line)
		}

		// Don't add newline after last line
		if i < contentEnd-1 || p.isTruncated {
			sb.WriteString("\n")
		}
	}

	if p.isTruncated {
		sb.WriteString(styles.Muted.Render("... (file truncated)"))
	}

	return sb.String()
}

// highlightLineMatches applies search match highlighting to a line.
func (p *Plugin) highlightLineMatches(lineNo int) string {
	// Get raw line (not syntax highlighted)
	if lineNo >= len(p.previewLines) {
		return ""
	}
	rawLine := p.previewLines[lineNo]

	// Find all matches on this line
	type lineMatch struct {
		matchIdx int // Index in contentSearchMatches (for current detection)
		startCol int
		endCol   int
	}
	var lineMatches []lineMatch

	for i, m := range p.contentSearchMatches {
		if m.LineNo == lineNo {
			lineMatches = append(lineMatches, lineMatch{
				matchIdx: i,
				startCol: m.StartCol,
				endCol:   m.EndCol,
			})
		}
	}

	if len(lineMatches) == 0 {
		// No matches on this line, use syntax highlighted version if available
		if lineNo < len(p.previewHighlighted) {
			return p.previewHighlighted[lineNo]
		}
		return rawLine
	}

	// Build highlighted line from raw text
	var result strings.Builder
	lastEnd := 0

	for _, m := range lineMatches {
		if m.startCol > len(rawLine) || m.endCol > len(rawLine) {
			continue
		}
		if m.startCol < lastEnd {
			continue // Overlapping match, skip
		}

		// Add text before match
		if m.startCol > lastEnd {
			result.WriteString(rawLine[lastEnd:m.startCol])
		}

		// Apply highlight style (current match vs other matches)
		matchText := rawLine[m.startCol:m.endCol]
		if m.matchIdx == p.contentSearchCursor {
			result.WriteString(styles.SearchMatchCurrent.Render(matchText))
		} else {
			result.WriteString(styles.SearchMatch.Render(matchText))
		}
		lastEnd = m.endCol
	}

	// Add remaining text
	if lastEnd < len(rawLine) {
		result.WriteString(rawLine[lastEnd:])
	}

	return result.String()
}

// truncatePath shortens a path to fit width.
func truncatePath(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}
	if maxWidth < 10 {
		return path[:maxWidth]
	}
	// Show ...end of path
	return "..." + path[len(path)-maxWidth+3:]
}

// formatSize formats a file size in human-readable form.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// renderQuickOpenModalContent renders the quick open modal box content.
func (p *Plugin) renderQuickOpenModalContent() string {
	// Modal dimensions
	modalWidth := p.width - 4
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	// Calculate max visible items based on available height
	// Leave room for: header (2 lines), footer (2 lines), border (2 lines), some padding
	maxListHeight := p.height - 8
	if maxListHeight < 5 {
		maxListHeight = 5
	}
	if maxListHeight > 20 {
		maxListHeight = 20
	}

	var sb strings.Builder

	// Header with search input
	cursor := "█"
	header := fmt.Sprintf("Quick Open: %s%s", p.quickOpenQuery, cursor)
	sb.WriteString(styles.ModalTitle.Render(header))
	sb.WriteString("\n\n")

	// Error message if scan was limited
	if p.quickOpenError != "" {
		sb.WriteString(styles.Muted.Render("⚠ " + p.quickOpenError))
		sb.WriteString("\n")
	}

	// Calculate modal position for hit region registration
	hPad := (p.width - modalWidth - 4) / 2
	if hPad < 0 {
		hPad = 0
	}
	modalX := hPad + 1 // +1 for modal border
	modalItemY := 2 + 3 // paddingTop(2) + border(1) + header(2)
	if p.quickOpenError != "" {
		modalItemY++ // Extra line for error message
	}

	if len(p.quickOpenMatches) == 0 {
		if p.quickOpenQuery != "" {
			sb.WriteString(styles.Muted.Render("No matches"))
		} else {
			sb.WriteString(styles.Muted.Render("Type to search files..."))
		}
	} else {
		// Determine visible range (scroll if cursor out of view)
		listHeight := maxListHeight
		if listHeight > len(p.quickOpenMatches) {
			listHeight = len(p.quickOpenMatches)
		}

		start := 0
		if p.quickOpenCursor >= listHeight {
			start = p.quickOpenCursor - listHeight + 1
		}
		end := start + listHeight
		if end > len(p.quickOpenMatches) {
			end = len(p.quickOpenMatches)
		}

		for i := start; i < end; i++ {
			match := p.quickOpenMatches[i]
			isSelected := i == p.quickOpenCursor

			// Register hit region for this quick open item
			itemY := modalItemY + (i - start)
			p.mouseHandler.HitMap.AddRect(regionQuickOpen, modalX, itemY, modalWidth-2, 1, i)

			// Build the display line with highlighted match chars
			line := p.renderQuickOpenMatch(match, modalWidth-4)

			if isSelected {
				sb.WriteString(styles.QuickOpenItemSelected.Render("> " + line))
			} else {
				sb.WriteString(styles.QuickOpenItem.Render("  " + line))
			}

			if i < end-1 {
				sb.WriteString("\n")
			}
		}
	}

	// Footer with match count
	if len(p.quickOpenMatches) > 0 {
		sb.WriteString(fmt.Sprintf("\n\n%s", styles.Muted.Render(fmt.Sprintf("(%d/%d)", p.quickOpenCursor+1, len(p.quickOpenMatches)))))
	} else if len(p.quickOpenFiles) > 0 {
		sb.WriteString(fmt.Sprintf("\n\n%s", styles.Muted.Render(fmt.Sprintf("(%d files)", len(p.quickOpenFiles)))))
	}

	// Wrap in modal box (centering handled by overlayModal)
	content := sb.String()
	return styles.ModalBox.
		Width(modalWidth).
		Render(content)
}

// renderQuickOpenMatch renders a single match with highlighted chars.
func (p *Plugin) renderQuickOpenMatch(match QuickOpenMatch, maxWidth int) string {
	path := match.Path

	// Truncate path if too long
	if len(path) > maxWidth {
		path = "..." + path[len(path)-maxWidth+3:]
		// Can't highlight properly after truncation, just return
		return path
	}

	// Apply match highlighting
	if len(match.MatchRanges) > 0 {
		return p.highlightFuzzyMatch(path, match.MatchRanges)
	}

	return path
}

// highlightFuzzyMatch applies highlighting to matched character ranges.
func (p *Plugin) highlightFuzzyMatch(text string, ranges []MatchRange) string {
	if len(ranges) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, r := range ranges {
		if r.Start > len(text) || r.End > len(text) {
			continue
		}
		if r.Start < lastEnd {
			continue // Skip overlapping
		}

		// Add text before match
		if r.Start > lastEnd {
			result.WriteString(text[lastEnd:r.Start])
		}

		// Add highlighted match
		result.WriteString(styles.FuzzyMatchChar.Render(text[r.Start:r.End]))
		lastEnd = r.End
	}

	// Add remaining text
	if lastEnd < len(text) {
		result.WriteString(text[lastEnd:])
	}

	return result.String()
}

// ============================================================================
// File Operation Modals
// ============================================================================

// registerFileOpButtonHitRegions registers mouse hit regions for modal buttons.
// contentLineCount is the number of lines in the modal content (before ModalBox wrapping).
// modalContentWidth is the content width passed to ModalBox.Width().
// confirmLabel and cancelLabel are the button text (e.g., " Confirm ", " Cancel ").
func (p *Plugin) registerFileOpButtonHitRegions(contentLineCount, modalContentWidth int, confirmLabel, cancelLabel string) {
	// ModalBox adds: border(1) + padding(1) vertically on each side
	// Total modal height = contentLineCount + 4 (top border + top padding + bottom padding + bottom border)
	modalHeight := contentLineCount + 4

	// ModalBox adds: border(1) + padding(2) horizontally on each side = 6 total
	actualModalWidth := modalContentWidth + 6

	// Calculate modal position (same as ui.OverlayModal)
	startX := (p.width - actualModalWidth) / 2
	startY := (p.height - modalHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Button Y position: startY + border(1) + padding(1) + last content line index + 1
	// Buttons are on the last line of content (index = contentLineCount - 1)
	// The +1 accounts for 0-based vs 1-based indexing in the rendered output
	buttonY := startY + 2 + (contentLineCount - 1) + 1

	// Button X positions: startX + border(1) + padding(2) = content start
	contentX := startX + 3

	// styles.Button adds Padding(0, 2) = 4 extra chars (2 left + 2 right)
	// So rendered button width = len(label) + 4
	confirmRenderedWidth := len(confirmLabel) + 4
	p.mouseHandler.HitMap.AddRect(regionFileOpConfirm, contentX, buttonY, confirmRenderedWidth, 1, nil)

	// Cancel button after confirm + gap(2 spaces between buttons)
	cancelX := contentX + confirmRenderedWidth + 2
	cancelRenderedWidth := len(cancelLabel) + 4
	p.mouseHandler.HitMap.AddRect(regionFileOpCancel, cancelX, buttonY, cancelRenderedWidth, 1, nil)
}

// renderRenameModal renders the rename modal for file/directory renaming.
func (p *Plugin) renderRenameModal() string {
	modalWidth := 50

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Rename"))
	sb.WriteString("\n\n")

	// Current name
	currentName := ""
	if p.fileOpTarget != nil {
		currentName = p.fileOpTarget.Name
	}
	sb.WriteString(styles.Muted.Render("Current: "))
	sb.WriteString(currentName)
	sb.WriteString("\n\n")

	// Input field
	sb.WriteString("New name: ")
	sb.WriteString(p.fileOpTextInput.View())

	// Track line count for hit region calculation
	lineCount := 5 // title + blank + current + blank + input

	// Error message if present
	if p.fileOpError != "" {
		sb.WriteString("\n\n")
		sb.WriteString(styles.StatusDeleted.Render(p.fileOpError))
		lineCount += 2 // blank + error
	}

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Confirm ", " Cancel ", p.fileOpButtonFocus, p.fileOpButtonHover))
	lineCount += 2 // blank + buttons

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Confirm ", " Cancel ")

	return styles.ModalBox.Width(modalWidth).Render(sb.String())
}

// renderDeleteModal renders the delete confirmation modal.
func (p *Plugin) renderDeleteModal() string {
	modalWidth := 50

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Delete"))
	sb.WriteString("\n\n")

	// Warning text
	sb.WriteString("Are you sure you want to\n")
	sb.WriteString("delete this ")

	itemType := "file"
	icon := "📄"
	if p.fileOpTarget != nil && p.fileOpTarget.IsDir {
		itemType = "directory"
		icon = "📁"
	}
	sb.WriteString(itemType)
	sb.WriteString("?")
	sb.WriteString("\n\n")

	// Item being deleted
	name := ""
	if p.fileOpTarget != nil {
		name = p.fileOpTarget.Name
	}
	sb.WriteString(icon + " ")
	sb.WriteString(name)

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Delete ", " Cancel ", p.fileOpButtonFocus, p.fileOpButtonHover))

	// Line count: title + blank + warning(2 lines) + blank + item + blank + buttons = 8
	lineCount := 8

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Delete ", " Cancel ")

	// Use a red border for delete warning
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Error).
		Padding(1, 2).
		Width(modalWidth).
		Render(sb.String())
}

// renderMoveModal renders the move modal for file/directory moving.
func (p *Plugin) renderMoveModal() string {
	modalWidth := 60 // Wider for paths

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Move"))
	sb.WriteString("\n\n")

	// Current name
	currentName := ""
	if p.fileOpTarget != nil {
		currentName = p.fileOpTarget.Name
	}
	sb.WriteString(styles.Muted.Render("Moving: "))
	sb.WriteString(currentName)
	sb.WriteString("\n\n")

	// Input field
	sb.WriteString("To: ")
	sb.WriteString(p.fileOpTextInput.View())

	// Track line count: title + blank + moving + blank + input = 5
	lineCount := 5

	// Path suggestions dropdown
	if p.fileOpShowSuggestions && len(p.fileOpSuggestions) > 0 {
		sb.WriteString("\n")
		for i, suggestion := range p.fileOpSuggestions {
			if i == p.fileOpSuggestionIdx {
				sb.WriteString(styles.ListItemSelected.Render("  → " + suggestion))
			} else {
				sb.WriteString(styles.Muted.Render("    " + suggestion))
			}
			if i < len(p.fileOpSuggestions)-1 {
				sb.WriteString("\n")
			}
		}
		lineCount += len(p.fileOpSuggestions)
	}

	// Error message if present
	if p.fileOpError != "" {
		sb.WriteString("\n\n")
		sb.WriteString(styles.StatusDeleted.Render(p.fileOpError))
		lineCount += 2 // blank + error
	}

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Confirm ", " Cancel ", p.fileOpButtonFocus, p.fileOpButtonHover))
	lineCount += 2 // blank + buttons

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Confirm ", " Cancel ")

	return styles.ModalBox.Width(modalWidth).Render(sb.String())
}

// renderMkdirModal renders the create directory modal.
func (p *Plugin) renderMkdirModal() string {
	modalWidth := 50

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Create Directory"))
	sb.WriteString("\n\n")

	// Parent directory
	parentPath := ""
	if p.fileOpTarget != nil {
		if p.fileOpTarget.IsDir {
			parentPath = p.fileOpTarget.Path
		} else if p.fileOpTarget.Parent != nil {
			parentPath = p.fileOpTarget.Parent.Path
		}
	}
	if parentPath == "" {
		parentPath = "/"
	}
	sb.WriteString(styles.Muted.Render("Parent: "))
	sb.WriteString(truncatePath(parentPath, modalWidth-12))
	sb.WriteString("\n\n")

	// Input field
	sb.WriteString("Name: ")
	sb.WriteString(p.fileOpTextInput.View())

	// Track line count: title + blank + parent + blank + name = 5
	lineCount := 5

	// Error message if present
	if p.fileOpError != "" {
		sb.WriteString("\n\n")
		sb.WriteString(styles.StatusDeleted.Render(p.fileOpError))
		lineCount += 2 // blank + error
	}

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Create ", " Cancel ", p.fileOpButtonFocus, p.fileOpButtonHover))
	lineCount += 2 // blank + buttons

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Create ", " Cancel ")

	return styles.ModalBox.Width(modalWidth).Render(sb.String())
}

// renderCreateFileModal renders the create file modal.
func (p *Plugin) renderCreateFileModal() string {
	modalWidth := 50

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Create File"))
	sb.WriteString("\n\n")

	// Parent directory
	parentPath := ""
	if p.fileOpTarget != nil {
		if p.fileOpTarget.IsDir {
			parentPath = p.fileOpTarget.Path
		} else if p.fileOpTarget.Parent != nil {
			parentPath = p.fileOpTarget.Parent.Path
		}
	}
	if parentPath == "" {
		parentPath = "/"
	}
	sb.WriteString(styles.Muted.Render("Parent: "))
	sb.WriteString(truncatePath(parentPath, modalWidth-12))
	sb.WriteString("\n\n")

	// Input field
	sb.WriteString("Name: ")
	sb.WriteString(p.fileOpTextInput.View())

	// Track line count: title + blank + parent + blank + name = 5
	lineCount := 5

	// Error message if present
	if p.fileOpError != "" {
		sb.WriteString("\n\n")
		sb.WriteString(styles.StatusDeleted.Render(p.fileOpError))
		lineCount += 2 // blank + error
	}

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Create ", " Cancel ", p.fileOpButtonFocus, p.fileOpButtonHover))
	lineCount += 2 // blank + buttons

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Create ", " Cancel ")

	return styles.ModalBox.Width(modalWidth).Render(sb.String())
}

// renderCreateDirConfirmModal renders the directory creation confirmation modal (during move).
func (p *Plugin) renderCreateDirConfirmModal() string {
	// Adaptive width based on path length
	modalWidth := len(p.fileOpConfirmPath) + 10
	if modalWidth < 40 {
		modalWidth = 40
	}
	if modalWidth > 70 {
		modalWidth = 70
	}

	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render("Create Directory?"))
	sb.WriteString("\n\n")

	// Message
	sb.WriteString("Directory does not exist:\n")
	sb.WriteString(truncatePath(p.fileOpConfirmPath, modalWidth-4))
	sb.WriteString("\n\n")
	sb.WriteString("Create it?")

	// Interactive buttons
	sb.WriteString("\n\n")
	sb.WriteString(ui.RenderButtonPair(" Yes ", " No ", p.fileOpButtonFocus, p.fileOpButtonHover))

	// Line count: title + blank + message(2 lines) + blank + question + blank + buttons = 8
	lineCount := 8

	// Register button hit regions
	p.registerFileOpButtonHitRegions(lineCount, modalWidth, " Yes ", " No ")

	return styles.ModalBox.Width(modalWidth).Render(sb.String())
}

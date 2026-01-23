package workspace

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/marcus/sidecar/internal/styles"
)

var ansiResetRe = regexp.MustCompile(`\x1b\[0?m`)

// expandTabs replaces tabs with spaces, preserving ANSI sequences and column widths.
func expandTabs(line string, tabWidth int) string {
	if tabWidth <= 0 || !strings.Contains(line, "\t") {
		return line
	}

	var sb strings.Builder
	sb.Grow(len(line))

	state := ansi.NormalState
	column := 0
	for len(line) > 0 {
		seq, width, n, newState := ansi.GraphemeWidth.DecodeSequenceInString(line, state, nil)
		if n <= 0 {
			sb.WriteString(line)
			break
		}
		if seq == "\t" && width == 0 {
			spaces := tabWidth - (column % tabWidth)
			if spaces == 0 {
				spaces = tabWidth
			}
			sb.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		} else {
			sb.WriteString(seq)
			column += width
		}
		state = newState
		line = line[n:]
	}

	return sb.String()
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var lines []string
	for _, para := range strings.Split(text, "\n") {
		if len(para) <= width {
			lines = append(lines, para)
			continue
		}

		// Simple word wrapping
		words := strings.Fields(para)
		var currentLine string
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}
	return strings.Join(lines, "\n")
}

// dimText renders dim placeholder text using theme style.
func dimText(s string) string {
	return styles.Muted.Render(s)
}

// formatRelativeTime formats a time as relative (e.g., "3m", "2h").
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// visualSubstring extracts a substring by visual column range [startCol, endCol).
// endCol is EXCLUSIVE (one past last included column).
// Handles ANSI escape codes (skipped in column counting).
// If endCol is -1, extracts to end of string.
// Returns plain text (ANSI stripped) for clipboard use.
func visualSubstring(s string, startCol, endCol int) string {
	if s == "" {
		return ""
	}

	var sb strings.Builder
	state := ansi.NormalState
	cumWidth := 0

	remaining := s
	for len(remaining) > 0 {
		seq, width, n, newState := ansi.GraphemeWidth.DecodeSequenceInString(remaining, state, nil)
		if n <= 0 {
			break
		}
		if width > 0 {
			charStart := cumWidth
			charEnd := cumWidth + width
			cumWidth = charEnd

			// Check if this character is within range
			inRange := false
			if endCol == -1 {
				inRange = charEnd > startCol
			} else {
				inRange = charStart < endCol && charEnd > startCol
			}
			if inRange {
				sb.WriteString(seq)
			}
			if endCol >= 0 && cumWidth >= endCol {
				break
			}
		}
		// Skip ANSI sequences (width == 0, not a visible character)
		state = newState
		remaining = remaining[n:]
	}

	return sb.String()
}

// injectCharacterRangeBackground applies selection background to visual columns
// [startCol, endCol] (inclusive) within the line. startCol and endCol are in
// absolute visual space (post-tab-expansion). Handles ANSI codes correctly.
// If endCol is -1, highlights to end of line.
func injectCharacterRangeBackground(line string, startCol, endCol int) string {
	if startCol == 0 && endCol == -1 {
		return injectSelectionBackground(line)
	}

	selBg := getSelectionBgANSI()
	var sb strings.Builder
	sb.Grow(len(line) + 64)

	state := ansi.NormalState
	cumWidth := 0
	inSelection := false

	remaining := line
	for len(remaining) > 0 {
		seq, width, n, newState := ansi.GraphemeWidth.DecodeSequenceInString(remaining, state, nil)
		if n <= 0 {
			sb.WriteString(remaining)
			break
		}

		if width > 0 {
			// Visible character
			charInRange := false
			if endCol == -1 {
				charInRange = cumWidth >= startCol
			} else {
				charInRange = cumWidth >= startCol && cumWidth <= endCol
			}

			if charInRange && !inSelection {
				sb.WriteString(selBg)
				inSelection = true
			} else if !charInRange && inSelection {
				sb.WriteString("\x1b[49m") // reset background only, preserve foreground
				inSelection = false
			}

			sb.WriteString(seq)
			cumWidth += width

			// Check if we've passed the end of selection
			if endCol >= 0 && cumWidth > endCol && inSelection {
				sb.WriteString("\x1b[49m") // reset background only, preserve foreground
				inSelection = false
			}
		} else {
			// ANSI sequence or control character
			sb.WriteString(seq)
			// If there's a reset within the selection, re-inject background
			if inSelection && ansiResetRe.MatchString(seq) {
				sb.WriteString(selBg)
			}
		}

		state = newState
		remaining = remaining[n:]
	}

	if inSelection {
		sb.WriteString("\x1b[49m") // reset background only at end of line
	}

	return sb.String()
}

func getSelectionBgANSI() string {
	theme := styles.GetCurrentTheme()
	hex := theme.Colors.BgTertiary
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		r, g, b = 55, 65, 81
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// injectSelectionBackground adds a selection background while preserving ANSI resets.
func injectSelectionBackground(s string) string {
	selectionBg := getSelectionBgANSI()
	result := selectionBg + s
	result = ansiResetRe.ReplaceAllString(result, "${0}"+selectionBg)
	return result + "\x1b[0m"
}

package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
)

// ConfirmDialog is a reusable confirmation modal with interactive buttons.
type ConfirmDialog struct {
	Title        string
	Message      string
	ConfirmLabel string         // e.g., " Confirm ", " Delete ", " Yes "
	CancelLabel  string         // e.g., " Cancel ", " No "
	BorderColor  lipgloss.Color // Modal border color
	Width        int            // Modal width (default 50)

	// State
	ButtonFocus int // 0=none, 1=confirm, 2=cancel
	ButtonHover int // 0=none, 1=confirm, 2=cancel
}

// NewConfirmDialog creates a dialog with sensible defaults.
func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		Title:        title,
		Message:      message,
		ConfirmLabel: " Confirm ",
		CancelLabel:  " Cancel ",
		BorderColor:  styles.Primary,
		Width:        ModalWidthMedium,
		ButtonFocus:  1, // Start with confirm focused
	}
}

// Render returns the modal content (without overlay - caller handles that).
func (d *ConfirmDialog) Render() string {
	var sb strings.Builder

	// Title
	sb.WriteString(styles.ModalTitle.Render(d.Title))
	sb.WriteString("\n\n")

	// Message
	sb.WriteString(d.Message)
	sb.WriteString("\n\n")

	// Buttons
	sb.WriteString(RenderButtonPair(d.ConfirmLabel, d.CancelLabel, d.ButtonFocus, d.ButtonHover))

	// Apply modal box styling
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.BorderColor).
		Padding(1, 2).
		Width(d.Width)

	return modalStyle.Render(sb.String())
}

// HandleKey processes keyboard input. Returns:
// - action: "confirm", "cancel", or "" (no action)
// - handled: whether the key was consumed
func (d *ConfirmDialog) HandleKey(key string) (action string, handled bool) {
	switch key {
	case "tab":
		// Cycle: 1 -> 2 -> 1
		if d.ButtonFocus == 1 {
			d.ButtonFocus = 2
		} else {
			d.ButtonFocus = 1
		}
		return "", true
	case "shift+tab":
		// Reverse cycle
		if d.ButtonFocus == 2 {
			d.ButtonFocus = 1
		} else {
			d.ButtonFocus = 2
		}
		return "", true
	case "enter":
		if d.ButtonFocus == 2 {
			return "cancel", true
		}
		return "confirm", true // Default to confirm
	case "y", "Y":
		return "confirm", true
	case "esc", "n", "N", "q":
		return "cancel", true
	}
	return "", false
}

// SetHover updates hover state. Index: 0=none, 1=confirm, 2=cancel
func (d *ConfirmDialog) SetHover(index int) {
	d.ButtonHover = index
}

// Reset resets the dialog state for reuse.
func (d *ConfirmDialog) Reset() {
	d.ButtonFocus = 1
	d.ButtonHover = 0
}

// ContentLineCount returns number of lines in content (for hit region calculation).
func (d *ConfirmDialog) ContentLineCount() int {
	messageLines := strings.Count(d.Message, "\n") + 1
	return 1 + 1 + messageLines + 1 + 1 // title + blank + message + blank + buttons
}

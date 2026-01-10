package gitstatus

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// renderPushMenu renders the push options popup menu.
func (p *Plugin) renderPushMenu() string {
	// Render the background (current view dimmed)
	background := p.renderThreePaneView()

	// Build menu content
	var sb strings.Builder

	// Title
	sb.WriteString(styles.Title.Render(" Push "))
	sb.WriteString("\n\n")

	// Menu options with focus/hover support
	options := []struct{ key, label string }{
		{"p", " Push to origin "},
		{"f", " Force push (--force-with-lease) "},
		{"u", " Push & set upstream (-u) "},
	}

	for i, opt := range options {
		// Determine style based on focus/hover
		style := ui.ResolveButtonStyle(p.pushMenuFocus, p.pushMenuHover, i)
		keyHint := styles.KeyHint.Render(" " + opt.key + " ")
		sb.WriteString(keyHint)
		sb.WriteString(" ")
		sb.WriteString(style.Render(opt.label))
		if i < len(options)-1 {
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(styles.Muted.Render("Tab/↑↓ to navigate, Enter to select, Esc to cancel"))

	// Create menu box
	menuWidth := ui.ModalWidthMedium

	menuContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(menuWidth).
		Render(sb.String())

	// Overlay menu on dimmed background
	return ui.OverlayModal(background, menuContent, p.width, p.height)
}

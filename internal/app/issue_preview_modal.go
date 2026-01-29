package app

import (
	"fmt"
	"strings"

	"github.com/marcus/sidecar/internal/modal"
	"github.com/marcus/sidecar/internal/mouse"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

func (m *Model) renderIssueInputOverlay(content string) string {
	m.ensureIssueInputModal()
	if m.issueInputModal == nil {
		return content
	}
	if m.issueInputMouseHandler == nil {
		m.issueInputMouseHandler = mouse.NewHandler()
	}
	rendered := m.issueInputModal.Render(m.width, m.height, m.issueInputMouseHandler)
	return ui.OverlayModal(content, rendered, m.width, m.height)
}

// issueSearchResultPrefix is the hit-region ID prefix for clickable search results.
const issueSearchResultPrefix = "issue-search-"

func (m *Model) ensureIssueInputModal() {
	modalW := 60
	if modalW > m.width-4 {
		modalW = m.width - 4
	}
	if modalW < 20 {
		modalW = 20
	}
	if m.issueInputModal != nil && m.issueInputModalWidth == modalW {
		return
	}
	m.issueInputModalWidth = modalW
	b := modal.New("Open Issue",
		modal.WithWidth(modalW),
		modal.WithHints(false),
	).
		AddSection(modal.Input("issue-id", &m.issueInputInput))

	// Status line — always present to avoid layout jumps
	if m.issueSearchLoading {
		b = b.AddSection(modal.Text(styles.Muted.Render("Searching...")))
	} else {
		b = b.AddSection(modal.Text(styles.Muted.Render(" ")))
	}

	// Search results dropdown — reserve minResultLines to reduce jumpiness
	const minResultLines = 5
	if len(m.issueSearchResults) > 0 {
		searchResults := m.issueSearchResults
		searchCursor := m.issueSearchCursor
		b = b.AddSection(modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
			var sb strings.Builder
			focusables := make([]modal.FocusableInfo, 0, len(searchResults))
			displayed := len(searchResults)
			if displayed > 10 {
				displayed = 10
			}
			for i := 0; i < displayed; i++ {
				r := searchResults[i]
				line := fmt.Sprintf("  %s  %s", r.ID, r.Title)
				if len(line) > contentWidth-2 {
					line = line[:contentWidth-5] + "..."
				}
				itemID := fmt.Sprintf("%s%d", issueSearchResultPrefix, i)
				isHovered := itemID == hoverID
				if i == searchCursor || isHovered {
					sb.WriteString(styles.ListItemSelected.Render(line))
				} else {
					sb.WriteString(styles.ListItemNormal.Render(line))
				}
				if i < displayed-1 {
					sb.WriteString("\n")
				}
				focusables = append(focusables, modal.FocusableInfo{
					ID:      itemID,
					OffsetX: 0,
					OffsetY: i,
					Width:   contentWidth,
					Height:  1,
				})
			}
			// Pad with empty lines to maintain minimum height
			for i := displayed; i < minResultLines; i++ {
				sb.WriteString("\n")
			}
			return modal.RenderedSection{Content: sb.String(), Focusables: focusables}
		}, nil))
	} else {
		// Reserve space for results even when empty
		b = b.AddSection(modal.Custom(func(contentWidth int, _, _ string) modal.RenderedSection {
			var sb strings.Builder
			for i := 0; i < minResultLines; i++ {
				if i > 0 {
					sb.WriteString("\n")
				}
			}
			return modal.RenderedSection{Content: sb.String()}
		}, nil))
	}

	// Buttons
	b = b.AddSection(modal.Spacer())
	b = b.AddSection(modal.Buttons(
		modal.Btn(" Open ", "open", modal.BtnPrimary()),
		modal.Btn(" Cancel ", "cancel"),
	))

	// Hint line
	hasResults := len(m.issueSearchResults) > 0
	b = b.AddSection(modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(styles.KeyHint.Render("enter"))
		sb.WriteString(styles.Muted.Render(" open  "))
		if hasResults {
			sb.WriteString(styles.KeyHint.Render("↑↓"))
			sb.WriteString(styles.Muted.Render(" select  "))
			sb.WriteString(styles.KeyHint.Render("tab"))
			sb.WriteString(styles.Muted.Render(" fill  "))
		}
		sb.WriteString(styles.KeyHint.Render("esc"))
		sb.WriteString(styles.Muted.Render(" cancel"))
		return modal.RenderedSection{Content: sb.String()}
	}, nil))

	m.issueInputModal = b
}

func (m *Model) renderIssuePreviewOverlay(content string) string {
	m.ensureIssuePreviewModal()
	if m.issuePreviewModal == nil {
		return content
	}
	if m.issuePreviewMouseHandler == nil {
		m.issuePreviewMouseHandler = mouse.NewHandler()
	}
	rendered := m.issuePreviewModal.Render(m.width, m.height, m.issuePreviewMouseHandler)
	return ui.OverlayModal(content, rendered, m.width, m.height)
}

func (m *Model) ensureIssuePreviewModal() {
	// Use 80% of terminal width so the issue is comfortable to read
	modalW := m.width * 4 / 5
	if modalW > m.width-4 {
		modalW = m.width - 4
	}
	if modalW < 30 {
		modalW = 30
	}

	// Cache check -- also invalidate when data/error/loading changes
	cacheKey := modalW
	if m.issuePreviewModal != nil && m.issuePreviewModalWidth == cacheKey {
		return
	}
	m.issuePreviewModalWidth = cacheKey

	if m.issuePreviewLoading {
		m.issuePreviewModal = modal.New("Loading...",
			modal.WithWidth(modalW),
			modal.WithHints(false),
		).
			AddSection(modal.Text("Fetching issue data..."))
		return
	}

	if m.issuePreviewError != nil {
		m.issuePreviewModal = modal.New("Error",
			modal.WithWidth(modalW),
			modal.WithVariant(modal.VariantDanger),
			modal.WithHints(false),
		).
			AddSection(modal.Text(m.issuePreviewError.Error())).
			AddSection(modal.Spacer()).
			AddSection(modal.Buttons(
				modal.Btn(" Close ", "cancel"),
			))
		return
	}

	if m.issuePreviewData == nil {
		m.issuePreviewModal = nil
		return
	}

	data := m.issuePreviewData

	// Build title
	title := data.ID
	if data.Title != "" {
		title += ": " + data.Title
	}

	// Build status line
	var metaParts []string
	if data.Status != "" {
		metaParts = append(metaParts, "["+data.Status+"]")
	}
	if data.Type != "" {
		metaParts = append(metaParts, data.Type)
	}
	if data.Priority != "" {
		metaParts = append(metaParts, data.Priority)
	}
	if data.Points > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%dp", data.Points))
	}
	statusLine := strings.Join(metaParts, "  ")

	// Build modal
	b := modal.New(title,
		modal.WithWidth(modalW),
		modal.WithHints(false),
	)

	if statusLine != "" {
		b = b.AddSection(modal.Text(statusLine))
	}

	if data.ParentID != "" {
		b = b.AddSection(modal.Text("Parent: " + data.ParentID))
	}

	if len(data.Labels) > 0 {
		b = b.AddSection(modal.Text("Labels: " + strings.Join(data.Labels, ", ")))
	}

	// Description — scale visible lines with terminal height
	// Reserve ~10 lines for title, meta, buttons, hints, borders
	if data.Description != "" {
		b = b.AddSection(modal.Spacer())
		maxDescLines := m.height - 10
		if maxDescLines < 5 {
			maxDescLines = 5
		}
		desc := data.Description
		lines := strings.Split(desc, "\n")
		if len(lines) > maxDescLines {
			lines = lines[:maxDescLines]
			lines = append(lines, "...")
		}
		b = b.AddSection(modal.Text(strings.Join(lines, "\n")))
	}

	b = b.AddSection(modal.Spacer())
	b = b.AddSection(modal.Buttons(
		modal.Btn(" Open in TD ", "open-in-td", modal.BtnPrimary()),
		modal.Btn(" Back ", "back"),
		modal.Btn(" Close ", "cancel"),
	))

	// Hint line
	b = b.AddSection(modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(styles.KeyHint.Render("o"))
		sb.WriteString(styles.Muted.Render(" open  "))
		sb.WriteString(styles.KeyHint.Render("b"))
		sb.WriteString(styles.Muted.Render(" back  "))
		sb.WriteString(styles.KeyHint.Render("y"))
		sb.WriteString(styles.Muted.Render(" yank  "))
		sb.WriteString(styles.KeyHint.Render("Y"))
		sb.WriteString(styles.Muted.Render(" yank key  "))
		sb.WriteString(styles.KeyHint.Render("esc"))
		sb.WriteString(styles.Muted.Render(" close"))
		return modal.RenderedSection{Content: sb.String()}
	}, nil))

	m.issuePreviewModal = b
}

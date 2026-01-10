package ui

import (
	"strings"
	"testing"
)

func TestNewConfirmDialog(t *testing.T) {
	d := NewConfirmDialog("Test Title", "Test message")

	if d.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", d.Title)
	}
	if d.Message != "Test message" {
		t.Errorf("expected message 'Test message', got %q", d.Message)
	}
	if d.ConfirmLabel != " Confirm " {
		t.Errorf("expected default confirm label ' Confirm ', got %q", d.ConfirmLabel)
	}
	if d.CancelLabel != " Cancel " {
		t.Errorf("expected default cancel label ' Cancel ', got %q", d.CancelLabel)
	}
	if d.ButtonFocus != 1 {
		t.Errorf("expected initial focus on confirm (1), got %d", d.ButtonFocus)
	}
	if d.Width != ModalWidthMedium {
		t.Errorf("expected width %d, got %d", ModalWidthMedium, d.Width)
	}
}

func TestConfirmDialog_Render(t *testing.T) {
	d := NewConfirmDialog("Delete File?", "Are you sure?")
	d.ConfirmLabel = " Delete "
	d.CancelLabel = " Cancel "

	output := d.Render()

	if !strings.Contains(output, "Delete File?") {
		t.Error("render should contain title")
	}
	if !strings.Contains(output, "Are you sure?") {
		t.Error("render should contain message")
	}
	if !strings.Contains(output, "Delete") {
		t.Error("render should contain confirm label")
	}
	if !strings.Contains(output, "Cancel") {
		t.Error("render should contain cancel label")
	}
}

func TestConfirmDialog_HandleKey_Tab(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")

	// Start at 1 (confirm)
	if d.ButtonFocus != 1 {
		t.Fatalf("expected initial focus 1, got %d", d.ButtonFocus)
	}

	// Tab should cycle to 2
	action, handled := d.HandleKey("tab")
	if !handled {
		t.Error("tab should be handled")
	}
	if action != "" {
		t.Errorf("tab should not return action, got %q", action)
	}
	if d.ButtonFocus != 2 {
		t.Errorf("expected focus 2 after tab, got %d", d.ButtonFocus)
	}

	// Tab again should cycle back to 1
	d.HandleKey("tab")
	if d.ButtonFocus != 1 {
		t.Errorf("expected focus 1 after second tab, got %d", d.ButtonFocus)
	}
}

func TestConfirmDialog_HandleKey_ShiftTab(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")
	d.ButtonFocus = 2

	// Shift+tab should cycle to 1
	action, handled := d.HandleKey("shift+tab")
	if !handled {
		t.Error("shift+tab should be handled")
	}
	if action != "" {
		t.Errorf("shift+tab should not return action, got %q", action)
	}
	if d.ButtonFocus != 1 {
		t.Errorf("expected focus 1 after shift+tab, got %d", d.ButtonFocus)
	}
}

func TestConfirmDialog_HandleKey_Enter(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")

	// Enter with focus on confirm
	d.ButtonFocus = 1
	action, handled := d.HandleKey("enter")
	if !handled {
		t.Error("enter should be handled")
	}
	if action != "confirm" {
		t.Errorf("expected 'confirm' action, got %q", action)
	}

	// Enter with focus on cancel
	d.ButtonFocus = 2
	action, handled = d.HandleKey("enter")
	if !handled {
		t.Error("enter should be handled")
	}
	if action != "cancel" {
		t.Errorf("expected 'cancel' action, got %q", action)
	}
}

func TestConfirmDialog_HandleKey_Shortcuts(t *testing.T) {
	tests := []struct {
		key            string
		expectedAction string
	}{
		{"y", "confirm"},
		{"Y", "confirm"},
		{"esc", "cancel"},
		{"n", "cancel"},
		{"N", "cancel"},
		{"q", "cancel"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			d := NewConfirmDialog("Test", "Message")
			action, handled := d.HandleKey(tt.key)
			if !handled {
				t.Errorf("%s should be handled", tt.key)
			}
			if action != tt.expectedAction {
				t.Errorf("expected %q action for %s, got %q", tt.expectedAction, tt.key, action)
			}
		})
	}
}

func TestConfirmDialog_HandleKey_Unhandled(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")

	action, handled := d.HandleKey("x")
	if handled {
		t.Error("'x' should not be handled")
	}
	if action != "" {
		t.Errorf("unhandled key should return empty action, got %q", action)
	}
}

func TestConfirmDialog_SetHover(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")

	d.SetHover(1)
	if d.ButtonHover != 1 {
		t.Errorf("expected hover 1, got %d", d.ButtonHover)
	}

	d.SetHover(2)
	if d.ButtonHover != 2 {
		t.Errorf("expected hover 2, got %d", d.ButtonHover)
	}
}

func TestConfirmDialog_Reset(t *testing.T) {
	d := NewConfirmDialog("Test", "Message")
	d.ButtonFocus = 2
	d.ButtonHover = 1

	d.Reset()

	if d.ButtonFocus != 1 {
		t.Errorf("expected focus reset to 1, got %d", d.ButtonFocus)
	}
	if d.ButtonHover != 0 {
		t.Errorf("expected hover reset to 0, got %d", d.ButtonHover)
	}
}

func TestConfirmDialog_ContentLineCount(t *testing.T) {
	tests := []struct {
		message  string
		expected int
	}{
		{"Single line", 5},           // title + blank + 1 line + blank + buttons
		{"Line 1\nLine 2", 6},        // title + blank + 2 lines + blank + buttons
		{"A\nB\nC\nD", 8},            // title + blank + 4 lines + blank + buttons
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			d := NewConfirmDialog("Title", tt.message)
			got := d.ContentLineCount()
			if got != tt.expected {
				t.Errorf("expected %d lines, got %d", tt.expected, got)
			}
		})
	}
}

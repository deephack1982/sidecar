package tdmonitor

import "time"

// Issue represents a task/issue from the TD database.
type Issue struct {
	ID          string
	Title       string
	Description string
	Status      string // open, in_progress, in_review, done
	Type        string // bug, feature, task, epic
	Priority    string // P0, P1, P2, P3, P4
	Labels      string
	ParentID    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Log represents a log entry for an issue.
type Log struct {
	ID        int64
	IssueID   string
	SessionID string
	Message   string
	Type      string
	Timestamp time.Time
}

// Session represents the current session info.
type Session struct {
	ID        string
	ContextID string
	Name      string
	StartedAt time.Time
}

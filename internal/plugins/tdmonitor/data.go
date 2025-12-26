package tdmonitor

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DataProvider handles read-only access to the TD SQLite database.
type DataProvider struct {
	dbPath string
	db     *sql.DB
}

// NewDataProvider creates a new data provider for the given project directory.
func NewDataProvider(workDir string) *DataProvider {
	return &DataProvider{
		dbPath: filepath.Join(workDir, ".todos", "issues.db"),
	}
}

// Open opens the database connection.
func (d *DataProvider) Open() error {
	// Check if database exists
	if _, err := os.Stat(d.dbPath); os.IsNotExist(err) {
		return err
	}

	db, err := sql.Open("sqlite", d.dbPath+"?mode=ro")
	if err != nil {
		return err
	}

	d.db = db
	return nil
}

// Close closes the database connection.
func (d *DataProvider) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// CurrentSession returns the current session info.
func (d *DataProvider) CurrentSession() (*Session, error) {
	if d.db == nil {
		return nil, nil
	}

	row := d.db.QueryRow(`
		SELECT id, context_id, COALESCE(name, ''), started_at
		FROM sessions
		WHERE ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`)

	var s Session
	err := row.Scan(&s.ID, &s.ContextID, &s.Name, &s.StartedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// InProgressIssues returns all issues with status in_progress.
func (d *DataProvider) InProgressIssues() ([]Issue, error) {
	return d.issuesByStatus("in_progress")
}

// ReadyIssues returns all open issues that are not blocked.
func (d *DataProvider) ReadyIssues(limit int) ([]Issue, error) {
	if d.db == nil {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, title, COALESCE(description, ''), status, type, priority,
		       COALESCE(labels, ''), COALESCE(parent_id, ''), created_at, updated_at
		FROM issues
		WHERE status = 'open'
		  AND deleted_at IS NULL
		  AND id NOT IN (
		      SELECT issue_id FROM issue_dependencies
		      WHERE depends_on_id IN (SELECT id FROM issues WHERE status != 'done' AND deleted_at IS NULL)
		  )
		ORDER BY
		  CASE priority
		    WHEN 'P0' THEN 0
		    WHEN 'P1' THEN 1
		    WHEN 'P2' THEN 2
		    WHEN 'P3' THEN 3
		    WHEN 'P4' THEN 4
		    ELSE 5
		  END,
		  updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

// ReviewableIssues returns all issues with status in_review.
func (d *DataProvider) ReviewableIssues() ([]Issue, error) {
	return d.issuesByStatus("in_review")
}

// RecentLogs returns the most recent log entries.
func (d *DataProvider) RecentLogs(limit int) ([]Log, error) {
	if d.db == nil {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, COALESCE(issue_id, ''), session_id, message, type, timestamp
		FROM logs
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.IssueID, &l.SessionID, &l.Message, &l.Type, &l.Timestamp); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	return logs, rows.Err()
}

// IssueByID returns a single issue by ID.
func (d *DataProvider) IssueByID(id string) (*Issue, error) {
	if d.db == nil {
		return nil, nil
	}

	row := d.db.QueryRow(`
		SELECT id, title, COALESCE(description, ''), status, type, priority,
		       COALESCE(labels, ''), COALESCE(parent_id, ''), created_at, updated_at
		FROM issues
		WHERE id = ? AND deleted_at IS NULL
	`, id)

	var i Issue
	err := row.Scan(&i.ID, &i.Title, &i.Description, &i.Status, &i.Type, &i.Priority,
		&i.Labels, &i.ParentID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &i, nil
}

// issuesByStatus returns issues with the given status.
func (d *DataProvider) issuesByStatus(status string) ([]Issue, error) {
	if d.db == nil {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, title, COALESCE(description, ''), status, type, priority,
		       COALESCE(labels, ''), COALESCE(parent_id, ''), created_at, updated_at
		FROM issues
		WHERE status = ? AND deleted_at IS NULL
		ORDER BY
		  CASE priority
		    WHEN 'P0' THEN 0
		    WHEN 'P1' THEN 1
		    WHEN 'P2' THEN 2
		    WHEN 'P3' THEN 3
		    WHEN 'P4' THEN 4
		    ELSE 5
		  END,
		  updated_at DESC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssues(rows)
}

// scanIssues scans rows into Issue slice.
func scanIssues(rows *sql.Rows) ([]Issue, error) {
	var issues []Issue
	for rows.Next() {
		var i Issue
		if err := rows.Scan(&i.ID, &i.Title, &i.Description, &i.Status, &i.Type, &i.Priority,
			&i.Labels, &i.ParentID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			continue
		}
		issues = append(issues, i)
	}
	return issues, rows.Err()
}

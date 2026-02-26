// Package store provides a SQLite-backed implementation of todo.Repository.
package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/MihkelHunter/mkToDo/internal/todo"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO required
)

// initialSchema sets up the base tables on a fresh database.
const initialSchema = `
CREATE TABLE IF NOT EXISTS tasks (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	title       TEXT    NOT NULL,
	description TEXT    NOT NULL DEFAULT '',
	done        INTEGER NOT NULL DEFAULT 0,
	priority    INTEGER NOT NULL DEFAULT 1,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER NOT NULL
);

INSERT INTO schema_version (version)
SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM schema_version);`

// migrations is an ordered list of SQL statements to apply incrementally.
// To add a schema change in the future, append a new entry here.
// Each entry is applied exactly once per database and never re-run.
//
// Example future migrations:
//
//	"ALTER TABLE tasks ADD COLUMN due_date TEXT;",
//	"ALTER TABLE tasks ADD COLUMN tags TEXT NOT NULL DEFAULT '';",
var migrations = []string{
	// v1 placeholder — add real migrations below this line as needed.
	`ALTER TABLE tasks ADD COLUMN closed_at TEXT`,
}

// SQLiteStore implements todo.Repository using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path,
// runs the initial schema, then applies any pending migrations.
func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec(initialSchema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// migrate applies any migrations that haven't been run yet.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("run migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, i+1); err != nil {
			tx.Rollback()
			return fmt.Errorf("update schema version: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}

	return nil
}

func (s *SQLiteStore) Create(t *todo.Task) error {
	res, err := s.db.Exec(
		`INSERT INTO tasks (title, description, done, priority, created_at, updated_at, closed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.Title, t.Description, boolToInt(t.Done), t.Priority,
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339), nil,
	)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) List() ([]*todo.Task, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, done, priority, created_at, updated_at, closed_at
		 FROM tasks ORDER BY done ASC, priority DESC, created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*todo.Task
	for rows.Next() {
		t := &todo.Task{}
		var done, priority int
		var createdAt, updatedAt string
		var closedAt sql.NullString

		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &done, &priority, &createdAt, &updatedAt, &closedAt); err != nil {
			return nil, err
		}

		t.Done = done != 0
		t.Priority = todo.Priority(priority)
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if closedAt.Valid {
			ts, err := time.Parse(time.RFC3339, closedAt.String)
			if err == nil {
				t.ClosedAt = &ts
			}
		} else {
			t.ClosedAt = nil
		}

		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *SQLiteStore) Update(t *todo.Task) error {
	now := time.Now()

	var closedAt interface{}

	if t.Done {
		// If marking done and not already closed, set timestamp
		if t.ClosedAt == nil {
			ts := now
			t.ClosedAt = &ts
		}
		closedAt = t.ClosedAt.Format(time.RFC3339)
	} else {
		// If marking undone, clear closed_at
		t.ClosedAt = nil
		closedAt = nil
	}

	_, err := s.db.Exec(
		`UPDATE tasks SET title=?, description=?, done=?, priority=?, updated_at=?, closed_at=? WHERE id=?`,
		t.Title, t.Description, boolToInt(t.Done), t.Priority,
		t.UpdatedAt.Format(time.RFC3339), closedAt, t.ID,
	)
	return err
}

func (s *SQLiteStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

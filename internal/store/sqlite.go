// Package store provides a SQLite-backed implementation of todo.Repository.
package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/todoapp/internal/todo"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO required
)

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	title       TEXT    NOT NULL,
	description TEXT    NOT NULL DEFAULT '',
	done        INTEGER NOT NULL DEFAULT 0,
	priority    INTEGER NOT NULL DEFAULT 1,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL
);`

// SQLiteStore implements todo.Repository using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path and returns a Store.
func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Create(t *todo.Task) error {
	res, err := s.db.Exec(
		`INSERT INTO tasks (title, description, done, priority, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.Title, t.Description, boolToInt(t.Done), t.Priority,
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) List() ([]*todo.Task, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, done, priority, created_at, updated_at
		 FROM tasks ORDER BY done ASC, priority DESC, created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*todo.Task
	for rows.Next() {
		t := &todo.Task{}
		var done int
		var priority int
		var createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &done, &priority, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		t.Done = done != 0
		t.Priority = todo.Priority(priority)
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *SQLiteStore) Update(t *todo.Task) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET title=?, description=?, done=?, priority=?, updated_at=? WHERE id=?`,
		t.Title, t.Description, boolToInt(t.Done), t.Priority,
		t.UpdatedAt.Format(time.RFC3339), t.ID,
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
